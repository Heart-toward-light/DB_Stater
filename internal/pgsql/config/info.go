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
type PgsqlInfo struct {
	Port      int    `ini:"port"`
	Host      string `ini:"host"`
	Socket    string `ini:"socket"`
	Username  string `ini:"username"`
	Password  string `ini:"password"`
	Database  string `ini:"database"`
	DeployDir string `ini:"deploydir"`
	DataDir   string `ini:"datadir"`
}

// SaveTo 将info信息保存到磁盘
func (p *PgsqlInfo) SlaveTo(filename string) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true})
	if err := ini.ReflectFrom(cfg, p); err != nil {
		return fmt.Errorf("部署配置映射到(%s)文件错误: %v", filename, err)
	}
	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("部署配置保存到(%s)文件错误: %v", filename, err)
	}
	return nil
}
