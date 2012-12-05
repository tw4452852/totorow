package controllers

import (
	"github.com/howeyc/fsnotify"
	"github.com/robfig/revel"
	"github.com/russross/blackfriday"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

//one record in posts list
type Record struct {
	Date time.Time
	Name string
}

//file meta infos db
type List []*Record

func NewList() List {
	//prepare 10 entries at first
	return make([]*Record, 0, 10)
}

//add a record into list
func (l *List) Add(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	name := filepath.Base(path)
	r := &Record{info.ModTime(), name}
	found := false
	//find it by name
	for i, record := range *l {
		if record.Name == name {
			//replace it with new one
			(*l)[i] = r
			found = true
			break
		}
	}
	if !found {
		//append new one
		*l = append(*l, r)
	}
	//reupdate it
	sort.Sort(*l)
	return nil
}

//remove a record from list
func (l *List) Remove(path string) {
	name := filepath.Base(path)
	found := false
	for i, record := range *l {
		if record.Name == name {
			*l = append((*l)[:i], (*l)[i+1:]...)
			found = true
			break
		}
	}
	if found {
		sort.Sort(*l)
	}
}

//let *List satisfy sort.Interface
func (l List) Less(i, j int) bool {
	return l[i].Date.Before(l[i].Date)
}

func (l List) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l List) Len() int {
	return len(l)
}

type Posts map[string]template.HTML

func NewPosts() Posts {
	return make(map[string]template.HTML)
}

func (p Posts) Add(path string) error {
	name := filepath.Base(path)
	content, err := generateHTML(path)
	if err != nil {
		return err
	}
	p[name] = content
	return nil
}

func (p Posts) Remove(path string) {
	name := filepath.Base(path)
	delete(p, name)
}

func (p Posts) Get(name string) (template.HTML, bool) {
	data, found := p[name]
	return data, found
}

//articles db
type articleDB struct {
	articles Posts             //storage
	list     List              //posts list, sorted by time
	watcher  *fsnotify.Watcher //watch posts
}

func newArticleDB() *articleDB {
	//ignore error by fsnotify.NewWatcher()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		rev.ERROR.Printf("New posts watcher failed: %s\n", err)
	}
	return &articleDB{
		articles: NewPosts(),
		list:     NewList(),
		watcher:  watcher,
	}
}

var filters = []*regexp.Regexp{
	regexp.MustCompile(".*.sw[px]"),
	regexp.MustCompile(".*~"),
}

//filter file type , return pass
func filetypeFilter(path string) (passed bool) {
	for _, filter := range filters {
		if filter.MatchString(path) {
			rev.INFO.Printf("ignore %s\n", path)
			return false
		}
	}
	return true
}

func (a *articleDB) watchLoop() {
	for {
		select {
		case ev := <-a.watcher.Event:
			path := ev.Name
			rev.INFO.Printf("%s: %s\n", path, ev)
			if filetypeFilter(path) {
				switch {
				case ev.IsDelete() || ev.IsRename():
					a.list.Remove(path)
					a.articles.Remove(path)
					a.watcher.RemoveWatch(path)
				case ev.IsModify() || ev.IsCreate():
					a.list.Add(path)
					a.articles.Add(path)
					a.watcher.Watch(path)
				default:
					//nothing
				}
			}
		case err := <-a.watcher.Error:
			rev.INFO.Println(err)
		}
	}
}

func (a *articleDB) init(topDir string) error {
	if err := filepath.Walk(topDir, func(path string, info os.FileInfo, err error) error {
		if err := a.watcher.Watch(path); err != nil {
			rev.ERROR.Printf("add watch(%q) failed: %s\n", path, err)
		}
		//skip dir itself
		if info.IsDir() {
			return nil
		}
		//translate article first
		if err := a.articles.Add(path); err != nil {
			return err
		}
		//then generate record in list
		if err := a.list.Add(path); err != nil {
			return err
		}
		rev.INFO.Printf("metadb add a file: %q\n", path)
		return nil
	}); err != nil {
		return err
	}
	//start watchloop
	go a.watchLoop()
	return nil
}

//use blackfriday to generate template.HTML
func generateHTML(path string) (template.HTML, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return template.HTML(blackfriday.MarkdownCommon(data)), err
}

//global db
var (
	storage *articleDB
)

type PostsPlugin struct {
	rev.EmptyPlugin
}

func (p PostsPlugin) OnAppStart() {
	//init articleDb
	storage = newArticleDB()
	//init metadb
	//assume posts in $GOPATH/src/totorow/app/posts/"
	gopath := os.Getenv("GOPATH")
	topDir := gopath + "/src/totorow/app/posts/"
	if err := storage.init(topDir); err != nil {
		rev.ERROR.Printf("init articles failed: err=%s\n", err)
		return
	}
}

//tempelate map func: trim filetype suffix
func TrimSuffix(path string) string {
	return strings.TrimRight(path, filepath.Ext(path))
}

const TimePattern = "2006-01-02"

//tempelate map func: translate time.Time.String() to year-month-day
func Ymd(t time.Time) string {
	return t.Format(TimePattern)
}

func init() {
	//register posts plugin
	rev.RegisterPlugin(PostsPlugin{})

	//register template func
	rev.TemplateFuncs["trim"] = TrimSuffix
	rev.TemplateFuncs["ymd"] = Ymd
}
