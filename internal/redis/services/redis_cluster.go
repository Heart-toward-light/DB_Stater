/*
@Author : WuWeiJian
@Date : 2021-07-27 15:20
*/

package services

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/redis/config"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path"
	"strings"
)

// 安装redis cluster的总控制逻辑
type RedisClusterDeploy struct {
	Option    config.RedisClusterOption
	ScpStatus map[string]bool
	masters   []*Instance
	slaves    []*Instance
	replica   int
}

func NewRedisClusterDeploy() *RedisClusterDeploy {
	return &RedisClusterDeploy{ScpStatus: make(map[string]bool), replica: 1}
}

func (d *RedisClusterDeploy) Run(c string) error {
	// 初始化参数和配置环节
	if err := global.YAMLLoadFromFile(c, &d.Option); err != nil {
		return err
	}

	if err := d.Option.Validator(); err != nil {
		return err
	}

	d.Option.SetDefault()
	d.GetHostList()

	if err := d.Option.CheckDuplicate(); err != nil {
		return err
	}

	if len(d.Option.Slave) == 0 {
		d.replica = 0
	}

	logger.Infof("初始化部署对象\n")
	if d.Option.SSHConfig.Password != "" {
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

	if err := d.CheckEnv(); err != nil {
		return err
	}

	if err := d.InstallAndInitSlave(); err != nil {
		if !d.Option.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			d.UNInstall()
		}
		return err
	}
	return nil
}

