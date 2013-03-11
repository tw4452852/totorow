package models

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

//Repository represent a repostory
type Repository interface { /*{{{*/
	//used for setup a repository
	Setup() error
	//used for uninstall a repostory
	Uninstall()
} /*}}}*/

//used for udpate a post in a repository
type Updater interface { /*{{{*/
	Update() error
} /*}}}*/

//used for Init a repository with a root path
type InitFunction func(root string) Repository

var supportedRepoTypes = make(map[string]InitFunction)

//RegisterRepoType register a support repository type
//If there is one, just update it
func RegisterRepoType(key string, f InitFunction) { /*{{{*/
	supportedRepoTypes[key] = f
} /*}}}*/

//UnregisterRepoType unregister a support repository type
func UnregisterRepoType(key string) { /*{{{*/
	delete(supportedRepoTypes, key)
} /*}}}*/

type repos map[string]Repository

func (rs repos) refresh(cfg *Configs) { /*{{{*/
	refreshed := make(map[string]bool)
	for key := range rs {
		refreshed[key] = false
	}

	for _, c := range cfg.Content {
		kind := c.Type
		root := c.Root
		key := kind + "-" + root
		_, found := rs[key]
		if !found {
			if initF, supported := supportedRepoTypes[kind]; supported {
				repo := initF(root)
				if err := repo.Setup(); err != nil {
					log.Printf("add repo: setup failed with err(%s)\n", err)
					continue
				}
				log.Printf("add a repo(%q)\n", key)
				rs[key] = repo
			} else {
				log.Printf("add repo: type(%s) isn't supported yet\n",
					kind)
			}
			continue
		}
		refreshed[key] = true
	}

	//uninstall the repos that have been remove
	for key, exist := range refreshed {
		if !exist {
			rs[key].Uninstall()
			delete(rs, key)
		}
	}
} /*}}}*/

var repositories repos

func initRepos() { /*{{{*/
	repositories = make(repos)
	go checkConfig(repositories)
} /*}}}*/

func checkConfig(r repos) { /*{{{*/
	//refresh every 10s
	timer := time.NewTicker(10 * time.Second)
	for _ = range timer.C {
		cfg, err := getConfig(filepath.Join(os.Getenv("GOPATH"), ConfigPath))
		if err != nil {
			//if there is some error(e.g. file doesn't exist) while reading
			//config file, just skip this refresh
			continue
		}
		r.refresh(cfg)
	}
	panic("not reach")
} /*}}}*/
