package controllers

import (
	"errors"
	"github.com/robfig/revel"
	"github.com/tw4452852/storage"
	"html/template"
	"io"
	"runtime"
	"sort"
)

func Init() {
	storage.Init("src/totorow/conf/repos.xml")
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//register posts plugin
	revel.OnAppStart(Init)
}

//Poster represent a post
type Poster interface { /*{{{*/
	Content() template.HTML
} /*}}}*/

//Lister represent a list entry
type Lister interface { /*{{{*/
	storage.Keyer
	Date() template.HTML
	Title() template.HTML
} /*}}}*/

type List struct { /*{{{*/
	Content []Lister
} /*}}}*/

type entry struct {
	storage.Poster
}

func (e entry) Date() template.HTML {
	return template.HTML(e.Poster.Date().Format(storage.TimePattern))
}

//GetFullList get entire posts list
func GetFullList() (*List, error) { /*{{{*/
	results, err := storage.Get()
	if err != nil {
		return nil, err
	}
	//sorted by date
	sort.Sort(results)

	l := &List{
		Content: make([]Lister, 0, len(results.Content)),
	}
	for _, v := range results.Content {
		var e interface{} = entry{v}
		if lister, ok := e.(Lister); ok {
			l.Content = append(l.Content, lister)
		}
	}
	if len(l.Content) == 0 {
		return nil, errors.New("There is no posts. Maybe is generating... Refresh after a while")
	}
	return l, nil
} /*}}}*/

type Post struct { /*{{{*/
	Content []Poster
} /*}}}*/

type postKey string

//Implement keyer
func (pk postKey) Key() string { /*{{{*/
	return string(pk)
} /*}}}*/

func GetPost(key string) (*Post, error) { /*{{{*/
	results, err := storage.Get(postKey(key))
	if err != nil {
		return nil, err
	}
	p := &Post{
		Content: make([]Poster, len(results.Content)),
	}
	for i, v := range results.Content {
		p.Content[i] = v.(Poster)
	}
	return p, nil
} /*}}}*/

func GetStaticReader(key, path string) (io.Reader, error) { /*{{{*/
	results, err := storage.Get(postKey(key))
	if err != nil {
		return nil, err
	}
	return results.Content[0].Static(path), nil
} /*}}}*/
