package controllers

import (
	"github.com/robfig/revel"
	"github.com/russross/blackfriday"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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
func (l *List) Add(r *Record) {
	found := false
	//find it by name
	for i, record := range *l {
		if record.Name == r.Name {
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
}

//remove a record from list
func (l *List) Remove(name string) {
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

func (p Posts) Add(name string, data template.HTML) {
	p[name] = data
}

func (p Posts) Delete(name string) {
	delete(p, name)
}

func (p Posts) Get(name string) (template.HTML, bool) {
	data, found := p[name]
	return data, found
}

//articles db
type articleDB struct {
	articles Posts //storage
	list     List  //posts list, sorted by time
}

func newArticleDB() *articleDB {
	return &articleDB{
		articles: NewPosts(),
		list:     NewList(),
	}
}

func (a *articleDB) init(topDir string) error {
	if err := filepath.Walk(topDir, func(path string, info os.FileInfo, err error) error {
		//skip dir itself
		if info.IsDir() {
			return nil
		}
		filename := info.Name()
		a.list.Add(&Record{Date: info.ModTime(), Name: filename})
		content, err := generateHTML(path)
		if err != nil {
			return err
		}
		a.articles.Add(filename, content)
		rev.INFO.Printf("metadb add a file: %q\n", path)
		return nil
	}); err != nil {
		return err
	}
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

type PostPlugin struct {
	rev.EmptyPlugin
}

func (d PostPlugin) OnAppStart() {
	//init metadb
	//assume posts in $GOPATH/src/totorow/app/posts/"
	gopath := os.Getenv("GOPATH")
	topDir := gopath + "/src/totorow/app/posts/"
	//init articleDb
	storage = newArticleDB()
	if err := storage.init(topDir); err != nil {
		rev.ERROR.Printf("init articles failed: err=%s\n", err)
		return
	}
}

//tempelate map func: trim filetype suffix
func TrimSuffix(path string) string {
	return strings.TrimRight(path, filepath.Ext(path))
}

func init() {
	//register post plugin
	rev.RegisterPlugin(PostPlugin{})

	//register TrimSuffix template func
	rev.TemplateFuncs["trim"] = TrimSuffix
}
