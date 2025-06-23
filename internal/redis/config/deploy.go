/*
@Author : WuWeiJian
@Date : 2021-04-13 11:09
*/

package config

import (
	"dbup/internal/environment"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type Server struct {
	Master   string `ini:"master"`
	Slaves   string `ini:"slaves"`
	SshPort  int    `ini:"ssh-port"`
	User     string `ini:"ssh-user"`
	Password string `ini:"ssh-password"`
	KeyFile  string `ini:"ssh-keyfile"`
	TmpDir   string `ini:"tmp-dir"`
}

func (s *Server) SetDefault() {
	if s.TmpDir == "" {
		s.TmpDir = DeployTmpDir
	}

	if s.Password == "" && s.KeyFile == "" {
		s.KeyFile = filepath.Join(environment.GlobalEnv().HomePath, ".ssh", "id_rsa")
	}
}

// 验证配置
func (s *Server) Validator() error {
	logger.Infof("验证 server 参数\n")
	if err := utils.IsIPv4(s.Master); err != nil {
		if !utils.IsHostName(s.Master) {
			return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", s.Master, err)
		}
	}

	slaves := strings.Split(s.Slaves, ",")
	if len(slaves) == 0 {
		return fmt.Errorf("从库不能为空")
	}
	if len(slaves) > 2 {
		return fmt.Errorf("暂时只支持最多两个从节点")
	}
	for _, slave := range slaves {
		if err := utils.IsIPv4(slave); err != nil {
			if !utils.IsHostName(slave) {
				return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败\n", slave, err)
			}
		}
	}

	// 端口
	if s.SshPort < 1 || s.SshPort > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", s.SshPort)
	}
	//if utils.PortInUse(s.SshPort) {
	//	return fmt.Errorf("端口号被占用: %d", s.SshPort)
	//}

	return nil
}

type Parameter struct {
	Server     Server     `ini:"server"`
	Redis      Parameters `ini:"redis"`
	Yes        bool       `ini:"yes" comment:"监听IP，如果没有特殊要求请勿修改"`
	NoRollback bool       `ini:"no-rollback" comment:"监听IP，如果没有特殊要求请勿修改"`
}

// Load 从配置文件加载配置到Prepare实例
func (p *Parameter) Load(filename string) error {
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

// SaveTo 将Parameter实例数据写入配置文件
func (p *Parameter) SlaveTo(filename string) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true})
	if err := ini.ReflectFrom(cfg, p); err != nil {
		return fmt.Errorf("部署配置映射到(%s)文件错误: %v", filename, err)
	}
	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("部署配置保存到(%s)文件错误: %v", filename, err)
	}
	return nil
}

// Load 从配置文件加载配置到Prepare实例
func (p *Parameter) Validator() error {
	if err := p.Server.Validator(); err != nil {
		return err
	}
	// if err := p.Redis.Validator(); err != nil {
	// 	return err
	// }
	return nil
}
