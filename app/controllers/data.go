package controllers

import (
	"github.com/robfig/revel"
	"os"
	"path/filepath"
	"time"
)

type Record struct {
	Date string
	Name string
}

//file infos db
var records []*Record

type DataPlugin struct {
	rev.EmptyPlugin
}

func (d DataPlugin) OnAppStart() {
	//assue data dir is in ../
	const topDir = "./data/"
	if err := filepath.Walk(topDir, func(path string, info os.FileInfo, err error) error {
		if path == topDir {
			return nil
		}
		if info.IsDir() {
			return filepath.SkipDir
		}
		records = append(records, &Record{info.ModTime().Format(time.ANSIC), info.Name()})
		return nil
	}); err != nil {
		rev.ERROR.Printf("walk %q failed: %s\n", topDir, err)
	}
}

func init() {
	rev.RegisterPlugin(DataPlugin{})
}
