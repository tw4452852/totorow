package controllers

import (
	"github.com/robfig/revel"
	"github.com/tw4452852/storage"
	"html/template"
	"runtime"
	"strings"
	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	revel.TemplateFuncs["formatTime"] = func(t time.Time) template.HTML {
		return template.HTML(t.Format(storage.TimePattern))
	}
	revel.TemplateFuncs["join"] = func(ss []string) template.HTML {
		return template.HTML(strings.Join(ss, " "))
	}
	revel.TemplateFuncs["highlight"] = func(search string, input template.HTML) template.HTML {
		inputS := string(input)
		index := strings.Index(inputS, search)
		if index == -1 {
			return input
		}
		r := inputS[:index] + "<span class=highlight>" + search + "</span>" +
			inputS[index+len(search):]
		return template.HTML(r)
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
