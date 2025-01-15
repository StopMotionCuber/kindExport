package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/go-jet/jet/v2/sqlite"
	"kindExport/generated/model"
	. "kindExport/generated/table"
	"kindExport/internal/db"
	"kindExport/internal/scrape"
	"log"
	_ "modernc.org/sqlite"
	"net/mail"
	"net/url"
	"strings"
)

var (
	registeredCommands map[int]*discordgo.ApplicationCommand
	commands           = []*discordgo.ApplicationCommand{
		{
			Name:        "mail",
			Description: "Recipient address of the mail for the Kindle exporter.",
			Contexts: &[]discordgo.InteractionContextType{
				discordgo.InteractionContextPrivateChannel,
				discordgo.InteractionContextBotDM,
			},
			IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
				discordgo.ApplicationIntegrationUserInstall,
			},
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "address",
					Description: "The mail address to send the exported epub to.",
					Required:    false,
				},
			},
		},
		{
			Name:        "export",
			Description: "Export a Substack newsletter to a epub. Will be sent to mail address if configured.",
			Contexts: &[]discordgo.InteractionContextType{
				discordgo.InteractionContextPrivateChannel,
				discordgo.InteractionContextBotDM,
			},
			IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
				discordgo.ApplicationIntegrationUserInstall,
			},
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "url",
					Description: "The URL of the Substack newsletter to export.",
					Required:    true,
				},
			},
		},
		{
			Name:        "session",
			Description: "Set the session cookie, needs to be the connect.sid",
			Contexts: &[]discordgo.InteractionContextType{
				discordgo.InteractionContextPrivateChannel,
				discordgo.InteractionContextBotDM,
			},
			IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
				discordgo.ApplicationIntegrationUserInstall,
			},
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "session_cookie",
					Description: "The value of the `connect.sid` cookie.",
					Required:    true,
				},
			},
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"mail":    handleMail,
		"export":  handleExport,
		"session": handleSession,
	}
)

func handleSession(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	sessionCookie := options[0].StringValue()

	// Get discord user id
	userID := i.Interaction.User.ID

	stmt := sqlite.SELECT(
		Users.AllColumns,
	).FROM(
		Users,
	).WHERE(
		Users.DiscordID.EQ(sqlite.String(userID)),
	).LIMIT(1)

	dbSession, _ := db.GetDB()
	var users []model.Users
	err := stmt.Query(dbSession, &users)

	if err != nil {
		log.Printf("Error querying user: %s", err.Error())
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An internal error occurred",
			},
		})
		return
	}

	if len(users) == 0 {
		// Create the user in the database
		_, err = Users.
			INSERT(Users.DiscordID, Users.Name, Users.SubstackSession).
			VALUES(userID, i.User.Username, sessionCookie).
			Exec(dbSession)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "An internal error occurred while creating new user",
				},
			})
			return
		}
	} else {
		_, err = Users.
			UPDATE(Users.SubstackSession).
			SET(sessionCookie).
			WHERE(Users.DiscordID.EQ(sqlite.String(userID))).
			Exec(dbSession)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "An internal error occurred while updating session for user",
				},
			})
			return
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Session cookie has been updated",
		},
	})
}

func normalizeUrl(urlValue string) string {
	u, err := url.Parse(urlValue)
	if err != nil {
		return urlValue
	}

	// Return the url without the query while normalizing the scheme to https
	// And remove trailing slash

	u.Scheme = "https"
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String()
}

