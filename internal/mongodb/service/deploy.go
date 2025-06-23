/*
@Author : WuWeiJian
@Date : 2021-05-13 15:49
*/

package service

import (
	"context"
	"dbup/internal/environment"
	"dbup/internal/mongodb/config"
	"dbup/internal/mongodb/dao"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type MongoDBDeploy struct {
	option  config.MongoDBDeployOptions
	master  *MongoDBInstance
	slaves  []*MongoDBInstance
	arbiter *MongoDBInstance
}

func NewMongoDBDeploy() *MongoDBDeploy {
	return &MongoDBDeploy{}
}

func (d *MongoDBDeploy) Run(c string, noRollback bool) error {
	// 初始化参数和配置环节
	if err := d.option.Load(c); err != nil {
		return err
	}
	d.option.Server.SetDefault()
	d.option.NoRollback = noRollback
	if err := d.option.Validator(); err != nil {
		return err
	}

	logger.Infof("初始化部署对象\n")
	if d.option.Server.Password != "" {
		if err := d.Init(); err != nil {
			return err
		}
	} else {
		if err := d.InitUseKeyFile(); err != nil {
			return err
		}
	}

	if d.option.MongoDB.Ipv6 {
		if err := d.CheckHost(); err != nil {
			return err
		}
	}

	if err := d.CheckTmpDir(); err != nil {
		return err
	}
	defer d.DropTmpDir()

	if err := d.Scp(); err != nil {
		return err
	}

	if err := d.CheckEnv(); err != nil {
		return err
	}
	if err := d.InstallAndInitSlave(); err != nil {
		if !d.option.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			d.UNInstall()
		}
		return err
	}
	return nil
}

