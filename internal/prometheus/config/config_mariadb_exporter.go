package config

import (
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/ini.v1"
)

type MariadbExporterConf struct {
	Port            int    `ini:"port" comment:"监听端口，如果没有特殊要求请勿修改"`
	Dir             string `ini:"dir" comment:"数据部署目录，请确认该目录存在，默认为/opt/exporters，如无特殊要求请勿修改"`
	MariadbAddr     string `ini:"mariadb-addr" comment:"mariadb 实例的地址 127.0.0.1"`
	MariadbUser     string `ini:"mariadb-user" comment:"mariadb 实例的用户"`
	MariadbPassword string `ini:"mariadb-password" comment:"mariadb 实例的密码"`
	MariadbPort     int    `ini:"mariadb-port" comment:"mariadb 实例的端口"`
}

// 确定配置文件位置
func (p *MariadbExporterConf) CfgPath(cfgFile string) (string, error) {
	if cfgFile != "" {
		if !utils.IsExists(cfgFile) {
			return cfgFile, fmt.Errorf("指定的配置文件不存在: %s", cfgFile)
		}
		return cfgFile, nil
	}
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("使用默认配置文件, 获取当前用户家目录失败: %v", err)
	}
	cfgFile = filepath.Join(home, DefaultPrometheusCfgFile)
	return cfgFile, nil
}

// Load 从配置文件加载配置到Prepare实例
func (p *MariadbExporterConf) Load(filename string) error {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		SpaceBeforeInlineComment: true,
	}, filename)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %v", err)
	}

	if err = cfg.MapTo(p); err != nil {
		return fmt.Errorf("配置文件映射到结构体失败: %v", err)
	}
	return nil
}

func (p *MariadbExporterConf) InitArgs() {
	logger.Infof("初始化安装参数\n")

	if p.Port == 0 {
		p.Port = utils.RandomPort(DefaultMariadbExporterPort)
	}

	if p.Dir == "" {
		p.Dir = DefaultExportersDir
	}
}

func (p *MariadbExporterConf) Validator() error {
	logger.Infof("验证参数\n")

	// 端口
	if err := p.validatePort(p.Port); err != nil {
		return err
	}

	if p.MariadbPort < 1025 || p.MariadbPort > 65535 {
		return fmt.Errorf("mariadb 实例端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", p.MariadbPort)
	}

	// 数据目录
	// if err := utils.ValidatorDir(p.Dir); err != nil {
	// 	return err
	// }
	return nil
}

func (p *MariadbExporterConf) validatePort(port int) error {
	// 端口
	if port < 1025 || port > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", port)
	}
	if utils.PortInUse(port) {
		return fmt.Errorf("端口号被占用: %d", port)
	}

	return nil
}
