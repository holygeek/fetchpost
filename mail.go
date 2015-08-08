package main

import (
	"fmt"
	"os"
	"time"

	"github.com/holygeek/maildir"
)

type Mails struct {
	md              maildir.Dir
	keys            []string
	keyForMessageId map[string]string
}

type Email struct {
	// TODO use map[string]interface{} instead - simplifies adding new
	// header later
	From      string
	Date      string
	XDate     string
	To        string
	Subject   string
	Body      string
	MessageId string
	Score     int
	InReplyTo string
}

func time2rfc8222(tstamp int64) string {
	return time.Unix(tstamp, 0).Format(time.RFC822Z)
}

func NewMail(parent, item Post) *Email {
	m := &Email{
		From:      item.Poster(),
		Date:      time2rfc8222(item.SortDate()),
		XDate:     time2rfc8222(item.Date()),
		To:        item.Recipient(),
		Subject:   item.Subject(),
		Body:      item.Body(),
		MessageId: item.MessageId(),
		Score:     item.Points(),
	}
	if item.PostType() != "comment" {
		m.Subject = fmt.Sprintf("(%d points) %s", item.Points(), m.Subject)
	}
	if parent != nil && parent.PostType() == "comment" {
		m.InReplyTo = parent.MessageId()
	}
	return m
}

func (e *Email) Header() string {
	format := "From: %s\n" +
		"To: %s\n" +
		"Subject: %s\n" +
		"Date: %s\n" +
		"X-Date: %s\n" +
		"Message-ID: %s\n" +
		"X-Score: %d\n" +
		`Content-Type: text/plain; charset="UTF-8"`
	values := []interface{}{
		e.From,
		e.To,
		e.Subject,
		e.Date,
		e.XDate,
		e.MessageId,
		e.Score,
	}

	if len(e.InReplyTo) > 0 {
		format = format + "\nIn-Reply-To: %s"
		values = append(values, e.InReplyTo)
	}
	return fmt.Sprintf(format, values...)
}

func (e *Email) Bytes() []byte {
	return []byte(e.String())
}

func (e *Email) String() string {
	return fmt.Sprintf("%s\n\n%s", e.Header(), e.Body)
}

func NewMails(dir string) (*Mails, error) {
	m := &Mails{md: maildir.Dir(dir)}
	m.keys = []string{}
	m.keyForMessageId = map[string]string{}
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		m.md.Create()
		return m, nil
	}
	keys, err := m.md.Keys()
	if err != nil {
		return nil, err
	}
	m.keys = keys
	for _, k := range m.keys {
		ma, err := m.md.Message(k)
		if err != nil {
			bail("error reading mail for %s: %v", k, err)
		}
		mid := ma.Header.Get("Message-ID")
		if len(mid) == 0 {
			bail("could not get message id for from %s", k)
		}
		m.keyForMessageId[mid] = k
	}
	return m, nil
}

func (mails *Mails) SaveMail(mail *Email) error {
	d, err := mails.md.NewDelivery()
	if err != nil {
		return fmt.Errorf("maildir NewDelivery: %v", err)
	}
	_, err = d.Write(mail.Bytes())
	if err != nil {
		return fmt.Errorf("maildir Write(): %v", err)
	}
	err = d.Close()
	if err != nil {
		return fmt.Errorf("maildir Close(): %v", err)
	}
	return nil
}
