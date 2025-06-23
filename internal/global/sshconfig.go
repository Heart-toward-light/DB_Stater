// Created by LiuSainan on 2021-11-24 15:48:38

package global

import (
	"dbup/internal/utils"
	"fmt"
)

type SSHConfig struct {
	Host     string
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	KeyFile  string `yaml:"keyfile"`
	TmpDir   string `yaml:"tmp-dir"`
}

func (o *SSHConfig) Validator() error {
	if err := utils.IsIPv4(o.Host); err != nil {
		if !utils.IsHostName(o.Host) {
			return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败\n", o.Host, err)
		}
	}
	// 端口
	if o.Port < 1 || o.Port > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", o.Port)
	}
	return nil
}
