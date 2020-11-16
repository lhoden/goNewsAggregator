package main

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/thedevsaddam/renderer"
)

/* This Go web application uses Go routines, channels, and synchronization
in order to maximize performance */

var wg sync.WaitGroup
var rnd *renderer.Render

type SitemapIndex struct {
	Locations []string `xml:"sitemap>loc"`
}

type News struct {
	Locations  []string `xml:"url>loc"`
	LastMod    []string `xml:"url>lastmod"`
	ChangeFreq []string `xml:"url>changefreq"`
}

type NewsMap struct {
	LastMod  string
	Location string
}

type NewsAggPage struct {
	Header string
	News   map[string]NewsMap
}

// Go routine function
func newsRoutine(c chan News, Location string) {
	defer wg.Done()
	var n News
	resp, _ := http.Get(strings.TrimSpace(Location))
	bytes, _ := ioutil.ReadAll(resp.Body)
	xml.Unmarshal(bytes, &n)
	resp.Body.Close()
	var size int = len(n.Locations)
	if size >= 1 {
		size = size - 1
	}

	c <- n // sending the value of n over to the channel

}

func newsAggHandler(w http.ResponseWriter, r *http.Request) {
	//Variables used to parse the XML returned from the HTTP request
	var s SitemapIndex

	//Making HTTP request
	resp, _ := http.Get("https://www.washingtonpost.com/sitemaps/index.xml")
	bytes, _ := ioutil.ReadAll(resp.Body)
	xml.Unmarshal(bytes, &s)
	newsMap := make(map[string]NewsMap)
	resp.Body.Close()
	queue := make(chan News, 30)

	for _, Location := range s.Locations {
		wg.Add(1)
		go newsRoutine(queue, Location)

	}

	wg.Wait()
	close(queue)

	for elem := range queue {
		for idx, _ := range elem.LastMod {
			newsMap[elem.Locations[idx]] = NewsMap{elem.LastMod[idx][0:10], elem.Locations[idx]}
		}
	}

	p := NewsAggPage{Header: "News Aggregator", News: newsMap}
	rnd.HTML(w, http.StatusOK, "agg", p)
}

func main() {

	http.HandleFunc("/", newsAggHandler)
	http.ListenAndServe(":4000", nil)
}

func init() {
	opts := renderer.Options{
		ParseGlobPattern: "github.com/lhoden/goNewsAggregator/tpl/*.html",
	}

	rnd = renderer.New(opts)
}
