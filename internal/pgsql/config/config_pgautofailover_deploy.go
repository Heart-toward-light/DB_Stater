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

type PGAutoFailoverServer struct {
	Monitor     string `ini:"monitor"`
	PGNode      string `ini:"pgnode"`
	NewPGnode   string `ini:"new-pgnode"`
	SshPort     int    `ini:"ssh-port"`
	SshUser     string `ini:"ssh-user"`
	SshPassword string `ini:"ssh-password"`
	SshKeyFile  string `ini:"ssh-keyfile"`
	TmpDir      string `ini:"tmp-dir"`
	SystemUser  string `ini:"system-user"`
	SystemGroup string `ini:"system-group"`
}

func (s *PGAutoFailoverServer) SetDefault() {
	if s.TmpDir == "" {
		s.TmpDir = DeployTmpDir
	}

	if s.SshPassword == "" && s.SshKeyFile == "" {
		s.SshKeyFile = filepath.Join(environment.GlobalEnv().HomePath, ".ssh", "id_rsa")
	}
}

// 验证节点配置
func (s *PGAutoFailoverServer) Validator() error {
	logger.Infof("验证 server 参数\n")
	if err := utils.IsIPv4(s.Monitor); err != nil {
		if !utils.IsHostName(s.Monitor) {
			return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", s.Monitor, err)
		}
	}

	pgnodes := strings.Split(s.PGNode, ",")
	newnodes := strings.Split(s.NewPGnode, ",")
	if len(pgnodes) == 0 {
		return fmt.Errorf("数据节点不能为空")
	}
	if len(pgnodes) < 2 {
		return fmt.Errorf("数据节点至少有两个节点，构成主从结构")
	}
	for _, pgnode := range pgnodes {
		if err := utils.IsIPv4(pgnode); err != nil {
			if !utils.IsHostName(pgnode) {
				return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", pgnode, err)
			}
		}
	}

	if s.NewPGnode != "" {
		for _, newnode := range newnodes {
			if err := utils.IsIPv4(newnode); err != nil {
				if !utils.IsHostName(newnode) {
					return fmt.Errorf("新增从节点 host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", newnode, err)
				}
			}
		}
	}

	// 端口
	if s.SshPort < 1 || s.SshPort > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", s.SshPort)
	}

	return nil
}

type PGAutoFailoverParameter struct {
	Server     PGAutoFailoverServer  `ini:"server"`
	Pgmonitor  PGAutoFailoverMonitor `ini:"monitor"`
	Pgnode     PGAutoFailoverPGNode  `ini:"pgnode"`
	Yes        bool                  `ini:"yes"`
	NoRollback bool                  `ini:"no-rollback"`
}

// Load 从配置文件加载配置到Prepare实例
func (p *PGAutoFailoverParameter) Load(filename string) error {
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

// 验证 monitor 与 pgnode 端口
func (p *PGAutoFailoverParameter) CheckPort() error {
	monitor_ins := fmt.Sprintf("%s:%d", p.Server.Monitor, p.Pgmonitor.Port)
	pgnodes := strings.Split(p.Server.PGNode, ",")
	for _, pgnode := range pgnodes {
		pgnode_ins := fmt.Sprintf("%s:%d", pgnode, p.Pgnode.Port)
		if pgnode_ins == monitor_ins {
			return fmt.Errorf("node 节点不能与监控节点 %s 一致", pgnode_ins)
		}
	}
	return nil
}

// 单独新增数据节点前进行相关集群验证
func (p *PGAutoFailoverParameter) NewPGCheck() error {
	monitor_ins := fmt.Sprintf("%s:%d", p.Server.Monitor, p.Pgmonitor.Port)
	pgnodes := strings.Split(p.Server.PGNode, ",")
	newpgnodes := strings.Split(p.Server.NewPGnode, ",")

	ok, _ := utils.TcpGather(monitor_ins)
	if !ok {
		return fmt.Errorf("PG_auto_failover  监控节点服务 %s:%d 端口异常, 请检查", p.Server.Monitor, p.Pgmonitor.Port)
	}

	for _, pgnode := range pgnodes {
		pgnode_ins := fmt.Sprintf("%s:%d", pgnode, p.Pgnode.Port)
		ok, _ := utils.TcpGather(pgnode_ins)
		if !ok {
			return fmt.Errorf("PG_auto_failover  数据节点服务 %s:%d 端口异常, 请先恢复主从状态,再添加从库", pgnode, p.Pgnode.Port)
		}
	}

	for _, newpgnode := range newpgnodes {
		// if strings.Contains(pgnodes,pgnode) {}
		if utils.ContainsString(pgnodes, newpgnode) {
			return fmt.Errorf("新增的数据节点服务 %s:%d 不能和 pgnode 配置节点冲突", newpgnode, p.Pgnode.Port)
		}
		pgnode_ins := fmt.Sprintf("%s:%d", newpgnode, p.Pgnode.Port)
		ok, _ := utils.TcpGather(pgnode_ins)
		if ok {
			return fmt.Errorf("新增的数据节点服务 %s:%d 不能存在, 已经存在的实例请添加到 pgnode 配置部分", newpgnode, p.Pgnode.Port)
		}
	}

	return nil
}

// Load 从配置文件加载配置到Prepare实例
func (p *PGAutoFailoverParameter) Validator() error {
	if err := p.Server.Validator(); err != nil {
		return err
	}

	if err := p.Pgnode.Validator(); err != nil {
		return err
	}

	if err := p.CheckPort(); err != nil {
		return err
	}
	return nil
}
