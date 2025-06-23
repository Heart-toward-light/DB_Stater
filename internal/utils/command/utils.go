/*
@Author : WuWeiJian
@Date : 2020-12-18 21:25
*/

package command

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func MoveFile(filename string) error {
	newFilename := path.Clean(filename) + ".bak." + time.Now().Format("20060102150405")
	cmd := fmt.Sprintf("mv %s %s", filename, newFilename)
	l := Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("mv命令操作失败: %s, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func CopyFile(filename string) error {
	newFilename := path.Clean(filename) + ".bak." + time.Now().Format("20060102150405")
	cmd := fmt.Sprintf("cp %s %s", filename, newFilename)
	l := Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("cp命令操作失败: %s, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func CopyDir(emoloyDir string, sourcedir string) error {
	// newFilename := path.Clean(filename) + ".bak." + time.Now().Format("20060102150405")
	cmd := fmt.Sprintf("cp -r %s %s", emoloyDir, sourcedir)
	l := Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("cp命令操作失败: %s, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func MoveDir(emoloyDir string, sourcedir string) error {
	// newFilename := path.Clean(filename) + ".bak." + time.Now().Format("20060102150405")
	cmd := fmt.Sprintf("mv %s %s", emoloyDir, sourcedir)
	l := Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("cp命令操作失败: %s, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func CopyFileDir(filename string, sourcedir string) error {
	// newFilename := path.Clean(filename) + ".bak." + time.Now().Format("20060102150405")
	cmd := fmt.Sprintf("cp %s %s", filename, sourcedir)
	l := Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("cp命令操作失败: %s, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func GetUserInfo(path string) (string, string, error) {
	// 获取目录的元数据信息
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", "", fmt.Errorf("无法获取目录信息: %s", err)
	}

	// 获取属主用户信息
	owner, err := user.LookupId(fmt.Sprint(fileInfo.Sys().(*syscall.Stat_t).Uid))
	if err != nil {
		return "", "", fmt.Errorf("无法获取属主用户信息: %s", err)
	}

	// 获取属组用户信息
	group, err := user.LookupGroupId(fmt.Sprint(fileInfo.Sys().(*syscall.Stat_t).Gid))
	if err != nil {
		return "", "", fmt.Errorf("无法获取属组用户信息: %s", err)
	}

	return owner.Username, group.Name, nil
}

func SystemdReload() error {
	cmd := "systemctl daemon-reload"
	l := Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("daemon-reload失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func SystemCtl(serviceName, action string) error {
	cmd := fmt.Sprintf("systemctl %s %s", action, serviceName)
	l := Local{Timeout: 300}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("执行(%s)失败: %v, 标准错误输出: %s", cmd, err, stderr)
	}
	return nil
}

func SystemResourceLimit(serviceName, limit string) error {
	cmd := fmt.Sprintf("systemctl set-property %s %s", serviceName, limit)
	l := Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("执行(%s)失败: %v, 标准错误输出: %s", cmd, err, stderr)
	}
	return nil
}

func VerifyDir(path string, verify []string) error {
	for _, dir := range verify {
		dirPath := filepath.Join(path, dir)
		_, err := os.Stat(dirPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("目录 %s 下不存在所需要的目录 %s,请检查指定的主路径是否正确", path, dir)
			} else {
				return fmt.Errorf("检查目录 %s 时出错：%v", dirPath, err)
			}
		}
	}
	return nil
}

func IsExists(path string) bool {
	_, err := os.Stat(path)

	if err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			return true
		}
	}

	return true
}

func ReplaceLineWithKeyword(filename string, keyword string, newContent string) error {
	// 打开文件
	file, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建一个临时文件来保存修改后的内容
	tmpfile, err := os.CreateTemp("/usr/lib/systemd/system/", "tempfile")
	if err != nil {
		return err
	}
	defer tmpfile.Close()

	// 读取文件内容并替换包含关键字的行
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, keyword) {
			line = newContent
		}
		fmt.Fprintln(tmpfile, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// 将临时文件内容写回原文件
	if err := os.Rename(tmpfile.Name(), filename); err != nil {
		return err
	}

	return nil
}

// 内存优化 systemctl 添加 libjemalloc
func AppendWithKeyword(filename string, keyword string, newContent string) error {
	// 打开文件
	file, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建一个临时文件来保存修改后的内容
	tmpfile, err := os.CreateTemp("", "tempfile")
	if err != nil {
		return err
	}
	defer tmpfile.Close()

	// 读取文件内容并替换包含关键字的行
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, keyword) {
			line = newContent
		}
		fmt.Fprintln(tmpfile, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// 将临时文件内容写回原文件
	if err := os.Rename(tmpfile.Name(), filename); err != nil {
		return err
	}

	return nil
}

