package models

import (
	"errors"
	"github.com/russross/blackfriday"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

func init() {
	RegisterRepoType("local", NewLocalRepo)
}

type localRepo struct {
	exitCh chan bool
	root   string
	posts  map[string]*localPost
}

func NewLocalRepo(root string) Repository {
	return &localRepo{
		exitCh: make(chan bool),
		root:   root,
		posts:  make(map[string]*localPost),
	}
}

//implement the Repository interface
func (lr *localRepo) Setup() error {
	//root Must be a dir
	fi, err := os.Stat(lr.root)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return errors.New("you can't specify a file as a repo root")
	}
	go lr.watch()
	return nil
}

func (lr *localRepo) Uninstall() {
	lr.exitCh <- true
}

func (lr *localRepo) watch() {
	timer := time.Tick(1 * time.Second)
	for {
		select {
		case <-lr.exitCh:
			return
		case <-timer:
			//delete the removed files
			lr.clean()
			//add newer post and update the exist post
			lr.update()
		}
	}
}

func (lr *localRepo) clean() {
	cleans := make([]string, 0)
	for relPath := range lr.posts {
		absPath := lr.root + relPath
		_, err := os.Stat(absPath)
		if err != nil && os.IsNotExist(err) {
			cleans = append(cleans, relPath)
		}
	}
	for _, relPath := range cleans {
		delete(lr.posts, relPath)
	}
}

func (lr *localRepo) update() {
	if err := filepath.Walk(lr.root, func(path string, info os.FileInfo, err error) error {
		//only watch the special filetype
		if info.IsDir() || !filetypeFilter(path) {
			return nil
		}
		relPath, _ := filepath.Rel(lr.root, path)
		post, found := lr.posts[relPath]
		if !found {
			lr.posts[relPath] = newLocalPost(path)
			return nil
		}
		//update a exist one
		var updater Updater = post
		updater.Update()
		return nil
	}); err != nil {
		log.Printf("updata local repo(%s) error: %s\n",
			lr.root, err)
	}
}

//supported filetype
var filters = []*regexp.Regexp{
	regexp.MustCompile(".*.md$"),
}

//filter file type , return pass
func filetypeFilter(path string) (passed bool) {
	for _, filter := range filters {
		if filter.MatchString(path) {
			return true
		}
	}
	return false
}

//represet a local post
type localPost struct {
	path string

	mutex      sync.RWMutex
	tile       string
	date       time.Time
	content    template.HTML
	lastUpdate time.Time
}

func newLocalPost(path string) *localPost {
	return &localPost{
		path: path,
	}
}

//implement Updater
func (lp *localPost) Update() error {
	file, err := os.Open(lp.path)
	if err != nil {
		return err
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		return err
	}
	if ut := fi.ModTime(); ut.After(lp.lastUpdate) {

	}
	return nil
}

func generateAll(file *os.File) (title string, date time.Time, content
template.HTML, err error) {
	c, e := ioutil.ReadAll(file)
	if e != nil {
		err = e
		return
	}
	firstAndContent := strings.SplitN(string(c), "\n", 1)
	
	template.HTML(blackfriday.MarkdownCommon(data)), err
}

//implement controllers.Poster