func (d *MongoDBDeploy) RemoveCluster(c string, yes bool) error {
	// 初始化参数和配置环节
	if err := d.option.Load(c); err != nil {
		return err
	}
	d.option.Server.SetDefault()
	if err := d.option.Server.Validator(); err != nil {
		return err
	}

	if d.option.MongoDB.Port == 0 {
		return fmt.Errorf("请指定要删除集群的端口号")
	}

	if d.option.MongoDB.Dir == "" {
		return fmt.Errorf("请指定要删除集群的数据目录")
	}

	for _, ip := range strings.Split(d.option.Server.Address, ",") {
		logger.Warningf("要删除的集群节点以及数据目录: %s:%d %s\n", ip, d.option.MongoDB.Port, d.option.MongoDB.Dir)
	}

	if !yes {
		logger.Warningf("删除集群是危险操作,会将整个集群中的数据完全删除, 不可恢复\n")
		logger.Warningf("是否确认删除[y|n]:")

		var yes string
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	logger.Infof("初始化删除对象\n")
	if d.option.Server.Password != "" {
		if err := d.Init(); err != nil {
			return err
		}
	} else {
		if err := d.InitUseKeyFile(); err != nil {
			return err
		}
	}
	if err := d.CheckTmpDir(); err != nil {
		return err
	}
	defer d.DropTmpDir()

	if err := d.Scp(); err != nil {
		return err
	}

	d.UNInstall()

	return nil
}

func (d *MongoDBDeploy) InstallAndInitSlave() error {
	if err := d.Install(); err != nil {
		return err
	}

	// 检查集群状态
	logger.Infof("等待集群状态:\n")
	stat := false
	for i := 1; i <= 20; i++ {
		time.Sleep(3 * time.Second)
		if err := d.CheckReplicaSetStatus(); err == nil {
			stat = true
			break
		} else {
			logger.Warningf("%v", err)
		}
	}

	if !stat {
		return fmt.Errorf("主从状态异常")
	}

	return d.Info()
}

func (d *MongoDBDeploy) Init() error {
	var err error
	ips := strings.Split(d.option.Server.Address, ",")
	if d.master, err = NewMongoDBInstance(d.option.Server.TmpDir,
		ips[0],
		d.option.Server.User,
		d.option.Server.Password,
		d.option.Server.SshPort,
		d.option.MongoDB); err != nil {
		return err
	}
	for _, slave := range ips[1:] {
		s, err := NewMongoDBInstance(d.option.Server.TmpDir,
			slave,
			d.option.Server.User,
			d.option.Server.Password,
			d.option.Server.SshPort,
			d.option.MongoDB)
		if err != nil {
			return err
		}
		s.Inst.Option.Join = ips[0]
		d.slaves = append(d.slaves, s)
	}

	if d.option.Server.Arbiter != "" {
		// 强制 arbiter 节点的内存等于1G
		if d.arbiter, err = NewMongoDBInstance(d.option.Server.TmpDir,
			d.option.Server.Arbiter,
			d.option.Server.User,
			d.option.Server.Password,
			d.option.Server.SshPort,
			d.option.MongoDB); err != nil {
			return err
		}
		d.arbiter.Inst.Option.Memory = 1
		d.arbiter.Inst.Option.Join = ips[0]
	}
	return nil
}

func (d *MongoDBDeploy) InitUseKeyFile() error {
	var err error
	ips := strings.Split(d.option.Server.Address, ",")
	if d.master, err = NewMongoDBInstanceUseKeyFile(d.option.Server.TmpDir,
		ips[0],
		d.option.Server.User,
		d.option.Server.KeyFile,
		d.option.Server.SshPort,
		d.option.MongoDB); err != nil {
		return err
	}
	for _, slave := range ips[1:] {
		s, err := NewMongoDBInstanceUseKeyFile(d.option.Server.TmpDir,
			slave,
			d.option.Server.User,
			d.option.Server.KeyFile,
			d.option.Server.SshPort,
			d.option.MongoDB)
		if err != nil {
			return err
		}
		s.Inst.Option.Join = ips[0]
		d.slaves = append(d.slaves, s)
	}

	if d.option.Server.Arbiter != "" {
		// 强制 arbiter 节点的内存等于1G
		if d.arbiter, err = NewMongoDBInstanceUseKeyFile(d.option.Server.TmpDir,
			d.option.Server.Arbiter,
			d.option.Server.User,
			d.option.Server.KeyFile,
			d.option.Server.SshPort,
			d.option.MongoDB); err != nil {
			fmt.Println(ips[0])
			fmt.Println("arbiter的主", ips[0])
			return err
		}
		d.arbiter.Inst.Option.Memory = 1
		d.arbiter.Inst.Option.Join = ips[0]
	}
	return nil
}

func (d *MongoDBDeploy) CheckTmpDir() error {
	logger.Infof("检查目标机器的临时目录\n")
	if err := d.master.CheckTmpDir(); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		if err := slave.CheckTmpDir(); err != nil {
			return err
		}
	}

	if d.arbiter != nil {
		if err := d.arbiter.CheckTmpDir(); err != nil {
			return err
		}
	}
	return nil
}

func (d *MongoDBDeploy) CheckHost() error {
	logger.Infof("检查目标机器Host文件\n")
	ips := strings.Split(d.option.Server.Address, ",")
	aips := strings.Split(d.option.Server.Arbiter, ",")
	mergedList := append(ips, aips...)
	if err := d.master.CheckHosts(mergedList); err != nil {
		return err
	}

	for _, slave := range d.slaves {
		if err := slave.CheckHosts(mergedList); err != nil {
			return err
		}
	}

	if d.arbiter != nil {
		if err := d.arbiter.CheckHosts(mergedList); err != nil {
			return err
		}
	}

	return nil
}

func (d *MongoDBDeploy) Scp() error {
	logger.Infof("将所需文件复制到目标机器\n")
	source := path.Join(environment.GlobalEnv().ProgramPath, "..")
	logger.Infof("复制到: %s\n", d.master.Host)
	if err := d.master.Scp(source); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		logger.Infof("复制到: %s\n", slave.Host)
		if err := slave.Scp(source); err != nil {
			return err
		}
	}

	if d.arbiter != nil {
		logger.Infof("复制到: %s\n", d.arbiter.Host)
		if err := d.arbiter.Scp(source); err != nil {
			return err
		}
	}
	return nil
}

func (d *MongoDBDeploy) DropTmpDir() {
	logger.Infof("删除目标机器的临时目录\n")
	_ = d.master.DropTmpDir()
	for _, slave := range d.slaves {
		_ = slave.DropTmpDir()
	}
	if d.arbiter != nil {
		_ = d.arbiter.DropTmpDir()
	}

}

