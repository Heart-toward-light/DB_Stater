/*
@Author : WuWeiJian
@Date : 2020-12-02 21:37
*/

package command

import (
	"bufio"
	"dbup/internal/utils"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Connection struct {
	Host         string
	Port         int
	User         string
	Password     string
	KeyFile      string
	SudoPassword string
	auth         []ssh.AuthMethod
	clientConfig *ssh.ClientConfig
	sshClient    *ssh.Client
	*sftp.Client
}

func NewConnection(host, user, password string, port int, timeout int64) (*Connection, error) {
	// fmt.Println(password)
	// fmt.Println(len(password))
	var err error
	conn := &Connection{Host: host, Port: port, User: user, Password: password}
	conn.auth = make([]ssh.AuthMethod, 0)
	conn.auth = append(conn.auth, ssh.Password(password))
	conn.clientConfig = &ssh.ClientConfig{
		User:            user,
		Auth:            conn.auth,
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
		Timeout:         time.Duration(timeout) * time.Second,
	}

	address := fmt.Sprintf("%s:%d", conn.Host, conn.Port)

	if conn.sshClient, err = ssh.Dial("tcp", address, conn.clientConfig); err != nil {
		return nil, err
	}

	if conn.Client, err = sftp.NewClient(conn.sshClient); err != nil {
		return nil, err
	}

	return conn, nil
}

func NewConnectionUseKeyFile(host, user, keyfile string, port int, timeout int64) (*Connection, error) {
	conn := &Connection{Host: host, Port: port, User: user, KeyFile: keyfile}
	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	conn.auth = make([]ssh.AuthMethod, 0)
	conn.auth = append(conn.auth, ssh.PublicKeys(signer))
	conn.clientConfig = &ssh.ClientConfig{
		User:            user,
		Auth:            conn.auth,
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
		Timeout:         time.Duration(timeout) * time.Second,
	}

	address := fmt.Sprintf("%s:%d", conn.Host, conn.Port)

	if conn.sshClient, err = ssh.Dial("tcp", address, conn.clientConfig); err != nil {
		return nil, err
	}

	if conn.Client, err = sftp.NewClient(conn.sshClient); err != nil {
		return nil, err
	}

	return conn, nil
}

func (conn *Connection) Run(cmd string) ([]byte, error) {
	cmd = fmt.Sprintf("PATH=$PATH:/usr/bin:/usr/sbin %s", cmd)
	var in io.WriteCloser
	var out io.Reader

	session, err := conn.sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,      // disable echoing
		ssh.TTY_OP_ISPEED: 144000, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 144000, // output speed = 14.4kbaud
	}

	err = session.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		return nil, err
	}

	if in, err = session.StdinPipe(); err != nil {
		return nil, err
	}

	if out, err = session.StdoutPipe(); err != nil {
		return nil, err
	}

	var output []byte

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		watcher(in, out, &output, conn.SudoPassword)
	}()

	if _, err := session.Output(cmd); err != nil {
		return nil, err
	}
	wg.Wait()
	return output, nil
}

func (conn *Connection) Sudo(cmd, user, password string) ([]byte, error) {
	var sudoStr string
	if user != "" {
		sudoStr = " -u " + user
	}
	conn.SudoPassword = password
	if conn.SudoPassword == "" {
		conn.SudoPassword = conn.Password
	}

	cmd = fmt.Sprintf("sudo -S -H %s /bin/bash -c \"cd; %s\"", sudoStr, cmd)
	return conn.Run(cmd)
}

