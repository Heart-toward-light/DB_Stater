/*
@Author : WuWeiJian
@Date : 2021-09-17 15:29
*/

package config

import (
	"fmt"
	"path/filepath"

	"gopkg.in/ini.v1"
)

// Config represent the data to generate systemd config
type MongoDBService struct {
	Cfg *ini.File
}

func NewMongoDBService(template string) (*MongoDBService, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowShadows:             true,
		SpaceBeforeInlineComment: true,
	}, template)
	if err != nil {
		return &MongoDBService{}, fmt.Errorf("加载ini文件(%s)失败: %v", template, err)
	}
	return &MongoDBService{Cfg: cfg}, err
}

func (s *MongoDBService) FormatBody(option *MongodbOptions, user, group string) error {
	var e struct {
		Environment []string `ini:"Environment,,allowshadow"`
	}
	e.Environment = append(e.Environment, fmt.Sprintf("\"OPTIONS=-f %s\"", filepath.Join(option.Dir, DefaultMongoDBConfigDir, DefaultMongoDBConfigFile)))

	s.Cfg.Section("Service").Key("User").SetValue(user)
	s.Cfg.Section("Service").Key("Group").SetValue(group)
	if err := s.Cfg.Section("Service").ReflectFrom(&e); err != nil { //导入多项 Environment
		return err
	}
	s.Cfg.Section("Service").Key("ExecStart").SetValue(fmt.Sprintf("%s $OPTIONS", filepath.Join(option.Dir, DefaultMongoDBBinDir, DefaultMongoDBBinFile)))
	s.Cfg.Section("Service").Key("PIDFile").SetValue(filepath.Join(option.Dir, "mongod.pid"))
	return nil
}

func (s *MongoDBService) FormatMongosBody(option *MongosOptions, user, group string) error {
	var e struct {
		Environment []string `ini:"Environment,,allowshadow"`
	}
	e.Environment = append(e.Environment, fmt.Sprintf("\"OPTIONS=-f %s\"", filepath.Join(option.Dir, DefaultMongoDBConfigDir, DefaultMongoSConfigFile)))

	s.Cfg.Section("Service").Key("User").SetValue(user)
	s.Cfg.Section("Service").Key("Group").SetValue(group)
	if err := s.Cfg.Section("Service").ReflectFrom(&e); err != nil { //导入多项 Environment
		return err
	}
	s.Cfg.Section("Service").Key("ExecStart").SetValue(fmt.Sprintf("%s $OPTIONS", filepath.Join(option.Dir, DefaultMongoDBBinDir, DefaultMongoSBinFile)))
	// s.Cfg.Section("Service").Key("PIDFile").SetValue(filepath.Join(option.Dir, "mongod.pid"))
	return nil
}

func (s *MongoDBService) SaveTo(filename string) error {
	return s.Cfg.SaveTo(filename)
}
