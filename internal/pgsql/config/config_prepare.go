/*
@Author : WuWeiJian
@Date : 2020-12-03 14:29
*/

package config

import (
	"dbup/internal/utils"
	"dbup/internal/utils/arrlib"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/validator"
	"github.com/mitchellh/go-homedir"
	"github.com/shirou/gopsutil/v3/mem"
	"gopkg.in/ini.v1"
)

// 安装时读取的配置文件
type Prepare struct {
	RepmgrDeployMode      string `ini:"repmgr-deploy-mode"`
	SystemUser            string `ini:"system-user"`
	SystemGroup           string `ini:"system-group"`
	BindIP                string `ini:"bind-ip" comment:"监听IP，如果没有特殊要求请勿修改"`
	Port                  int    `ini:"port" comment:"监听端口，如果没有特殊要求请勿修改"`
	Dir                   string `ini:"dir" comment:"数据部署目录，请确认该目录存在，默认为/opt/pgsql+端口号，如无特殊要求请勿修改"`
	AdminPassword         string `ini:"admin-password" comment:"超级管理员密码, 必须填写"`
	AdminPasswordExpireAt string `ini:"admin-password-expire-at" comment:"超级管理员密码的过期时间"`
	Username              string `ini:"username" comment:"程序用于连接数据库的用户名，默认为pguser+端口号，如无特殊要求请勿修改"`
	Password              string `ini:"password" comment:"程序用于连接数据库的用户名的密码（为username参数所设置的用户的密码），留空会随机生成密码"`
	AdminAddress          string `ini:"admin-address" comment:"IP白名单，列入白名单的IP地址能够连接该数据库，无特殊要求请勿修改"`
	Address               string `ini:"address" comment:"IP白名单，列入白名单的IP地址能够连接该数据库，无特殊要求请勿修改"`
	MemorySize            string `ini:"memory-size" comment:"内存配置，建议内存配置不超过系统物理内存总量的50%，避免使用过程中系统物理内存耗尽造成内存溢出，默认为操作系统的50%，请根据实际部署环境进行调整，单位后缀可以为{MB,GB}"`
	ResourceLimit         string `ini:"resource-limit"`
	Ipv6                  bool   `ini:"ipv6"`
	Libraries             string `ini:"libraries"`
	RepmgrOwnerIP         string `ini:"repmgr-owner-ip"`
	RepmgrNodeID          int    `ini:"repmgr-node-id"`
	RepmgrUser            string `ini:"repmgr-user"`
	RepmgrPassword        string `ini:"repmgr-password"`
	RepmgrDBName          string `ini:"repmgr-dbname"`
	Yes                   bool   `ini:"yes" comment:"监听IP，如果没有特殊要求请勿修改"`
	NoRollback            bool   `ini:"no-rollback" comment:"监听IP，如果没有特殊要求请勿修改"`
}

// 初始化生成配置文件
func (p *Prepare) MakeConfigFile(cfgFile string) error {
	var err error
	if cfgFile, err = p.CfgPath(cfgFile); err != nil {
		return err
	}

	if utils.IsExists(cfgFile) {
		return fmt.Errorf("配置文件 ( %s ) 已存在, 请根据需要调整配置\n执行: [ dbup pgsql install --config=%s ] 命令安装pgsql程序", cfgFile, cfgFile)
	}

	p.InitArgs()

	// 生成配置文件就不验证参数了,验证参数在安装的时候
	//if err := p.Validator(); err != nil {
	//	return err
	//}

	if err := p.SlaveTo(cfgFile); err != nil {
		return err
	}
	logger.Successf("准备完成, 请根据需要调整配置文件: %s \n", cfgFile)
	logger.Successf("调整完成后,请执行: [ dbup pgsql install --config=%s ] 进行安装\n", cfgFile)
	return nil
}

// 确定配置文件位置
func (p *Prepare) CfgPath(cfgFile string) (string, error) {
	if cfgFile != "" {
		if !utils.IsExists(cfgFile) {
			return cfgFile, fmt.Errorf("指定的配置文件不存在: %s", cfgFile)
		}
		return cfgFile, nil
	}
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("使用默认配置文件, 获取当前用户家目录失败: %v", err)
	}
	cfgFile = filepath.Join(home, DefaultPGCfgFile)
	return cfgFile, nil
}

