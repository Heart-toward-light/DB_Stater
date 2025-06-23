// Created by LiuSainan on 2022-06-14 11:01:02

package config

import (
	"dbup/internal/utils/command"
	"fmt"
	"path/filepath"

	"gopkg.in/ini.v1"
)

type MariaDBService struct {
	Cfg *ini.File
}

func NewMariaDBService(template string) (*MariaDBService, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowShadows:             true,
		SpaceBeforeInlineComment: true,
	}, template)
	if err != nil {
		return &MariaDBService{}, fmt.Errorf("加载ini文件(%s)失败: %v", template, err)
	}
	return &MariaDBService{Cfg: cfg}, err
}

func (s *MariaDBService) FormatBody(option *MariaDBOptions) error {
	startup := fmt.Sprintf("%s --defaults-file=%s --basedir=%s --datadir=%s --plugin-dir=%s --log-error=%s --open-files-limit=65535 --pid-file=%s --socket=%s --port=%d",
		filepath.Join(option.Dir, "bin", "mariadbd"),
		filepath.Join(option.Dir, "config", DefaultMariaDBConfigFile),
		option.Dir,
		filepath.Join(option.Dir, "data"),
		filepath.Join(option.Dir, "lib", "plugin"),
		filepath.Join(option.Dir, "logs", "error.log"),
		filepath.Join(option.Dir, "data", "mariadb.pid"),
		fmt.Sprintf("/tmp/.mariadb%d.sock", option.Port),
		option.Port)
	description := fmt.Sprintf("MariaDB %s database server", DefaultMariaDBVersion)
	s.Cfg.Section("Unit").Key("Description").SetValue(description)

	s.Cfg.Section("Service").Key("User").SetValue(option.SystemUser)
	s.Cfg.Section("Service").Key("Group").SetValue(option.SystemGroup)
	s.Cfg.Section("Service").Key("ExecStart").SetValue(startup)

	//  systemctl service 添加 libjemalloc.so 如下示例:
	// Environment          = LD_PRELOAD=/data1/mariadb3327/lib/el7/amd64/libjemalloc.so.1
	os, arch, jemalloc, err := command.GetOsArchInfo()
	if err != nil {
		return fmt.Errorf("获取操作系统,cpu架构信息发生错误: %v", err)
	}
	jemallocPath := fmt.Sprintf("LD_PRELOAD=%s", filepath.Join(option.Dir, "lib", os, arch, jemalloc))
	s.Cfg.Section("Service").Key("Environment").SetValue(jemallocPath)

	return nil
}

func (s *MariaDBService) GaleraFormatBody(option *MariaDBOptions) error {
	startup := fmt.Sprintf("%s --defaults-file=%s --basedir=%s --datadir=%s --plugin-dir=%s --log-error=%s --open-files-limit=65535 --pid-file=%s --socket=%s --port=%d --wsrep-new-cluster",
		filepath.Join(option.Dir, "bin", "mariadbd"),
		filepath.Join(option.Dir, "config", DefaultMariaDBConfigFile),
		option.Dir,
		filepath.Join(option.Dir, "data"),
		filepath.Join(option.Dir, "lib", "plugin"),
		filepath.Join(option.Dir, "logs", "error.log"),
		filepath.Join(option.Dir, "data", "mariadb.pid"),
		fmt.Sprintf("/tmp/.mariadb%d.sock", option.Port),
		option.Port)
	description := fmt.Sprintf("MariaDB %s database server", DefaultMariaDBVersion)

	s.Cfg.Section("Unit").Key("Description").SetValue(description)
	s.Cfg.Section("Service").Key("User").SetValue(option.SystemUser)
	s.Cfg.Section("Service").Key("Group").SetValue(option.SystemGroup)
	s.Cfg.Section("Service").Key("ExecStart").SetValue(startup)

	return nil
}

func (s *MariaDBService) SaveTo(filename string) error {
	return s.Cfg.SaveTo(filename)
}
