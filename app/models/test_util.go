package models

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
)

var (
	pathNotFound error
)

func init() {
	if runtime.GOOS == "windows" {
		pathNotFound = syscall.ENOTDIR
	} else {
		pathNotFound = syscall.ENOENT
	}
}

func matchError(expect, real error) error { /*{{{*/
	if expect != real {
		if expect == nil {
			return fmt.Errorf("expect err(nil), but get err(%s)\n", real.Error())
		}
		if real == nil {
			return fmt.Errorf("expect err(%s), but get err(nil)\n", expect.Error())
		}
		if strings.Contains(real.Error(), expect.Error()) {
			return nil
		}
		return fmt.Errorf("expect err(%s), but get err(%s)\n",
			expect.Error(), real.Error())
	}
	return nil
} /*}}}*/
