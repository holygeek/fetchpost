package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type PostCache struct {
	Url   string
	Items map[string]*item
}

var postCache = PostCache{Items: map[string]*item{}}

const DEFAULT_FOLDER = "<from title>"
const HOST_HACKER_NEWS = "news.ycombinator.com"

var Verbose *bool
var Quiet *bool

func main() {
	cacheFile := flag.String("readfrom", "", "Read posts from a json file instead of querying firebase (mainly used for testing)")
	dump := flag.Bool("dump", false, "Fetch the given url only and dump the response. Do not fetch its children")
	folder := flag.String("o", DEFAULT_FOLDER, "Save mails into the given directory instead of the default one")
	Verbose = flag.Bool("v", false, "Verbose")
	Quiet = flag.Bool("q", false, "Be quiet")
	flag.Parse()
	if *Quiet {
		*Verbose = false
	}

	var arg string
	if *cacheFile != "" {
		if fileExists(*cacheFile) {
			readCacheFile(*cacheFile)
		} else {
			bail("readfrom: file %s does not exist", *cacheFile)
		}
		arg = postCache.Url
	} else {
		if flag.NArg() != 1 {
			printError("Usage: post2mail <url|dir>")
			os.Exit(1)
		}
		arg = flag.Arg(0)
	}

	var url string
	if strings.HasPrefix(arg, "https://") {
		url = arg
		postCache.Url = url
	} else {
		url = readUrlFile(arg)
		if strings.HasPrefix(url, "https://") {
			postCache.Url = url
		}
	}

	/**
	1. Get post
	2. If mail for post doesn't exist:
		2.1 write mail in maildir
	3. else if post body or title changed:
		3.1 Update subject or body and mark mail as new
	4. Repeat from step 1 for each kid in post.
	*/
	post := mustGetPost(url)
	if *dump {
		buf, err := json.Marshal(post)
		if err != nil {
			bail("dump: %v", err)
		}
		fmt.Printf("%s\n", string(buf))
		os.Exit(0)
	}

	saveComments(post, arg, arg, *folder)

	postDir := getPostDir(post)
	fmt.Println(postDir)
	postTestFile := filepath.Join(postDir, postDir+".json")
	buf, err := json.Marshal(postCache)
	if err != nil {
		printError("Marshal postCache: %v", err)
	}
	err = ioutil.WriteFile(postTestFile, append(buf, '\n'), 0644)
	if err != nil {
		printError("Write postCache: %v", err)
	}
}

func saveComments(post Post, arg, url string, folder string) {
	dir := getPostDir(post)
	if folder == DEFAULT_FOLDER && arg != url && arg != dir {
		renamePostDir(arg, dir)
	}

	mails, err := NewMails(dir)
	if err != nil {
		bail("saveComments(): error: %v", err)
	}
	maybeSavePostUrl(dir, url)
	mails.SavePost(nil, post, 0)
}

func renamePostDir(olddir, newdir string) {
	if fileExists(newdir) {
		bail("renamePostDir(): post title changed but directory for new title"+
			" already exist\n\told title: %s\n\tnew title: %s",
			olddir, newdir)
	}
	err := os.Rename(olddir, newdir)
	if err != nil {
		bail("renamePostDir(): Rename: %v", err)
	}
	printNotice("post title has changed:\n\told: %s\n\tnew: %s", olddir, newdir)

}

func readCacheFile(f string) {
	buf, err := ioutil.ReadFile(f)
	if err != nil {
		bail("Read %s: %v", f, err)
	}
	err = json.Unmarshal(buf, &postCache)
	if err != nil {
		bail("Unmarshal %s: %v", f, err)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			bail("fileExists: %s: %v", path, err)
		}
	}
	return true
}

func (mails *Mails) SavePost(parent, child Post, threadLevel int) {
	mid := child.MessageId()
	key, exist := mails.keyForMessageId[mid]
	if exist {
		msg, err := mails.md.Message(key)
		if err != nil {
			bail("SavePost(): Error getting mail for key %s: %v", key, err)
		}

		path, err := mails.md.Filename(key)
		if err != nil {
			bail("SavePost(): could not get Filename for key %s", key)
		}
		contentChanged := updateMail(parent, child, path, msg)
		if contentChanged {
			p := strings.Replace(path, "/cur/", "/new/", 1)
			if path != p {
				err = os.Rename(path, p)
				if err != nil {
					bail("SavePost(): Rename(%s, %s): %v", path, p, err)
				}
				verbose("Renamed %s to %s", path, p)
			}
		}
		showThreadLevel(threadLevel, "*")
	} else {
		mail := NewMail(parent, child)
		err := mails.SaveMail(mail)
		if err != nil {
			bail("SavePost(): %v", err)
		}
		showThreadLevel(threadLevel, "+")
	}

	mails.SavePostChildren(child, threadLevel+1)
}

func (mails *Mails) SavePostChildren(p Post, threadLevel int) {
	for _, idStr := range p.Children() {
		child, err := fetchHNPost(idStr)
		if err != nil {
			printError("error: %v", err)
			continue
		}
		mails.SavePost(p, child, threadLevel)
	}
}

func showThreadLevel(level int, symbol string) {
	for i := 0; i < level; i++ {
		fmt.Print(" ")
	}
	fmt.Println(symbol)
}

func maybeSavePostUrl(dir, url string) {
	if !fileExists(dir) {
		os.Mkdir(dir, 0755)
	}
	f := filepath.Join(dir, "url.txt")
	if fileExists(f) {
		return
	}
	err := ioutil.WriteFile(f, []byte(url), 0644)
	if err != nil {
		bail("%s: %v", f, err)
	}
}

func mustGetPost(href string) Post {
	url, err := url.Parse(href)
	if err != nil {
		bail("bad url: %s", err)
	}

	switch strings.ToLower(url.Host) {
	case HOST_HACKER_NEWS:
		return GetHNPost(url)
	default:
		bail("Unsupported url: %s", url.Host)
	}
	return nil
}

func bail(format string, args ...interface{}) {
	printError(format, args...)
	os.Exit(1)
}

func readUrlFile(arg string) string {
	urlFile := filepath.Join(arg, "url.txt")
	buf, err := ioutil.ReadFile(urlFile)
	if err != nil {
		bail("readUrlFile: error: %v", err)
	}
	return string(buf)
}

func verbose(format string, args ...interface{}) {
	if !*Verbose {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func printNotice(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "notice:"+format+"\n", args...)
}
