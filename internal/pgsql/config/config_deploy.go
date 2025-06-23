/*
@Author : WuWeiJian
@Date : 2021-02-26 14:50
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
				return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", slave, err)
			}
		}
	}

	// 端口
	if s.SshPort < 1 || s.SshPort > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", s.SshPort)
	}

	// mshost := append(slaves, s.Master)
	// // 如果为IPV6环境，则验证
	// if err := utils.Ipv6Check(mshost); err != nil {
	// 	return err
	// }

	return nil
}

/*// 安装时读取的配置文件
type Pgsql struct {
	BindIP     string `ini:"bind_ip" comment:"监听IP，如果没有特殊要求请勿修改"`
	Port       int    `ini:"port" comment:"监听端口，如果没有特殊要求请勿修改"`
	Dir        string `ini:"dir" comment:"数据部署目录，请确认该目录存在，默认为/opt/pgsql+端口号，如无特殊要求请勿修改"`
	Username   string `ini:"username" comment:"程序用于连接数据库的用户名，默认为pguser+端口号，如无特殊要求请勿修改"`
	Password   string `ini:"password" comment:"程序用于连接数据库的用户名的密码（为username参数所设置的用户的密码），留空会随机生成密码"`
	Address    string `ini:"address" comment:"IP白名单，列入白名单的IP地址能够连接该数据库，无特殊要求请勿修改"`
	MemorySize string `ini:"memory_size" comment:"内存配置，建议内存配置不超过系统物理内存总量的50%，避免使用过程中系统物理内存耗尽造成内存溢出，默认为操作系统的50%，请根据实际部署环境进行调整，单位后缀可以为{MB,GB}"`
}

// 验证配置
func (pg *Pgsql) Validator() error {
	logger.Infof("验证参数\n")
	// 用户名
	r, _ := regexp.Compile(RegexpUsername)
	if ok := r.MatchString(pg.Username); !ok {
		return fmt.Errorf("用户名(%s)不符合规则: 2到63位小写字母,数字,下划线; 不能以数字开头", pg.Username)
	}

	if pg.Username == DefaultPGAdminUser {
		return fmt.Errorf("禁止以 %s 做为用户名", pg.Username)
	}

	// 密码 TODO 密码是否需要验证？ (目前看PGSQL本身只要随意一个字符就可以设置为密码)

	// 数据目录
	if err := utils.ValidatorDir(pg.Dir); err != nil {
		return err
	}

	// 端口
	if pg.Port < 1025 || pg.Port > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", pg.Port)
	}
	if utils.PortInUse(pg.Port) {
		return fmt.Errorf("端口号被占用: %d", pg.Port)
	}

	// 内存不大于机器最大内存
	if err := pg.ValidatorMemorySize(); err != nil {
		return err
	}

	// 绑定IP, 是不是一个IP地址
	if pg.BindIP != "localhost" && pg.BindIP != "*" {
		if err := utils.IsIPv4(pg.BindIP); err != nil {
			return err
		}
	}

	if err := pg.ValidatorAddress(); err != nil {
		return err
	}
	return nil
}

// 验证内存
func (pg *Pgsql) ValidatorMemorySize() error {
	var max int
	var memory int
	var suffix string
	var err error
	if v, err := mem.VirtualMemory(); err != nil {
		fmt.Println("获取机器内存信息失败, 默认配最大512G")
		max = 512 * 1024 // 512GB
	} else {
		max = int(v.Total / 1024 / 1024)
	}

	r, _ := regexp.Compile(RegexpMemorySuffix)
	index := r.FindStringIndex(pg.MemorySize)
	if index == nil {
		return fmt.Errorf("内存参数必须包含单位后缀(MB 或 GB)")
	}

	if memory, err = strconv.Atoi(pg.MemorySize[:index[0]]); err != nil {
		return err
	}

	suffix = strings.ToUpper(pg.MemorySize[index[0]:])

	switch suffix {
	case "M", "MB":
		suffix = "MB"
	case "G", "GB":
		suffix = "GB"
		memory = memory * 1024 //转换为MB单位
	default:
		return fmt.Errorf("不支持的内存后缀单位")
	}

	if memory > max {
		return fmt.Errorf("PG内存不能超过操作系统最大物理内存大小")
	}

	// 验证函数不应该包含赋值操作
	//p.MemorySize = strconv.Itoa(memory) + suffix
	return nil
}

// 验证授权IP
func (pg *Pgsql) ValidatorAddress() error {
	addrs := strings.Split(pg.Address, ",")
	for _, addr := range addrs {
		if addr == "localhost" {
			continue
		}
		ipMask := strings.Split(addr, "/")
		switch len(ipMask) {
		case 0:
			fmt.Println("空字符串忽略")
		case 1:
			return utils.IsIPv4(addr)
		case 2:
			return utils.IsIPv4Mask(addr)
		default:
			return fmt.Errorf("授权IP字符串不符合规则")
		}
	}
	return nil
}*/

type Parameter struct {
	Server     Server  `ini:"server"`
	Pgsql      Prepare `ini:"pgsql"`
	Yes        bool    `ini:"yes"`
	NoRollback bool    `ini:"no-rollback"`
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
	if err := p.Pgsql.Validator(); err != nil {
		return err
	}
	return nil
}