func PGsqlVersion(path string) (ver string, err error) {
	var version string
	cli := filepath.Join(path, "server/bin/postgres")
	if !IsExists(cli) {
		return "", fmt.Errorf("未找到 %s 文件,请确认指定路径是否为安装主路径", cli)
	}

	cmd := fmt.Sprintf("%s --version", cli)
	l := Local{}
	if v, stderr, err := l.Run(cmd); err != nil {
		return "", fmt.Errorf("执行(%s)失败: %v, 标准错误输出: %s", cmd, err, stderr)
	} else {
		// var version string
		version = strings.Split(string(v), " ")[2]
		if strings.Contains(version, "dbup") {
			version = strings.Split(strings.Split(string(v), " ")[2], "dbup")[0]
		}

		version = strings.Replace(version, "\n", "", -1)
		return version, nil
	}
}

func MariadbVersion(mpath string) (ver string, err error) {
	cli := filepath.Join(mpath, "/bin/mariadb")
	if !IsExists(cli) {
		return "", fmt.Errorf("未找到 %s 文件,请确认指定路径是否为安装主路径", cli)
	}

	cmd := fmt.Sprintf("%s --version", cli)
	l := Local{}
	if v, stderr, err := l.Run(cmd); err != nil {
		return "", fmt.Errorf("执行(%s)失败: %v, 标准错误输出: %s", cmd, err, stderr)
	} else {
		v := strings.Split(strings.Split(string(v), " ")[5], "-")[0]
		return v, nil
	}
}

func RedisVersion(rpath string) (ver string, err error) {
	serverdir := filepath.Join(rpath, "/server")
	cli := filepath.Join(rpath, "/server/bin/redis-cli")

	if !IsExists(cli) || !IsExists(serverdir) {
		return "", fmt.Errorf("未找到 %s 相关执行文件,请确认指定路径是否为安装主路径", cli)
	}
	// var ver string
	cmd := fmt.Sprintf("%s --version", cli)
	l := Local{}
	if v, stderr, err := l.Run(cmd); err != nil {
		return "", fmt.Errorf("执行(%s)失败: %v, 标准错误输出: %s", cmd, err, stderr)
	} else {
		v := strings.TrimSuffix(strings.Split(string(v), " ")[1], "\n")
		return v, nil
		// str := strings.ReplaceAll(v, ".", "")
		// ver, err = strconv.ParseFloat(str, 64)
		// if err != nil {
		// 	return 0, fmt.Errorf("%v 转换浮点数报错了: %s", v, err)
		// }
	}

}

func CompareVersion(OldVersion, NewVersion string) int {
	OldParts := strings.Split(OldVersion, ".")
	NewParts := strings.Split(NewVersion, ".")

	for i := 0; i < len(OldParts) && i < len(NewParts); i++ {
		golangPart := OldParts[i]
		redisPart := NewParts[i]

		golangNum := 0
		redisNum := 0

		fmt.Sscanf(golangPart, "%d", &golangNum)
		fmt.Sscanf(redisPart, "%d", &redisNum)

		if golangNum > redisNum {
			return 1
		} else if golangNum < redisNum {
			return -1
		}
	}

	if len(OldParts) > len(NewParts) {
		return 1
	} else if len(OldParts) < len(NewParts) {
		return -1
	}

	return 0
}

// 获取 操作系统 和 cpu架构
func GetOsArchInfo() (string, string, string, error) {
	var (
		os       string
		arch     string
		jemalloc string
	)
	cmd := exec.Command("uname", "-a")
	output, err := cmd.Output()
	if err != nil {
		return os, arch, jemalloc, err
	}
	kernelVersion := strings.TrimSpace(string(output))

	switch {
	case strings.Contains(kernelVersion, "el7"):
		os = "el7"
	case strings.Contains(kernelVersion, "uel20"):
		os = "el7"
	case strings.Contains(kernelVersion, "el8"):
		os = "el8"
	case strings.Contains(kernelVersion, "ubuntu22"):
		os = "ubuntu22"
	case strings.Contains(kernelVersion, "ky10"):
		os = "ky10"
	}

	arch = runtime.GOARCH

	if os == "el7" {
		jemalloc = "libjemalloc.so.1"
	} else {
		jemalloc = "libjemalloc.so.2"
	}

	return os, arch, jemalloc, err
}

// 配置 pg 认证信息
func FlushPGPass(AuthFile string, AuthUserinfo []string) error {
	// 打开文件，如果文件不存在则创建，如果文件已存在则截断文件
	file, err := os.OpenFile(AuthFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建一个带缓冲的写入器
	writer := bufio.NewWriter(file)

	for _, line := range AuthUserinfo {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	// 刷新缓冲区，确保所有数据被写入文件
	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

// 验证系统命令是否存在
func CheckCommandExists(cmdName string) bool {
	cmd := exec.Command("command", "-v", cmdName)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
