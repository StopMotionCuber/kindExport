package scrape

type scraper interface {
	Scrape(targetUrl string) error
}
