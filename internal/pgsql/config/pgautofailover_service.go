/*
@Author : WuWeiJian
@Date : 2023-10-16
*/

package config

import (
	"fmt"

	"gopkg.in/ini.v1"
)

// Config represent the data to generate systemd config
type AutoPGService struct {
	Cfg                *ini.File
	WorkingDirectory   string
	DataPath           string
	User               string
	ServiceProcessName string
}

func NewAutoPGService(template string) (*AutoPGService, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowShadows:             true,
		SpaceBeforeInlineComment: true,
	}, template)
	if err != nil {
		return &AutoPGService{}, fmt.Errorf("加载ini文件(%s)失败: %v", template, err)
	}
	return &AutoPGService{Cfg: cfg}, err
}

func (s *AutoPGService) FormatBody() error {
	s.Cfg.Section("Service").Key("WorkingDirectory").SetValue(s.WorkingDirectory)
	s.Cfg.Section("Service").Key("Environment").SetValue(fmt.Sprintf("PGDATA=%s", s.DataPath))
	s.Cfg.Section("Service").Key("User").SetValue(s.User)
	s.Cfg.Section("Service").Key("ExecStart").SetValue(fmt.Sprintf("%s  run", s.ServiceProcessName))
	s.Cfg.Section("Service").Key("ExecReload").SetValue(fmt.Sprintf("%s  reload", s.ServiceProcessName))
	return nil
}

func (s *AutoPGService) SaveTo(filename string) error {
	return s.Cfg.SaveTo(filename)
}
