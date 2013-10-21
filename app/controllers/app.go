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
