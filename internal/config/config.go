package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config is a struct that holds the configuration for the application
type Config struct {
	// DatabasePath is the path to the sqlite database
	DatabasePath string
	// OutputDirectory is the directory where the output files are stored
	OutputDirectory string
	// DiscordToken is the token for the discord bot
	DiscordToken string
	// MailServer is the address of the mail server
	MailServer string
	// MailPort is the port of the mail server
	MailPort int
	// MailUser is the username for the mail server
	MailUser string
	// MailPassword is the password for the mail server
	MailPassword string
}

var (
	instance  *Config
	once      sync.Once
	initError error
)

// GetConfig returns the configuration for the application
func GetConfig() (*Config, error) {
	if instance == nil {
		once.Do(func() {
			instance = &Config{
				DatabasePath:    "./kindExport.sqlite",
				OutputDirectory: "./output",
				DiscordToken:    "",
				MailServer:      "",
				MailPort:        587,
				MailUser:        "",
				MailPassword:    "",
			}
			if os.Getenv("OUTPUT_DIRECTORY") != "" {
				instance.OutputDirectory = strings.TrimRight(os.Getenv("OUTPUT_DIRECTORY"), "/")
			}
			if os.Getenv("DATABASE_PATH") != "" {
				instance.DatabasePath = os.Getenv("DATABASE_PATH")
			}
			if os.Getenv("DISCORD_TOKEN") != "" {
				instance.DiscordToken = os.Getenv("DISCORD_TOKEN")
			} else {
				initError = errors.New("DISCORD_TOKEN is not set, it is required")
				return
			}
			if os.Getenv("MAIL_SERVER") != "" {
				instance.MailServer = os.Getenv("MAIL_SERVER")
			} else {
				initError = errors.New("MAIL_SERVER is not set, it is required")
				return
			}
			if os.Getenv("MAIL_PORT") != "" {
				port, err := strconv.Atoi(os.Getenv("MAIL_PORT"))
				if err != nil {
					initError = errors.New("MAIL_PORT is not a valid number")
					return
				}
				instance.MailPort = port
			}
			if os.Getenv("MAIL_USER") != "" {
				instance.MailUser = os.Getenv("MAIL_USER")
			} else {
				initError = errors.New("MAIL_USER is not set, it is required")
				return
			}
			if os.Getenv("MAIL_PASSWORD") != "" {
				instance.MailPassword = os.Getenv("MAIL_PASSWORD")
			} else {
				initError = errors.New("MAIL_PASSWORD is not set, it is required")
				return
			}
		})
	}
	return instance, initError
}
