package models

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

const repoRoot = "./testdata/localRepo/"

func TestLocalSetup(t *testing.T) { /*{{{*/
	cases := []struct {
		root   string
		expect error
	}{
		{
			"./testdata/noexist/",
			pathNotFound,
		},
		{
			"./testdata/localRepo.file",
			errors.New("you can't specify a file as a repo root"),
		},
		{
			"./testdata/localRepo/",
			nil,
		},
	}

	for _, c := range cases {
		lr := NewLocalRepo(c.root)
		if e := matchError(c.expect, lr.Setup()); e != nil {
			t.Error(e)
		}
		r := lr.(*localRepo)
		if r.root != c.root {
			t.Errorf("expect repo root(%s), but get(%s)\n",
				c.root, r.root)
		}
	}
} /*}}}*/

func TestLocalRepo(t *testing.T) { /*{{{*/
	repo := NewLocalRepo(repoRoot)
	if err := repo.Setup(); err != nil {
		t.Fatal(err)
	}
	repo.Uninstall()
	lr := repo.(*localRepo)
	cases := []struct {
		prepare func()
		check   func()
	}{
		{
			prepare: func() {
				os.Create(repoRoot + "1.md")
				os.Create(repoRoot + "2.mkd")
				os.Create(repoRoot + "3")
				os.Mkdir(repoRoot+"level1/", 0777)
				os.Create(repoRoot + "level1/" + "1.md")
				os.Create(repoRoot + "level1/" + "2.mkd")
				os.Create(repoRoot + "level1/" + "3")
			},
			check: func() {
				defer func() {
					os.RemoveAll(repoRoot + "level1/")
					os.Remove(repoRoot + "1.md")
					os.Remove(repoRoot + "2.mkd")
					os.Remove(repoRoot + "3")
				}()
				expect := map[string]*localPost{
					"1.md": &localPost{path: repoRoot + "1.md"},
					"level1/1.md": &localPost{path: repoRoot + "level1/" +
						"1.md"},
				}
				lr.update()
				if err := checkLocalPosts(expect, lr.posts); err != nil {
					t.Error(err)
				}
			},
		},

		{
			prepare: func() {
				os.Create(repoRoot + "1.md")
				os.Mkdir(repoRoot+"level1", 0777)
				lr.posts["1.md"] = &localPost{path: repoRoot + "1.md"}
				lr.posts["2.md"] = &localPost{path: repoRoot + "2.md"}
				lr.posts["level1/1.md"] = &localPost{path: repoRoot + "level1/" + "1.md"}
			},
			check: func() {
				defer func() {
					os.Remove(repoRoot + "1.md")
					os.Remove(repoRoot + "level1")
				}()
				expect := map[string]*localPost{
					"1.md": &localPost{path: repoRoot + "1.md"},
				}
				lr.clean()
				if err := checkLocalPosts(expect, lr.posts); err != nil {
					t.Error(err)
				}
			},
		},
	}

	for _, c := range cases {
		if c.prepare != nil {
			c.prepare()
		}
		if c.check != nil {
			c.check()
		}
	}
} /*}}}*/

func checkLocalPosts(expect, real map[string]*localPost) error { /*{{{*/
	if len(real) != len(expect) {
		return fmt.Errorf("length of posts isn't equal: expect %d but get %d\n",
			len(expect), len(real))
	}
	for k, v := range expect {
		if *v != *real[k] {
			return fmt.Errorf("expect post %v but get %v\n", *v, *real[k])
		}
	}
	return nil
} /*}}}*/

func TestLocalPostUpdate(t *testing.T) { /*{{{*/
	type Expect struct {
		path, title, date, content string
	}
	type Case struct {
		prepare   func()
		clean     func()
		path      string
		updateErr error
		expect    *Expect
	}
	cases := []*Case{
		{
			nil,
			nil,
			"./testdata/noexist/1.md",
			pathNotFound,
			nil,
		},
		{
			func() {
				ioutil.WriteFile(repoRoot+"1.md",
					[]byte("hello world | 2012-12-01 \n# title hello world \n"), 0777)
			},
			func() {
				os.Remove(repoRoot + "1.md")
			},
			repoRoot + "1.md",
			nil,
			&Expect{
				path:    repoRoot + "1.md",
				title:   "hello world",
				date:    "2012-12-01",
				content: "<h1>title hello world</h1>\n",
			},
		},
		{
			func() {
				ioutil.WriteFile(repoRoot+"1.md",
					[]byte("hello world | 2012-12-01 \n "), 0777)
			},
			func() {
				os.Remove(repoRoot + "1.md")
			},
			repoRoot + "1.md",
			nil,
			&Expect{
				path:    repoRoot + "1.md",
				title:   "hello world",
				date:    "2012-12-01",
				content: "",
			},
		},
		{
			func() {
				ioutil.WriteFile(repoRoot+"1.md",
					[]byte(" hello world | 2012-12-01"), 0777)
			},
			func() {
				os.Remove(repoRoot + "1.md")
			},
			repoRoot + "1.md",
			errors.New("generateAll: there must be at least one line"),
			nil,
		},
		{
			func() {
				ioutil.WriteFile(repoRoot+"1.md",
					[]byte(" hello world & 2012-12-01\n"), 0777)
			},
			func() {
				os.Remove(repoRoot + "1.md")
			},
			repoRoot + "1.md",
			errors.New("generateAll: can't find seperator"),
			nil,
		},
		{
			func() {
				ioutil.WriteFile(repoRoot+"1.md",
					[]byte(" hello world || 2012-12-01\n"), 0777)
			},
			func() {
				os.Remove(repoRoot + "1.md")
			},
			repoRoot + "1.md",
			errors.New("parsing time"),
			nil,
		},
	}

	runCase := func(c *Case) error {
		if c.clean != nil {
			defer c.clean()
		}
		if c.prepare != nil {
			c.prepare()
		}
		lp := newLocalPost(c.path)
		if err := matchError(c.updateErr, lp.Update()); err != nil {
			return err
		}
		if c.updateErr != nil && c.expect == nil {
			return nil
		}
		real := &Expect{
			path:    lp.path,
			title:   string(lp.Title()),
			date:    string(lp.Date()),
			content: string(lp.Content()),
		}
		if *real != *c.expect {
			return fmt.Errorf("expect %v, but get %v\n", *c.expect, *real)
		}
		return nil
	}

	for i, c := range cases {
		if err := runCase(c); err != nil {
			t.Errorf("case %d error: %s\n", i, err)
		}
	}
} /*}}}*/