func (d *MongoDBDeploy) CheckEnv() error {
	logger.Infof("检查环境\n")
	if err := d.master.Install(true, false, d.option.NoRollback, d.option.MongoDB.Ipv6); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		if err := slave.Install(true, false, d.option.NoRollback, d.option.MongoDB.Ipv6); err != nil {
			return err
		}
	}
	if d.arbiter != nil {
		if err := d.arbiter.Install(true, true, d.option.NoRollback, d.option.MongoDB.Ipv6); err != nil {
			return err
		}
	}
	return nil
}

func (d *MongoDBDeploy) Install() error {
	logger.Infof("开始安装\n")
	if err := d.master.Install(false, false, d.option.NoRollback, d.option.MongoDB.Ipv6); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		if err := slave.Install(false, false, d.option.NoRollback, d.option.MongoDB.Ipv6); err != nil {
			return err
		}
	}
	if d.arbiter != nil {
		if err := d.arbiter.Install(false, true, d.option.NoRollback, d.option.MongoDB.Ipv6); err != nil {
			return err
		}
	}
	return nil
}

func (d *MongoDBDeploy) UNInstall() {
	logger.Infof("开始卸载清理\n")
	if err := d.master.UNInstall(); err != nil {
		logger.Warningf("卸载节点: %s 失败: %v\n", d.master.Host, err)
	}
	for _, slave := range d.slaves {
		if err := slave.UNInstall(); err != nil {
			logger.Warningf("卸载节点: %s 失败: %v\n", slave.Host, err)
		}
	}

	if d.arbiter != nil {
		if err := d.arbiter.UNInstall(); err != nil {
			logger.Warningf("卸载节点: %s 失败: %v\n", d.arbiter.Host, err)
		}
	}
}

func (d *MongoDBDeploy) CheckReplicaSetStatus() error {
	conn, err := dao.NewMongoClient(d.master.Host, d.master.Inst.Option.Port, d.master.Inst.Option.Username, d.master.Inst.Option.Password, "admin")
	if err != nil {
		return err
	}
	defer conn.Conn.Disconnect(context.Background())

	status, err := conn.GetReplStatus()
	if err != nil {
		return err
	}

	primary := 0
	secondary := 0
	arrlib := 0
	for _, member := range status["members"].(bson.A) {
		logger.Infof("%s %s\n", member.(bson.M)["name"].(string), member.(bson.M)["stateStr"].(string))

		if member.(bson.M)["stateStr"].(string) == "PRIMARY" {
			primary += 1
		}
		if member.(bson.M)["stateStr"].(string) == "SECONDARY" {
			secondary += 1
		}
		if member.(bson.M)["stateStr"].(string) == "ARBITER" {
			arrlib += 1
		}
	}

	if primary != 1 || secondary != len(d.slaves) {
		return fmt.Errorf("主从状态异常")
	}

	if d.arbiter != nil && arrlib == 0 {
		return fmt.Errorf("arbiter 节点状态异常")
	}

	return nil
}

func (d *MongoDBDeploy) Info() error {
	hostPorts := fmt.Sprintf("%s:%d", d.master.Host, d.option.MongoDB.Port)
	for _, slave := range d.slaves {
		hostPorts = hostPorts + fmt.Sprintf(",%s:%d", slave.Host, d.option.MongoDB.Port)
	}

	conn, err := dao.NewMongoClient(d.master.Host, d.master.Inst.Option.Port, d.master.Inst.Option.Username, d.master.Inst.Option.Password, "admin")
	if err != nil {
		return err
	}
	defer conn.Conn.Disconnect(context.Background())

	status, err := conn.GetReplStatus()
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("mongodb://%s:%s@%s/?authSource=admin&replicaSet=%s",
		d.option.MongoDB.Username,
		d.option.MongoDB.Password,
		hostPorts,
		status["set"].(string))

	logger.Successf("从库正常\n")
	logger.Successf("集群搭建成功\n")
	logger.Successf("集群连接字符串(URI): %s \n", uri)
	return nil
}
