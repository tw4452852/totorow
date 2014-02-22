package controllers

import (
	"errors"
	"github.com/robfig/revel"
)

type Application struct {
	*revel.Controller
}

func (c Application) Index() revel.Result {
	l, err := GetFullList()
	if err != nil {
		return c.RenderError(err)
	}
	c.RenderArgs["tags"] = GetTags(l)
	c.RenderArgs["list"] = l
	return c.Render()
}

func (c Application) Posts(key string) revel.Result {
	p, err := GetPost(key)
	if err != nil {
		return c.RenderError(err)
	}
	c.RenderArgs["post"] = p
	return c.Render()
}

func (c Application) Slides(key string) revel.Result {
	p, err := GetPost(key)
	if err != nil {
		return c.RenderError(err)
	}
	c.RenderArgs["post"] = p
	return c.Render()
}

func (c Application) Static(key, path string) revel.Result {
	reader, err := GetStaticReader(key, path)
	if err != nil {
		return c.RenderError(err)
	}
	return &revel.BinaryResult{
		Reader:   reader,
		Delivery: "",
		Length:   -1,
	}
}

func (c Application) RSS() revel.Result {
	rss, err := GetRSS()
	if err != nil {
		return c.RenderError(err)
	}
	return c.RenderXml(rss)
}

// A dump Controller.Action
// just for the play filter's recognization
func (c Application) Play() revel.Result {
	return c.RenderError(errors.New("not reachable!!"))
}

func (c Application) Search() revel.Result {
	search := c.Params.Get("q")
	if search == "" {
		return c.Redirect("/")
	}
	p, err := GetFullList()
	if err != nil {
		return c.RenderError(err)
	}
	c.RenderArgs["search"] = search
	c.RenderArgs["tags"] = GetTags(p)
	c.RenderArgs["list"] = Filter(p, CheckAll(search))
	return c.Render()
}

func (c Application) Tag(tag string) revel.Result {
	if tag == "" {
		return c.Redirect("/")
	}
	p, err := GetFullList()
	if err != nil {
		return c.RenderError(err)
	}
	c.RenderArgs["tag"] = tag
	c.RenderArgs["tags"] = GetTags(p)
	c.RenderArgs["list"] = Filter(p, CheckTags(tag))
	return c.Render()
}
