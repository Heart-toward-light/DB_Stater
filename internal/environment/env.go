/*
@Author : WuWeiJian
@Date : 2020-12-23 16:25
*/

package environment

import (
	"fmt"
	"os/user"
)

var _env *Environment

// SetGlobalEnv the global env used.
func SetGlobalEnv(env *Environment) {
	_env = env
}

// GlobalEnv Get the global env used.
func GlobalEnv() *Environment {
	return _env
}

func MustRoot() error {
	if GlobalEnv().GOOS == "linux" {
		if u, err := user.Current(); err == nil {
			if u.Uid != "0" {
				return fmt.Errorf("必须以root用户执行")
			}
		} else {
			return err
		}
	}
	return nil
}
