// Created by LiuSainan on 2022-06-09 15:50:59

package config

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/utils"
	"dbup/internal/utils/arrlib"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/mem"
	"gopkg.in/ini.v1"
)

// type MariaDBGaleraOptions struct {
// 	SystemUser            string `ini:"system-user"`
// 	SystemGroup           string `ini:"system-group"`
// 	Port                  int    `ini:"port"`
// 	Dir                   string `ini:"dir"`
// 	Password              string `ini:"password"`
// 	Memory                string `ini:"memory"`
// 	OwnerIP               string `ini:"owner-ip"`
// 	ResourceLimit         string `ini:"resource-limit"`
// 	wsrep_node_name       string `ini:"wsrep_node_name"`
// 	wsrep_node_address    string `ini:"wsrep_node_address"`
// 	wsrep_cluster_address string `ini:"wsrep_cluster_address"`
// }

type MariaDBOptions struct {
	SystemUser          string `ini:"system-user"`
	SystemGroup         string `ini:"system-group"`
	Clustermode         string `ini:"cluster-mode"`
	Port                int    `ini:"port"`
	Dir                 string `ini:"dir"`
	Repluser            string `ini:"repluser"`
	ReplPassword        string `ini:"replpassword"`
	Password            string `ini:"password"`
	Memory              string `ini:"memory"`
	Role                string `ini:"role"`
	OwnerIP             string `ini:"owner-ip"`
	TxIsolation         string
	Join                string `ini:"join" validate:"ipPort"`
	ResourceLimit       string `ini:"resource-limit"`
	BackupData          bool   `ini:"no"`
	AddSlave            bool   `ini:"no"`
	AutoIncrement       int    `ini:"no"`
	Yes                 bool   `ini:"yes"`
	NoRollback          bool   `ini:"no-rollback"`
	Wsrepclusteraddress string `ini:"wsrep_cluster_address"`
	Backupuser          string
	BackupPassword      string
	Galera              bool
	Onenode             bool
}

// 初始化参数
func (option *MariaDBOptions) Parameter() {
	if option.Port == 0 {
		option.Port = utils.RandomPort(DefaultMariaDBPort)
	}

	if option.Dir == "" {
		option.Dir = fmt.Sprintf(DefaultMariaDBBaseDir, option.Port)
	}

	if option.SystemUser == "" {
		option.SystemUser = DefaultMariaDBSystemUser
	}

	if option.SystemGroup == "" {
		option.SystemGroup = DefaultMariaDBSystemGroup
	}

	if option.Memory == "" {
		option.Memory = "512M"
	}

	if option.Role == "" {
		option.Role = MariaDBMasterRole
	}

	if option.Password == "" {
		option.Password = utils.GeneratePasswd(DefaultMariaDBPassLength)
	}

	if option.Backupuser == "" {
		option.Backupuser = MariaDBBackupUser
	}

	if option.BackupPassword == "" {
		option.BackupPassword = DefaultMariaDBBakPassword
	}

	if option.Repluser == "" {
		option.Repluser = MariaDBReplicationUser
	}

	if option.ReplPassword == "" {
		option.ReplPassword = DefaultMariaDBReplPassword
	}
}

func (option *MariaDBOptions) GaleraParameter() {
	if option.Port == 0 {
		option.Port = utils.RandomPort(DefaultMariaDBPort)
	}

	if option.Dir == "" {
		option.Dir = fmt.Sprintf(DefaultMariaDBBaseDir, option.Port)
	}

	if option.SystemUser == "" {
		option.SystemUser = DefaultMariaDBSystemUser
	}

	if option.SystemGroup == "" {
		option.SystemGroup = DefaultMariaDBSystemGroup
	}

	if option.Memory == "" {
		option.Memory = "512M"
	}

	if option.Role == "" {
		option.Role = MariaDBMasterRole
	}

	if option.Password == "" {
		option.Password = utils.GeneratePasswd(DefaultMariaDBPassLength)
	}
}

