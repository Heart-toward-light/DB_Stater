/*
@Author : WuWeiJian
@Date : 2021-07-27 15:52
*/

package config

import (
	"dbup/internal/environment"
	"dbup/internal/utils"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type RedisClusterSSHConfig struct {
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	KeyFile  string `yaml:"keyfile"`
	TmpDir   string `yaml:"tmp-dir"`
}

func (s *RedisClusterSSHConfig) SetDefault() {
	if s.TmpDir == "" {
		s.TmpDir = RedisClusterDeployTmpDir
	}

	if s.Password == "" && s.KeyFile == "" {
		s.KeyFile = filepath.Join(environment.GlobalEnv().HomePath, ".ssh", "id_rsa")
	}
}

type RedisClusterConfig struct {
	SystemUser    string `yaml:"system-user"`
	SystemGroup   string `yaml:"system-group"`
	IPV6          bool   `yaml:"ipv6"`
	Password      string `yaml:"password"`
	Memory        string `yaml:"memory"`
	Module        string `yaml:"module"`
	ResourceLimit string `yaml:"resource-limit"`
	// Appendonly      string `yaml:"appendonly"`
	MaxmemoryPolicy string `yaml:"maxmemory-policy"`
}

func (c *RedisClusterConfig) Validator() error {
	if err := c.ValidatorMemory(); err != nil {
		return err
	}

	if c.Module != "" {
		modules := strings.Split(c.Module, ",")
		for _, value := range modules {
			switch value {
			case "redisbloom", "redisearch", "redisgraph":
				continue
			default:
				return fmt.Errorf("不支持的模块: %s\n", value)
			}
		}
	}
	return nil
}

// 验证内存
func (c *RedisClusterConfig) ValidatorMemory() error {
	r, _ := regexp.Compile(RegexpMemorySuffix)
	index := r.FindStringIndex(c.Memory)
	if index == nil {
		return fmt.Errorf("内存参数必须包含单位后缀(MB 或 GB)")
	}
	if _, err := strconv.Atoi(c.Memory[:index[0]]); err != nil {
		return fmt.Errorf("内存必须为整数值, 并且必须包含单位后缀(MB 或 GB): %v", err)
	}
	return nil
}

type RedisClusterNode struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Dir  string `yaml:"dir"`
}

func (n *RedisClusterNode) Validator() error {
	if err := utils.IsIPv4(n.Host); err != nil {
		if !utils.IsHostName(n.Host) {
			return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败\n", n.Host, err)
		}
	}

	if n.Port < 1025 || n.Port > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间\n", n.Port)
	}

	if n.Dir != "" && !path.IsAbs(n.Dir) {
		return fmt.Errorf("路径(%s) 为空 或非绝对路径, 为空时默认/opt/redis<port>\n", n.Dir)
	}

	return nil
}

func (n *RedisClusterNode) SetDefault() {
	if n.Dir == "" {
		n.Dir = fmt.Sprintf("%s%d", DefaultRedisDir, n.Port)
	}
}

type RedisClusterOption struct {
	SSHConfig   RedisClusterSSHConfig `yaml:"ssh-config"`
	RedisConfig RedisClusterConfig    `yaml:"redis-config"`
	Master      []RedisClusterNode    `yaml:"master"`
	Slave       []RedisClusterNode    `yaml:"slave"`
	NoRollback  bool                  `yaml:"no-rollback"`
}

func (o *RedisClusterOption) Validator() error {
	if err := o.RedisConfig.Validator(); err != nil {
		return err
	}

	if len(o.Master) != len(o.Slave) && len(o.Slave) != 0 {
		return fmt.Errorf("部署集群如果存在从库, 从库数量必须与主库数量一致\n")
	}

	for _, node := range o.Master {
		if err := node.Validator(); err != nil {
			return err
		}
	}
	for _, node := range o.Slave {
		if err := node.Validator(); err != nil {
			return err
		}
	}

	return nil
}

func (o *RedisClusterOption) SetDefault() {
	o.SSHConfig.SetDefault()
	for _, node := range o.Master {
		node.SetDefault()
	}
	for _, node := range o.Slave {
		node.SetDefault()
	}
}

func (o *RedisClusterOption) CheckDuplicate() error {
	var ins []RedisClusterNode
	ins = append(ins, o.Master...)
	ins = append(ins, o.Slave...)
	for i := 0; i < len(ins)-1; i++ {
		for j := i + 1; j < len(ins); j++ {
			if ins[i].Host == ins[j].Host && ins[i].Port == ins[j].Port {
				return fmt.Errorf("有重复的 <Host:Port>: %s:%d 请检查\n", ins[i].Host, ins[i].Port)
			}

			if ins[i].Host == ins[j].Host && ins[i].Dir == ins[j].Dir {
				return fmt.Errorf("有重复的 <Host:Dir>: %s:%s 请检查\n", ins[i].Host, ins[i].Dir)
			}
		}
	}
	return nil
}

type RedisClusterAddNodeOption struct {
	SSHConfig  RedisClusterSSHConfig
	Parameter  Parameters
	Host       string
	Cluster    string
	Master     string
	TmpDir     string
	IPV6       bool
	NoRollback bool
}

func (o *RedisClusterAddNodeOption) Validator() error {
	if err := utils.IsIPv4(o.Host); err != nil {
		if !utils.IsHostName(o.Host) {
			return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败\n", o.Host, err)
		}
	}

	if err := ValidateIPPort(o.Cluster); err != nil {
		return err
	}

	if o.Master != "" {
		if err := ValidateIPPort(o.Master); err != nil {
			return err
		}
	}

	return nil
}

func (o *RedisClusterAddNodeOption) ValidatorHost() error {
	if err := utils.IsIPv4(o.Host); err != nil {
		if !utils.IsHostName(o.Host) {
			return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败\n", o.Host, err)
		}
	}
	return nil
}

// ValidateIPPort 验证IP:PORT格式
func ValidateIPPort(s string) error {
	ipPort := strings.Split(s, ":")
	if len(ipPort) != 2 {
		return fmt.Errorf("host (%s) 不是 <IP>:<PORT> 格式\n", s)
	}

	if err := utils.IsIPv4(ipPort[0]); err != nil {
		if !utils.IsHostName(ipPort[0]) {
			return fmt.Errorf("host (%s) 的IP部分,即不是一个 IP 地址(%v), 又解析主机名失败\n", s, err)
		}
	}
	if len(ipPort) > 1 {
		port, err := strconv.Atoi(ipPort[1])
		if err != nil {
			return fmt.Errorf("host (%s) 不是 <IP>:<PORT> 格式\n", s)
		}
		if port <= 0 || port >= 65536 {
			return fmt.Errorf("host (%s) 端口号范围不正确\n", s)
		}
	}
	return nil
}
