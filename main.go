package main

import (
	"kindExport/internal/config"
	"kindExport/internal/db"
	"kindExport/internal/discord"
	"kindExport/internal/scrape"
	"log"
)

func mainSubstack() {
	link := "https://newsletter.pragmaticengineer.com/p/leaving-big-tech"
	scraper := scrape.SubstackScraper{}

	//_, err := scraper.CheckPaywallAccessible("https://newsletter.pragmaticengineer.com/p/the-pulse-118")
	//return
	_, err := scraper.Scrape(&link)
	if err != nil {
		print(err.Error())
		return
	}
}

func main() {
	//mainSubstack()
	//return
	log.Printf("Initializing database")
	dbSession, err := db.GetDB()
	if err != nil {
		log.Printf("Error initializing database: %s", err.Error())
		return
	}
	if dbSession != nil {
		defer dbSession.Close()
	}
	log.Printf("Checking mail configuration")
	err = discord.CheckMailConfig()
	if err != nil {
		log.Printf("Error checking mail configuration: %s", err.Error())
		return
	}
	log.Printf("Starting listener")
	conf, err := config.GetConfig()
	if err != nil {
		log.Printf("Error getting configuration: %s", err.Error())
		return
	}
	listener, err := discord.NewListener(conf.DiscordToken)
	if err != nil {
		log.Printf("Error creating listener: %s", err.Error())
		return
	}

	listener.Listen()
}