func (conn *Connection) Scp(source, target string) error {
	// 如果source是文件, target是文件, 直接调用copy
	// 如果source是文件, target是目录, target 需要加文件名: path.Join(target, path.Base(source))
	// 如果source是目录, target是文件, 报错,目标不是一个路径
	// 如果source是目录, target是目录, 直接遍历copy, 如果target不存在,则自动创建。 如果存在,则创建下一级与 path.Join(target, path.Base(source)) 同名的目录(如果存在, 则报错)

	// 通过ssh协议传输文件的目标机器全都是linux系统, 所以将目标路径强制转换为Linux格式
	target = filepath.ToSlash(target)

	if source == "" {
		return fmt.Errorf("源文件不能为空\n")
	}
	if target == "" {
		return fmt.Errorf("目标路径不能为空\n")
	}
	if !utils.IsExists(source) {
		return fmt.Errorf("文件 %s 不存在\n", source)
	}

	if !utils.IsDir(source) {
		if conn.IsDir(target) {
			target = filepath.ToSlash(path.Join(target, path.Base(filepath.ToSlash(source))))
		}
		return conn.Copy(source, target)
	}

	if !conn.IsExists(target) {
		return conn.LoopCopy(source, target)
	}

	if !conn.IsDir(target) {
		return fmt.Errorf("远程已经存在同名文件: %s\n", target)
	}

	target = filepath.ToSlash(path.Join(target, path.Base(filepath.ToSlash(source))))
	return conn.LoopCopy(source, target)
}

func (conn *Connection) singleCopy(source, target string, path string, info os.FileInfo) error {
	relative, err := filepath.Rel(source, path)
	if err != nil {
		fmt.Println("获取相对路径失败: ", err)
	}

	if info.IsDir() {
		if err := conn.MkdirAll(filepath.ToSlash(filepath.Join(target, relative))); err != nil {
			return err
		}
	} else {
		dir, _ := filepath.Split(filepath.Join(target, relative))
		if err := conn.MkdirAll(filepath.ToSlash(dir)); err != nil {
			return err
		}
		if err := conn.Copy(path, filepath.ToSlash(filepath.Join(target, relative))); err != nil {
			return err
		}
	}
	return err
}

func (conn *Connection) LoopCopy(source, target string) error {
	if err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return conn.singleCopy(source, target, path, info)
	}); err != nil {
		return err
	}
	return nil
}

func (conn *Connection) Copy(source, target string) error {
	var sf *os.File
	var df *sftp.File
	var err error

	// 通过ssh协议传输文件的目标机器全都是linux系统, 所以将目标路径强制转换为Linux格式
	target = filepath.ToSlash(target)

	if sf, err = os.Open(source); err != nil {
		return err
	}
	defer sf.Close()

	if df, err = conn.Create(target); err != nil {
		return err
	}
	defer df.Close()
	if _, err = df.ReadFrom(sf); err != nil {
		return err
	}
	return nil
}

func (conn *Connection) IsExists(path string) bool {
	_, err := conn.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func (conn *Connection) IsDir(path string) bool {
	s, err := conn.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func (conn *Connection) IsEmpty(path string) (bool, error) {
	fs, err := conn.ReadDir(path)
	if err != nil {
		return false, err
	}
	if len(fs) == 0 {
		return true, nil
	}
	return false, err
}

func (conn *Connection) Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (conn *Connection) Hostsanalysis(hostnamelist []string) error {
	file, err := conn.Open("/etc/hosts")
	if err != nil {
		return fmt.Errorf("无法打开/etc/hosts文件：%v \n ", err)
	}
	defer file.Close()

	// 创建一个Scanner来逐行读取文件内容
	scanner := bufio.NewScanner(file)
	hlist := []string{}
	// 逐行检查域名解析
	for scanner.Scan() {
		line := scanner.Text()
		// 跳过注释行和空行
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue // 跳过无效行
		}
		ip := fields[0]
		hostname := fields[1]
		hlist = append(hlist, hostname)
		addrs, err := net.LookupHost(hostname)
		if err != nil {
			fmt.Printf("无法解析域名 %s: %v\n", hostname, err)
			continue
		}

		if !conn.Contains(addrs, ip) {
			return fmt.Errorf("域名 %s 解析到的IP地址与期望的地址 %s 不一致\n", hostname, ip)
		}
	}

	for _, hn := range hostnamelist {
		if hn != "" {
			if !conn.Contains(hlist, hn) {
				return fmt.Errorf("域名 %s 在主机 %s 的 /etc/hosts 文件未配置解析\n", hn, conn.Host)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取 /etc/hosts 文件时发生错误: %v", err)
	}

	return nil
}
