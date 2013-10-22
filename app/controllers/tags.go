package controllers

import (
	"github.com/tw4452852/storage"
	"html/template"
)

type TagEntry struct {
	Name  template.HTML
	Count int
}

type Tags struct {
	Content []*TagEntry
}

func NewTags() *Tags {
	return &Tags{make([]*TagEntry, 0)}
}

func (t *Tags) Add(name template.HTML) {
	for i, tag := range t.Content {
		if name == tag.Name {
			t.Content[i].Count++
			return
		}
	}
	t.Content = append(t.Content, &TagEntry{name, 1})
}

func GetTags(all *storage.Result) *Tags {
	tags := NewTags()
	if all == nil || len(all.Content) == 0 {
		return tags
	}
	for _, p := range all.Content {
		for _, t := range p.Tags() {
			tags.Add(template.HTML(t))
		}
	}
	return tags
}
