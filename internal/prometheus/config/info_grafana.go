/*
@Author : WuWeiJian
@Date : 2020-12-22 20:37
*/

package config

import (
	"fmt"

	"gopkg.in/ini.v1"
)

// info 信息
type GrafanaInfo struct {
	Port        int    `ini:"port"`
	User        string `ini:"user"`
	Password    string `ini:"password"`
	InstallPath string `ini:"install_path"`
}

// SaveTo 将info信息保存到磁盘
func (p *GrafanaInfo) SaveTo(filename string) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true})
	if err := ini.ReflectFrom(cfg, p); err != nil {
		return fmt.Errorf("部署配置映射到(%s)文件错误: %v", filename, err)
	}
	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("部署配置保存到(%s)文件错误: %v", filename, err)
	}
	return nil
}
