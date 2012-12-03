package controllers

import "github.com/robfig/revel"

type Application struct {
	*rev.Controller
}

func (c Application) Index() rev.Result {
	return c.Render(posts)
}
func (c Application) Posts(fileName string) rev.Result {
	article, ok := articles[fileName]
	if !ok {
		return c.NotFound("Can't find article " + fileName)
	}
	return c.Render(article)
}
