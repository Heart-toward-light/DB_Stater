package config

import (
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-playground/validator"
	"gopkg.in/ini.v1"
)

type PGAutoFailoverMonitor struct {
	SystemUser            string `ini:"system-user"`
	SystemGroup           string `ini:"system-group"`
	Host                  string `ini:"host" comment:"机器的ip或主机名"`
	Port                  int    `ini:"port" comment:"监听端口，如果没有特殊要求请勿修改"`
	Dir                   string `ini:"dir" comment:"数据部署目录，请确认该目录存在，默认为/opt/pgsql+端口号，如无特殊要求请勿修改"`
	AdminPassword         string `ini:"admin-password" comment:"超级管理员密码, 必须填写"`
	AdminPasswordExpireAt string `ini:"admin-password-expire-at" comment:"超级管理员密码的过期时间"`
	// Auth                  string `ini:"auth" comment:"实例认证模式"`
	// SslSelfSigned         bool   `ini:"ssl-self-signed" comment:"是否开启网络机密，默认开启"`
	Yes        bool `ini:"yes" comment:"监听IP，如果没有特殊要求请勿修改"`
	NoRollback bool `ini:"no-rollback" comment:"监听IP，如果没有特殊要求请勿修改"`
}

// 验证配置
func (m *PGAutoFailoverMonitor) Validator() error {
	if m.Port == 0 {
		return fmt.Errorf("monitor 需要指定节点端口号")
	}

	if utils.PortInUse(m.Port) {
		return fmt.Errorf("monitor 节点端口号已经被占用: %d", m.Port)
	}

	if m.Host == "" {
		return fmt.Errorf("monitor 节点需要指定本地ip")
	}

	if m.Dir == "" {
		m.Dir = fmt.Sprintf("%s%d", DefaultPGMonitorDir, m.Port)
	}

	if m.SystemUser == "" {
		m.SystemUser = DefaultPGAdminUser
	}

	if m.SystemGroup == "" {
		m.SystemGroup = DefaultPGAdminUser
	}

	if err := Checkfile(m.SystemUser, filepath.Join(m.Dir, "data")); err != nil {
		return err
	}

	return nil
}

type PGAutoFailoverPGNode struct {
	SystemUser            string `ini:"system-user"`
	SystemGroup           string `ini:"system-group"`
	Mhost                 string `ini:"Mhost" comment:"监控机实例的IP地址或域名"`
	Mport                 int    `ini:"Mpost" comment:"监控机实例的端口"`
	BindIP                string `ini:"bind-ip" comment:"监听IP，如果没有特殊要求请勿修改"`
	Port                  int    `ini:"port" comment:"监听端口，如果没有特殊要求请勿修改"`
	Host                  string `ini:"host" comment:"机器的ip或主机名"`
	Dir                   string `ini:"dir" comment:"数据部署目录，请确认该目录存在，默认为/opt/pgsql+端口号，如无特殊要求请勿修改"`
	AdminPassword         string `ini:"admin-password" comment:"超级管理员密码, 必须填写"`
	AdminPasswordExpireAt string `ini:"admin-password-expire-at" comment:"超级管理员密码的过期时间"`
	Username              string `ini:"username" comment:"程序用于连接数据库的用户名，默认为pguser+端口号，如无特殊要求请勿修改"`
	Password              string `ini:"password" comment:"程序用于连接数据库的用户名的密码（为username参数所设置的用户的密码），留空会随机生成密码"`
	AdminAddress          string `ini:"admin-address" comment:"IP白名单，列入白名单的IP地址能够连接该数据库，无特殊要求请勿修改"`
	Address               string `ini:"address" comment:"IP白名单，列入白名单的IP地址能够连接该数据库，无特殊要求请勿修改"`
	MemorySize            string `ini:"memory-size" comment:"内存配置，建议内存配置不超过系统物理内存总量的50%,单位后缀可以为{MB,GB}"`
	ResourceLimit         string `ini:"resource-limit"`
	Libraries             string `ini:"libraries"`
	AllNode               string `ini:"allnode"`
	Yes                   bool   `ini:"yes" comment:"监听IP，如果没有特殊要求请勿修改"`
	NoRollback            bool   `ini:"no-rollback" comment:"监听IP，如果没有特殊要求请勿修改"`
	Onenode               bool
}

// Load 从配置文件加载配置到数据节点的实例
func (d *PGAutoFailoverPGNode) Load(filename string) error {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		SpaceBeforeInlineComment: true,
	}, filename)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %v", err)
	}

	if err = cfg.MapTo(d); err != nil {
		return fmt.Errorf("配置文件映射到结构体失败: %v", err)
	}
	return nil
}

