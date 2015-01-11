package controllers

import (
	"errors"
	"sort"

	"github.com/tw4452852/storage"
	"golang.org/x/tools/blog/atom"
)

var waitErr = errors.New("Is generating... Wait a minute and refresh")

//GetRSS get rss object in atom format
func GetRSS() (interface{}, error) {
	r, err := storage.Get()
	if err != nil {
		return nil, err
	}
	if len(r.Content) == 0 {
		return nil, waitErr
	}

	//sorted by time
	sort.Sort(r)

	//Init a common infos
	feed := &atom.Feed{
		Title: "Tw's blog",
		Link:  []atom.Link{atom.Link{Href: "/rss"}},
		ID:    "/rss",
		Author: &atom.Person{
			Name:  "Tw",
			Email: "tw19881113@gmail.com",
		},
		//use the newest post time
		Updated: atom.Time(r.Content[0].Date()),
	}

	//fill the entries
	feed.Entry = make([]*atom.Entry, len(r.Content))
	for i := range feed.Entry {
		p := r.Content[i]
		feed.Entry[i] = &atom.Entry{
			Title:   string(p.Title()),
			ID:      p.Key(),
			Updated: atom.Time(p.Date()),
			Link:    []atom.Link{atom.Link{Href: "/posts/" + p.Key()}},
			Content: &atom.Text{Body: string(p.Content()), Type: "html"},
		}
	}
	return feed, nil
}
