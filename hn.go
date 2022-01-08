package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const HN_API_URL = "https://hacker-news.firebaseio.com/v0/item/%s.json"
const DELETED = "-Deleted-"

type item struct {
	// Reference: https://github.com/HackerNews/API
	Id          int    // The item's unique id.
	Deleted     bool   // true if the item is deleted.
	Type        string // The type of item. One of "job", "story", "comment", "poll", or "pollopt".
	By          string // The username of the item's author.
	Time        int64  // Creation date of the item, in Unix Time.
	Text        string // The comment, story or poll text. HTML.
	Dead        bool   // true if the item is dead.
	Parent      int    // The item's parent. For comments, either another comment or the relevant story. For pollopts, the relevant poll.
	Kids        []int  // The ids of the item's comments, in ranked display order.
	Url         string // The URL of the story.
	Score       int    // The story's score, or the votes for a pollopt.
	Title       string // The title of the story, poll or job.
	Parts       []int  // A list of related pollopts, in display order.
	Descendants int    // In the case of stories or polls, the total comment count.
}

func (i *item) Poster() string {
	if i.Deleted {
		return DELETED
	}
	return i.By
}

func (i *item) Recipient() string {
	return "everyone@example.com"
}

func (i *item) Subject() (subject string) {
	if i.Deleted {
		subject = DELETED
		return
	}
	if len(i.Title) == 0 {
		s := i.Text
		if len(s) > 80 {
			s = string([]rune(s)[0:80])
		}
		subject = strings.Replace(s, "\n", " ", -1)
	} else {
		subject = i.Title
	}
	subject = html.UnescapeString(subject)
	return
}

func (i *item) Body() (body string) {
	if len(i.Text) == 0 {
		body = i.Url
	} else {
		body = i.Text
	}
	body = html.UnescapeString(body)
	body = strings.Replace(body, "<p>", "\n\n", -1)
	return
}

var c int64 = 0

func (i *item) SortDate() int64 {
	c++
	return c
}

func (i *item) Date() int64 {
	return i.Time
}

func (i *item) MessageId() string {
	return fmt.Sprintf("<hackernews-%d>", i.Id)
}

func (i *item) Points() int {
	return i.Score
}

func (i *item) Children() (idStr []string) {
	for _, v := range i.Kids {
		idStr = append(idStr, strconv.Itoa(v))
	}
	return
}

func (i *item) PostType() string {
	return i.Type
}

func (i *item) Removed() bool {
	return i.Deleted
}

func GetHNPost(url *url.URL) Post {
	idStr := url.Query().Get("id")
	if len(idStr) == 0 {
		return nil
	}

	item, err := fetchHNPost(idStr)
	if err != nil {
		bail("%v", err)
	}
	return item
}

func fetchHNPost(id string) (r *item, err error) {
	if rr, exist := postCache.Items[id]; exist {
		return rr, nil
	}
	url := fmt.Sprintf(HN_API_URL, id)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	verbose("%v", url)
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &r)
	if err != nil {
		return nil, err
	}
	verbose("\t%d bytes", len(body))
	if _, exist := postCache.Items[id]; !exist {
		postCache.Items[id] = r
	}
	return r, nil
}
