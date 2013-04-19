package controllers

import (
	"github.com/robfig/revel"
	"github.com/tw4452852/storage"
	"html/template"
	"io"
	"runtime"
	"sort"
)

type PostsPlugin struct {
	revel.EmptyPlugin
}

func (p PostsPlugin) OnAppStart() {
	storage.Init("src/totorow/conf/repos.xml")
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//register posts plugin
	revel.RegisterPlugin(PostsPlugin{})
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
	Free    storage.Releaser
	Content []Lister
} /*}}}*/

//GetFullList get entire posts list
func GetFullList() (*List, error) { /*{{{*/
	results, err := storage.Get()
	if err != nil {
		return nil, err
	}
	//sorted by date
	sort.Sort(results)

	l := &List{
		Content: make([]Lister, len(results.Content)),
		Free:    results,
	}
	for i, v := range results.Content {
		l.Content[i] = v.(Lister)
	}
	return l, nil
} /*}}}*/

type Post struct { /*{{{*/
	Free    storage.Releaser
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
		Free:    results,
		Content: make([]Poster, len(results.Content)),
	}
	for i, v := range results.Content {
		p.Content[i] = v.(Poster)
	}
	return p, nil
} /*}}}*/

type StaticReader struct { /*{{{*/
	storage.Releaser
	io.Reader
} /*}}}*/

//implememt io.Closer
func (sr *StaticReader) Close() error { /*{{{*/
	if v, ok := sr.Reader.(io.Closer); ok {
		v.Close()
	}
	sr.Releaser.Release()
	return nil
} /*}}}*/

func GetStaticReader(key, path string) (*StaticReader, error) { /*{{{*/
	results, err := storage.Get(postKey(key))
	if err != nil {
		return nil, err
	}
	return &StaticReader{
		Releaser: results,
		Reader:   results.Content[0].Static(path),
	}, nil
} /*}}}*/
