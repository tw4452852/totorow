package controllers

import (
	"github.com/tw4452852/storage"
	"io"
	"sort"
)

//GetFullList get entire posts list
func GetFullList() (*storage.Result, error) { /*{{{*/
	results, err := storage.Get()
	if err != nil {
		return nil, err
	}
	//sorted by date
	sort.Sort(results)
	return results, nil
} /*}}}*/

type postKey string

//Implement keyer
func (pk postKey) Key() string { /*{{{*/
	return string(pk)
} /*}}}*/

func GetPost(key string) (*storage.Result, error) { /*{{{*/
	result, err := storage.Get(postKey(key))
	if err != nil {
		return nil, err
	}
	return result, nil
} /*}}}*/

func GetStaticReader(key, path string) (io.Reader, error) { /*{{{*/
	results, err := storage.Get(postKey(key))
	if err != nil {
		return nil, err
	}
	return results.Content[0].Static(path), nil
} /*}}}*/
