package controllers

import (
	"github.com/robfig/revel"
	"github.com/tw4452852/storage"
	"html/template"
	"io"
	"runtime"
	"sort"
	"time"
)

func Init() {
	storage.Init("src/totorow/conf/repos.xml")
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//register posts plugin
	revel.OnAppStart(Init)

	revel.TemplateFuncs["formatTime"] = func(t time.Time) template.HTML {
		return template.HTML(t.Format(storage.TimePattern))
	}
}

//GetFullList get entire posts list
func GetFullList() (*storage.Result, error) { /*{{{*/
	results, err := storage.Get()
	if err != nil {
		return nil, err
	}
	//sorted by date
	sort.Sort(results)
	return results, nil
} /*}}}*/

type postKey string

//Implement keyer
func (pk postKey) Key() string { /*{{{*/
	return string(pk)
} /*}}}*/

func GetPost(key string) (*storage.Result, error) { /*{{{*/
	result, err := storage.Get(postKey(key))
	if err != nil {
		return nil, err
	}
	return result, nil
} /*}}}*/

func GetStaticReader(key, path string) (io.Reader, error) { /*{{{*/
	results, err := storage.Get(postKey(key))
	if err != nil {
		return nil, err
	}
	return results.Content[0].Static(path), nil
} /*}}}*/
