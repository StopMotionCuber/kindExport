package scrape

type scraper interface {
	Scrape() error
}
