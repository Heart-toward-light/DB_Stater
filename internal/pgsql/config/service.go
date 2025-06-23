/*
@Author : WuWeiJian
@Date : 2020-12-09 15:56
*/

package config

import (
	"fmt"
	"gopkg.in/ini.v1"
)

// Config represent the data to generate systemd config
type PostgresService struct {
	Cfg                *ini.File
	Description        string
	User               string
	Group              string
	ServiceProcessName string
	DataPath           string
	LibPath            string
	Version            string
}

func NewPostgresService(template string) (*PostgresService, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowShadows:             true,
		SpaceBeforeInlineComment: true,
	}, template)
	if err != nil {
		return &PostgresService{}, fmt.Errorf("加载ini文件(%s)失败: %v", template, err)
	}
	return &PostgresService{Cfg: cfg}, err
}

func (s *PostgresService) FormatBody() error {
	var e struct {
		Environment []string `ini:"Environment,,allowshadow"`
	}
	e.Environment = append(e.Environment, fmt.Sprintf("PGDATA=%s", s.DataPath), fmt.Sprintf("LD_LIBRARY_PATH=%s", s.LibPath), "PG_OOM_ADJUST_FILE=/proc/self/oom_score_adj", "PG_OOM_ADJUST_VALUE=0")

	s.Cfg.Section("Unit").Key("Description").SetValue(s.Description)
	s.Cfg.Section("Service").Key("User").SetValue(s.User)
	if err := s.Cfg.Section("Service").ReflectFrom(&e); err != nil { //导入多项 Environment
		return err
	}
	s.Cfg.Section("Service").Key("ExecStart").SetValue(fmt.Sprintf("%s start -D ${PGDATA}", s.ServiceProcessName))
	s.Cfg.Section("Service").Key("ExecStop").SetValue(fmt.Sprintf("%s stop -D ${PGDATA}", s.ServiceProcessName))
	return nil
}

func (s *PostgresService) SaveTo(filename string) error {
	return s.Cfg.SaveTo(filename)
}
