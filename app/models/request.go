package models

type cmd int

const (
	ADD cmd = iota
	REMOVE
	GET
)

type request struct { /*{{{*/
	cmd    cmd
	args   []interface{}
	result chan []interface{}
	err    chan error
} /*}}}*/

//Add add something into the dataCenter
//If the things are exist, update it
//Some internal error will be returned
func Add(args ...interface{}) error { /*{{{*/
	r := &request{
		cmd:  ADD,
		args: args,
		err:  make(chan error),
	}
	dataCenter.requestCh <- r
	return <-r.err
} /*}}}*/

//Remove remove something from the dataCenter
//If the things are not exist, do nothing
//Some internal error will be returned
func Remove(args ...interface{}) error { /*{{{*/
	r := &request{
		cmd:  REMOVE,
		args: args,
		err:  make(chan error),
	}
	dataCenter.requestCh <- r
	return <-r.err
} /*}}}*/

//Response for the request
type Result struct { /*{{{*/
	Content []interface{}
} /*}}}*/

//Satisfy the Releaser
//Release the reference
func (r *Result) Release() { /*{{{*/
	dataCenter.waiter.Done()
} /*}}}*/

//Get may get something from the dataCenter
//If you want get sth special, give the filter arg
//Otherwise, get all
//Some internal error will be returned
func Get(args ...interface{}) (*Result, error) { /*{{{*/
	r := &request{
		cmd:    GET,
		args:   args,
		result: make(chan []interface{}, 1),
		err:    make(chan error),
	}
	dataCenter.requestCh <- r
	if err := <-r.err; err != nil {
		return nil, err
	}
	return &Result{<-r.result}, nil
} /*}}}*/
