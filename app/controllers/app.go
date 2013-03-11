package controllers

import (
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
