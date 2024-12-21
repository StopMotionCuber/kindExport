package main

import (
	"kindExport/internal/scrape"
)

func main() {
	err := scrape.SubstackScraper{}.Scrape("https://newsletter.pragmaticengineer.com/p/leaving-big-tech")
	if err != nil {
		print(err.Error())
		return
	}
}
