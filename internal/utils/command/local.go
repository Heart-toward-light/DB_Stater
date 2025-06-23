/*
@Author : WuWeiJian
@Date : 2020-12-02 18:23
*/

package command

import (
	"bytes"
	"context"
	"dbup/internal/utils"
	"fmt"
	"os/exec"
	"time"
)

// Local execute the command at local host.
type Local struct {
	Timeout int
	User    string // sudo 用户名。 默认为空时, 执行sudo不传-u参数, 以默认root执行
	Locale  string // the locale used when executing the command
}

func (l *Local) Run(cmd string) ([]byte, []byte, error) {
	// set a basic PATH in case it's empty on login
	cmd = fmt.Sprintf("PATH=$PATH:/usr/bin:/usr/sbin %s", cmd)

	if l.Locale != "" {
		cmd = fmt.Sprintf("export LANG=%s; %s", l.Locale, cmd)
	}

	if l.Timeout == 0 {
		l.Timeout = 60
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(l.Timeout)*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, "/bin/sh", "-c", cmd)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command.Stdout = stdout
	command.Stderr = stderr

	err := command.Run()

	if err != nil {
		return stdout.Bytes(), stderr.Bytes(), err
	}

	return stdout.Bytes(), stderr.Bytes(), nil
}

func (l *Local) Sudo(cmd string) ([]byte, []byte, error) {
	var sudoStr string
	if l.User != "" {
		sudoStr = " -u " + l.User
	}
	cmd = fmt.Sprintf("sudo -S -H %s /bin/bash -l -c \"cd; %s\"", sudoStr, cmd)
	return l.Run(cmd)
}

func (l *Local) WinRun(cmd string) ([]byte, []byte, error) {
	if l.Timeout == 0 {
		l.Timeout = 60
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(l.Timeout)*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, "cmd", "/c", cmd)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command.Stdout = stdout
	command.Stderr = stderr

	err := command.Run()

	stdoutBytes, _ := utils.GbkToUtf8(stdout.Bytes())
	stderrBytes, _ := utils.GbkToUtf8(stderr.Bytes())

	if err != nil {
		return stdoutBytes, stderrBytes, err
	}

	return stdoutBytes, stderrBytes, nil
}
