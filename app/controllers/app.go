package controllers

import "github.com/robfig/revel"

type Application struct {
	*revel.Controller
}

func (c Application) Index() revel.Result {
	c.RenderArgs["list"] = storage.list.records
	return c.Render()
}
func (c Application) Posts(fileName string) revel.Result {
	data, ok := storage.articles.Get(fileName)
	if !ok {
		return c.NotFound("Can't find article " + fileName)
	}
	c.RenderArgs["data"] = data
	c.RenderArgs["title"] = fileName
	return c.Render()
}
