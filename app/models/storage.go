package models

import (
	"errors"
	"sync"
)

var dataCenter *storage

type storage struct { /*{{{*/
	requestCh chan *request //for outcoming request
	closeCh   chan bool     //for exit

	waiter sync.WaitGroup         //count the reference
	data   map[string]interface{} //internal data storage
} /*}}}*/

func (d *storage) serve() { /*{{{*/
	for {
		select {
		case req := <-d.requestCh:
			d.handleRequest(req)
		case <-d.closeCh:
			return
		}
	}
} /*}}}*/

//Keyer used for generate the key in the map
type Keyer interface { /*{{{*/
	Key() string
} /*}}}*/

func (d *storage) handleRequest(req *request) { /*{{{*/
	loopArgs := func(action func(key string, arg interface{}) error) error {
		for _, arg := range req.args {
			//Only accept the things implement keyer
			if keyer, ok := arg.(Keyer); ok {
				key := keyer.Key()
				if err := action(key, arg); err != nil {
					return err
				}
			} else {
				return errors.New("arg is not a keyer")
			}
		}
		return nil
	}

	switch req.cmd {
	case ADD:
		req.err <- loopArgs(func(key string, arg interface{}) error {
			if _, found := d.data[key]; found {
				//wait previous reference done
				d.waiter.Wait()
			}
			//add or update it, here only myself refer the map
			d.data[key] = arg
			return nil
		})
		return
	case REMOVE:
		req.err <- loopArgs(func(key string, arg interface{}) error {
			if _, found := d.data[key]; found {
				//wait previous reference done
				d.waiter.Wait()
			}
			//remove it, here only myself refer the map
			delete(d.data, key)
			return nil
		})
		return
	case GET:
		content := make([]interface{}, 0)
		err := loopArgs(func(key string, arg interface{}) error {
			if v, found := d.data[key]; found {
				content = append(content, v)
				return nil
			}
			//not found
			return errors.New("can't find want you want")
		})

		//some internal error
		if err != nil {
			req.err <- err
			return
		}

		//get all
		if len(content) == 0 {
			for _, v := range d.data {
				content = append(content, v)
			}
		}

		//add a reference
		d.waiter.Add(1)
		req.result <- content
		req.err <- nil
	}
} /*}}}*/

func (d *storage) reset() { /*{{{*/
	d.waiter = sync.WaitGroup{}
	d.data = make(map[string]interface{})
} /*}}}*/

func (d *storage) find(key string) interface{} { /*{{{*/
	if v, found := d.data[key]; found {
		return v
	}
	return nil
} /*}}}*/
