/*
@Author : WuWeiJian
@Date : 2021-01-05 15:56
*/

package config

import (
	"dbup/internal/environment"
	"dbup/internal/utils"
	"dbup/internal/utils/arrlib"
	"dbup/internal/utils/logger"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

// 安装时读取的配置文件
type Parameters struct {
	SystemUser      string `ini:"system-user" comment:"监听端口，如果没有特殊要求请勿修改"`
	SystemGroup     string `ini:"system-group" comment:"监听端口，如果没有特殊要求请勿修改"`
	Port            int    `ini:"port" comment:"监听端口，如果没有特殊要求请勿修改"`
	Dir             string `ini:"dir" comment:"数据部署目录，请确认该目录存在，默认为/opt/pgsql+端口号，如无特殊要求请勿修改"`
	Password        string `ini:"password" comment:"程序用于连接数据库的用户名的密码，密码规则为：16位长度，需要包含数字、大写字母、小写字母，留空即为生成随机密码"`
	MemorySize      string `ini:"memory-size" comment:"内存配置，建议内存配置不超过系统物理内存总量的50%，避免使用过程中系统物理内存耗尽造成内存溢出，默认为操作系统的1/4，请根据实际部署环境进行调整，单位后缀可以为{MB,GB}"`
	Appendonly      string `ini:"appendonly"`
	MaxmemoryPolicy string `ini:"maxmemory-policy"`
	Ipv6            bool   `ini:"ipv6"`
	Module          string `ini:"module" comment:"监听IP，如果没有特殊要求请勿修改"`
	Master          string `ini:"master" comment:"单机安装同步主库的IP:PORT"`
	Cluster         bool   `ini:"cluster" comment:"是否为集群模式"`
	ResourceLimit   string `ini:"resource-limit"`
	Yes             bool   `ini:"yes" comment:"监听IP，如果没有特殊要求请勿修改"`
	NoRollback      bool   `ini:"no-rollback" comment:"监听IP，如果没有特殊要求请勿修改"`
}

// Load 从配置文件加载配置到Prepare实例
func (p *Parameters) Load(filename string) error {
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

// 初始化参数
func (p *Parameters) InitArgs() {
	logger.Infof("初始化安装参数\n")

	if p.SystemUser == "" {
		p.SystemUser = DefaultRedisSystemUser
	}

	if p.SystemGroup == "" {
		p.SystemGroup = DefaultRedisSystemGroup
	}

	if p.Appendonly == "" {
		p.Appendonly = "yes"
	}

	if p.MaxmemoryPolicy == "" {
		p.MaxmemoryPolicy = "noeviction"
	}

	if p.Password == "" {
		p.Password = utils.GeneratePasswd(DefaultRedisPassLength)
	}

	if p.Port == 0 {
		p.Port = utils.RandomPort(DefaultRedisPort)
	}

	if p.Dir == "" {
		p.Dir = fmt.Sprintf("%s%d", DefaultRedisDir, p.Port)
	}

	if p.MemorySize == "" {
		mem := environment.GlobalEnv().Memory.Total / 1024 / 1024 / 4
		if mem > 1024 {
			p.MemorySize = fmt.Sprintf("%vGB", mem/1024)
		} else {
			p.MemorySize = fmt.Sprintf("%vMB", mem)
		}
	}
}

// 初始化 PORT && Dir
func (p *Parameters) InitPortDir() {
	if p.Port == 0 {
		p.Port = utils.RandomPort(DefaultRedisPort)
	}

	if p.Dir == "" {
		p.Dir = fmt.Sprintf("%s%d", DefaultRedisDir, p.Port)
	}
}

// 验证配置
func (p *Parameters) Validator() error {
	logger.Infof("验证参数\n")

	// 端口
	if p.Port < 1025 || p.Port > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", p.Port)
	}

	if p.Module != "" {
		modules := strings.Split(p.Module, ",")
		for _, value := range modules {
			switch value {
			case "redisbloom", "redisearch", "redisgraph":
				continue
			default:
				return fmt.Errorf("不支持的模块: %s\n", value)
			}
		}
	}

	if p.Cluster && p.Master != "" {
		return fmt.Errorf("cluster 模式不允许指定主从关系: %s\n", p.Master)
	}

	if p.Master != "" {
		ipPort := strings.Split(p.Master, ":")
		// var conn *dao.PgConn

		if err := utils.IsIPv4(ipPort[0]); err != nil {
			if !utils.IsHostName(ipPort[0]) {
				return fmt.Errorf("%s 不是可用的IP地址或主机名不可访问", p.Master)
			}
		}
		if len(ipPort) > 1 {
			port, err := strconv.Atoi(ipPort[1])
			if err != nil {
				return fmt.Errorf("%s, 不是可用的端口", p.Master)
			}
			if port <= 0 || port >= 65536 {
				return fmt.Errorf("%s, 不是可用的端口", p.Master)
			}
		}
	}

	if p.Appendonly != "yes" && p.Appendonly != "no" {
		return fmt.Errorf("%s, 不支持的appendonly值", p.Appendonly)
	}

	policy := []string{"volatile-lru", "allkeys-lru", "volatile-random", "allkeys-random", "volatile-ttl", "noeviction"}
	if !arrlib.InArray(p.MaxmemoryPolicy, policy) {
		return fmt.Errorf("%s, 不支持的key淘汰策略", p.MaxmemoryPolicy)
	}

	return nil
}

// 验证内存
func (p *Parameters) ValidatorMemorySize() error {
	var max int
	var memory int
	var suffix string
	var err error
	max = int(environment.GlobalEnv().Memory.Total / 1024 / 1024)

	r, _ := regexp.Compile(RegexpMemorySuffix)
	index := r.FindStringIndex(p.MemorySize)
	if index == nil {
		return fmt.Errorf("内存参数必须包含单位后缀(MB 或 GB)")
	}

	if memory, err = strconv.Atoi(p.MemorySize[:index[0]]); err != nil {
		return err
	}

	suffix = strings.ToUpper(p.MemorySize[index[0]:])

	switch suffix {
	case "M", "MB":
		suffix = "MB"
	case "G", "GB":
		suffix = "GB"
		memory = memory * 1024 //转换为MB单位
	default:
		return fmt.Errorf("不支持的内存后缀单位")
	}

	if memory+ReplBacklogSizeMB > max {
		return fmt.Errorf("redis内存 + 同步缓冲区(512M), 不能超过操作系统最大物理内存大小")
	}
	return nil
}

// 检查环境
func (p *Parameters) CheckEnv() error {
	// 数据目录
	if err := utils.ValidatorDir(p.Dir); err != nil {
		return err
	}

	//if utils.PortInUse(p.Port) {
	//	return fmt.Errorf("端口号被占用: %d", p.Port)
	//}
	// 检查本地 ipv6 地址解析情况
	if p.Ipv6 {
		if err := utils.Ipv6Check(); err != nil {
			return err
		}
	}
	// 替换 utils.PortInUse() 是为了查看端口号被占用的具体错误信息, 排查总被占用问题
	c, err := net.Listen("tcp", fmt.Sprintf(":%d", p.Port))
	if err != nil {
		logger.Warningf("尝试启动端口(%d)失败: %v\n", p.Port, err)
		return fmt.Errorf("端口号被占用: %d", p.Port)
	}
	defer c.Close()

	if p.Cluster {
		if utils.PortInUse(p.Port + 10000) {
			return fmt.Errorf("redis cluster 集群通信端口号( port + 10000)被占用: %d", p.Port+10000)
		}
	}

	// 内存不大于机器最大内存
	if err := p.ValidatorMemorySize(); err != nil {
		return err
	}
	return nil
}
