package controllers

import (
	"github.com/tw4452852/storage"
	"runtime"
	"sort"
	"strings"
	"sync"
)

var workers = runtime.NumCPU()

// Filter return the Posters than contain `require`
// search sequence: title, tag, content
func Filter(all *storage.Result, require string) *storage.Result {
	if all == nil {
		return nil
	}
	originCount := len(all.Content)
	if originCount == 0 {
		return all
	}
	in, out := make(chan storage.Poster, originCount), make(chan storage.Poster, originCount)
	// fill the input channel
	for _, p := range all.Content {
		in <- p
	}
	close(in)
	// start workers
	waiter := new(sync.WaitGroup)
	for i := 0; i < workers; i++ {
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			work(in, out, require)
		}()
	}
	// collect the result
	waiter.Wait()
	resultCount := len(out)
	r := make([]storage.Poster, resultCount)
	for i := 0; i < resultCount; i++ {
		r[i] = <-out
	}
	ret := &storage.Result{r}
	sort.Sort(ret)
	return ret
}

func work(in <-chan storage.Poster, out chan<- storage.Poster, require string) {
	for p := range in {
		if strings.Contains(string(p.Title()), require) {
			out <- p
			continue
		}
		// TODO: tag
		if strings.Contains(string(p.Content()), require) {
			out <- p
			continue
		}

	}
}