// Load 从配置文件加载配置到Prepare实例
func (p *Prepare) Load(filename string) error {
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

// SaveTo 将Prepare实例数据写入配置文件
func (p *Prepare) SlaveTo(filePath string) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true})
	if err := ini.ReflectFrom(cfg, p); err != nil {
		return fmt.Errorf("部署配置映射到(%s)文件错误: %v", filePath, err)
	}
	if err := cfg.SaveTo(filePath); err != nil {
		return fmt.Errorf("部署配置保存到(%s)文件错误: %v", filePath, err)
	}
	return nil
}

// 初始化参数
func (p *Prepare) InitArgs() {
	logger.Infof("初始化安装参数\n")

	if p.Username == "" {
		p.Username = DefaultPGUser
	}

	if p.Password == "" {
		p.Password = utils.GeneratePasswd(DefaultPGPassLength)
	}

	if p.Port == 0 {
		p.Port = p.RandomPort(DefaultPGPort)
	}

	if p.Dir == "" {
		p.Dir = fmt.Sprintf("%s%d", DefaultPGDir, p.Port)
	}

	if p.MemorySize == "" {
		//p.MemorySize = strconv.Itoa(p.InitMemory()) + "GB"
		p.MemorySize = "512M"
	}

	if p.BindIP == "" {
		p.BindIP = DefaultPGBindIP
	}

	if p.Address == "" {
		p.Address = DefaultPGAddress
	}

	if p.SystemUser == "" {
		p.SystemUser = DefaultPGAdminUser
	}

	if p.SystemGroup == "" {
		p.SystemGroup = DefaultPGAdminUser
	}
}

// 确定加入集群用哪个IP
func (p *Prepare) GetOwner() error {
	ips, err := utils.LocalIP()
	if err != nil {
		return err
	}

	if p.RepmgrOwnerIP == "" {
		if len(ips) == 1 {
			p.RepmgrOwnerIP = ips[0]
		} else {
			return fmt.Errorf("本机配置了多个IP地址, 请通过参数 --repmgr-owner-ip 手动指定使用哪个IP地址进行postgresql通信")
		}
	} else {
		if err := utils.IsIP(p.RepmgrOwnerIP); err != nil {
			h, e := os.Hostname()
			if e != nil {
				return fmt.Errorf("获取主机名失败")
			}

			if p.RepmgrOwnerIP == h {
				return nil
			} else {
				return fmt.Errorf("参数 --repmgr-owner-ip 不是正确的IP地址格式, 也不是本机主机名")
			}
		}

		if !arrlib.InArray(p.RepmgrOwnerIP, ips) {
			return fmt.Errorf("参数 --repmgr-owner-ip 手动指定的IP地址, 不是本机配置的IP地址, 请指定正确的本机地址")
		}
	}
	return nil
}

// 初始化参数
func (p *Prepare) InitSlaveArgs() {
	logger.Infof("初始化安装参数\n")
	if p.Port == 0 {
		p.Port = p.RandomPort(DefaultPGPort)
	}

	if p.Dir == "" {
		p.Dir = fmt.Sprintf("%s%d", DefaultPGDir, p.Port)
	}
}

