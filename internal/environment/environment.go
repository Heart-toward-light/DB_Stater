/*
@Author : WuWeiJian
@Date : 2020-12-03 14:09
*/

package environment

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mitchellh/go-homedir"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// 磁盘好像暂时还没用
type Environment struct {
	GOOS         string
	GOARCH       string
	Memory       *mem.VirtualMemoryStat
	Disk         int64
	TmpPath      string
	HomePath     string
	ProgramPath  string
	Program      string
	CurrentPath  string
	DbupInfoPath string
	HostInfo     *host.InfoStat
}

func NewEnvironment() (*Environment, error) {
	env := &Environment{
		GOOS:    runtime.GOOS,
		GOARCH:  runtime.GOARCH,
		TmpPath: "/tmp",
	}

	if hi, err := host.Info(); err != nil {
		return env, err
	} else {
		env.HostInfo = hi
	}

	// if env.GOOS == "linux" {
	// 	if u, err := user.Current(); err == nil {
	// 		if u.Username != "root" {
	// 			return env, fmt.Errorf("必须以root用户执行\n")
	// 		}
	// 	} else {
	// 		return env, err
	// 	}
	// }

	if err := env.SetMemory(); err != nil {
		return env, err
	}
	if err := env.SetHomePath(); err != nil {
		return env, err
	}

	if p, err := os.Executable(); err != nil {
		return env, err
	} else {
		env.Program = p
	}
	env.ProgramPath = filepath.Dir(env.Program)

	if err := env.SetCurrentPath(); err != nil {
		return env, err
	}
	env.DbupInfoPath = filepath.Join(env.HomePath, ".dbup")
	return env, nil
}

func (e *Environment) SetMemory() error {
	var err error
	if e.Memory, err = mem.VirtualMemory(); err != nil {
		return fmt.Errorf("获取机器内存信息失败: %v", err)
	}
	return nil
}

func (e *Environment) SetHomePath() error {
	var err error
	if e.HomePath, err = homedir.Dir(); err != nil {
		return fmt.Errorf("获取当前用户家目录失败: %v", err)
	}
	return nil
}

//func (e *Environment) SetProgramPath() error {
//	var err error
//	if e.ProgramPath, err = utils.GetExecDir(); err != nil {
//		return fmt.Errorf("获取执行程序目录失败: %v", err)
//	}
//	return nil
//}

func (e *Environment) SetCurrentPath() error {
	var err error
	if e.CurrentPath, err = os.Getwd(); err != nil {
		return fmt.Errorf("获取当前目录失败: %v", err)
	}
	return nil
}

func IsWindows() bool {
	return GlobalEnv().GOOS == "windows"
}
