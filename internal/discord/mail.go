package discord

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/wneessen/go-mail"
	"kindExport/internal/config"
	"mime/multipart"
	"net/http"
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

func toBytes(fileContent []byte, fileName string, subject string, toAddr string) []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(fmt.Sprintf("Subject: %s\n", subject))
	buf.WriteString(fmt.Sprintf("To: %s\n", toAddr))

	buf.WriteString("MIME-Version: 1.0\n")
	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\n", boundary))
	buf.WriteString(fmt.Sprintf("--%s\n", boundary))

	buf.WriteString("See attached for the newsletter article:\n")
	buf.WriteString(fmt.Sprintf("\n\n--%s\n", boundary))
	buf.WriteString(fmt.Sprintf("Content-Type: %s\n", http.DetectContentType(fileContent)))
	buf.WriteString("Content-Transfer-Encoding: base64\n")
	buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%s\n", fileName))

	b := make([]byte, base64.StdEncoding.EncodedLen(len(fileContent)))
	base64.StdEncoding.Encode(b, fileContent)
	buf.Write(b)
	buf.WriteString(fmt.Sprintf("\n--%s", boundary))
	buf.WriteString("--")

	return buf.Bytes()
}

func sendMail(address string, epubPath string) error {
	conf, _ := config.GetConfig()

	message := mail.NewMsg()
	address = "stopmotioncuber@gmail.com"
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
