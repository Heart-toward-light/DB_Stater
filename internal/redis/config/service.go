/*
@Author : WuWeiJian
@Date : 2021-01-06 11:39
*/

package config

import (
	"fmt"

	"gopkg.in/ini.v1"
)

type RedisService struct {
	Cfg                *ini.File
	User               string
	PidFile            string
	ServiceProcessName string
	ConfigFile         string
	RedisCli           string
	Port               int
	Password           string
}

func NewRedisService(template string) (*RedisService, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowShadows:             true,
		IgnoreInlineComment:      true,
		SpaceBeforeInlineComment: true,
	}, template)
	if err != nil {
		return &RedisService{}, fmt.Errorf("加载ini文件(%s)失败: %v", template, err)
	}
	return &RedisService{Cfg: cfg}, err
}

func (s *RedisService) FormatBody() {
	s.Cfg.Section("Service").Key("User").SetValue(s.User)
	s.Cfg.Section("Service").Key("PIDFile").SetValue(s.PidFile)
	s.Cfg.Section("Service").Key("ExecStart").SetValue(fmt.Sprintf("%s %s", s.ServiceProcessName, s.ConfigFile))
	s.Cfg.Section("Service").Key("ExecStop").SetValue(fmt.Sprintf("%s -p %d -a %s shutdown", s.RedisCli, s.Port, s.Password))
}

func (s *RedisService) SaveTo(filename string) error {
	return s.Cfg.SaveTo(filename)
}
