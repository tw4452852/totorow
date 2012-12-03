package controllers

import (
	"github.com/robfig/revel"
	"os"
	"path/filepath"
	"time"
)

type Post struct {
	Date string
	Name string
}

//file infos db
var posts []*Post

type PostPlugin struct {
	rev.EmptyPlugin
}

func (d PostPlugin) OnAppStart() {
	//assue data dir is in ../
	gopath := os.Getenv("GOPATH")
	topDir := gopath + "/src/totorow/app/posts/"
	if err := filepath.Walk(topDir, func(path string, info os.FileInfo, err error) error {
		if path == topDir {
			return nil
		}
		if info.IsDir() {
			return filepath.SkipDir
		}
		rev.INFO.Printf("walk %q\n", path)
		posts = append(posts, &Post{info.ModTime().Format(time.ANSIC), info.Name()})
		return nil
	}); err != nil {
		rev.ERROR.Printf("walk %q failed: %s\n", topDir, err)
	}
}

func init() {
	rev.RegisterPlugin(PostPlugin{})
}