// 到3万还没选出来,就用默认5432吧
func (p *Prepare) RandomPort(port int) int {
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

func (p *Prepare) InitMemory() int {
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
func (p *Prepare) Validator() error {
	logger.Infof("验证参数\n")
	// 用户名
	libs := strings.Split(p.Libraries, ",")
	IsLib := true
	for _, lib := range libs {
		if lib != "timescaledb" {
			IsLib = false
			break
		}
	}

	if p.Libraries != "" && !IsLib {
		return fmt.Errorf("目前只支持 timescaledb 插件")
	}

	r, _ := regexp.Compile(RegexpUsername)
	if ok := r.MatchString(p.Username); !ok {
		return fmt.Errorf("用户名(%s)不符合规则: 2到63位小写字母,数字,下划线; 不能以数字开头", p.Username)
	}

	if p.Username == DefaultPGAdminUser {
		return fmt.Errorf("禁止以 %s 做为用户名", p.Username)
	}

	// 密码
	if p.AdminPassword == "" {
		return fmt.Errorf("请指定 pgsql 的超级管理员密码, 以方便日后维护数据库")
	}
	if err := utils.CheckPasswordLever(p.AdminPassword); err != nil {
		return err
	}

	if p.AdminPasswordExpireAt != "" {
		ex := strings.Fields(p.AdminPasswordExpireAt)
		if len(ex) > 3 {
			return fmt.Errorf("过期时间格式错误, 正确示例: <2021-01-01> <2021-01-01 24:00+8> <2021-01-01 24:00:00+15:59:59>")
		} else if len(ex) == 3 {
			ex[1] = ex[1] + ex[2]
		}

		r1, _ := regexp.Compile("^[12][0-9]{3}-(0?[1-9]|1[0-2])-((0?[1-9])|((1|2)[0-9])|30|31)$")
		if ok := r1.MatchString(ex[0]); !ok {
			return fmt.Errorf("过期时间格式错误, 正确示例: <2021-01-01> <2021-01-01 24:00+8> <2021-01-01 24:00:00+15:59:59>")
		}
		fmt.Println(ex[1])
		if len(ex) == 2 && ex[1] != "" {
			r2, _ := regexp.Compile("^^((0?[1-9])|(1[0-9])|(2[0-3])):((0?[0-9])|([1-5][0-9]))(:((0?[0-9])|([1-5][0-9])))?([-+]((0?[1-9])|(1[0-5]))(:((0?[0-9])|([1-5][0-9]))){0,2})?$")
			if ok := r2.MatchString(ex[1]); !ok {
				return fmt.Errorf("过期时间格式错误, 正确示例: <2021-01-01> <2021-01-01 24:00+8> <2021-01-01 24:00:00+15:59:59>")
			}
		}
	}

	// 端口
	if p.Port < 1025 || p.Port > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", p.Port)
	}

	// 绑定IP, 是不是一个IP地址
	if p.BindIP != "localhost" && p.BindIP != "*" {
		if err := utils.IsIPv4(p.BindIP); err != nil {
			return err
		}
	}

	if err := p.ValidatorAddress(); err != nil {
		return err
	}

	if err := p.ValidatorAdminAddress(); err != nil {
		return err
	}

	if strings.Contains(p.Libraries, "repmgr") {
		if p.RepmgrNodeID == 0 || p.RepmgrUser == "" || p.RepmgrPassword == "" || p.RepmgrDBName == "" {
			return fmt.Errorf("请正确设置repmgr相关参数")
		}
	}

	return nil
}

// 验证配置
func (p *Prepare) ValidatorSlave() error {
	if p.Username == "" {
		return fmt.Errorf("请指定主从同步用的用户名")
	}

	// 端口
	if p.Port < 1025 || p.Port > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", p.Port)
	}

	return nil
}

// 验证内存
func (p *Prepare) ValidatorMemorySize() error {
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

	if memory > max {
		return fmt.Errorf("PG内存不能超过操作系统最大物理内存大小")
	}

	// 验证函数不应该包含赋值操作
	//p.MemorySize = strconv.Itoa(memory) + suffix
	return nil
}

// 验证授权IP
func (p *Prepare) ValidatorAddress() error {
	addrs := strings.Split(p.Address, ",")
	v := validator.New()
	for _, addr := range addrs {
		if addr == "localhost" || addr == "local" {
			continue
		}
		//ipMask := strings.Split(addr, "/")
		//switch len(ipMask) {
		//case 0:
		//	fmt.Println("空字符串忽略")
		//case 1:
		//	return utils.IsIPv4(addr)
		//case 2:
		//	return utils.IsIPv4Mask(addr)
		//default:
		//	return fmt.Errorf("授权IP字符串不符合规则")
		//}
		if err := utils.CheckAddressFormat(addr); err != nil {
			if e := v.Var(addr, "hostname"); e != nil {
				return fmt.Errorf("%s 不是有效IP地址, 也不是规范主机名", addr)
			}
		}
	}
	return nil
}

// 验证授权IP
func (p *Prepare) ValidatorAdminAddress() error {
	if p.AdminAddress == "" {
		return nil
	}
	addrs := strings.Split(p.AdminAddress, ",")
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
func (p *Prepare) CheckEnv() error {
	// 数据目录
	if err := utils.ValidatorDir(p.Dir); err != nil {
		return err
	}

	if utils.PortInUse(p.Port) {
		return fmt.Errorf("端口号被占用: %d", p.Port)
	}

	// 检查本地 ipv6 地址解析情况
	if p.Ipv6 {
		if err := utils.Ipv6Check(); err != nil {
			return err
		}
	}

	// 内存不大于机器最大内存
	if err := p.ValidatorMemorySize(); err != nil {
		return err
	}
	return nil
}
