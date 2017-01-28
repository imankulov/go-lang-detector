package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	pb "gopkg.in/cheggaaa/pb.v1"

	"github.com/artyom/autoflags"
	"github.com/imankulov/go-lang-detector/langdet"
)

// Doc is a document element
type Doc struct {
	Abstract string `xml:"abstract"`
}

var help = `
langdet command is used to load language statistics from Wikipedia abstracts

Usage example to load first 10k definitions for English language, and to
store them in an en.json file:

langdet -url https://dumps.wikimedia.org/enwiki/20170120/enwiki-20170120-abstract.xml -lang en -file en.json -limit 10000
`

func main() {
	config := struct {
		URL   string `flag:"url,URL with wikipedia abstract pages"`
		Lang  string `flag:"lang,Language to parse"`
		File  string `flag:"file,Output filename"`
		Depth int    `flag:"depth,Occurence map depth"`
		Limit int    `flag:"limit,Maximum number of abstracts to process"`
		Help  bool   `flag:"help,This help"`
	}{
		Depth: 3,
		Limit: 20000,
	}
	autoflags.Define(&config)
	flag.Parse()

	if config.Help {
		fmt.Println(help)
		return
	}

	// validate parameters
	if config.URL == "" {
		log.Fatalf("-url is a required argument\n%s", help)
	}
	if config.Lang == "" {
		log.Fatalf("-lang is a required argument\n%s", help)
	}
	if config.File == "" {
		log.Fatalf("-file is a required argument\n%s", help)
	}

	// Create lang structure
	occurenceMap := make(map[string]int)

	// download and parse Wikipedia article
	resp, err := http.Get(config.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	decoder := xml.NewDecoder(resp.Body)
	bar := pb.StartNew(config.Limit)
	for processed := 0; processed < config.Limit; {
		t, _ := decoder.Token()
		if t == nil {
			break
		}

		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "doc" {
				var d Doc
				err = decoder.DecodeElement(&d, &se)
				if err != nil {
					log.Fatal(err)
				}
				// for every abstract record, update occurrence map
				langdet.UpdateOccurenceMap(occurenceMap, d.Abstract, config.Depth)
				processed++
				bar.Increment()
			}
		}
	}

	// bulid a language object
	ranked := langdet.CreateRankLookupMap(occurenceMap)
	lang := langdet.Language{Name: config.Lang, Profile: ranked}

	// save it to the file
	langJSON, err := json.Marshal(lang)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(config.File, langJSON, 0644)
	if err != nil {
		log.Fatal(err)
	}

	bar.FinishPrint("Languge processing is done")

}