// Validator 检查参数是否合理
func (option *MariaDBOptions) Validator() error {
	// 端口
	if option.Port < 1025 || option.Port > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", option.Port)
	}

	if err := utils.CheckPasswordLever(option.Password); err != nil {
		return err
	}

	if err := utils.CheckPasswordLever(option.ReplPassword); err != nil {
		return err
	}

	if option.Join != "" {
		option.Role = MariaDBSlaveRole
		ipPort := strings.Split(option.Join, ":")
		if err := utils.IsIPv4(ipPort[0]); err != nil {
			if !utils.IsHostName(ipPort[0]) {
				return fmt.Errorf("--join 参数的地址部分即不是IP地址, 也不是有效的主机名")
			}
		}
		if len(ipPort) > 1 {
			port, err := strconv.Atoi(ipPort[1])
			if err != nil {
				return fmt.Errorf("--join 参数的端口部分不是有效的端口号")
			}
			if port <= 0 || port >= 65536 {
				return fmt.Errorf("--join 参数的端口部分不是有效的端口号")
			}
		}
	}

	// 验证并设置 transaction-isolation 为 READ-COMMITTED || REPEATABLE-READ
	if option.TxIsolation == "" {
		option.TxIsolation = "RC"
	}

	if option.TxIsolation != "RR" && option.TxIsolation != "RC" {
		return fmt.Errorf("事务隔离级别(%s), 必须为 RR 或 RC ", option.TxIsolation)
	}

	if option.TxIsolation == "RC" {
		option.TxIsolation = DefaultMariaDBtxisolation
	} else if option.TxIsolation == "RR" {
		option.TxIsolation = "REPEATABLE-READ"
	}

	return nil
}

// 检查本地操作系统环境
func (option *MariaDBOptions) Environment() error {
	// 数据目录
	if err := utils.ValidatorDir(option.Dir); err != nil {
		return err
	}

	if utils.PortInUse(option.Port) {
		return fmt.Errorf("端口号被占用: %d", option.Port)
	}

	serviceFileName := fmt.Sprintf(ServiceFileName, option.Port)
	serviceFileFullName := filepath.Join(global.ServicePath, serviceFileName)
	if utils.IsExists(serviceFileFullName) {
		return fmt.Errorf("启动文件(%s)已经存在, 停止安装", serviceFileFullName)
	}

	if err := option.ValidatorMemorySize(); err != nil {
		return err
	}

	if option.Galera {
		if err := option.GaleraCheckCmd(); err != nil {
			return err
		}
	}

	if err := option.GetOwner(); err != nil {
		return err
	}

	return nil
}

// 验证 galera 本地是否安装数据拉取必备命令 socat
func (option *MariaDBOptions) GaleraCheckCmd() error {
	if !command.CheckCommandExists("socat") {
		return fmt.Errorf("galera 依赖的数据同步工具 socat 未安装, 请安装: yum install socat")
	}
	return nil
}

