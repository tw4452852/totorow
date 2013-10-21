package controllers

import (
	"github.com/robfig/revel"
	"github.com/tw4452852/storage"
	"html/template"
	"runtime"
	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	revel.TemplateFuncs["formatTime"] = func(t time.Time) template.HTML {
		return template.HTML(t.Format(storage.TimePattern))
	}
	// forbid sequent handlers for go playground
	playFilter := func(c *revel.Controller, fc []revel.Filter) {
		c.Result = PlayResult{}
		return
	}
	revel.FilterAction(Application.Play).
		Insert(playFilter, revel.BEFORE, revel.ParamsFilter)
	//register posts plugin
	revel.OnAppStart(onStart)

}

func onStart() {
	storage.Init("src/totorow/conf/repos.xml")
}
