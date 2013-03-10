package controllers

import (
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

//Poster represent a post
type Poster interface {
	Date() time.Time
	Content() template.HTML
	Tile() string
}

//one record in posts list
type Record struct {
	Date time.Time
	Name string
}

//file meta infos db
type List struct {
	basePath string
	records  []*Record
}

func NewList(topDir string) *List {
	return &List{
		basePath: topDir,
		records:  make([]*Record, 0),
	}
}

//add a record into list
func (l *List) Add(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	//ignore the err, we sure path is under basePath
	name, _ := filepath.Rel(l.basePath, path)
	now := info.ModTime()
	found := false
	//find it by name
	for _, record := range l.records {
		if record.Name == name {
			//replace it with new one
			if now.After(record.Date) {
				record.Date = now
				revel.INFO.Printf("db update: %q\n", path)
			}
			found = true
			break
		}
	}
	if !found {
		//append new one
		r := &Record{now, name}
		l.records = append(l.records, r)
		revel.INFO.Printf("db add: %q\n", path)
	}
	//reupdate it
	sort.Sort(l)
	return nil
}

//remove a record from list
func (l *List) Remove(path string) {
	//ignore the err, we sure path is under basePath
	name, _ := filepath.Rel(l.basePath, path)
	found := false
	for i, record := range l.records {
		if record.Name == name {
			l.records = append(l.records[:i], l.records[i+1:]...)
			found = true
			break
		}
	}
	if found {
		revel.INFO.Printf("db delete: %q\n", path)
		sort.Sort(l)
	}
}

//let *List satisfy sort.Interface
func (l *List) Less(i, j int) bool {
	return l.records[i].Date.After(l.records[j].Date)
}

func (l *List) Swap(i, j int) {
	l.records[i], l.records[j] = l.records[j], l.records[i]
}

func (l *List) Len() int {
	return len(l.records)
}

type Posts struct {
	basePath string
	posts    map[string]template.HTML
}

func NewPosts(topDir string) *Posts {
	return &Posts{
		basePath: topDir,
		posts:    make(map[string]template.HTML),
	}
}

func (p *Posts) Add(path string) error {
	//ignore the err, we sure path is under basePath
	name, _ := filepath.Rel(p.basePath, path)
	content, err := generateHTML(path)
	if err != nil {
		return err
	}
	p.posts[name] = content
	return nil
}

func (p *Posts) Remove(path string) {
	//ignore the err, we sure path is under basePath
	name, _ := filepath.Rel(p.basePath, path)
	delete(p.posts, name)
}

func (p *Posts) Get(name string) (template.HTML, bool) {
	data, found := p.posts[name]
	return data, found
}

//articles db
type articleDB struct {
	basePath string //topdir of the db
	articles *Posts //storage
	list     *List  //posts list, sorted by time
}

func newArticleDB(topdir string) *articleDB {
	//ignore error by fsnotify.NewWatcher()
	a := &articleDB{
		basePath: topdir,
		articles: NewPosts(topdir),
		list:     NewList(topdir),
	}
	go a.watchLoop()
	return a
}

//add/update/remove with articleDB
func (a *articleDB) watchLoop() {
	c := time.Tick(1 * time.Second)
	for _ = range c {
		//delte the removed files
		a.clean()
		//update and add new files
		a.update()
	}
}

//add/update file with db
func (a *articleDB) update() {
	if err := filepath.Walk(a.basePath, func(path string, info os.FileInfo, err error) error {
		//only watch my filetype
		if info.IsDir() || !filetypeFilter(path) {
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
		return nil
	}); err != nil {
		revel.WARN.Println(err)
	}
}

//remove files in db
func (a *articleDB) clean() {
	var paths []string
	for _, record := range a.list.records {
		path := a.basePath + record.Name
		_, err := os.Stat(path)
		if err != nil && os.IsNotExist(err) {
			paths = append(paths, path)
		}
	}
	for _, path := range paths {
		a.articles.Remove(path)
		a.list.Remove(path)
	}
}

//supported filetype
var filters = []*regexp.Regexp{
	regexp.MustCompile(".*.md$"),
}

//filter file type , return pass
func filetypeFilter(path string) (passed bool) {
	for _, filter := range filters {
		if filter.MatchString(path) {
			return true
		}
	}
	return false
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
	revel.EmptyPlugin
}

func (p PostsPlugin) OnAppStart() {
	//assume posts in $GOPATH/src/totorow/app/posts/"
	gopath := os.Getenv("GOPATH")
	topDir := gopath + "/src/totorow/app/posts/"
	//init articleDb
	storage = newArticleDB(topDir)
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
	revel.RegisterPlugin(PostsPlugin{})

	//register template func
	revel.TemplateFuncs["trim"] = TrimSuffix
	revel.TemplateFuncs["ymd"] = Ymd
}
