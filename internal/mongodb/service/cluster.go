package service

import (
	"context"
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/mongodb/config"
	"dbup/internal/mongodb/dao"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type MongoDBClusterDeploy struct {
	option  config.MongoDBDeployOptions
	coption config.MongoDBClusterOptions
	Conn    *command.Connection
	master  *MongoDBInstance
	slaves  []*MongoDBInstance
	arbiter *MongoDBInstance
	mongos  []*MongoSInstance
}

func NewMongoClusterDeploy() *MongoDBClusterDeploy {
	return &MongoDBClusterDeploy{}
}

func (d *MongoDBClusterDeploy) Run(c string) error {
	// 初始化参数和配置环节
	if err := global.YAMLLoadFromFile(c, &d.coption); err != nil {
		return err
	}

	d.coption.SetDefault()

	// 验证集群的配置参数规范
	if err := d.coption.Validators(); err != nil {
		return err
	}

	// 验证集群的IPV6环境配置
	if d.coption.MongoConfig.Ipv6 {
		logger.Infof("验证Mongodb集群IPV6环境\n")
		if err := d.CheckClusterHosts(); err != nil {
			return err
		}
	}

	logger.Infof("初始化集群部署对象\n")

	if err := d.ClusterInstall(config.MongoShards, config.Mongoclusterinstall); err != nil {
		return err
	}

	if err := d.ClusterInstall(config.MongoConfig, config.Mongoclusterinstall); err != nil {
		return err
	}

	if err := d.ClusterInstall(config.Mongos, config.Mongoclusterinstall); err != nil {
		return err
	}

	return nil
}

func (d *MongoDBClusterDeploy) RemoveCluster(c string, yes bool) error {
	// 初始化参数和配置环节
	if err := global.YAMLLoadFromFile(c, &d.coption); err != nil {
		return err
	}

	d.option.Server.SetDefault()

	if err := d.coption.Validator(); err != nil {
		return err
	}

	for _, Mongosnode := range d.coption.Mongos {
		if Mongosnode.Port == 0 {
			return fmt.Errorf("请指定要删除集群 mongos 节点 %s 的端口号", Mongosnode.Host)
		}
		if Mongosnode.Dir == "" {
			return fmt.Errorf("请指定要删除集群 mongos 节点 %s 的数据目录", Mongosnode.Host)
		}
		logger.Warningf("要删除的集群 mongos 节点以及数据目录: %s:%d %s\n", Mongosnode.Host, Mongosnode.Port, Mongosnode.Dir)
	}

	for _, Confignode := range d.coption.MongoCfg {
		if Confignode.Port == 0 {
			return fmt.Errorf("请指定要删除集群 mongoconfig 节点 %s 的端口号", Confignode.Host)
		}
		if Confignode.Dir == "" {
			return fmt.Errorf("请指定要删除集群 mongoconfig 节点 %s 的数据目录", Confignode.Host)
		}
		logger.Warningf("要删除的集群 mongoconfig 节点以及数据目录: %s:%d %s\n", Confignode.Host, Confignode.Port, Confignode.Dir)
	}

	for _, Shardlist := range d.coption.MongoShard {
		for _, Shardnode := range Shardlist.Shard {
			if Shardnode.Port == 0 {
				return fmt.Errorf("请指定要删除集群 mongoshard 节点 %s 的端口号", Shardnode.Host)
			}
			if Shardnode.Dir == "" {
				return fmt.Errorf("请指定要删除集群 mongoshard 节点 %s 的数据目录", Shardnode.Host)
			}
			logger.Warningf("要删除的集群 mongoshard 节点以及数据目录: %s:%d %s\n", Shardnode.Host, Shardnode.Port, Shardnode.Dir)
		}
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

	logger.Infof("初始化 Mongodb 分片集群的删除对象与删除操作\n")

	if err := d.ClusterInstall(config.Mongos, config.Mongoclusterdelete); err != nil {
		return err
	}

	if err := d.ClusterInstall(config.MongoConfig, config.Mongoclusterdelete); err != nil {
		return err
	}

	if err := d.ClusterInstall(config.MongoShards, config.Mongoclusterdelete); err != nil {
		return err
	}

	return nil
}

func (d *MongoDBClusterDeploy) ClusterInstall(role, mongoswitch string) error {
	// 初始化副本配置
	d.option.Server.Arbiter = ""
	d.option.MongoDB.Ipv6 = d.coption.MongoConfig.Ipv6
	d.option.MongoDB.BindIP = d.coption.MongoConfig.Bind_ip
	d.option.MongoDB.Username = d.coption.MongoConfig.Username
	d.option.MongoDB.Password = d.coption.MongoConfig.Password
	d.option.MongoDB.SystemUser = d.coption.MongoConfig.System_user
	d.option.MongoDB.SystemGroup = d.coption.MongoConfig.System_group
	d.option.MongoDB.ResourceLimit = d.coption.MongoConfig.Resource_limit
	// d.option.Server.

	switch role {
	case config.MongoShards:
		d.option.MongoDB.Memory = d.coption.MongoConfig.Shard_memory
		switch mongoswitch {
		case config.Mongoclusterinstall:
			if err := d.MongoShardInit(); err != nil {
				return err
			}
		case config.Mongoclusterdelete:
			if err := d.MongoShardRemove(); err != nil {
				return err
			}
		}
	case config.MongoConfig:
		d.option.MongoDB.Memory = d.coption.MongoConfig.Config_memory
		switch mongoswitch {
		case config.Mongoclusterinstall:
			if err := d.MongoConfigInit(); err != nil {
				return err
			}
		case config.Mongoclusterdelete:
			if err := d.MongoConfigRemove(); err != nil {
				return err
			}
		}
	case config.Mongos:
		// 初始化 mongos 基础配置
		d.coption.Mongosoption.Ipv6 = d.coption.MongoConfig.Ipv6
		d.coption.Mongosoption.BindIP = d.coption.MongoConfig.Bind_ip
		d.coption.Mongosoption.Username = d.coption.MongoConfig.Username
		d.coption.Mongosoption.Password = d.coption.MongoConfig.Password
		d.coption.Mongosoption.SystemUser = d.coption.MongoConfig.System_user
		d.coption.Mongosoption.SystemGroup = d.coption.MongoConfig.System_group
		d.coption.Mongosoption.ResourceLimit = d.coption.MongoConfig.Resource_limit
		switch mongoswitch {
		case config.Mongoclusterinstall:
			if err := d.MongoSInit(); err != nil {
				return err
			}
		case config.Mongoclusterdelete:
			if err := d.MongoSRmove(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *MongoDBClusterDeploy) MongoShardInit() error {

	for n, shard := range d.coption.MongoShard {

		if d.coption.SSHConfig.Password != "" {
			if err := d.Sinit(n, shard.Shard); err != nil {
				return err
			}
		} else {
			if err := d.SinitUseKeyFile(n, shard.Shard); err != nil {
				return err
			}
		}

		if err := d.CheckTmpDir(); err != nil {
			d.DropTmpDir()
			return err
		}

		if err := d.Scp(); err != nil {
			return err
		}

		if err := d.CheckEnv(); err != nil {
			d.DropTmpDir()
			return err
		}

		if err := d.InstallAndInitSlave(config.MongoShards); err != nil {
			if !d.option.NoRollback {
				logger.Warningf("安装失败, 开始回滚\n")
				d.UNInstall()
			}
			return err
		}

		d.DropTmpDir()

		d.master = nil
		d.slaves = nil

	}

	return nil
}

func (d *MongoDBClusterDeploy) MongoConfigInit() error {
	if d.coption.SSHConfig.Password != "" {
		if err := d.Cinit(d.coption.MongoCfg); err != nil {
			return err
		}
	} else {
		if err := d.CinitUseKeyFile(d.coption.MongoCfg); err != nil {
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

	if err := d.InstallAndInitSlave(config.MongoConfig); err != nil {
		if !d.option.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			d.UNInstall()
		}
		return err
	}

	return nil
}

func (d *MongoDBClusterDeploy) MongoSInit() error {
	if d.coption.SSHConfig.Password != "" {
		if err := d.MSinit(d.coption.Mongos); err != nil {
			return err
		}
	} else {
		if err := d.MSinitUseKeyFile(d.coption.Mongos); err != nil {
			return err
		}
	}

	if err := d.MSCheckTmpDir(); err != nil {
		return err
	}
	defer d.MSDropTmpDir()

	if err := d.MScp(); err != nil {
		return err
	}

	if err := d.MSCheckEnv(); err != nil {
		return err
	}

	if err := d.MSInstallandInit(); err != nil {
		if !d.coption.Mongosoption.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			d.MSUNInstall()
		}
		return err
	}

	return nil
}

func (d *MongoDBClusterDeploy) MongoSRmove() error {
	if d.coption.SSHConfig.Password != "" {
		if err := d.MSinit(d.coption.Mongos); err != nil {
			return err
		}
	} else {
		if err := d.MSinitUseKeyFile(d.coption.Mongos); err != nil {
			return err
		}
	}

	if err := d.MSCheckTmpDir(); err != nil {
		return err
	}
	defer d.MSDropTmpDir()

	if err := d.MScp(); err != nil {
		return err
	}

	d.MSUNInstall()

	return nil
}

func (d *MongoDBClusterDeploy) MongoConfigRemove() error {
	if d.coption.SSHConfig.Password != "" {
		if err := d.Cinit(d.coption.MongoCfg); err != nil {
			return err
		}
	} else {
		if err := d.CinitUseKeyFile(d.coption.MongoCfg); err != nil {
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

func (d *MongoDBClusterDeploy) MongoShardRemove() error {

	d.slaves = nil

	for n, shard := range d.coption.MongoShard {

		if d.coption.SSHConfig.Password != "" {
			if err := d.Sinit(n, shard.Shard); err != nil {
				return err
			}
		} else {
			if err := d.SinitUseKeyFile(n, shard.Shard); err != nil {
				return err
			}
		}

		if err := d.CheckTmpDir(); err != nil {
			return err
		}
		// defer d.DropTmpDir()
		if err := d.Scp(); err != nil {
			return err
		}

		d.UNInstall()
		d.DropTmpDir()

		d.master = nil
		d.slaves = nil

	}

	return nil
}

func (d *MongoDBClusterDeploy) MSInstallandInit() error {
	sharlist := d.coption.Mongosoption.Shardlist
	// logger.Infof("初始化分片 %s\n", sharlist)

	if err := d.MSInstall(); err != nil {
		return err
	}

	logger.Infof("初始化分片到 Mongos\n")
	time.Sleep(3 * time.Second)
	stat := false
	for _, shardDB := range sharlist {
		for i := 1; i <= 20; i++ {
			time.Sleep(3 * time.Second)
			if err := d.MongosInitShard(shardDB); err == nil {
				stat = true
				break
			} else {
				logger.Warningf("%v\n", err)
			}
		}
	}

	if !stat {
		return fmt.Errorf("mongos 连接异常")
	}

	d.MongoInfo()

	return nil
}

func (d *MongoDBClusterDeploy) MongosInitverify() error {
	m := d.coption.Mongos[:1]
	msnode := m[len(m)-1]
	sharlist := d.coption.Mongosoption.Shardlist
	conn, err := dao.NewMongoClient(msnode.Host, msnode.Port, d.coption.Mongosoption.Username, d.coption.Mongosoption.Password, "admin")
	if err != nil {
		return err
	}

	// 验证添加shard是否成功
	cfg, err := conn.ShardingList()
	if err != nil {
		return err
	}
	for _, info := range cfg["shards"].(bson.A) {
		stat := false
		online_host := info.(bson.M)["host"].(string)
		for i := 1; i <= 20; i++ {
			time.Sleep(2 * time.Second)
			if !d.Conn.Contains(sharlist, online_host) {
				logger.Warningf("添加shard %s 失败,开始重新添加", online_host)
				if err := conn.ShardingAdd(online_host); err != nil {
					return err
				}
			} else {
				stat = true
				break
			}
		}
		if !stat {
			return fmt.Errorf("mongos 多次添加shard %s 失败", online_host)
		}
	}
	conn.Conn.Disconnect(context.Background())

	return nil
}

func (d *MongoDBClusterDeploy) MongoInfo() {
	hostPorts := []string{}
	for _, mongosnode := range d.coption.Mongos {
		hostPorts = append(hostPorts, fmt.Sprintf("%s:%d", mongosnode.Host, mongosnode.Port))
	}
	hostPort := strings.Join(hostPorts, ",")
	uri := fmt.Sprintf("mongodb://%s:%s@%s/?authSource=admin", d.coption.MongoConfig.Username, d.coption.MongoConfig.Password, hostPort)
	logger.Successf("分片集群 Mongos 搭建成功\n")
	logger.Successf("分片集群全部节点搭建成功\n")
	logger.Successf("分片集群 Mongos 连接字符串(URI): \n%s\n", uri)
}

func (d *MongoDBClusterDeploy) MongosInitShard(shardDB string) error {

	m := d.coption.Mongos[:1]
	msnode := m[len(m)-1]

	conn, err := dao.NewMongoClient(msnode.Host, msnode.Port, d.coption.Mongosoption.Username, d.coption.Mongosoption.Password, "admin")
	if err != nil {
		return err
	}
	defer conn.Conn.Disconnect(context.Background())
	if err := conn.ShardingAdd(shardDB); err != nil {
		return err
	}

	return nil
}

func (d *MongoDBClusterDeploy) MSInstall() error {
	logger.Infof("开始安装 Mongos\n")
	for _, mongosnode := range d.mongos {
		if err := mongosnode.Install(false, d.coption.Mongosoption.Ipv6); err != nil {
			return err
		}
	}

	return nil
}

func (d *MongoDBClusterDeploy) MSUNInstall() error {
	logger.Infof("开始卸载 Mongos\n")
	for _, mongosnode := range d.mongos {
		if err := mongosnode.UNInstall(); err != nil {
			return err
		}
	}

	return nil
}

func (d *MongoDBClusterDeploy) MSCheckTmpDir() error {
	logger.Infof("检查 Mongos 目标机器的临时目录\n")

	for _, mongosnode := range d.mongos {
		if err := mongosnode.CheckTmpDir(); err != nil {
			return err
		}
	}

	return nil
}

func (d *MongoDBClusterDeploy) MSDropTmpDir() {
	logger.Infof("删除目标机器的临时目录\n")

	for _, mongosnode := range d.mongos {
		_ = mongosnode.DropTmpDir()
	}

}

func (d *MongoDBClusterDeploy) MScp() error {
	logger.Infof("将 Mongos 所需文件复制到目标机器\n")
	source := path.Join(environment.GlobalEnv().ProgramPath, "..")
	for _, mongosnode := range d.mongos {
		logger.Infof("复制到: %s\n", mongosnode.Host)
		if err := mongosnode.Scp(source); err != nil {
			return err
		}
	}
	return nil
}

func (d *MongoDBClusterDeploy) MSCheckEnv() error {
	logger.Infof("检查环境\n")
	for _, mongosnode := range d.mongos {
		if err := mongosnode.Install(true, d.coption.Mongosoption.Ipv6); err != nil {
			return err
		}
	}
	return nil
}

func (d *MongoDBClusterDeploy) MSinit(Nodelist []config.MongosNode) error {

	logger.Infof("开始初始化分片集群 Mongos 路由节点\n")

	for _, mongosnode := range Nodelist {
		d.coption.Mongosoption.Dir = mongosnode.Dir
		d.coption.Mongosoption.Port = mongosnode.Port
		d.coption.Mongosoption.Owner = mongosnode.Host

		ms, err := NewMongoSInstance(d.coption.SSHConfig.TmpDir,
			mongosnode.Host,
			d.coption.SSHConfig.Username,
			d.coption.SSHConfig.Password,
			d.coption.SSHConfig.Port,
			d.coption.Mongosoption)
		if err != nil {
			return err
		}
		d.mongos = append(d.mongos, ms)
	}

	return nil
}

func (d *MongoDBClusterDeploy) MSinitUseKeyFile(Nodelist []config.MongosNode) error {

	logger.Infof("开始初始化分片集群 Mongos 路由节点\n")

	for _, mongosnode := range Nodelist {

		d.coption.Mongosoption.Dir = mongosnode.Dir
		d.coption.Mongosoption.Port = mongosnode.Port
		d.coption.Mongosoption.Owner = mongosnode.Host

		ms, err := NewMongoSInstanceUseKeyFile(d.coption.SSHConfig.TmpDir,
			mongosnode.Host,
			d.coption.SSHConfig.Username,
			d.coption.SSHConfig.Password,
			d.coption.SSHConfig.Port,
			d.coption.Mongosoption)
		if err != nil {
			return err
		}
		d.mongos = append(d.mongos, ms)
	}

	return nil
}

func (d *MongoDBClusterDeploy) InstallAndInitSlave(role string) error {
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
			logger.Warningf("%v\n", err)
		}
	}

	if !stat {
		return fmt.Errorf("主从状态异常")
	}

	return d.Info(role)
}

func (d *MongoDBClusterDeploy) Sinit(num int, Nodelist []config.MongoShardNode) error {
	var err error

	ShardDB := []string{}
	m := Nodelist[:1]
	master := m[len(m)-1]

	d.option.MongoDB.ReplSetName = fmt.Sprintf("Shard%d-%d", num+1, master.Port)
	logger.Infof("开始初始化分片集群 Shard 副本:%s\n", d.option.MongoDB.ReplSetName)

	head := fmt.Sprintf("%s/%s:%d", d.option.MongoDB.ReplSetName, master.Host, master.Port)
	ShardDB = append(ShardDB, head)

	d.option.MongoDB.Dir = master.Dir
	d.option.MongoDB.Port = master.Port
	if d.master, err = NewMongoDBInstance(d.coption.SSHConfig.TmpDir,
		master.Host,
		d.coption.SSHConfig.Username,
		d.coption.SSHConfig.Password,
		d.coption.SSHConfig.Port,
		d.option.MongoDB); err != nil {
		return err
	}
	for _, slave := range Nodelist[1:] {
		d.option.MongoDB.Dir = slave.Dir
		d.option.MongoDB.Port = slave.Port
		s, err := NewMongoDBInstance(d.coption.SSHConfig.TmpDir,
			slave.Host,
			d.coption.SSHConfig.Username,
			d.coption.SSHConfig.Password,
			d.coption.SSHConfig.Port,
			d.option.MongoDB)
		if err != nil {
			return err
		}
		ipport := fmt.Sprintf("%s:%d", slave.Host, slave.Port)
		s.Inst.Option.Join = master.Host
		d.slaves = append(d.slaves, s)
		ShardDB = append(ShardDB, ipport)
	}

	if d.option.Server.Arbiter != "" {
		// 强制 arbiter 节点的内存等于1G
		if d.arbiter, err = NewMongoDBInstance(d.coption.SSHConfig.TmpDir,
			d.option.Server.Arbiter,
			d.coption.SSHConfig.Username,
			d.coption.SSHConfig.Password,
			d.coption.SSHConfig.Port,
			d.option.MongoDB); err != nil {
			return err
		}
		d.arbiter.Inst.Option.Memory = 1
		d.arbiter.Inst.Option.Join = master.Host
	}

	d.coption.Mongosoption.Shardlist = append(d.coption.Mongosoption.Shardlist, strings.Join(ShardDB, ","))

	return nil
}

func (d *MongoDBClusterDeploy) Cinit(Nodelist []config.MongoConfigNode) error {
	var err error

	m := Nodelist[:1]
	master := m[len(m)-1]

	d.option.MongoDB.ReplSetName = fmt.Sprintf("Config-%d", master.Port)
	logger.Infof("开始初始化分片集群 Config 副本:%s\n", d.option.MongoDB.ReplSetName)

	head := fmt.Sprintf("%s/%s:%d", d.option.MongoDB.ReplSetName, master.Host, master.Port)
	ConfigDB := []string{}
	ConfigDB = append(ConfigDB, head)

	d.option.MongoDB.Dir = master.Dir
	d.option.MongoDB.Port = master.Port
	if d.master, err = NewMongoDBInstance(d.coption.SSHConfig.TmpDir,
		master.Host,
		d.coption.SSHConfig.Username,
		d.coption.SSHConfig.Password,
		d.coption.SSHConfig.Port,
		d.option.MongoDB); err != nil {
		return err
	}
	for _, slave := range Nodelist[1:] {
		d.option.MongoDB.Dir = slave.Dir
		d.option.MongoDB.Port = slave.Port
		s, err := NewMongoDBInstance(d.coption.SSHConfig.TmpDir,
			slave.Host,
			d.coption.SSHConfig.Username,
			d.coption.SSHConfig.Password,
			d.coption.SSHConfig.Port,
			d.option.MongoDB)
		if err != nil {
			return err
		}
		ipport := fmt.Sprintf("%s:%d", slave.Host, slave.Port)
		s.Inst.Option.Join = master.Host
		d.slaves = append(d.slaves, s)
		ConfigDB = append(ConfigDB, ipport)
	}

	d.coption.Mongosoption.ConfigDB = strings.Join(ConfigDB, ",")

	if d.option.Server.Arbiter != "" {
		// 强制 arbiter 节点的内存等于1G
		if d.arbiter, err = NewMongoDBInstance(d.coption.SSHConfig.TmpDir,
			d.option.Server.Arbiter,
			d.coption.SSHConfig.Username,
			d.coption.SSHConfig.Password,
			d.coption.SSHConfig.Port,
			d.option.MongoDB); err != nil {
			return err
		}
		d.arbiter.Inst.Option.Memory = 1
		d.arbiter.Inst.Option.Join = master.Host
	}
	return nil
}

func (d *MongoDBClusterDeploy) SinitUseKeyFile(num int, Nodelist []config.MongoShardNode) error {
	var err error

	ShardDB := []string{}
	m := Nodelist[:1]
	master := m[len(m)-1]

	d.option.MongoDB.ReplSetName = fmt.Sprintf("Shard%d-%d", num+1, master.Port)
	logger.Infof("开始初始化分片集群 Shard 副本:%s\n", d.option.MongoDB.ReplSetName)

	head := fmt.Sprintf("%s/%s:%d", d.option.MongoDB.ReplSetName, master.Host, master.Port)
	ShardDB = append(ShardDB, head)
	d.option.MongoDB.Dir = master.Dir
	d.option.MongoDB.Port = master.Port

	if d.master, err = NewMongoDBInstanceUseKeyFile(d.coption.SSHConfig.TmpDir,
		master.Host,
		d.option.Server.User,
		d.option.Server.KeyFile,
		d.option.Server.SshPort,
		d.option.MongoDB); err != nil {
		return err
	}
	for _, slave := range Nodelist[1:] {
		d.option.MongoDB.Dir = slave.Dir
		d.option.MongoDB.Port = slave.Port
		s, err := NewMongoDBInstanceUseKeyFile(d.coption.SSHConfig.TmpDir,
			slave.Host,
			d.option.Server.User,
			d.option.Server.KeyFile,
			d.option.Server.SshPort,
			d.option.MongoDB)
		if err != nil {
			return err
		}
		ipport := fmt.Sprintf("%s:%d", slave.Host, slave.Port)
		s.Inst.Option.Join = master.Host
		d.slaves = append(d.slaves, s)
		ShardDB = append(ShardDB, ipport)
	}

	d.coption.Mongosoption.Shardlist = append(d.coption.Mongosoption.Shardlist, strings.Join(ShardDB, ","))
	if d.option.Server.Arbiter != "" {
		// 强制 arbiter 节点的内存等于1G
		if d.arbiter, err = NewMongoDBInstanceUseKeyFile(d.coption.SSHConfig.TmpDir,
			d.option.Server.Arbiter,
			d.option.Server.User,
			d.option.Server.KeyFile,
			d.option.Server.SshPort,
			d.option.MongoDB); err != nil {
			fmt.Println(master.Host)
			fmt.Println("arbiter的主", master.Host)
			return err
		}
		d.arbiter.Inst.Option.Memory = 1
		d.arbiter.Inst.Option.Join = master.Host
	}
	return nil
}

func (d *MongoDBClusterDeploy) CinitUseKeyFile(Nodelist []config.MongoConfigNode) error {
	var err error

	m := Nodelist[:1]
	master := m[len(m)-1]

	d.option.MongoDB.ReplSetName = fmt.Sprintf("Config-%d", master.Port)
	logger.Infof("开始初始化分片集群 Shard 副本:%s\n", d.option.MongoDB.ReplSetName)

	d.option.MongoDB.Dir = master.Dir
	d.option.MongoDB.Port = master.Port

	if d.master, err = NewMongoDBInstanceUseKeyFile(d.coption.SSHConfig.TmpDir,
		master.Host,
		d.option.Server.User,
		d.option.Server.KeyFile,
		d.option.Server.SshPort,
		d.option.MongoDB); err != nil {
		return err
	}
	for _, slave := range Nodelist[1:] {
		d.option.MongoDB.Dir = slave.Dir
		d.option.MongoDB.Port = slave.Port
		s, err := NewMongoDBInstanceUseKeyFile(d.coption.SSHConfig.TmpDir,
			slave.Host,
			d.option.Server.User,
			d.option.Server.KeyFile,
			d.option.Server.SshPort,
			d.option.MongoDB)
		if err != nil {
			return err
		}
		s.Inst.Option.Join = master.Host
		d.slaves = append(d.slaves, s)
	}

	if d.option.Server.Arbiter != "" {
		// 强制 arbiter 节点的内存等于1G
		if d.arbiter, err = NewMongoDBInstanceUseKeyFile(d.coption.SSHConfig.TmpDir,
			d.option.Server.Arbiter,
			d.option.Server.User,
			d.option.Server.KeyFile,
			d.option.Server.SshPort,
			d.option.MongoDB); err != nil {
			fmt.Println(master.Host)
			fmt.Println("arbiter的主", master.Host)
			return err
		}
		d.arbiter.Inst.Option.Memory = 1
		d.arbiter.Inst.Option.Join = master.Host
	}
	return nil
}

func (d *MongoDBClusterDeploy) CheckClusterHosts() error {
	mongonode := []string{}
	for _, mongos := range d.coption.Mongos {
		if !d.Conn.Contains(mongonode, mongos.Host) {
			mongonode = append(mongonode, mongos.Host)
		}
	}
	for _, mongocfg := range d.coption.MongoCfg {
		if !d.Conn.Contains(mongonode, mongocfg.Host) {
			mongonode = append(mongonode, mongocfg.Host)
		}
	}
	for _, shards := range d.coption.MongoShard {
		for _, shardnode := range shards.Shard {
			if !d.Conn.Contains(mongonode, shardnode.Host) {
				mongonode = append(mongonode, shardnode.Host)
			}
		}
	}
	if d.coption.SSHConfig.Password != "" {
		for _, cnode := range mongonode {
			mnode, err := NewMongoDBInstance(
				d.coption.SSHConfig.TmpDir,
				cnode,
				d.coption.SSHConfig.Username,
				d.coption.SSHConfig.Password,
				d.coption.SSHConfig.Port,
				d.option.MongoDB)
			if err != nil {
				return err
			}
			if err := mnode.CheckHosts(mongonode); err != nil {
				return err
			}
		}
	} else {
		for _, knode := range mongonode {
			mnode, err := NewMongoDBInstance(
				d.coption.SSHConfig.TmpDir,
				knode,
				d.coption.SSHConfig.Username,
				d.coption.SSHConfig.Password,
				d.coption.SSHConfig.Port,
				d.option.MongoDB)
			if err != nil {
				return err
			}
			if err := mnode.CheckHosts(mongonode); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *MongoDBClusterDeploy) CheckTmpDir() error {
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

func (d *MongoDBClusterDeploy) Scp() error {
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

func (d *MongoDBClusterDeploy) DropTmpDir() {
	logger.Infof("删除目标机器的临时目录\n")
	_ = d.master.DropTmpDir()
	for _, slave := range d.slaves {
		_ = slave.DropTmpDir()
	}
	if d.arbiter != nil {
		_ = d.arbiter.DropTmpDir()
	}

}

func (d *MongoDBClusterDeploy) CheckEnv() error {
	logger.Infof("检查环境\n")
	if err := d.master.Install(true, false, false, d.coption.MongoConfig.Ipv6); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		if err := slave.Install(true, false, false, d.coption.MongoConfig.Ipv6); err != nil {
			return err
		}
	}
	if d.arbiter != nil {
		if err := d.arbiter.Install(true, true, false, d.coption.MongoConfig.Ipv6); err != nil {
			return err
		}
	}
	return nil
}

func (d *MongoDBClusterDeploy) Install() error {
	logger.Infof("开始安装\n")
	if err := d.master.Install(false, false, false, d.coption.MongoConfig.Ipv6); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		if err := slave.Install(false, false, false, d.coption.MongoConfig.Ipv6); err != nil {
			return err
		}
	}
	if d.arbiter != nil {
		if err := d.arbiter.Install(false, true, false, d.coption.MongoConfig.Ipv6); err != nil {
			return err
		}
	}
	return nil
}

func (d *MongoDBClusterDeploy) UNInstall() {
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

func (d *MongoDBClusterDeploy) CheckReplicaSetStatus() error {
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

func (d *MongoDBClusterDeploy) Info(role string) error {

	conn, err := dao.NewMongoClient(d.master.Host, d.master.Inst.Option.Port, d.master.Inst.Option.Username, d.master.Inst.Option.Password, "admin")
	if err != nil {
		return err
	}
	defer conn.Conn.Disconnect(context.Background())

	status, err := conn.GetReplStatus()
	if err != nil {
		return err
	}

	switch role {
	case config.MongoShards:
		logger.Successf("分片集群 Shard 副本集 %s 从库正常\n", status["set"].(string))
		logger.Successf("分片集群 Shard 副本集 %s 搭建成功\n", status["set"].(string))
	case config.MongoConfig:
		logger.Successf("分片集群 Config 副本集 %s 从库正常\n", status["set"].(string))
		logger.Successf("分片集群 Config 副本集 %s 搭建成功\n", status["set"].(string))
	}

	return nil
}
