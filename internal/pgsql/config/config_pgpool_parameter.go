/*
@Author : WuWeiJian
@Date : 2021-04-16 10:37
*/

package config

import (
	"dbup/internal/utils"
	"dbup/internal/utils/arrlib"
	"dbup/internal/utils/logger"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator"
	"github.com/shirou/gopsutil/mem"
	"gopkg.in/ini.v1"
)

// 安装pgpool时读取命令行参数
type PgPoolParameter struct {
	Port          int    `ini:"port"`
	PcpPort       int    `ini:"pcp-port"`
	WDPort        int    `ini:"wd-port"`
	HeartPort     int    `ini:"heart-port"`
	BindIP        string `ini:"bind-ip"`
	PcpBindIP     string `ini:"pcp-bind-ip"`
	PGPoolIP      string `ini:"pgpool-ip"`
	Dir           string `ini:"dir"`
	NodeID        int    `ini:"node-id"`
	Username      string `ini:"username"`
	Password      string `ini:"password"`
	Address       string `ini:"address"`
	PGMaster      string `ini:"pg-master"`
	PGSlave       string `ini:"pg-slave"`
	PGPort        int    `ini:"pg-port"`
	PGDir         string `ini:"pg-dir"`
	ResourceLimit string `ini:"resource-limit"`
	Yes           bool   `ini:"yes"`
	NoRollback    bool   `ini:"no-rollback"`
}

