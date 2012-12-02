package controllers

import (
	"github.com/robfig/revel"
	"os/exec"
)

var name string

type DataPlugin struct {
	rev.EmptyPlugin
}

func (d DataPlugin) OnAppStart() {
	o, _ := exec.Command("pwd").CombinedOutput()
	name = string(o)
}

func init() {
	rev.RegisterPlugin(DataPlugin{})
}
