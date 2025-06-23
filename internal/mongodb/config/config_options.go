/*
@Author : WuWeiJian
@Date : 2021-05-10 16:36
*/

package config

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/validator"
	"gopkg.in/ini.v1"
)

// 安装时读取的配置文件
type MongodbOptions struct {
	SystemUser  string `ini:"system-user"`
	SystemGroup string `ini:"system-group"`
	Port        int    `ini:"port" validate:"required,gt=1024,lt=65535"`
	Dir         string `ini:"dir" validate:"required"`
	Username    string `ini:"username" validate:"required,min=3,max=50"`
	Password    string `ini:"password" validate:"required,min=6,max=50"`
	Memory      int    `ini:"memory" validate:"required"`
	ReplSetName string `ini:"replSetName" validate:"max=32"`
	Arbiter     bool   `ini:"arbiter"`
	Ipv6        bool   `ini:"ipv6"`
	// BindIP      string `ini:"bind-ip" validate:"required,ip"`
	BindIP        string `ini:"bind-ip"`
	Join          string `ini:"join" validate:"ipPort"`
	Owner         string `ini:"owner"`
	ResourceLimit string `ini:"resource-limit"`
	Yes           bool   `ini:"yes"`
	NoRollback    bool   `ini:"no-rollback"`
}

// 初始化参数
func (option *MongodbOptions) InitArgs() {
	if option.Port == 0 {
		option.Port = utils.RandomPort(DefaultMongoDBPort)
	}

	if option.Dir == "" {
		option.Dir = fmt.Sprintf(DefaultMongoDBBaseDir, option.Port)
	}

	// option.Join 为空 表示单机版安装, 不做为从库加入其他集群, 所以直接随机生成一个副本集名称
	if option.ReplSetName == "" && option.Join == "" {
		option.ReplSetName = fmt.Sprintf(DefaultMongoDBReplSet, option.Port, utils.GenerateString(6))
	}

	if !option.Ipv6 && option.BindIP == "" {
		option.BindIP = DefaultMongoDBBindIP
	}

	if option.Ipv6 {
		option.BindIP = DefaultMongoDBIpv6BindIP
	}

	if option.SystemUser == "" {
		option.SystemUser = DefaultMongoDBSystemUser
	}

	if option.SystemGroup == "" {
		option.SystemGroup = DefaultMongoDBSystemGroup
	}
}

// CheckEnv 检查环境
func (option *MongodbOptions) CheckSpecialChar() error {
	r, _ := regexp.Compile(MongodbURISpecialChar)
	index := r.FindStringIndex(option.Password)
	if index != nil {
		return fmt.Errorf("密码不能包含特殊字符(: / + @ ? & =), 随机示例: %s\n", utils.GeneratePasswd(16))
	}

	index = r.FindStringIndex(option.Username)
	if index != nil {
		return fmt.Errorf("用户名不能包含特殊字符(: / + @ ? & =)\n")
	}

	index = r.FindStringIndex(option.ReplSetName)
	if index != nil {
		return fmt.Errorf("副本集名不能包含特殊字符(: / + @ ? & =)\n")
	}
	return nil
}

// 检查环境
func (option *MongodbOptions) CheckEnv() error {
	// 数据目录
	if err := utils.ValidatorDir(option.Dir); err != nil {
		return err
	}

	if utils.PortInUse(option.Port) {
		return fmt.Errorf("端口号被占用: %d", option.Port)
	}

	// 内存不大于机器最大内存
	max := int(environment.GlobalEnv().Memory.Total / 1024 / 1024 / 1024)
	if option.Memory >= max {
		return fmt.Errorf("memory: %d GB 必须小于机器最大内存: %d GB", option.Memory, max)
	}
	return nil
}

// IPV6环境检查
func (option *MongodbOptions) CheckIPV6() error {
	ips, err := net.LookupIP(option.Owner)
	if err != nil {
		return fmt.Errorf("获取IPV6地址失败: %v\n", err)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("获取网络接口失败：%v\n", err)
	}

	if ips[0].To4() != nil { // 检查是否为IPv6地址
		return fmt.Errorf("%s 不是IPV6域名地址或/etc/hosts解析映射本地地址异常,请检查\n", option.Owner)
	} else {
		// 判断是否与本地IPV6地址匹配
		found := false
		for _, iface := range ifaces {
			addrs, err := iface.Addrs()
			if err != nil {
				// fmt.Println("获取地址失败：", err)
				continue
			}

			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() == nil {
					if ipnet.IP.Equal(ips[0]) {
						found = true
						break
					}
				}
			}
		}
		if !found {
			return fmt.Errorf("%s 域名解析的IPV6地址与本地地址不匹配,请检查\n", option.Owner)
		}

	}

	return nil
}

