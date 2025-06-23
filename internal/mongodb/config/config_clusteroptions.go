package config

import (
	"dbup/internal/environment"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"strings"
)

type MongosOptions struct {
	Shardlist     []string
	ConfigDB      string
	SystemUser    string
	SystemGroup   string
	Port          int
	Dir           string
	Username      string
	Password      string
	Ipv6          bool
	BindIP        string
	Owner         string
	ResourceLimit string
	Yes           bool
	NoRollback    bool
}

// 初始化参数
func (option *MongosOptions) InitArgs() {
	if option.Port == 0 {
		option.Port = utils.RandomPort(DefaultMongoSPort)
	}

	if option.Dir == "" {
		option.Dir = fmt.Sprintf(DefaultMongoDBBaseDir, option.Port)
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

// 检查configdb配置
func (option *MongosOptions) CheckConfigDB() error {
	res1 := strings.Split(option.ConfigDB, "/")
	fmt.Println(res1)
	if len(res1) == 2 {
		node := strings.Split(res1[1], ",")
		if len(node) < 3 {
			return fmt.Errorf("config 副本节点不能少于三个: %s \n", res1[1])
		} else {
			for _, n := range node {
				if len(strings.Split(n, ":")) != 2 {
					return fmt.Errorf("config 副本配置ip与端口不符合规范: %s \n", n)
				} else {
					ok, _ := utils.TcpGather(n)
					if !ok {
						return fmt.Errorf("config 副本配置ip与端口服务 %s 连接异常\n", n)
					}
				}
			}
		}
	} else {
		return fmt.Errorf("config 副本配置规则不规范: %s \n", option.ConfigDB)
	}

	return nil
}

// 检查环境
func (option *MongosOptions) CheckEnv() error {
	// 数据目录
	if err := utils.ValidatorDir(option.Dir); err != nil {
		return err
	}

	if utils.PortInUse(option.Port) {
		return fmt.Errorf("端口号被占用: %d", option.Port)
	}

	return nil
}

// IPV6环境检查
func (option *MongosOptions) CheckIPV6() error {
	ips, err := net.LookupIP(option.Owner)
	if err != nil {
		return fmt.Errorf("获取IPV6地址失败: %v\n", err)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("获取网络接口失败：%v\n", err)
	}

	if ips[0].To4() != nil { // 检查是否为IPv6地址
		return fmt.Errorf("mongos 节点 %s 不是IPV6域名地址或/etc/hosts解析映射本地地址异常,请检查\n", option.Owner)
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
			return fmt.Errorf("mongos 节点 %s 域名解析的IPV6地址与本地地址不匹配,请检查\n", option.Owner)
		}
	}

	return nil
}

type MongoDBClusterOptions struct {
	Mongosoption MongosOptions
	SSHConfig    Ssh_config        `yaml:"ssh-config"`
	MongoConfig  Mongo_config      `yaml:"mongo-config"`
	Mongos       []MongosNode      `yaml:"mongos"`
	MongoCfg     []MongoConfigNode `yaml:"mongo-cfg"`
	MongoShard   []MongoShard      `yaml:"mongo-shard"`
}

func (M *MongoDBClusterOptions) SetDefault() {
	if M.SSHConfig.TmpDir == "" {
		M.SSHConfig.TmpDir = DeployTmpDir
	}

	if M.SSHConfig.Password == "" && M.SSHConfig.KeyFile == "" {
		M.SSHConfig.KeyFile = filepath.Join(environment.GlobalEnv().HomePath, ".ssh", "id_rsa")
	}

}

// CheckEnv 检查环境
func (M *MongoDBClusterOptions) CheckSpecialChar() error {
	r, _ := regexp.Compile(MongodbURISpecialChar)
	index := r.FindStringIndex(M.MongoConfig.Password)
	if index != nil {
		return fmt.Errorf("密码不能包含特殊字符(: / + @ ? & =), 随机示例: %s \n", utils.GeneratePasswd(16))
	}

	index = r.FindStringIndex(M.MongoConfig.Username)
	if index != nil {
		return fmt.Errorf("用户名不能包含特殊字符")
	}

	return nil
}

// 验证集群配置
func (M *MongoDBClusterOptions) Validator() error {
	logger.Infof("验证 MongoCluster 配置参数规则\n")

	// mongos验证
	if len(M.Mongos) < 1 {
		return fmt.Errorf("mongos 配置地址至少为1个")
	} else {

		for _, Mongosnode := range M.Mongos {
			if err := M.Ipv6Verify(Mongosnode.Host); err != nil {
				return err
			}
			if err := utils.IsIPv4(Mongosnode.Host); err != nil {
				if !utils.IsHostName(Mongosnode.Host) {
					return fmt.Errorf("mongos host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", Mongosnode.Host, err)
				}
			}
		}
	}

	// mongoconfig 验证
	if len(M.MongoCfg) != 3 {
		return fmt.Errorf("mongo Config 部署实例数必须为三个")
	} else {
		cfgport := []int{}
		cfgdir := []string{}
		for _, Mongoconfig := range M.MongoCfg {
			if err := M.Ipv6Verify(Mongoconfig.Host); err != nil {
				return err
			}
			cfgport = append(cfgport, Mongoconfig.Port)
			cfgdir = append(cfgdir, Mongoconfig.Dir)
			if err := utils.IsIPv4(Mongoconfig.Host); err != nil {
				if !utils.IsHostName(Mongoconfig.Host) {
					return fmt.Errorf("mongoconfig host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", Mongoconfig.Host, err)
				}
			}
		}
		if M.CheckPort(cfgport) {
			return fmt.Errorf("mongoconfig 配置端口 %d 需要保持一致", cfgport)
		}
		if M.Checkdir(cfgdir) {
			return fmt.Errorf("mongoconfig 配置目录 %s 需要保持一致", cfgdir)
		}
	}

	// mongoshard 验证
	for _, Mongoshard := range M.MongoShard {
		if len(Mongoshard.Shard) != 3 {
			return fmt.Errorf("mongo 每个 Shard 部署实例数数必须为三个")
		} else {
			shardport := []int{}
			sharddir := []string{}
			for _, shardnode := range Mongoshard.Shard {
				if err := M.Ipv6Verify(shardnode.Host); err != nil {
					return err
				}
				shardport = append(shardport, shardnode.Port)
				sharddir = append(sharddir, shardnode.Dir)
				if err := utils.IsIPv4(shardnode.Host); err != nil {
					if !utils.IsHostName(shardnode.Host) {
						return fmt.Errorf("mongoconfig host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", shardnode.Host, err)
					}
				}
			}
			if M.CheckPort(shardport) {
				return fmt.Errorf("MongoShard 配置端口 %d 需要保持一致", shardport)
			}
			if M.Checkdir(sharddir) {
				return fmt.Errorf("MongoShard 配置目录 %s 需要保持一致", sharddir)
			}
		}
	}

	// ssh 端口验证
	if M.SSHConfig.Port < 1 || M.SSHConfig.Port > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", M.SSHConfig.Port)
	}

	return nil
}

