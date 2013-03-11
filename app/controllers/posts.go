package controllers

import (
	"github.com/robfig/revel"
	"github.com/tw4452852/totorow/app/models"
	"html/template"
	"runtime"
	"sort"
	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

type PostsPlugin struct {
	revel.EmptyPlugin
}

func (p PostsPlugin) OnAppStart() {
	models.Init()
}

func init() {
	//register posts plugin
	revel.RegisterPlugin(PostsPlugin{})

}

//Poster represent a post
type Poster interface { /*{{{*/
	Content() template.HTML
} /*}}}*/

//Lister represent a list entry
type Lister interface { /*{{{*/
	models.Keyer
	Date() template.HTML
	Title() template.HTML
} /*}}}*/

//Releaser release a reference
type Releaser interface { /*{{{*/
	Release() string
} /*}}}*/

type List struct { /*{{{*/
	Free    Releaser
	Content []Lister
} /*}}}*/

//List satisfy sort.Interface
func (l *List) Len() int {
	return len(l.Content)
}

func (l *List) Less(i, j int) bool {
	ti, _ := time.Parse(models.TimePattern, string(l.Content[i].Date()))
	tj, _ := time.Parse(models.TimePattern, string(l.Content[j].Date()))
	return ti.After(tj)
}

func (l *List) Swap(i, j int) {
	l.Content[i], l.Content[j] = l.Content[j], l.Content[i]
}

//GetFullList get entire posts list
func GetFullList() (*List, error) { /*{{{*/
	results, err := models.Get()
	if err != nil {
		return nil, err
	}
	l := &List{
		Content: make([]Lister, len(results.Content)),
		Free:    results,
	}
	for i, v := range results.Content {
		l.Content[i] = v.(Lister)
	}
	sort.Sort(l)
	return l, nil
} /*}}}*/

type Post struct { /*{{{*/
	Free    Releaser
	Content []Poster
} /*}}}*/

type postKey string

//Implement keyer
func (pk postKey) Key() string { /*{{{*/
	return string(pk)
} /*}}}*/

func GetPost(key string) (*Post, error) { /*{{{*/
	results, err := models.Get(postKey(key))
	if err != nil {
		return nil, err
	}
	p := &Post{
		Free:    results,
		Content: make([]Poster, len(results.Content)),
	}
	for i, v := range results.Content {
		p.Content[i] = v.(Poster)
	}
	return p, nil
} /*}}}*/
