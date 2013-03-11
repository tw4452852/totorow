package models

//Init init the dataCenter and repositories
func Init() { /*{{{*/
	dataCenter = &storage{
		requestCh: make(chan *request),
		data:      make(map[string]interface{}),
	}
	go dataCenter.serve()
	initRepos()
} /*}}}*/

const (
	TitleAndDateSeperator = "|"
	TimePattern           = "2006-01-02"
	//get repo config file
	//Must be: $GOPATH/src/github.com/tw4452852/totorow/conf/repos.xml
	ConfigPath = "src/github.com/tw4452852/totorow/conf/repos.xml"
)

//Releaser release a reference
type Releaser interface { /*{{{*/
	Release() string
} /*}}}*/
