package controllers

import (
	"github.com/robfig/revel"
	"github.com/russross/blackfriday"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Article struct {
	Title string
	Data  string
}

//articles db
type articleDB map[string]*Article

func NewArticleDB() articleDB {
	return make(map[string]*Article)
}

func (a articleDB) init(topDir string) error {
	for _, post := range posts {
		data, err := generateHTML(topDir + post.Name)
		if err != nil {
			return err
		}
		a[post.Name] = &Article{TrimSuffix(post.Name), data}
	}
	return nil
}

func generateHTML(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(blackfriday.MarkdownCommon(data)), err
}

type Post struct {
	Date string
	Name string
}

//file meta infos db
type metaDB []*Post

func NewMetaDB() metaDB {
	//prepare 10 entries
	return make([]*Post, 0, 10)
}

func (m *metaDB) init(topDir string) error {
	if err := filepath.Walk(topDir, func(path string, info os.FileInfo, err error) error {
		if path == topDir {
			return nil
		}
		if info.IsDir() {
			return filepath.SkipDir
		}
		rev.INFO.Printf("metadb add a file: %q\n", path)
		*m = append(*m, &Post{info.ModTime().Format(time.ANSIC), info.Name()})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

var (
	posts    metaDB
	articles articleDB
)

type PostPlugin struct {
	rev.EmptyPlugin
}

func (d PostPlugin) OnAppStart() {
	//init metadb
	//assume posts in $GOPATH/src/totorow/app/posts/"
	gopath := os.Getenv("GOPATH")
	topDir := gopath + "/src/totorow/app/posts/"
	posts = NewMetaDB()
	if err := posts.init(topDir); err != nil {
		rev.ERROR.Printf("init posts failed: path=%q err=%s\n", topDir, err)
		return
	}

	//init articleDb
	articles = NewArticleDB()
	if err := articles.init(topDir); err != nil {
		rev.ERROR.Printf("init articles failed: err=%s\n", err)
		return
	}
}

//for tempelate map func
func TrimSuffix(path string) string {
	return strings.TrimRight(path, filepath.Ext(path))
}

func init() {
	//register post plugin
	rev.RegisterPlugin(PostPlugin{})

	//register TrimSuffix template func
	rev.TemplateFuncs["trim"] = TrimSuffix
}
