package models

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func init() {
	Init()
}

type entry struct {
	data string
}

//implement Keyer interface
func (e *entry) Key() string {
	return e.data
}

type invalidEntry struct {
	data string
}

type testCase struct {
	prepare func() error
	input   []interface{}
	err     error
	checker func(r *Result) error
}

var (
	noKeyerErr = errors.New("arg is not a keyer")
	noFound    = errors.New("can't find want you want")

	ents = []*entry{
		&entry{"1"},
		&entry{"1"},
		&entry{"2"},
	}
	inents = []*invalidEntry{
		&invalidEntry{"1"},
		&invalidEntry{"1"},
		&invalidEntry{"2"},
	}
)

func matchError(expect, real error, t *testing.T) { /*{{{*/
	if expect != real {
		if expect == nil {
			t.Errorf("expect err(nil), but get err(%s)\n", real.Error())
			return
		}
		if real == nil {
			t.Errorf("expect err(%s), but get err(nil)\n", expect.Error())
			return
		}
		if strings.Contains(real.Error(), expect.Error()) {
			return
		}
		t.Errorf("expect err(%s), but get err(%s)\n",
			expect.Error(), real.Error())
		return
	}
	return
} /*}}}*/

func TestAdd(t *testing.T) { /*{{{*/
	cases := []testCase{
		//add
		{
			nil,
			[]interface{}{ents[0]},
			nil,
			func(*Result) error {
				if dataCenter.find(ents[0].data) != ents[0] {
					return errors.New("add valid one failed\n")
				}
				return nil
			},
		},
		{
			nil,
			[]interface{}{inents[0]},
			noKeyerErr,
			func(*Result) error {
				if dataCenter.find(inents[0].data) != nil {
					return errors.New("add invalid one failed\n")
				}
				return nil
			},
		},
		{
			nil,
			[]interface{}{inents[0], ents[0]},
			noKeyerErr,
			func(*Result) error {
				if dataCenter.find(inents[1].data) != nil {
					return errors.New("invalid+valid add: invaled is found\n")
				}
				if dataCenter.find(ents[0].data) != nil {
					return errors.New("invalid+valid add: valied is found\n")
				}
				return nil
			},
		},
		{
			nil,
			[]interface{}{ents[0], inents[0]},
			noKeyerErr,
			func(*Result) error {
				if dataCenter.find(ents[0].data) != ents[0] {
					return errors.New("valied+valid add: valied is not found\n")
				}
				if dataCenter.find(inents[0].data) == inents[0] {
					return errors.New("valied+valid add: invalied is found\n")
				}
				return nil
			},
		},
	}
	for _, c := range cases {
		dataCenter.reset()
		if c.prepare != nil {
			if err := c.prepare(); err != nil {
				t.Fatal(err)
			}
		}
		matchError(c.err, Add(c.input...), t)
		if c.checker != nil {
			if err := c.checker(nil); err != nil {
				t.Fatal(err)
			}
		}
	}
} /*}}}*/

func TestUpdate(t *testing.T) { /*{{{*/
	cases := []testCase{
		//update
		{
			nil,
			[]interface{}{ents[0], ents[1]},
			nil,
			func(*Result) error {
				if dataCenter.find(ents[0].data) != ents[1] {
					return errors.New("update valid+valid: not update\n")
				}
				return nil
			},
		},
		{
			nil,
			[]interface{}{inents[0], inents[1]},
			noKeyerErr,
			func(*Result) error {
				if dataCenter.find(inents[0].data) != nil {
					return errors.New("update invalid+invalid: find first\n")
				}
				if dataCenter.find(inents[1].data) != nil {
					return errors.New("update invalid+invalid: find second\n")
				}
				return nil
			},
		},
		{
			nil,
			[]interface{}{inents[0], ents[1]},
			noKeyerErr,
			func(*Result) error {
				if dataCenter.find(inents[0].data) != nil {
					return errors.New("update invalid+valid: find first\n")
				}
				if dataCenter.find(ents[1].data) != nil {
					return errors.New("update invalid+valid: find second\n")
				}
				return nil
			},
		},
		{
			nil,
			[]interface{}{ents[0], inents[1]},
			noKeyerErr,
			func(*Result) error {
				if dataCenter.find(ents[0].data) != ents[0] {
					return errors.New("update valid+invalid: can't find first\n")
				}
				if dataCenter.find(inents[1].data) == inents[1] {
					return errors.New("update valid+invalid: find second\n")
				}
				return nil
			},
		},
	}
	for _, c := range cases {
		dataCenter.reset()
		if c.prepare != nil {
			if err := c.prepare(); err != nil {
				t.Fatal(err)
			}
		}
		matchError(c.err, Add(c.input...), t)
		if c.checker != nil {
			if err := c.checker(nil); err != nil {
				t.Fatal(err)
			}
		}
	}
} /*}}}*/