func (d *RedisClusterDeploy) RemoveCluster(c string, yes bool) error {
	// 初始化参数和配置环节
	if err := global.YAMLLoadFromFile(c, &d.Option); err != nil {
		return err
	}

	for _, node := range d.Option.Master {
		if err := node.Validator(); err != nil {
			return err
		}
	}
	for _, node := range d.Option.Slave {
		if err := node.Validator(); err != nil {
			return err
		}
	}

	d.Option.SetDefault()
	d.GetHostList()

	if err := d.Option.CheckDuplicate(); err != nil {
		return err
	}

	for _, node := range d.Option.Master {
		logger.Warningf("要删除的集群节点以及数据目录: %s:%d %s\n", node.Host, node.Port, node.Dir)
	}
	for _, node := range d.Option.Slave {
		logger.Warningf("要删除的集群节点以及数据目录: %s:%d %s\n", node.Host, node.Port, node.Dir)
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
	if d.Option.SSHConfig.Password != "" {
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

func (d *RedisClusterDeploy) GetHostList() {
	for _, node := range d.Option.Master {
		d.ScpStatus[node.Host] = false
	}
	for _, node := range d.Option.Slave {
		d.ScpStatus[node.Host] = false
	}
}

func (d *RedisClusterDeploy) Init() error {
	for _, node := range d.Option.Master {
		m, err := NewInstance(d.Option.SSHConfig.TmpDir,
			node.Host,
			d.Option.SSHConfig.Username,
			d.Option.SSHConfig.Password,
			d.Option.SSHConfig.Port,
			config.Parameters{
				SystemUser:      d.Option.RedisConfig.SystemUser,
				SystemGroup:     d.Option.RedisConfig.SystemGroup,
				Port:            node.Port,
				Dir:             node.Dir,
				Password:        d.Option.RedisConfig.Password,
				MemorySize:      d.Option.RedisConfig.Memory,
				Module:          d.Option.RedisConfig.Module,
				ResourceLimit:   d.Option.RedisConfig.ResourceLimit,
				MaxmemoryPolicy: d.Option.RedisConfig.MaxmemoryPolicy,
			})
		if err != nil {
			return err
		}
		d.masters = append(d.masters, m)
	}

	for _, node := range d.Option.Slave {
		s, err := NewInstance(d.Option.SSHConfig.TmpDir,
			node.Host,
			d.Option.SSHConfig.Username,
			d.Option.SSHConfig.Password,
			d.Option.SSHConfig.Port,
			config.Parameters{
				SystemUser:    d.Option.RedisConfig.SystemUser,
				SystemGroup:   d.Option.RedisConfig.SystemGroup,
				Port:          node.Port,
				Dir:           node.Dir,
				Password:      d.Option.RedisConfig.Password,
				MemorySize:    d.Option.RedisConfig.Memory,
				Module:        d.Option.RedisConfig.Module,
				ResourceLimit: d.Option.RedisConfig.ResourceLimit,
			})
		if err != nil {
			return err
		}
		d.slaves = append(d.slaves, s)
	}
	return nil
}

func (d *RedisClusterDeploy) InitUseKeyFile() error {
	for _, node := range d.Option.Master {
		m, err := NewInstanceUseKeyFile(d.Option.SSHConfig.TmpDir,
			node.Host,
			d.Option.SSHConfig.Username,
			d.Option.SSHConfig.KeyFile,
			d.Option.SSHConfig.Port,
			config.Parameters{
				SystemUser:      d.Option.RedisConfig.SystemUser,
				SystemGroup:     d.Option.RedisConfig.SystemGroup,
				Port:            node.Port,
				Dir:             node.Dir,
				Password:        d.Option.RedisConfig.Password,
				MemorySize:      d.Option.RedisConfig.Memory,
				Module:          d.Option.RedisConfig.Module,
				ResourceLimit:   d.Option.RedisConfig.ResourceLimit,
				MaxmemoryPolicy: d.Option.RedisConfig.MaxmemoryPolicy,
			})
		if err != nil {
			return err
		}
		d.masters = append(d.masters, m)
	}

	for _, node := range d.Option.Slave {
		s, err := NewInstanceUseKeyFile(d.Option.SSHConfig.TmpDir,
			node.Host,
			d.Option.SSHConfig.Username,
			d.Option.SSHConfig.KeyFile,
			d.Option.SSHConfig.Port,
			config.Parameters{
				SystemUser:    d.Option.RedisConfig.SystemUser,
				SystemGroup:   d.Option.RedisConfig.SystemGroup,
				Port:          node.Port,
				Dir:           node.Dir,
				Password:      d.Option.RedisConfig.Password,
				MemorySize:    d.Option.RedisConfig.Memory,
				Module:        d.Option.RedisConfig.Module,
				ResourceLimit: d.Option.RedisConfig.ResourceLimit,
			})
		if err != nil {
			return err
		}
		d.slaves = append(d.slaves, s)
	}
	return nil
}

func (d *RedisClusterDeploy) CheckTmpDir() error {
	logger.Infof("检查目标机器的临时目录\n")
	for _, slave := range d.masters {
		if err := slave.CheckTmpDir(); err != nil {
			return err
		}
	}
	for _, slave := range d.slaves {
		if err := slave.CheckTmpDir(); err != nil {
			return err
		}
	}
	return nil
}

func (d *RedisClusterDeploy) DropTmpDir() {
	logger.Infof("删除目标机器的临时目录\n")
	for _, master := range d.masters {
		_ = master.DropTmpDir()
	}
	for _, slave := range d.slaves {
		_ = slave.DropTmpDir()
	}
}

func (d *RedisClusterDeploy) Scp() error {
	logger.Infof("将所需文件复制到目标机器\n")
	source := path.Join(environment.GlobalEnv().ProgramPath, "..")
	for _, master := range d.masters {
		if d.ScpStatus[master.Host] {
			continue
		}
		logger.Infof("复制到: %s\n", master.Host)
		if err := master.Scp(source); err != nil {
			return err
		}
	}
	for _, slave := range d.slaves {
		if d.ScpStatus[slave.Host] {
			continue
		}
		logger.Infof("复制到: %s\n", slave.Host)
		if err := slave.Scp(source); err != nil {
			return err
		}
	}
	return nil
}

func (d *RedisClusterDeploy) CheckEnv() error {
	logger.Infof("检查环境\n")
	for _, master := range d.masters {
		if err := master.Install(true, true, false); err != nil {
			return err
		}
	}
	for _, slave := range d.slaves {
		if err := slave.Install(true, true, false); err != nil {
			return err
		}
	}
	return nil
}

func (d *RedisClusterDeploy) InstallAndInitSlave() error {
	if err := d.Install(); err != nil {
		return err
	}

	if err := d.CreateCluster(); err != nil {
		return err
	}

	//logger.Infof("5秒后检查集群状态\n")
	//time.Sleep(5 * time.Second)
	//for _, slave := range d.slaves {
	//	if err := slave.CheckSlaves(); err != nil {
	//		return err
	//	}
	//}
	//
	//logger.Successf("从库正常\n")
	logger.Successf("集群搭建成功\n")
	return nil
}

func (d *RedisClusterDeploy) Install() error {
	logger.Infof("开始安装\n")
	for _, master := range d.masters {
		if err := master.Install(true, false, false); err != nil {
			return err
		}
	}
	for _, slave := range d.slaves {
		if err := slave.Install(true, false, false); err != nil {
			return err
		}
	}
	return nil
}

func (d *RedisClusterDeploy) UNInstall() {
	logger.Infof("开始卸载清理\n")
	for _, master := range d.masters {
		if err := master.UNInstall(); err != nil {
			logger.Warningf("卸载节点: %s 失败: %v\n", master.Host, err)
		}
	}
	for _, slave := range d.slaves {
		if err := slave.UNInstall(); err != nil {
			logger.Warningf("卸载节点: %s 失败: %v\n", slave.Host, err)
		}
	}
}

func (d *RedisClusterDeploy) CreateCluster() error {
	logger.Infof("初始化集群\n")
	nodes := ""
	for _, node := range d.Option.Master {
		if d.Option.RedisConfig.IPV6 {
			host := utils.Ipv6conversion(node.Host)
			nodes += fmt.Sprintf(" %s:%d", host, node.Port)
		} else {
			nodes += fmt.Sprintf(" %s:%d", node.Host, node.Port)
		}
	}
	for _, node := range d.Option.Slave {
		if d.Option.RedisConfig.IPV6 {
			host := utils.Ipv6conversion(node.Host)
			nodes += fmt.Sprintf(" %s:%d", host, node.Port)
		} else {
			nodes += fmt.Sprintf(" %s:%d", node.Host, node.Port)
		}
	}
	if err := d.masters[0].ClusterCreate(nodes, d.replica); err != nil {
		return err
	}
	return nil
}
