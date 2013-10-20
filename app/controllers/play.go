package controllers

import (
	"github.com/robfig/revel"
	"io"
	"net/http"
)

const baseURL = "http://play.golang.org"

type PlayResult struct{}

func (PlayResult) Apply(req *revel.Request, resp *revel.Response) {
	defer req.Body.Close()
	url := baseURL + req.URL.Path
	r, err := http.DefaultClient.Post(url, req.Header.Get("Content-type"), req.Body)
	if err != nil {
		revel.ErrorResult{Error: err}.Apply(req, resp)
		return
	}
	defer r.Body.Close()
	_, err = io.Copy(resp.Out, r.Body)
	if err != nil {
		revel.ErrorResult{Error: err}.Apply(req, resp)
		return
	}
}
