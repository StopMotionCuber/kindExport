package scrape

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-epub"
	"github.com/gocolly/colly/v2"
	"strings"
)

type SubstackScraper struct {
	Book *epub.Epub
}

func NewSubstackScraper() SubstackScraper {
	return SubstackScraper{
		Book: &epub.Epub{},
	}
}

func (s SubstackScraper) Scrape(url string) error {

	// We want to scrape the URL and create an EPUB file from it
	// First, we get the HTML content of the URL

	book, err := epub.NewEpub("Placeholder")
	s.Book = book

	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.OnHTML(".post-title", func(e *colly.HTMLElement) {
		fmt.Println(e.Text)
		s.Book.SetTitle(e.Text)
	})

	c.OnHTML("meta[name=\"author\"]", func(e *colly.HTMLElement) {
		fmt.Println(e.Attr("content"))
		if e.Attr("content") != "Substack" {
			s.Book.SetAuthor(e.Attr("content"))
		}
	})

	c.OnHTML(".available-content", func(e *colly.HTMLElement) {
		// This should be parsed into the book
		// Replace all <source> tags content with raw <source> tag
		e.DOM.Find("source").Each(func(i int, selection *goquery.Selection) {
			selection.ReplaceWithHtml(selection.Text())
		})

		// Delete all elements with class .pencraft
		e.DOM.Find(".pencraft").Each(func(i int, selection *goquery.Selection) {
			selection.Remove()
		})

		e.DOM.Find("img").Each(func(i int, selection *goquery.Selection) {
			imgSrc, _ := selection.Attr("src")
			// Add image to epub
			image, err := book.AddImage(imgSrc, "")
			if err != nil {
				return
			}
			selection.ReplaceWithHtml(fmt.Sprintf("<img src=\"%s\" alt=\"placeholder\"/>", image))
			selection.SetAttr("src", image)
			// Remove all attributes except src and alt
		})
		content, err := e.DOM.Html()
		if err != nil {
			return
		}
		content = fmt.Sprintf("<h1>%s</h1>By %s\n<hr/>%s", s.Book.Title(), s.Book.Author(), content)
		// Lowercase book title

		_, err = s.Book.AddSection(content, s.Book.Title(), fmt.Sprintf("%s.xhtml", strings.ReplaceAll(strings.ToLower(s.Book.Title()), " ", "-")), "")
		if err != nil {
			return
		}
	})

	err = c.Visit(url)
	if err != nil {
		return err
	}

	// Next, we parse the HTML content
	// We are searching for a few elements in the HTML content
	// - Title
	// - Author
	// - Content
	// - Title image

	// We can now create an EPUB from the parsed HTML content
	err = s.Book.Write(fmt.Sprintf("output/%s.epub", s.Book.Title()))

	if err != nil {
		return err
	}

	return nil
}
