package main

import (
	"io/ioutil"
	"net/mail"
	"regexp"
)

type Post interface {
	Poster() string
	Recipient() string
	Subject() string
	SortDate() int64
	Date() int64
	MessageId() string
	Points() int
	Body() string
	Children() []string
	PostType() string
	Removed() bool
}

var nonTextRe = regexp.MustCompile(`[^0-9a-zA-Z_-]+`)

func getPostDir(p Post) string {
	dir := p.Subject()
	if len(dir) == 0 {
		dir = p.MessageId() + p.Body()[0:20]
	}
	dir = nonTextRe.ReplaceAllString(dir, "_")
	return dir
}

func updateMail(parent, child Post, path string, msg *mail.Message) bool {
	changed := false
	newTime := time2rfc8222(child.Date())
	oldTime := msg.Header.Get("X-Date")
	if oldTime != newTime {
		verbose("Date changed %s:\n\told: '%s'\n\tnew: '%s'\n", child.MessageId(), oldTime, newTime)
		changed = true
	}

	oldBody, err := ioutil.ReadAll(msg.Body)
	if err != nil {
		bail("updateMail(): error reading mail body for %v", child.MessageId())
	}

	newBody := child.Body()
	if newBody != string(oldBody) {
		changed = true
	}
	mail := NewMail(parent, child)
	err = ioutil.WriteFile(path, mail.Bytes(), 0744)
	if err != nil {
		bail("updateMail(): %v", err)
	}

	return changed
}
