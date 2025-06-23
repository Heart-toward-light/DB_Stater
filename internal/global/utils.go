/*
@Author : WuWeiJian
@Date : 2020-12-22 17:27
*/

package global

import (
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v2"
)

// 获取文件中记录的md5值
func GetMd5(filename, kind, packageName string) (string, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		SpaceBeforeInlineComment: true,
	}, filename)
	if err != nil {
		return "", fmt.Errorf("读取md5文件失败: %v", err)
	}

	s := cfg.Section(kind).Key(packageName).MustString("")
	return s, nil
}

// CheckPackage 检查tar.gz包是否存在, 并验证md5值
func CheckPackage(path, packageFullName, kinds string) error {
	logger.Infof("检查安装包文件: %s\n", packageFullName)

	packageName := filepath.Base(packageFullName)
	if !utils.IsExists(packageFullName) {
		return fmt.Errorf("在指定路径(%s)未找到安装包, 停止安装", packageFullName)
	}

	md5Filename := filepath.Join(path, PackagePath, Md5FileName)
	md5Vlues, err := GetMd5(md5Filename, kinds, packageName)
	if err != nil {
		return err
	}

	m, err := utils.CheckMd5sumByFile(packageFullName)
	if err != nil {
		return err
	}

	CurrentVlues := utils.CheckMd5sumByByte([]byte(Salt + m + "\n"))
	if CurrentVlues != md5Vlues {
		return fmt.Errorf("安装包md5值验证不正确: \n检测当前配置md5值为(%s) \n检测当前包实际md5值为(%s)", md5Vlues, CurrentVlues)
	}
	return nil
}

// INILoadFromFile 从INI配置文件加载配置到结构体
func INILoadFromFile(filename string, config interface{}, iniConfig ini.LoadOptions) error {
	cfg, err := ini.LoadSources(iniConfig, filename)
	if err != nil {
		return fmt.Errorf("加载ini文件(%s)失败: %v", filename, err)
	}

	if err = cfg.MapTo(config); err != nil {
		return fmt.Errorf("将ini文件(%s)映射到结构体对象(%t)失败: %v", filename, config, err)
	}
	return nil
}

// INISaveToFile 保存结构体数据到操作系统INI配置文件
func INISaveToFile(filename string, config interface{}) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true}) //AllowNestedValues: true 允许嵌套值,应该没用
	if err := ini.ReflectFrom(cfg, config); err != nil {
		return fmt.Errorf("结构体对象(%t)映射到ini对象(%s) 错误: %v", config, filename, err)
	}
	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("对象(%t)保存到(%s)文件错误: %v", config, filename, err)
	}
	return nil
}

// YAMLLoadFromFile 从 YAML 配置文件加载配置到结构体
func YAMLLoadFromFile(filename string, config interface{}) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(content, config)
}

// YAMLSaveToFile 保存结构体数据到操作系统 YAML 配置文件
func YAMLSaveToFile(filename string, config interface{}) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0777)
}

type OsVersion struct {
	OSName  string `ini:"ID"`
	Version string `ini:"VERSION_ID"`
}

// 检查系统版本
func OsLoad() (OsVersion, error) {
	var p OsVersion
	cfg, err := ini.LoadSources(ini.LoadOptions{
		SpaceBeforeInlineComment: true,
	}, "/etc/os-release")
	if err != nil {
		return p, fmt.Errorf("加载配置文件失败: %v", err)
	}

	if err = cfg.MapTo(&p); err != nil {
		return p, fmt.Errorf("配置文件映射到结构体失败: %v", err)
	}
	return p, nil
}

// 判断是否为 x86 架构
func Osamd() bool {
	arch := runtime.GOARCH
	if arch == "386" || arch == "amd64" {
		return true
	}
	return false
}

type MissSoLibrariesfile struct {
	Name   string
	Info   string
	Repair string
}

func Checkldd(c string) (ms []MissSoLibrariesfile, err error) {
	cmd := fmt.Sprintf("ldd %s", c)
	l := command.Local{}
	stdout, stderr, err := l.Run(cmd)
	if err != nil {
		return ms, fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}

	for _, line := range strings.Split(string(stdout), "\n") {
		if !strings.Contains(line, "not found") {
			continue
		}

		var m MissSoLibrariesfile
		m.Info = line
		for k, v := range MissSoLibrariesAndRepairPlanList {
			if strings.Contains(line, k) {
				m.Name = k
				m.Repair = v
				break
			}
		}
		ms = append(ms, m)
	}

	return ms, nil
}
