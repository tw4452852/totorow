package controllers

import "github.com/robfig/revel"

type Application struct {
	*rev.Controller
}

func (c Application) Index() rev.Result {
	c.RenderArgs["list"] = storage.list.records
	return c.Render()
}
func (c Application) Posts(fileName string) rev.Result {
	data, ok := storage.articles.Get(fileName)
	if !ok {
		return c.NotFound("Can't find article " + fileName)
	}
	c.RenderArgs["data"] = data
	c.RenderArgs["title"] = fileName
	return c.Render()
}
