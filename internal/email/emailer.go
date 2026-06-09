package email

import (
	"fmt"

	"gopkg.in/gomail.v2"
)

type EmailConfig struct {
	Sender      string
	Receivers   []string
	AppPassword string
}

func SendEmail(subject, body string, config EmailConfig, attachmentPath string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", config.Sender)
	m.SetHeader("To", config.Receivers[0])
	if len(config.Receivers) > 1 {
		m.SetHeader("Cc", config.Receivers[1:]...)
	}
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)
	m.Attach(attachmentPath)

	d := gomail.NewDialer("smtp.gmail.com", 587, config.Sender, config.AppPassword)
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("erro enviando email: %w", err)
	}
	return nil
}
