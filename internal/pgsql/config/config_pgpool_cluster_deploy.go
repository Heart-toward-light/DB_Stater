/*
@Author : WuWeiJian
@Date : 2021-04-25 11:55
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

type PGPoolClusterServer struct {
	Master      string `ini:"pg-master"`
	Slave       string `ini:"pg-slave"`
	PGPools     string `ini:"pg-pools"`
	SshPort     int    `ini:"ssh-port"`
	SshUser     string `ini:"ssh-user"`
	SshPassword string `ini:"ssh-password"`
	SshKeyFile  string `ini:"ssh-keyfile"`
	TmpDir      string `ini:"tmp-dir"`
}

func (s *PGPoolClusterServer) SetDefault() {
	if s.TmpDir == "" {
		s.TmpDir = DeployTmpDir
	}

	if s.SshPassword == "" && s.SshKeyFile == "" {
		s.SshKeyFile = filepath.Join(environment.GlobalEnv().HomePath, ".ssh", "id_rsa")
	}
}

// 验证配置
func (s *PGPoolClusterServer) Validator() error {
	logger.Infof("验证 server 参数\n")
	if err := utils.IsIPv4(s.Master); err != nil {
		if !utils.IsHostName(s.Master) {
			return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", s.Master, err)
		}
	}
	if s.Slave == "" {
		return fmt.Errorf("从库不能为空")
	}
	slaves := strings.Split(s.Slave, ",")
	if len(slaves) > 1 {
		return fmt.Errorf("pgpool 集群暂时只支持一个从库")
	}

	if err := utils.IsIPv4(s.Slave); err != nil {
		if !utils.IsHostName(s.Slave) {
			return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", s.Slave, err)
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

type PGPoolClusterParameter struct {
	Server     PGPoolClusterServer `ini:"server"`
	Pgsql      Prepare             `ini:"pgsql"`
	PGPool     PgPoolParameter     `ini:"pgpool"`
	Yes        bool                `ini:"yes"`
	NoRollback bool                `ini:"no-rollback"`
}

// Load 从配置文件加载配置到Prepare实例
func (p *PGPoolClusterParameter) Load(filename string) error {
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
func (p *PGPoolClusterParameter) SlaveTo(filename string) error {
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
func (p *PGPoolClusterParameter) Validator() error {
	if err := p.Server.Validator(); err != nil {
		return err
	}
	if err := p.Pgsql.Validator(); err != nil {
		return err
	}
	if err := p.PGPool.Validator(); err != nil {
		return err
	}
	return nil
}
