package controllers

import "github.com/robfig/revel"

type Application struct {
	*rev.Controller
}

func (c Application) Index() rev.Result {
	list := storage.list
	return c.Render(list)
}
func (c Application) Posts(fileName string) rev.Result {
	data, ok := storage.articles.Get(fileName)
	if !ok {
		return c.NotFound("Can't find article " + fileName)
	}
	return c.Render(data)
}