func TestRemove(t *testing.T) { /*{{{*/
	cases := []testCase{
		//remove
		{
			func() error {
				if err := Add(ents[0], ents[1], ents[2]); err != nil {
					return err
				}
				return nil
			},
			[]interface{}{ents[0]},
			nil,
			func(*Result) error {
				if dataCenter.find(ents[0].data) != nil {
					return errors.New("remove exist one: not remove\n")
				}
				if dataCenter.find(ents[2].data) != ents[2] {
					return errors.New("remove exist one: remove another\n")
				}
				return nil
			},
		},
		{
			nil,
			[]interface{}{ents[0]},
			nil,
			func(*Result) error {
				if dataCenter.find(ents[0].data) != nil {
					return errors.New("remove no exist one: not remove\n")
				}
				return nil
			},
		},
	}
	for _, c := range cases {
		dataCenter.reset()
		if c.prepare != nil {
			if err := c.prepare(); err != nil {
				t.Fatal(err)
			}
		}
		matchError(c.err, Remove(c.input...), t)
		if c.checker != nil {
			if err := c.checker(nil); err != nil {
				t.Fatal(err)
			}
		}
	}
} /*}}}*/

func TestGet(t *testing.T) { /*{{{*/
	type Releaser interface {
		Release()
	}

	cases := []testCase{
		//get
		{
			func() error {
				if err := Add(ents[0], ents[1], ents[2]); err != nil {
					return err
				}
				return nil
			},
			[]interface{}{},
			nil,
			func(r *Result) error {
				if len(r.Content) != 2 {
					return fmt.Errorf("get all: result len(%d) != expect(%d)\n",
						len(r.Content), 2)
				}
				if err := compareTwo(ents[1:], r.Content); err != nil {
					return err
				}

				done := make(chan bool, 1)
				go func() {
					dataCenter.waiter.Wait()
					done <- true
				}()
				var ri interface{} = r
				if rr, ok := ri.(Releaser); ok {
					rr.Release()
					if <-done != true {
						return errors.New("get all: wait failed\n")
					}
				} else {
					return errors.New("get all: result is not a releaser\n")
				}
				return nil
			},
		},

		{
			func() error {
				if err := Add(ents[0], ents[1]); err != nil {
					return err
				}
				return nil
			},
			[]interface{}{ents[0]},
			nil,
			func(r *Result) error {
				if len(r.Content) != 1 {
					return fmt.Errorf("get some: result len(%d) != expect(%d)\n",
						len(r.Content), 1)
				}
				if r.Content[0] != ents[1] {
					return noFound
				}
				return nil
			},
		},

		{
			func() error {
				if err := Add(ents[0], ents[1]); err != nil {
					return err
				}
				return nil
			},
			[]interface{}{ents[0], ents[1]},
			nil,
			func(r *Result) error {
				if len(r.Content) != 2 {
					return fmt.Errorf("get some: result len(%d) != expect(%d)\n",
						len(r.Content), 2)
				}
				if r.Content[0] != ents[1] || r.Content[1] != ents[1] {
					return noFound
				}
				return nil
			},
		},

		{
			func() error {
				if err := Add(ents[0], ents[1]); err != nil {
					return err
				}
				return nil
			},
			[]interface{}{ents[0], ents[2]},
			noFound,
			func(r *Result) error {
				if r != nil {
					return errors.New("add some: result should be nil\n")
				}
				return nil
			},
		},
	}
	for _, c := range cases {
		dataCenter.reset()
		if c.prepare != nil {
			if err := c.prepare(); err != nil {
				t.Fatal(err)
			}
		}
		result, err := Get(c.input...)
		matchError(c.err, err, t)
		if c.checker != nil {
			if err := c.checker(result); err != nil {
				t.Fatal(err)
			}
		}
	}
} /*}}}*/

func compareTwo(expects []*entry, reals []interface{}) error { /*{{{*/
check:
	for _, expect := range expects {
		for _, real := range reals {
			if expect == real {
				continue check
			}
		}
		return fmt.Errorf("get all: expect %v not in result\n",
			expect)
	}
	return nil
} /*}}}*/
