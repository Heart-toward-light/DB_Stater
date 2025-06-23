/*
@Author : WuWeiJian
@Date : 2021-05-10 16:09
*/

package service

import (
	"context"
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/mongodb/config"
	"dbup/internal/mongodb/dao"
	"dbup/internal/utils"
	"dbup/internal/utils/arrlib"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/ini.v1"
)

// 安装pgsql的总控制逻辑
type MongoDBInstall struct {
	Option          *config.MongodbOptions
	Config          *config.MongoDBConfig
	ShardConfig     *config.MongoDBShardConfig
	Service         *config.MongoDBService
	KeyFileContent  string
	Role            string
	JoinIP          string
	JoinPort        int
	Owner           string
	ReplSetName     string
	SysUser         string
	SysGroup        string
	PackageFullName string
}

func NewMongoDBInstall(option *config.MongodbOptions) *MongoDBInstall {
	return &MongoDBInstall{
		Option:          option,
		Role:            config.MongoDBPrimary,
		PackageFullName: filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultMongoDBVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)),
	}
}

// 检查service文件是否存在, 如果存在退出安装, 检查mongodb安装包是否存在且md5正确
func (i *MongoDBInstall) CheckEnv() error {
	if err := i.Option.CheckEnv(); err != nil {
		return err
	}
	serviceFileName := fmt.Sprintf(config.ServiceFileName, i.Option.Port)
	serviceFileFullName := filepath.Join(global.ServicePath, serviceFileName)
	if utils.IsExists(serviceFileFullName) {
		return fmt.Errorf("启动文件(%s)已经存在, 停止安装", serviceFileFullName)
	}

	if err := i.GetOwner(); err != nil {
		return err
	}

	return global.CheckPackage(environment.GlobalEnv().ProgramPath, i.PackageFullName, config.Kinds)
}

// 确定加入集群用哪个IP
func (i *MongoDBInstall) GetOwner() error {
	h, e := os.Hostname()
	if e != nil {
		return fmt.Errorf("获取主机名失败")
	}
	if i.Option.Ipv6 {
		if i.Option.Owner != h {
			return fmt.Errorf("开启IPV6部署功能需要指定本地主机名进行mongodb通信")
		}

		if err := i.Option.CheckIPV6(); err != nil {
			return err
		}
	}

	ips, err := utils.LocalIP()
	if err != nil {
		return err
	}

	if i.Option.Owner == "" {
		if len(ips) == 1 {
			i.Owner = ips[0]
		} else {
			return fmt.Errorf("本机配置了多个IP地址, 请通过参数 --owner 手动指定使用哪个IP地址进行mongodb通信")
		}
	} else {
		if err := utils.IsIP(i.Option.Owner); err != nil {

			if i.Option.Owner == h {
				i.Owner = i.Option.Owner
				return nil
			} else {
				return fmt.Errorf("参数 --owner 不是正确的IP地址格式, 也不是本机主机名")
			}
		}

		if arrlib.InArray(i.Option.Owner, ips) {
			i.Owner = i.Option.Owner
		} else {
			return fmt.Errorf("参数 --owner 手动指定的IP地址, 不是本机配置的IP地址, 请指定正确的本机地址")
		}
	}
	return nil
}

// 检查service文件是否存在, 如果存在退出安装, 检查mongodb安装包是否存在且md5正确
func (i *MongoDBInstall) GetKeyFileContent(filename string) (string, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		SpaceBeforeInlineComment: true,
	}, filename)
	if err != nil {
		return "", fmt.Errorf("获取keyfile信息失败: %v", err)
	}

	s := cfg.Section(config.Kinds).Key("key_file").MustString("")
	return s, nil
}

func (i *MongoDBInstall) Run() error {
	if err := i.HandleArgs(); err != nil {
		return err
	}
	if !i.Option.Yes {
		var yes string
		if i.Option.Join != "" {
			logger.Successf("\n")
			logger.Successf("本次安装实例为(SECONDARY)从节点\n")
			logger.Successf("要加入的集群为: %s\n", i.Option.Join)
			logger.Successf("\n")
		}
		logger.Successf("端口: %d\n", i.Option.Port)
		logger.Successf("副本集名称: %s\n", i.ReplSetName)
		logger.Successf("用户: %s\n", i.Option.Username)
		logger.Successf("密码: %s\n", i.Option.Password)
		logger.Successf("安装路径: %s\n", i.Option.Dir)
		logger.Successf("是否确认安装[y|n]:")
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	if err := i.InstallAndInitDB(); err != nil {
		if !i.Option.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			uninstall := MongoDBUNInstall{Port: i.Option.Port, BasePath: i.Option.Dir}
			uninstall.Uninstall()
		}
		return err
	}

	// 整个过程结束，生成连接信息文件, 并返回MongoDB用户名、密码、授权IP
	i.Info()
	return nil
}