// 初始化参数
func (d *PGAutoFailoverPGNode) InitArgs() {
	logger.Infof("初始化安装参数\n")

	if d.Username == "" {
		d.Username = DefaultPGUser
	}

	if d.Password == "" {
		d.Password = utils.GeneratePasswd(DefaultPGPassLength)
	}

	if d.Port == 0 {
		d.Port = d.RandomPort(DefaultPGPort)
	}

	if d.Dir == "" {
		d.Dir = fmt.Sprintf("%s%d", DefaultPGDir, d.Port)
	}

	if d.MemorySize == "" {
		//p.MemorySize = strconv.Itoa(p.InitMemory()) + "GB"
		d.MemorySize = "512M"
	}

	if d.BindIP == "" {
		d.BindIP = DefaultPGBindIP
	}

	if d.Address == "" {
		d.Address = DefaultPGAddress
	}

	if d.SystemUser == "" {
		d.SystemUser = DefaultPGAdminUser
	}

	if d.SystemGroup == "" {
		d.SystemGroup = DefaultPGAdminUser
	}

}

// 到3万还没选出来,就用默认5432吧
func (d *PGAutoFailoverPGNode) RandomPort(port int) int {
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

// 验证授权IP
func (d *PGAutoFailoverPGNode) ValidatorAdminAddress() error {
	if d.AdminAddress == "" {
		return nil
	}
	addrs := strings.Split(d.AdminAddress, ",")
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

// 验证授权IP
func (d *PGAutoFailoverPGNode) ValidatorAddress() error {
	addrs := strings.Split(d.Address, ",")
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

// 验证主机名与IP
func (d *PGAutoFailoverPGNode) CheckHost() error {
	if err := utils.IsIP(d.Host); err != nil {
		h, e := os.Hostname()
		if e != nil {
			return fmt.Errorf("获取主机名失败")
		}
		if d.Host != h {
			return fmt.Errorf("%s 不是正确的IP地址格式, 也不是本机主机名", d.Host)
		}
	}

	return nil
}

func (d *PGAutoFailoverPGNode) Checkservice() error {
	if d.Mhost == "" {
		return fmt.Errorf("必须指定监控节点IP或域名")
	}

	if d.Mport == 0 {
		return fmt.Errorf("必须指定监控节点端口")
	}

	if d.Host == "" {
		return fmt.Errorf("必须指定本地节点的IP地址或域名")
	} else {
		if err := d.CheckHost(); err != nil {
			return fmt.Errorf("指定的参数 --host  %s ", err)
		}
	}

	if utils.PortInUse(d.Port) {
		return fmt.Errorf("pgdata 端口号已经被占用: %d", d.Port)
	}

	return nil
}

func (d *PGAutoFailoverPGNode) Checkmonitor() error {
	PGbaseport := fmt.Sprintf("%s:%d", d.Mhost, d.Mport)
	ok, _ := utils.TcpGather(PGbaseport)
	if !ok {
		return fmt.Errorf("PG_auto_failover  监控节点服务 %s:%d 不存在, 请检查", d.Mhost, d.Mport)
	}
	return nil
}

// 验证配置
func (d *PGAutoFailoverPGNode) Validator() error {
	logger.Infof("验证参数\n")

	if d.AllNode == "" {
		return fmt.Errorf("需要指定集群中所有数据节点进行弱密码加密,示例 IP:PORT,IP:PORT")
	}

	if d.Libraries != "timescaledb" && d.Libraries != "" {
		return fmt.Errorf("目前只支持 timescaledb 插件")
	}

	r, _ := regexp.Compile(RegexpUsername)
	if ok := r.MatchString(d.Username); !ok {
		return fmt.Errorf("用户名(%s)不符合规则: 2到63位小写字母,数字,下划线; 不能以数字开头", d.Username)
	}

	if err := utils.CheckPasswordLever(d.Password); err != nil {
		return err
	}

	if d.Username == DefaultPGAdminUser {
		return fmt.Errorf("禁止以 %s 做为用户名", d.Username)
	}

	if d.AdminPassword == "" {
		return fmt.Errorf("请指定 pgsql 的超级管理员密码, 以方便日后维护数据库")
	}
	if err := utils.CheckPasswordLever(d.AdminPassword); err != nil {
		return err
	}

	if d.AdminPasswordExpireAt != "" {
		ex := strings.Fields(d.AdminPasswordExpireAt)
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
	if d.Port < 1025 || d.Port > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", d.Port)
	}

	// 绑定IP, 是不是一个IP地址
	if d.BindIP != "localhost" && d.BindIP != "*" {
		if err := utils.IsIPv4(d.BindIP); err != nil {
			return err
		}
	}

	if err := d.ValidatorAddress(); err != nil {
		return err
	}

	if err := d.ValidatorAdminAddress(); err != nil {
		return err
	}

	if err := Checkfile(d.SystemUser, filepath.Join(d.Dir, "data")); err != nil {
		return err
	}

	return nil
}

// 判断实例默认文件是否存在
func Checkfile(user, datadir string) error {
	var filelist []string
	Cfile := fmt.Sprintf(Config_file, user, datadir)
	Sfile := fmt.Sprintf(State_file, user, datadir)
	Ifile := fmt.Sprintf(Init_file, user, datadir)
	filelist = append(filelist, Cfile, Sfile, Ifile)

	for _, filename := range filelist {
		if utils.IsExists(filename) {
			return fmt.Errorf("安装前默认文件 %s 不能存在,请检查", filename)
		}
	}
	return nil
}