// 验证内存
func (option *MariaDBOptions) ValidatorMemorySize() error {
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
	index := r.FindStringIndex(option.Memory)
	if index == nil {
		return fmt.Errorf("内存参数必须包含单位后缀(MB 或 GB)")
	}

	if memory, err = strconv.Atoi(option.Memory[:index[0]]); err != nil {
		return err
	}

	suffix = strings.ToUpper(option.Memory[index[0]:])

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

// 确定加入集群用哪个IP
func (option *MariaDBOptions) GetOwner() error {
	ips, err := utils.LocalIP()
	if err != nil {
		return err
	}

	if option.OwnerIP == "" {
		if len(ips) == 1 {
			option.OwnerIP = ips[0]
		} else {
			return fmt.Errorf("本机配置了多个IP地址, 请通过参数 --owner-ip 手动指定使用哪个IP地址进行MariaDB通信")
		}
	} else {
		if err := utils.IsIP(option.OwnerIP); err != nil {
			h, e := os.Hostname()
			if e != nil {
				return fmt.Errorf("获取主机名失败")
			}

			if option.OwnerIP != h {
				return fmt.Errorf("参数 --owner-ip 不是正确的IP地址格式, 也不是本机主机名")
			}
		}

		if !arrlib.InArray(option.OwnerIP, ips) {
			return fmt.Errorf("参数 --owner-ip 手动指定的IP地址, 不是本机配置的IP地址, 请指定正确的本机地址")
		}
	}
	return nil
}

type Server struct {
	Address  string `ini:"address"`
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

func (s *Server) Checkport() error {
	// 端口
	if s.SshPort < 1 || s.SshPort > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", s.SshPort)
	}

	return nil
}

func (s *Server) CheckGaleraNode() error {

	s.SetDefault()

	logger.Infof("验证 Galera server 参数\n")
	ips := strings.Split(s.Address, ",")

	if len(ips) < 3 {
		return fmt.Errorf("部署 Galera 集群必须为三个或以上地址")
	}

	for _, ip := range ips {
		if err := utils.IsIPv4(ip); err != nil {
			if !utils.IsHostName(ip) {
				return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", ip, err)
			}
		}
	}

	// 端口
	if err := s.Checkport(); err != nil {
		return err
	}

	return nil
}

// 验证配置
func (s *Server) Validator() error {
	logger.Infof("验证 server 参数\n")
	ips := strings.Split(s.Address, ",")

	if len(ips) < 2 {
		return fmt.Errorf("部署IP必须为两个或以上地址")
	}

	for _, ip := range ips {
		if err := utils.IsIPv4(ip); err != nil {
			if !utils.IsHostName(ip) {
				return fmt.Errorf("host (%s) 即不是一个 IP 地址(%v), 又解析主机名失败", ip, err)
			}
		}
	}

	// 端口
	if err := s.Checkport(); err != nil {
		return err
	}

	return nil
}

type MariaDBDeployOptions struct {
	Server  Server         `ini:"server"`
	MariaDB MariaDBOptions `ini:"mariadb"`
	// Clustermode string         `ini:"cluster-mode"`
	Yes        bool `ini:"yes" comment:"监听IP，如果没有特殊要求请勿修改"`
	NoRollback bool `ini:"no-rollback" comment:"监听IP，如果没有特殊要求请勿修改"`
}

// Load 从配置文件加载配置到Prepare实例
func (o *MariaDBDeployOptions) Load(filename string) error {
	return global.INILoadFromFile(filename, o, ini.LoadOptions{
		SpaceBeforeInlineComment: true,
	})
}

// 检查集群的模式
func (o *MariaDBDeployOptions) ClusterModeCheck() error {
	if o.MariaDB.Clustermode == "" {
		o.MariaDB.Clustermode = MariaDBModeMS
	} else {

		if !strings.Contains("MMS", o.MariaDB.Clustermode) {
			return fmt.Errorf("cluster-mode 部署模式的必须选择 MS 或 MM")
		}

		if o.MariaDB.Clustermode == "MM" {
			ips := strings.Split(o.Server.Address, ",")

			if len(ips) != 2 {
				return fmt.Errorf("部署双主模式的 Mariadb , IP 必须为两个")
			}
		}
	}

	return nil
}

func (o *MariaDBDeployOptions) Validator() error {

	if err := o.Server.Validator(); err != nil {
		return err
	}

	if err := o.ClusterModeCheck(); err != nil {
		return err
	}

	if o.MariaDB.Password == "" {
		return fmt.Errorf("主从环境为保证一致性 root 账号密码不能为空")
	} else {
		if err := utils.CheckPasswordLever(o.MariaDB.Password); err != nil {
			return err
		}
	}

	return nil
}

func (o *MariaDBDeployOptions) GaleraPortCheck() error {
	// 检查 galera 成员通信端口
	ips := strings.Split(o.Server.Address, ",")
	for _, ip := range ips {
		galerabaseport := fmt.Sprintf("%s:%d", ip, DefaultGalerabaseport)
		ok, _ := utils.TcpGather(galerabaseport)
		if ok {
			return fmt.Errorf("galera 集群成员 %s 的通信端口 %d 已存在, 请检查", ip, DefaultGalerabaseport)
		}
	}

	return nil
}

func (o *MariaDBDeployOptions) GaleraValidator() error {

	if err := o.Server.CheckGaleraNode(); err != nil {
		return err
	}

	// 检查 galera 成员通信端口
	if err := o.GaleraPortCheck(); err != nil {
		return err
	}

	// 检查配置的 root 密码
	if o.MariaDB.Password == "" {
		return fmt.Errorf("主从环境为保证一致性 root 账号密码不能为空")
	} else {
		if err := utils.CheckPasswordLever(o.MariaDB.Password); err != nil {
			return err
		}
	}

	return nil
}