// IPV6 验证
func (o *MongoDBClusterOptions) Ipv6Verify(mongonode string) error {
	if o.MongoConfig.Ipv6 {
		// for _, domain := range mongonode {
		ip := net.ParseIP(mongonode)
		if ip != nil {
			return fmt.Errorf("无效IP %s , ipv6环境需要配置主机名", mongonode)
		}
		ips, err := net.LookupIP(mongonode)
		if err != nil {
			return fmt.Errorf("获取ipv6地址失败: %v\n", err)
		}
		if ips[0].To4() != nil {
			return fmt.Errorf("%s 不是ipv6域名地址或/etc/hosts解析映射本地地址异常,请检查\n", mongonode)
		}
		// }
	} else {
		// for _, domain := range mongonode {
		ips, err := net.LookupIP(mongonode)
		if err != nil {
			return fmt.Errorf("获取IPV6地址失败: %v\n", err)
		}
		if ips[0].To4() == nil {
			return fmt.Errorf("%s 是IPV6域名地址但是你没有打开IPV6配置,请检查配置文件\n", mongonode)
		}
		// }
	}
	return nil
}

func (M *MongoDBClusterOptions) CheckPort(pl []int) bool {
	flags := false
	tt := pl[len(pl)-1]
	for _, i := range pl {
		if tt != i {
			flags = true
		}
	}
	return flags
}

func (M *MongoDBClusterOptions) Checkdir(dl []string) bool {
	flags := false
	tt := dl[len(dl)-1]
	for _, i := range dl {
		if tt != i {
			flags = true
		}
	}
	return flags
}

func (M *MongoDBClusterOptions) Validators() error {
	if err := M.Validator(); err != nil {
		return err
	}

	if err := utils.CheckPasswordLever(M.MongoConfig.Password); err != nil {
		return err
	}

	if err := M.CheckSpecialChar(); err != nil {
		return err
	}

	return nil
}
