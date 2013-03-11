package models

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/russross/blackfriday"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

func init() {
	RegisterRepoType("local", NewLocalRepo)
}

type localRepo struct { /*{{{*/
	exitCh chan bool
	root   string
	posts  map[string]*localPost
} /*}}}*/

func NewLocalRepo(root string) Repository { /*{{{*/
	return &localRepo{
		exitCh: make(chan bool),
		root:   root,
		posts:  make(map[string]*localPost),
	}
} /*}}}*/

//implement the Repository interface
func (lr *localRepo) Setup() error { /*{{{*/
	//root Must be a dir
	fi, err := os.Stat(lr.root)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return errors.New("you can't specify a file as a repo root")
	}
	go lr.watch()
	return nil
} /*}}}*/

func (lr *localRepo) Uninstall() { /*{{{*/
	lr.exitCh <- true
} /*}}}*/

func (lr *localRepo) watch() { /*{{{*/
	timer := time.Tick(1 * time.Second)
	for {
		select {
		case <-lr.exitCh:
			return
		case <-timer:
			//delete the removed files
			lr.clean()
			//add newer post and update the exist post
			lr.update()
		}
	}
} /*}}}*/

//clean the noexist posts
func (lr *localRepo) clean() { /*{{{*/
	cleans := make([]string, 0)
	for relPath := range lr.posts {
		absPath := lr.root + relPath
		_, err := os.Stat(absPath)
		if err != nil && os.IsNotExist(err) {
			cleans = append(cleans, relPath)
		}
	}
	for _, relPath := range cleans {
		lp := lr.posts[relPath]
		if err := Remove(lp); err != nil {
			log.Printf("remove local post failed: %s\n", err)
			continue
		}
		delete(lr.posts, relPath)
	}
} /*}}}*/

//update add new post or update the exist ones
func (lr *localRepo) update() { /*{{{*/
	if err := filepath.Walk(lr.root, func(path string, info os.FileInfo, err error) error {
		//only watch the special filetype
		if info.IsDir() || !filetypeFilter(path) {
			return nil
		}
		relPath, _ := filepath.Rel(lr.root, path)
		post, found := lr.posts[relPath]
		if !found {
			lp := newLocalPost(lr.root + relPath)
			lr.posts[relPath] = lp
			return nil
		}
		//update a exist one
		var updater Updater = post
		updater.Update()
		return nil
	}); err != nil {
		log.Printf("update local repo(%s) error: %s\n",
			lr.root, err)
	}
} /*}}}*/

//supported filetype
var filters = []*regexp.Regexp{ /*{{{*/
	regexp.MustCompile(".*.md$"),
} /*}}}*/

//filter file type , return pass
func filetypeFilter(path string) (passed bool) { /*{{{*/
	for _, filter := range filters {
		if filter.MatchString(path) {
			return true
		}
	}
	return false
} /*}}}*/

//represet a local post
type localPost struct { /*{{{*/
	path string

	mutex      sync.RWMutex
	key        string
	title      string
	date       time.Time
	content    template.HTML
	lastUpdate time.Time
} /*}}}*/

func newLocalPost(path string) *localPost { /*{{{*/
	return &localPost{
		path: path,
	}
} /*}}}*/

//implement Updater
func (lp *localPost) Update() error { /*{{{*/
	file, err := os.Open(lp.path)
	if err != nil {
		return err
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		return err
	}
	if ut := fi.ModTime(); ut.After(lp.lastUpdate) {
		key, title, date, content, err := generateAll(file)
		if err != nil {
			return err
		}
		lp.mutex.Lock()
		lp.key, lp.title, lp.date, lp.content, lp.lastUpdate =
			key, title, date, content, ut
		lp.mutex.Unlock()
		//update the content in dataCenter
		if err := Add(lp); err != nil {
			log.Printf("update a local post failed: %s\n", err)
		}
		log.Printf("update a local post: path(%s), key(%x), date(%s), lastUpdate(%s)\n",
			lp.path, lp.Key(), lp.Date(), lp.lastUpdate)
	}
	return nil
} /*}}}*/

func generateAll(file *os.File) (key, title string, date time.Time, content template.HTML, err error) { /*{{{*/
	c, e := ioutil.ReadAll(file)
	if e != nil {
		err = e
		return
	}
	//generate title and date
	firstLineIndex := strings.Index(string(c), "\n")
	if firstLineIndex == -1 {
		err = errors.New("generateAll: there must be at least one line\n")
		return
	}
	firstLine := strings.TrimSpace(string(c[:firstLineIndex]))
	remain := strings.TrimSpace(string(c[firstLineIndex+1:]))
	sepIndex := strings.Index(firstLine, TitleAndDateSeperator)
	if sepIndex == -1 {
		err = errors.New("generateAll: can't find seperator for title and date\n")
		return
	}
	t, e := time.Parse(TimePattern, strings.TrimSpace(firstLine[sepIndex+1:]))
	if e != nil {
		err = e
		return
	}

	//generate key
	h := md5.New()
	io.WriteString(h, string(c))
	key = fmt.Sprintf("%x", h.Sum(nil))

	title = strings.TrimSpace(firstLine[:sepIndex])
	date = t
	content = template.HTML(blackfriday.MarkdownCommon([]byte(remain)))
	return
} /*}}}*/

//implement controllers.Poster
func (lp *localPost) Date() template.HTML { /*{{{*/
	lp.mutex.RLock()
	defer lp.mutex.RUnlock()
	return template.HTML(lp.date.Format(TimePattern))
} /*}}}*/

func (lp *localPost) Content() template.HTML { /*{{{*/
	lp.mutex.RLock()
	defer lp.mutex.RUnlock()
	return lp.content
} /*}}}*/

func (lp *localPost) Title() template.HTML { /*{{{*/
	lp.mutex.RLock()
	defer lp.mutex.RUnlock()
	return template.HTML(lp.title)
} /*}}}*/

//implement Keyer
func (lp *localPost) Key() string { /*{{{*/
	lp.mutex.RLock()
	defer lp.mutex.RUnlock()
	return lp.key
} /*}}}*/
