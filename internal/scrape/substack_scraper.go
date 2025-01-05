package scrape

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-epub"
	"github.com/gocolly/colly/v2"
	"github.com/google/uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

type Book struct {
	Book      *epub.Epub
	Path      *string
	Permalink *string
	Paid      bool
}

type ArticleEmbeddedImage struct {
	URL string `json:"url"`
}

type ArticleEmbeddedAuthor struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type ArticleSchema struct {
	Title       string                  `json:"headline"`
	URL         string                  `json:"url"`
	Description string                  `json:"description"`
	Image       []ArticleEmbeddedImage  `json:"image"`
	Published   string                  `json:"datePublished"`
	Free        bool                    `json:"isAccessibleForFree"`
	Author      []ArticleEmbeddedAuthor `json:"author"`
	Publisher   ArticleEmbeddedAuthor   `json:"publisher"`
}

type SubstackScraper struct {
	SubstackLoginCookie *string
}

func generateUUID(filename string) string {
	imgUUID := uuid.New().String()
	// Get file ending from src
	if strings.Contains(filename, ".") {
		splitted := strings.Split(filename, ".")
		ending := splitted[len(splitted)-1]
		imgUUID = fmt.Sprintf("%s.%s", imgUUID, ending)
	} else {
		imgUUID = fmt.Sprintf("%s.jpg", imgUUID)
	}
	return imgUUID
}

func (s SubstackScraper) setCookies(c *colly.Collector, targetUrl string) {
	if s.SubstackLoginCookie == nil {
		return
	}
	val, _ := url.Parse(targetUrl)
	if val != nil {
		c.SetCookies(val.Scheme+"://"+val.Host, []*http.Cookie{{
			Name:  "connect.sid",
			Value: *s.SubstackLoginCookie,
		}})
	}
}

func normalizeStr(str string) string {
	// We want to have a lowercase string with space replaced by - and all special characters removed
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	str, _, _ = transform.String(t, str)
	str = strings.ToLower(str)
	str = strings.ReplaceAll(str, " ", "-")
	str = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= '0' && r <= '9' {
			return r
		}
		if r == '-' || r == '_' {
			return r
		}
		return -1
	}, str)
	return str
}

func (s SubstackScraper) CheckPaywallAccessible(targetUrl string) (bool, error) {
	// We want to scrape the URL and check for paywall newsletters
	// whether they are accessible by the scraper or not by checking for the paywall-title class

	c := colly.NewCollector()
	s.setCookies(c, targetUrl)

	c.OnRequest(func(r *colly.Request) {
		_ = c
		fmt.Println("Visiting", r.URL)
	})

	paywallAccessible := true

	c.OnHTML(".paywall-title", func(e *colly.HTMLElement) {
		if e.Text == "This post is for paid subscribers" {
			paywallAccessible = false
		}
	})

	err := c.Visit(targetUrl)
	if err != nil {
		return false, err
	}

	return paywallAccessible, nil
}

func (s SubstackScraper) Scrape(url *string) (*Book, error) {

	// We want to scrape the URL and create an EPUB file from it
	// First, we get the HTML content of the URL

	book, err := epub.NewEpub("Placeholder")
	article := ArticleSchema{}

	permalink := *url

	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	sectionFound := false

	// Next, we parse the HTML content
	// We are searching for a few elements in the HTML content
	// - Title
	// - Author
	// - Content
	// - Title image
	// - Description (optional)
	// - Permalink (optional)
	// - Release date
	// - Paid (optional)

	c.OnHTML("script[type=\"application/ld+json\"]", func(e *colly.HTMLElement) {
		fmt.Println("Found ld-json")
		// Parse the JSON content
		err := json.Unmarshal([]byte(e.Text), &article)
		if err != nil {
			return
		}
		book.SetTitle(article.Title)
		book.SetDescription(article.Description)
		book.SetIdentifier(article.URL)
		book.SetAuthor(fmt.Sprintf("%s - %s", article.Publisher.Name, article.Author[0].Name))
		// Set release date

		// Cover could be set, but that leads to the article name not being shown in kindle
		// imgPath, _ := book.AddImage(article.Image[0].URL, generateUUID(article.Image[0].URL))
		// _ = book.SetCover(imgPath, "")
		permalink = article.URL
	})

	c.OnHTML(".available-content", func(e *colly.HTMLElement) {
		// This should be parsed into the Book
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
			// Get file ending from src
			image, err := book.AddImage(imgSrc, generateUUID(imgSrc))
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

		// Parse releaseDate to time.Time
		releaseDate, err := time.Parse(time.RFC3339, article.Published)
		if err != nil {
			return
		}

		// We want to format the releasedate for 25th Februrary 2025 to "Feb 25, 2025"
		content = fmt.Sprintf("<h1>%s</h1><p>By %s<em><br/>Published at %s</em></p><hr/>%s", book.Title(), book.Author(), releaseDate.Format("Jan 01, 2006"), content)

		_, err = book.AddSection(content, book.Title(), fmt.Sprintf("%s.xhtml", normalizeStr(book.Title())), "")
		if err != nil {
			return
		}
		sectionFound = true
	})

	paywallAccessible := true

	c.OnHTML(".paywall-title", func(e *colly.HTMLElement) {
		if e.Text == "This post is for paid subscribers" {
			paywallAccessible = false
		}
	})

	err = c.Visit(*url)
	if err != nil {
		return nil, err
	}

	if !article.Free && !paywallAccessible {
		return nil, fmt.Errorf("the article is behind a paywall and not accessible")
	}

	// Check whether everything was parsed correctly
	if book.Title() == "" || book.Author() == "" || !sectionFound {
		return nil, fmt.Errorf("failed to parse the Substack newsletter correctly")
	}

	// We can now create an EPUB from the parsed HTML content
	epubPath := fmt.Sprintf("output/%s.epub", book.Title())
	err = book.Write(epubPath)

	if err != nil {
		return nil, err
	}

	return &Book{
		Book:      book,
		Path:      &epubPath,
		Permalink: &permalink,
		Paid:      !article.Free,
	}, nil
}
