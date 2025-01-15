package discord

import (
	"crypto/tls"
	"github.com/wneessen/go-mail"
	"kindExport/internal/config"
	"net/smtp"
	"strconv"
)

func createClient() (*smtp.Client, error) {
	conf, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	// Try to connect to the mail server
	client, err := smtp.Dial(conf.MailServer + ":" + strconv.Itoa(conf.MailPort))
	if err != nil {
		return nil, err
	}

	// StartTLS
	err = client.StartTLS(&tls.Config{
		ServerName: conf.MailServer,
	})
	if err != nil {
		return nil, err
	}

	err = client.Auth(smtp.PlainAuth("", conf.MailUser, conf.MailPassword, conf.MailServer))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func CheckMailConfig() error {
	client, err := createClient()
	if err != nil {
		return err
	}
	defer client.Close()
	return nil
}

func sendMail(address string, epubPath string) error {
	conf, _ := config.GetConfig()

	message := mail.NewMsg()
	message.From("kindle@sim0ns.de")
	message.To(address)
	message.Subject("Your newsletter article is ready")
	message.SetBodyString(mail.TypeTextPlain, "See attached for the newsletter article")
	message.AttachFile(epubPath)
	client, err := mail.NewClient(conf.MailServer, mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(conf.MailUser), mail.WithPassword(conf.MailPassword), mail.WithPort(conf.MailPort))
	if err != nil {
		return err
	}
	err = client.DialAndSend(message)
	if err != nil {
		return err
	}
	return nil
}