func handleExport(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	urlValue := options[0].StringValue()

	// Get discord user id
	userID := i.Interaction.User.ID

	_, err := url.Parse(urlValue)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid URL",
			},
		})
		return
	}

	stmt := sqlite.SELECT(
		Users.AllColumns,
	).FROM(
		Users,
	).WHERE(
		Users.DiscordID.EQ(sqlite.String(userID)),
	).LIMIT(1)

	dbSession, _ := db.GetDB()
	var users []model.Users
	err = stmt.Query(dbSession, &users)

	if err != nil {
		log.Printf("Error querying user: %s", err.Error())
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An internal error occurred",
			},
		})
		return
	}

	// Check whether the epub is already in the database
	stmt = sqlite.SELECT(
		Articles.AllColumns,
	).FROM(
		Articles,
	).WHERE(
		Articles.URL.EQ(sqlite.String(normalizeUrl(urlValue))),
	).LIMIT(1)

	var articles []model.Articles
	println(stmt.Sql())
	err = stmt.Query(dbSession, &articles)

	if err != nil {
		log.Printf("Error querying articles: %s", err.Error())
		articles = []model.Articles{}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An internal error occurred",
			},
		})
		return
	}

	ebookPath := ""

	scraper := scrape.SubstackScraper{}
	if len(users) > 0 && users[0].SubstackSession != nil {
		scraper.SubstackLoginCookie = users[0].SubstackSession
	}

	// Fetch the epub (if necessary)
	if len(articles) > 0 {
		// Epub already exists in the database
		// Todo Check if the epub is paid
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Newsletter article already exists in the database, no need" +
					" to fetch it again",
			},
		})
		ebookPath = articles[0].LocalPath
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Fetching newsletter, this may take a few seconds...",
			},
		})

		book, err := scraper.Scrape(&urlValue)
		if err != nil {
			log.Printf("Error scraping newsletter: %s", err.Error())
			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: "Error fetching newsletter: " + err.Error(),
			})
			return
		}
		// Insert the book into the database
		db.InsertBook(*book)
		ebookPath = *book.Path
	}

	// Send the epub to the user's kindle mail address

	if len(users) == 0 || *users[0].KindleMail == "" || users[0].KindleMail == nil {
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: "Mail address is not configured." +
				" Epub has been fetched, but cannot be sent to kindle mail.",
		})
	} else {
		// Check for paywall on article
		paywallAccessible, err := scraper.CheckPaywallAccessible(urlValue)
		if err != nil {
			log.Printf("Error checking paywall: %s", err.Error())
			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: "Error checking paywall: " + err.Error(),
			})
			return
		}
		if !paywallAccessible {
			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: "Article is behind a paywall. As no session with a subscription is provided, the article will not be sent to the Kindle mail address." +
					" To access the article, please subscribe to the newsletter and provide the session cookie to the bot via the `/session` command",
			})
			return
		}
		err = sendMail(*users[0].KindleMail, ebookPath)
		if err != nil {
			log.Printf("Error sending mail: %s", err.Error())
			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: "Error sending epub to kindle mail address: " + err.Error(),
			})
			return
		}
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: "Sent epub to kindle mail address",
		})

	}

	if err != nil {
		log.Printf("Error scraping newsletter: %s", err.Error())
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error scraping newsletter: " + err.Error(),
			},
		})
		return
	}

}

func handleMail(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var address string
	if options == nil {
		address = ""
	} else {
		address = options[0].StringValue()
	}

	// Get discord user id
	userID := i.Interaction.User.ID

	if address != "" {
		_, err := mail.ParseAddress(address)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Invalid mail address",
				},
			})
			return
		}
	}

	stmt := sqlite.SELECT(
		Users.ID, Users.DiscordID, Users.Name, Users.KindleMail,
	).FROM(
		Users,
	).WHERE(
		Users.DiscordID.EQ(sqlite.String(userID)),
	).LIMIT(1)

	var dest []model.Users

	dbSession, err := db.GetDB()
	if err != nil {
		log.Printf("Error getting database session: %s", err.Error())
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An internal error occurred",
			},
		})
		return
	}

	err = stmt.Query(dbSession, &dest)
	if err != nil {
		log.Printf("Error querying user: %s", err.Error())
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An internal error occurred",
			},
		})
		return
	}

	if address == "" {
		// Return the current mail address
		if len(dest) == 0 || dest[0].KindleMail == nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Mail address is not set",
				},
			})
			return
		}
		address = *dest[0].KindleMail
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Your ebooks are exported to " + address,
			},
		})
	}

	if len(dest) == 0 {
		// Create a new user in the database
		_, err = Users.
			INSERT(Users.DiscordID, Users.Name, Users.KindleMail).
			VALUES(userID, i.User.Username, address).
			Exec(dbSession)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "An internal error occurred while creating new user",
				},
			})
			return
		}
	} else {
		// Update the existing user
		_, err = Users.
			UPDATE(Users.KindleMail).
			SET(address).
			WHERE(Users.DiscordID.EQ(sqlite.String(userID))).
			Exec(dbSession)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "An internal error occurred while updating mail for user",
				},
			})
			return
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Mail address has been updated to " + address,
		},
	})
}

func initCommands(session *discordgo.Session) error {
	registeredCommands = make(map[int]*discordgo.ApplicationCommand)
	for i, v := range commands {
		cmd, err := session.ApplicationCommandCreate(session.State.User.ID, "", v)
		if err != nil {
			return err
		}
		registeredCommands[i] = cmd
	}
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	return nil
}

func removeCommands(session *discordgo.Session) error {
	for _, v := range registeredCommands {
		log.Println("Removing command: ", v.Name)
		err := session.ApplicationCommandDelete(session.State.User.ID, "", v.ID)
		if err != nil {
			return err
		}
	}
	return nil
}
