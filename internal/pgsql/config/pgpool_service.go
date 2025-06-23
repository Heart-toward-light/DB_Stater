/*
@Author : WuWeiJian
@Date : 2021-09-17 14:38
*/

package config

import (
	"fmt"
	"gopkg.in/ini.v1"
)

// Config represent the data to generate systemd config
type PgPoolService struct {
	Cfg *ini.File
}

func NewPgPoolService(template string) (*PgPoolService, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowShadows:             true,
		SpaceBeforeInlineComment: true,
	}, template)
	if err != nil {
		return &PgPoolService{}, fmt.Errorf("加载ini文件(%s)失败: %v", template, err)
	}
	return &PgPoolService{Cfg: cfg}, err
}

// HandleConfig 调整配置
func (s *PgPoolService) HandleConfig(env []string, user, start, stop string) error {
	var e struct {
		Environment []string `ini:"Environment,,allowshadow"`
	}
	e.Environment = env

	s.Cfg.Section("Service").Key("User").SetValue(user)
	if err := s.Cfg.Section("Service").ReflectFrom(&e); err != nil { //导入多项 Environment
		return err
	}
	s.Cfg.Section("Service").Key("ExecStart").SetValue(start)
	s.Cfg.Section("Service").Key("ExecStop").SetValue(stop)
	return nil
}

func (s *PgPoolService) SaveTo(filename string) error {
	return s.Cfg.SaveTo(filename)
}