func (i *MongoDBInstall) HandleArgs() error {
	var err error
	i.SysUser = i.Option.SystemUser
	i.SysGroup = i.Option.SystemGroup

	if i.KeyFileContent, err = i.GetKeyFileContent(filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, global.Md5FileName)); err != nil {
		return err
	}
	i.ReplSetName = i.Option.ReplSetName

	if i.Option.Arbiter && i.Option.Join == "" {
		return fmt.Errorf("创建arbiter节点, 必须要指定要加入的集群IP")
	}

	if i.Option.Join != "" {
		if i.Option.Arbiter {
			i.Role = config.MongoDBArbiter
		} else {
			i.Role = config.MongoDBSecondary
		}
		ipPort := strings.Split(i.Option.Join, ":")
		var port int
		if len(ipPort) > 1 {
			port, _ = strconv.Atoi(ipPort[1]) // 在自定义验证器 ValidateIPPort 里已经验证过了, 不需要返回错误
		} else {
			port = i.Option.Port
		}
		conn, err := dao.NewMongoClient(ipPort[0], port, i.Option.Username, i.Option.Password, "admin")
		if err != nil {
			return err
		}
		defer conn.Conn.Disconnect(context.Background())

		r, err := conn.DBisMaster()
		if err != nil {
			return fmt.Errorf("访问 join 节点( %s )失败: %v", i.Option.Join, err)
		}
		i.JoinIP = strings.Split(r["primary"].(string), ":")[0]
		if i.JoinPort, err = strconv.Atoi(strings.Split(r["primary"].(string), ":")[1]); err != nil {
			return err
		}
		i.ReplSetName = r["setName"].(string)
		if i.Option.ReplSetName != "" && i.Option.ReplSetName != i.ReplSetName {
			return fmt.Errorf("指定的副本集名称与要加入的集群副本集名称不一致")
		}
	}

	if ok, _ := regexp.MatchString("Shard.*-", i.ReplSetName); ok {
		i.ShardConfig = config.NewMongoDBShardConfig(i.Option, i.ReplSetName)
	} else if strings.Contains(i.ReplSetName, "Config-") {
		i.ShardConfig = config.NewMongoDBShardConfig(i.Option, i.ReplSetName)
	} else {
		i.Config = config.NewMongoDBConfig(i.Option, i.ReplSetName)
	}

	if i.Service, err = config.NewMongoDBService(filepath.Join(environment.GlobalEnv().ProgramPath, global.ServiceTemplatePath, config.MongoDBServiceTemplateFile)); err != nil {
		return err
	}
	return i.Service.FormatBody(i.Option, i.SysUser, i.SysGroup)
}

func (i *MongoDBInstall) InstallAndInitDB() error {
	service := fmt.Sprintf(config.ServiceFileName, i.Option.Port)
	if err := i.Install(service); err != nil {
		return err
	}
	logger.Infof("启动实例\n")
	if err := command.SystemCtl(service, "start"); err != nil {
		return err
	}
	logger.Infof("初始化\n")
	switch i.Role {
	case config.MongoDBPrimary:
		if err := i.InitPrimary(); err != nil {
			return err
		}
	case config.MongoDBSecondary:
		if err := i.InitSecondary(); err != nil {
			return err
		}
	case config.MongoDBArbiter:
		if err := i.InitArbiter(); err != nil {
			return err
		}
	}
	return nil
}

func (i *MongoDBInstall) CreateUser() error {
	logger.Infof("创建启动用户: %s\n", i.SysUser)
	u, err := user.Lookup(i.SysUser)
	if err == nil { // 如果用户已经存在,则i.adminGroup设置为真正的所属组名
		g, _ := user.LookupGroupId(u.Gid)
		i.SysGroup = g.Name
		return nil
	}
	// groupadd -f <group-name>
	groupAdd := fmt.Sprintf("%s -f %s", command.GroupAddCmd, i.SysGroup)

	// useradd -g <group-name> <user-name>
	userAdd := fmt.Sprintf("%s -g %s %s", command.UserAddCmd, i.SysGroup, i.SysUser)

	l := command.Local{}
	if _, stderr, err := l.Run(groupAdd); err != nil {
		return fmt.Errorf("创建用户组(%s)失败: %v, 标准错误输出: %s", i.SysGroup, err, stderr)
	}
	if _, stderr, err := l.Run(userAdd); err != nil {
		return fmt.Errorf("创建用户(%s)失败: %v, 标准错误输出: %s", i.SysUser, err, stderr)
	}
	return nil
}

