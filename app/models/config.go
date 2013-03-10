package models

import (
	"encoding/xml"
	"log"
	"os"
)

type Config struct {
	Type string `xml:"type"`
	Root string `xml:"root"`
}

type Configs struct {
	Content []Config `xml:"repo"`
}

func getConfig(path string) (*Configs, error) {
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
	clean := make([]int, 0)
	for i, c := range cfg.Content {
		if c.Type == "" || c.Root == "" {
			clean = append(clean, i)
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
}
