package models

import (
	"encoding/xml"
	"log"
	"os"
	"path/filepath"
)

type Config struct { /*{{{*/
	Type string `xml:"type"`
	Root string `xml:"root"`
} /*}}}*/

type Configs struct { /*{{{*/
	Content []Config `xml:"repo"`
} /*}}}*/

func getConfig(path string) (*Configs, error) { /*{{{*/
	file, err := os.Open(path)
	if err != nil {
		log.Printf("open config file error: %s\n", err)
		return nil, err
	}
	defer file.Close()
	decoder := xml.NewDecoder(file)
	cfg := new(Configs)
	if err := decoder.Decode(cfg); err != nil {
		log.Printf("parse config file error: %s\n", err)
		return nil, err
	}

	//filter the empty repo
	//and join the $GOPATH to the rel local root path
	clean := make([]int, 0)
	for i, c := range cfg.Content {
		if c.Type == "" || c.Root == "" {
			clean = append(clean, i)
			continue
		}
		if c.Type == "local" && !filepath.IsAbs(c.Root) {
			cfg.Content[i].Root = filepath.Join(os.Getenv("GOPATH"), c.Root)
		}
	}
	if len(clean) > 0 {
		cc := cfg.Content
		for numDeleted, i := range clean {
			index := i - numDeleted
			if index < len(cc)-1 {
				copy(cc[index:], cc[index+1:])
			}
			cc = cc[:len(cc)-1]
		}
		cfg.Content = cc
	}
	return cfg, nil
} /*}}}*/