// ValidateIPPort 验证IP:PORT格式
func ValidateIPPort(fl validator.FieldLevel) bool {
	if fl.Field().String() == "" {
		return true
	}
	ipPort := strings.Split(fl.Field().String(), ":")
	if err := utils.IsIPv4(ipPort[0]); err != nil {
		if utils.IsHostName(ipPort[0]) {
			return true
		}
		logger.Warningf("主机名不可访问")
		return false
	}
	if len(ipPort) > 1 {
		port, err := strconv.Atoi(ipPort[1])
		if err != nil {
			return false
		}
		if port <= 0 || port >= 65536 {
			return false
		}
	}
	return true
}

type Server struct {
	Address  string `ini:"address"`
	Arbiter  string `ini:"arbiter"`
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
	ips := strings.Split(s.Address, ",")
	if s.Arbiter != "" {
		if err := utils.IsIPv4(s.Arbiter); err != nil {
			if !utils.IsHostName(s.Arbiter) {
				return fmt.Errorf("arbiter host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", s.Arbiter, err)
			}
		}
	}

	if s.Arbiter != "" {
		if len(ips) < 2 {
			return fmt.Errorf("部署IP至少要有两数据节点")
		}
	} else if len(ips) < 3 {
		return fmt.Errorf("部署IP至少要有三个地址")
	}
	for _, ip := range ips {
		if err := utils.IsIPv4(ip); err != nil {
			if !utils.IsHostName(ip) {
				return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", ip, err)
			}
		}
	}

	// 端口
	if s.SshPort < 1 || s.SshPort > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", s.SshPort)
	}
	return nil
}

type MongoDBDeployOptions struct {
	Server     Server         `ini:"server"`
	MongoDB    MongodbOptions `ini:"mongodb"`
	Yes        bool           `ini:"yes" comment:"监听IP，如果没有特殊要求请勿修改"`
	NoRollback bool           `ini:"no-rollback" comment:"监听IP，如果没有特殊要求请勿修改"`
}

// Load 从配置文件加载配置到Prepare实例
func (o *MongoDBDeployOptions) Load(filename string) error {
	return global.INILoadFromFile(filename, o, ini.LoadOptions{
		SpaceBeforeInlineComment: true,
	})
}

// SaveTo 将Parameter实例数据写入配置文件
func (o *MongoDBDeployOptions) SlaveTo(filename string) error {
	return global.INISaveToFile(filename, o)
}

// IPV6 验证
func (o *MongoDBDeployOptions) Ipv6Verify() error {
	// logger.Infof("验证 IPV6 Server 参数\n")
	iplist := strings.Split(o.Server.Address, ",")
	if o.MongoDB.Ipv6 {
		for _, domain := range iplist {
			ips, err := net.LookupIP(domain)
			if err != nil {
				return fmt.Errorf("获取IPV6地址失败: %v\n", err)
			}
			if ips[0].To4() != nil {
				return fmt.Errorf("%s 不是IPV6域名地址或/etc/hosts解析映射本地地址异常,请检查\n", domain)
			}
		}
	} else {
		for _, domain := range iplist {
			ips, err := net.LookupIP(domain)
			if err != nil {
				return fmt.Errorf("获取IPV6地址失败: %v\n", err)
			}
			if ips[0].To4() == nil {
				return fmt.Errorf("%s 是IPV6域名地址但是你没有打开IPV6配置,请检查配置文件\n", domain)
			}
		}
	}
	return nil
}

func (o *MongoDBDeployOptions) Validator() error {
	if err := o.Server.Validator(); err != nil {
		return err
	}
	if err := utils.CheckPasswordLever(o.MongoDB.Password); err != nil {
		return err
	}
	if err := o.MongoDB.CheckSpecialChar(); err != nil {
		return err
	}
	if err := o.Ipv6Verify(); err != nil {
		return err
	}
	return nil
}