func (i *MongoDBInstall) Mkdir() error {
	logger.Infof("创建数据目录和程序目录\n")
	if err := os.MkdirAll(environment.GlobalEnv().DbupInfoPath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(i.Option.Dir, config.DefaultMongoDBConfigDir), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(i.Option.Dir, config.DefaultMongoDBLogDir), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(i.Option.Dir, config.DefaultMongoDBDataDir), 0755); err != nil {
		return err
	}

	return nil
}

func (i *MongoDBInstall) ChownDir(path string) error {
	cmd := fmt.Sprintf("chown -R %s:%s %s", i.SysUser, i.SysGroup, path)
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (i *MongoDBInstall) Install(service string) error {
	logger.Infof("开始安装\n")
	serviceFile := filepath.Join(global.ServicePath, service)
	//检查并创建 mongod 账号
	if err := i.CreateUser(); err != nil {
		return err
	}

	// 创建子目录
	if err := i.Mkdir(); err != nil {
		return err
	}

	// 解压安装包
	logger.Infof("解压安装包: %s 到 %s \n", i.PackageFullName, i.Option.Dir)
	if err := utils.UntarGz(i.PackageFullName, i.Option.Dir); err != nil {
		return err
	}

	// 检查依赖
	if missLibs, err := global.Checkldd(filepath.Join(i.Option.Dir, config.DefaultMongoDBBinDir, config.DefaultMongoDBBinFile)); err != nil {
		return err
	} else {
		if len(missLibs) != 0 {
			if err := i.LibComplement(missLibs); err != nil {
				return err
			}
		}
	}

	if ok, _ := regexp.MatchString("Shard.*-", i.ReplSetName); ok {
		if err := global.YAMLSaveToFile(filepath.Join(i.Option.Dir, config.DefaultMongoDBConfigDir, config.DefaultMongoDBConfigFile), i.ShardConfig); err != nil {
			return err
		}
	} else if strings.Contains(i.ReplSetName, "Config-") {
		if err := global.YAMLSaveToFile(filepath.Join(i.Option.Dir, config.DefaultMongoDBConfigDir, config.DefaultMongoDBConfigFile), i.ShardConfig); err != nil {
			return err
		}
	} else {
		if err := global.YAMLSaveToFile(filepath.Join(i.Option.Dir, config.DefaultMongoDBConfigDir, config.DefaultMongoDBConfigFile), i.Config); err != nil {
			return err
		}
	}

	if err := utils.WriteToFile(filepath.Join(i.Option.Dir, "data", "keyfile"), i.KeyFileContent); err != nil {
		return err
	}

	if err := os.Chmod(filepath.Join(i.Option.Dir, "data", "keyfile"), 0400); err != nil {
		return err
	}

	if err := i.ChownDir(i.Option.Dir); err != nil {
		return err
	}

	// 生成 service 启动文件
	//if err := global.INISaveToFile(serviceFile, i.Service); err != nil {
	//	return err
	//}
	if err := i.Service.SaveTo(serviceFile); err != nil {
		return err
	}

	// service reload 并 设置开机自启动
	if err := command.SystemdReload(); err != nil {
		return err
	}

	logger.Infof("设置开机自启动\n")
	if err := command.SystemCtl(service, "enable"); err != nil {
		return err
	}

	if i.Option.ResourceLimit != "" {
		logger.Infof("设置资源限制启动\n")
		if err := command.SystemResourceLimit(service, i.Option.ResourceLimit); err != nil {
			return err
		}
	}

	return nil
}

func (i *MongoDBInstall) LibComplement(NoLiblist []global.MissSoLibrariesfile) error {
	LibList := []string{"libssl.so.10", "libcrypto.so.10", "libtinfo.so.5", "libncurses.so.5"}
	for _, missLib := range NoLiblist {
		re := regexp.MustCompile(`\s+`)
		result := re.ReplaceAllString(missLib.Info, "")
		Libname := strings.Split(result, "=")[0]
		for _, s := range LibList {
			if strings.Contains(s, Libname) {
				logger.Warningf("安装出现缺失的Lib文件 %s 开始进行自动补齐\n", Libname)
				Libfullname := filepath.Join(i.Option.Dir, config.DefaultMongoDBLibDir, Libname)
				if utils.IsExists(Libfullname) {
					if err := command.CopyFileDir(Libfullname, "/lib64"); err != nil {
						return err
					}
				}
			}
		}
	}

	if missLibs, err := global.Checkldd(filepath.Join(i.Option.Dir, config.DefaultMongoDBBinDir, config.DefaultMongoDBBinFile)); err != nil {
		return err
	} else {
		if len(missLibs) != 0 {
			errInfo := ""
			for _, missLib := range missLibs {
				errInfo = errInfo + fmt.Sprintf("%s, 缺少: %s, 需要: %s\n", missLib.Info, missLib.Name, missLib.Repair)
			}
			return errors.New(errInfo)
		}
	}

	return nil
}

func (i *MongoDBInstall) InitPrimary() error {
	conn, err := dao.NewMongoClient("127.0.0.1", i.Option.Port, "", "", "admin")
	if err != nil {
		return err
	}
	defer conn.Conn.Disconnect(context.Background())

	if err := conn.ReplSetInitiate(); err != nil {
		logger.Warningf("初始化副本集报错了: %v\n", err)
		//return err
	}
	//time.Sleep(30 * time.Second)
	stat := false
	for j := 1; j <= 20; j++ {
		time.Sleep(3 * time.Second)
		c1, err := dao.NewMongoClient("127.0.0.1", i.Option.Port, "", "", "admin")
		if err != nil {
			logger.Warningf("初始化副本集后创建连接失败: %v\n", err)
			continue
		}
		status, err := c1.GetReplStatus()
		if err != nil {
			logger.Warningf("初始化副本集后获取集群状态失败: %v\n", err)
			continue
		}
		c1.Conn.Disconnect(context.Background())

		logger.Infof("状态: %s\n", status["members"].(bson.A)[0].(bson.M)["stateStr"].(string))
		if status["members"].(bson.A)[0].(bson.M)["stateStr"].(string) == "PRIMARY" {
			stat = true
			break
		}
	}
	if !stat {
		return fmt.Errorf("初始化主库失败, 等待60秒节点状态仍未 PRIMARY")
	}

	time.Sleep(3 * time.Second)

	if err := conn.CreateUser(i.Option.Username, i.Option.Password, "admin"); err != nil {
		logger.Warningf("创建用户报错了: %v", err)
	}

	time.Sleep(3 * time.Second)

	conn2, err := dao.NewMongoClient("127.0.0.1", i.Option.Port, i.Option.Username, i.Option.Password, "admin")
	if err != nil {
		return err
	}
	defer conn2.Conn.Disconnect(context.Background())
	logger.Infof("新用户连接成功\n")
	time.Sleep(3 * time.Second)

	cfg, err := conn2.GetReplConfig()
	if err != nil {
		return err
	}

	cfg["version"] = cfg["version"].(int32) + 1
	cfg["members"].(bson.A)[0].(bson.M)["host"] = fmt.Sprintf("%s:%d", i.Owner, i.Option.Port)
	//json, err1 := bson.MarshalExtJSON(cfg, true, true)
	//logger.Warningf("添加日志 - 打印重置配置转json错误: %v\n", err1)
	//logger.Warningf("添加日志 - 打印重置后的配置信息: %s\n", string(json))
	if err = conn2.ReplReConfig(cfg); err != nil {
		return err
	}

	for n := 1; n <= 20; n++ {
		time.Sleep(3 * time.Second)
		if i.CheckReplicaSetStatus() {
			return nil
		}
	}

	return fmt.Errorf("节点状态异常\n")
}

func (i *MongoDBInstall) InitSecondary() error {
	var max int32 = 0
	conn, err := dao.NewMongoClient(i.JoinIP, i.JoinPort, i.Option.Username, i.Option.Password, "admin")
	if err != nil {
		return err
	}
	defer conn.Conn.Disconnect(context.Background())

	cfg, err := conn.GetReplConfig()
	if err != nil {
		return err
	}

	for _, member := range cfg["members"].(bson.A) {
		if member.(bson.M)["_id"].(int32) > max {
			max = member.(bson.M)["_id"].(int32)
		}
	}

	cfg["version"] = cfg["version"].(int32) + 1
	cfg["members"] = append(cfg["members"].(bson.A), bson.M{"_id": max + 1, "host": fmt.Sprintf("%s:%d", i.Owner, i.Option.Port)})

	if err = conn.ReplReConfig(cfg); err != nil {
		return err
	}

	for n := 1; n <= 20; n++ {
		time.Sleep(3 * time.Second)
		if i.CheckReplicaSetStatus() {
			return nil
		}
	}

	return fmt.Errorf("节点状态异常\n")
}

func (i *MongoDBInstall) InitArbiter() error {
	var max int32 = 0
	conn, err := dao.NewMongoClient(i.JoinIP, i.JoinPort, i.Option.Username, i.Option.Password, "admin")
	if err != nil {
		return err
	}
	defer conn.Conn.Disconnect(context.Background())

	cfg, err := conn.GetReplConfig()
	if err != nil {
		return err
	}

	for _, member := range cfg["members"].(bson.A) {
		if member.(bson.M)["_id"].(int32) > max {
			max = member.(bson.M)["_id"].(int32)
		}
	}

	cfg["version"] = cfg["version"].(int32) + 1
	cfg["members"] = append(cfg["members"].(bson.A), bson.M{"_id": max + 1, "host": fmt.Sprintf("%s:%d", i.Owner, i.Option.Port), "arbiterOnly": true})

	if err = conn.ReplReConfig(cfg); err != nil {
		return err
	}

	for n := 1; n <= 20; n++ {
		time.Sleep(3 * time.Second)
		if i.CheckReplicaSetStatus() {
			return nil
		}
	}

	return fmt.Errorf("节点状态异常\n")
}

func (i *MongoDBInstall) CheckReplicaSetStatus() bool {
	var conn *dao.MongoClient
	var err error
	if i.Option.Join == "" {
		conn, err = dao.NewMongoClient("127.0.0.1", i.Option.Port, i.Option.Username, i.Option.Password, "admin")
		if err != nil {
			return false
		}
	} else {
		conn, err = dao.NewMongoClient(i.JoinIP, i.JoinPort, i.Option.Username, i.Option.Password, "admin")
		if err != nil {
			return false
		}
	}

	defer conn.Conn.Disconnect(context.Background())

	status, err := conn.GetReplStatus()
	if err != nil {
		return false
	}

	nodeIsOk := false
	if i.Option.Join == "" {
		for _, member := range status["members"].(bson.A) {
			logger.Infof("%s %s\n", member.(bson.M)["name"].(string), member.(bson.M)["stateStr"].(string))
			if member.(bson.M)["stateStr"].(string) == "PRIMARY" && member.(bson.M)["name"].(string) == fmt.Sprintf("%s:%d", i.Owner, i.Option.Port) {
				nodeIsOk = true
				break
			}
		}
	} else {
		for _, member := range status["members"].(bson.A) {
			logger.Infof("%s %s\n", member.(bson.M)["name"].(string), member.(bson.M)["stateStr"].(string))
			if (member.(bson.M)["stateStr"].(string) == "SECONDARY" || member.(bson.M)["stateStr"].(string) == "ARBITER" || member.(bson.M)["stateStr"].(string) == "STARTUP2") && member.(bson.M)["name"].(string) == fmt.Sprintf("%s:%d", i.Owner, i.Option.Port) {
				nodeIsOk = true
				break
			}
		}
	}
	return nodeIsOk
}

func (i *MongoDBInstall) Info() {
	logger.Successf("\n")
	logger.Successf("MongoDB初始化[完成]\n")
	logger.Successf("MongoDB端 口:%d\n", i.Option.Port)
	logger.Successf("MongoDB用 户:%s\n", i.Option.Username)
	logger.Successf("MongoDB密 码:%s\n", i.Option.Password)
	logger.Successf("数据目录:%s\n", i.Option.Dir)
	logger.Successf("启动用户:%s\n", i.SysUser)
	logger.Successf("启动方式:systemctl start %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
	logger.Successf("关闭方式:systemctl stop %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
	logger.Successf("重启方式:systemctl restart %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
	logger.Successf("登录命令: %s --authenticationDatabase admin -u %s -p '%s' --host 127.0.0.1 --port %d\n", filepath.Join(i.Option.Dir, "bin", "mongo"), i.Option.Username, i.Option.Password, i.Option.Port)
	if i.Option.Join != "" {
		logger.Successf("\n")
		logger.Successf("请自行检查主从数据同步进度\n")
	}
}