// Load 从配置文件加载配置到Prepare实例
func (p *PgPoolParameter) Load(filename string) error {
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
func (p *PgPoolParameter) InitArgs() {
	logger.Infof("初始化安装参数\n")

	if p.Port == 0 {
		p.Port = PGPoolPort
	}

	if p.PcpPort == 0 {
		p.PcpPort = PGPoolPCPPort
	}

	if p.WDPort == 0 {
		p.WDPort = PGPoolWDPort
	}

	if p.HeartPort == 0 {
		p.HeartPort = PGPoolHeartPort
	}

	if p.BindIP == "" {
		p.BindIP = "*"
	}

	if p.PcpBindIP == "" {
		p.PcpBindIP = "localhost"
	}

	if p.Dir == "" {
		p.Dir = fmt.Sprintf("%s%d", DefaultPGPoolDir, p.Port)
	}

	if p.Username == "" {
		p.Username = DefaultPGUser
	}

	if p.PGPort == 0 {
		p.PGPort = DefaultPGPort
	}
}

// 到3万还没选出来,就用默认5432吧
func (p *PgPoolParameter) RandomPort(port int) int {
	for i := port; i <= 30000; i++ {
		socket1 := fmt.Sprintf("/var/run/postgresql/.s.PGSQL.%d", i)
		socket2 := fmt.Sprintf("/tmp/.s.PGSQL.%d", i)
		if !utils.PortInUse(i) && !utils.IsExists(socket1) && !utils.IsExists(socket2) {
			port = i
			break
		}
	}
	return port
}

func (p *PgPoolParameter) InitMemory() int {
	var memSize int
	if v, err := mem.VirtualMemory(); err != nil {
		logger.Warningf("获取机器内存信息失败, 默认配置1G\n")
		memSize = 1
	} else {
		memSize = int(v.Total / 1024 / 1024 / 1024 / 2)
		if memSize == 0 {
			memSize = 1
		}
	}
	return memSize
}

// 验证配置
func (p *PgPoolParameter) Validator() error {
	logger.Infof("验证参数\n")

	// 绑定IP, 是不是一个IP地址
	if p.BindIP != "localhost" && p.BindIP != "*" {
		if err := utils.IsIPv4(p.BindIP); err != nil {
			return err
		}
	}
	// 绑定IP, 是不是一个IP地址
	if p.PcpBindIP != "localhost" && p.PcpBindIP != "*" {
		if err := utils.IsIPv4(p.PcpBindIP); err != nil {
			return fmt.Errorf("pcp-bind-ip: %s: %v\n", p.PcpBindIP, err)
		}
	}

	if err := p.ValidatorPort(); err != nil {
		return err
	}

	if err := p.ValidatorIP(); err != nil {
		return err
	}

	// 用户名
	r, _ := regexp.Compile(RegexpUsername)
	if ok := r.MatchString(p.Username); !ok {
		return fmt.Errorf("用户名(%s)不符合规则: 2到63位小写字母,数字,下划线; 不能以数字开头", p.Username)
	}

	if p.Username == DefaultPGAdminUser {
		return fmt.Errorf("禁止以 %s 做为用户名", p.Username)
	}

	if p.Username == "" || p.Password == "" {
		return fmt.Errorf("用户名和密码不能为空")
	}

	return nil
}

func (p *PgPoolParameter) ValidatorPort() error {
	if p.Port < 1025 || p.Port > 65535 {
		return fmt.Errorf("port (%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", p.Port)
	}

	if p.PcpPort < 1025 || p.PcpPort > 65535 {
		return fmt.Errorf("pcp-port 端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", p.PcpPort)
	}

	if p.WDPort < 1025 || p.WDPort > 65535 {
		return fmt.Errorf("wd-port (%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", p.WDPort)
	}

	if p.HeartPort < 1025 || p.HeartPort > 65535 {
		return fmt.Errorf("heart-port(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", p.HeartPort)
	}

	if arrlib.IsDoubleInt(p.Port, p.PcpPort, p.WDPort, p.HeartPort) {
		return fmt.Errorf("port(%d), pcp-port(%d), wd-port(%d), heart-port(%d) 不能重复", p.Port, p.PcpPort, p.WDPort, p.HeartPort)
	}
	return nil
}

func (p *PgPoolParameter) ValidatorIP() error {
	if err := p.ValidatorAddress(); err != nil {
		return fmt.Errorf("address参数格式不正确: %v\n", err)
	}

	pools := strings.Split(p.PGPoolIP, ",")
	if len(pools) != 3 {
		return fmt.Errorf("pgpool安装地址必须是三个")
	}
	for _, pool := range pools {
		if err := utils.IsIPv4(pool); err != nil {
			if !utils.IsHostName(pool) {
				return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败\n", pool, err)
			}
		}
	}
	if arrlib.IsDoubleString(pools...) {
		return fmt.Errorf("pgpool安装地址重复\n")
	}

	if err := utils.IsIPv4(p.PGMaster); err != nil {
		if !utils.IsHostName(p.PGMaster) {
			return fmt.Errorf("pg-master: (%s) 即不是一个 IP 地址(%v), 又解析主机名失败\n", p.PGMaster, err)
		}
		//return fmt.Errorf("pg-master: %s: %v\n", p.PGMaster, err)
	}

	if err := utils.IsIPv4(p.PGSlave); err != nil {
		if !utils.IsHostName(p.PGSlave) {
			return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败\n", p.PGSlave, err)
		}
		//return fmt.Errorf("pg-slave: %s: %v\n", p.PGSlave, err)
	}
	if p.PGMaster == p.PGSlave {
		return fmt.Errorf("pgsql主(%s), 从(%s)IP重复\n", p.PGMaster, p.PGSlave)
	}
	return nil
}

// 验证内存
func (p *PgPoolParameter) ValidatorMemorySize() error {
	var max int
	var memory int
	var suffix string
	//var err error
	//if v, err := mem.VirtualMemory(); err != nil {
	//	fmt.Println("获取机器内存信息失败, 默认配最大512G")
	//	max = 512 * 1024 // 512GB
	//} else {
	//	max = int(v.Total / 1024 / 1024)
	//}
	//
	//r, _ := regexp.Compile(RegexpMemorySuffix)
	//index := r.FindStringIndex(p.MemorySize)
	//if index == nil {
	//	return fmt.Errorf("内存参数必须包含单位后缀(MB 或 GB)")
	//}
	//
	//if memory, err = strconv.Atoi(p.MemorySize[:index[0]]); err != nil {
	//	return err
	//}
	//
	//suffix = strings.ToUpper(p.MemorySize[index[0]:])

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
func (p *PgPoolParameter) ValidatorAddress() error {
	addrs := strings.Split(p.Address, ",")
	v := validator.New()
	for _, addr := range addrs {
		if addr == "localhost" || addr == "local" {
			continue
		}
		if err := utils.CheckAddressFormat(addr); err != nil {
			if e := v.Var(addr, "hostname"); e != nil {
				return fmt.Errorf("%s 不是有效IP地址, 也不是规范主机名", addr)
			}
		}
	}
	return nil
}

// 检查环境
func (p *PgPoolParameter) CheckEnv() error {
	if utils.PortInUse(p.Port) {
		return fmt.Errorf("port 被占用: %d", p.Port)
	}

	if utils.PortInUse(p.PcpPort) {
		return fmt.Errorf("pcp-port 被占用: %d", p.PcpPort)
	}

	if utils.PortInUse(p.WDPort) {
		return fmt.Errorf("wd-port 被占用: %d", p.WDPort)
	}

	if utils.PortInUse(p.HeartPort) {
		return fmt.Errorf("heart-port被占用: %d", p.HeartPort)
	}

	// 安装目录
	return utils.ValidatorDir(p.Dir)
}
