/*
@Author : WuWeiJian
@Date : 2021-04-13 11:02
*/

package services

import (
	"dbup/internal/environment"
	"dbup/internal/redis/config"
	"dbup/internal/redis/dao"
	"dbup/internal/utils/newssh"
	"fmt"
	"path"
	"path/filepath"
)

type Instance struct {
	DbupCmd string
	Host    string
	TmpDir  string
	Inst    *Install
	Conn    *newssh.Connection
	//spool
}

func NewInstance(tmp, host, user, password string, port int, pre config.Parameters) (*Instance, error) {
	conn, err := newssh.NewConnection(host, user, password, port, 600)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewInstall()
	if err := inst.HandleParam(pre, ""); err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 初始化install失败: %v", host, err)
	}
	inst.Init()

	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &Instance{DbupCmd: cmd, Host: host, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func NewInstanceUseKeyFile(tmp, host, user, keyfile string, port int, pre config.Parameters) (*Instance, error) {
	conn, err := newssh.NewConnectionUseKeyFile(host, user, keyfile, port, 600)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewInstall()
	if err := inst.HandleParam(pre, ""); err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 初始化install失败: %v", host, err)
	}
	inst.Init()

	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &Instance{DbupCmd: cmd, Host: host, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func (i *Instance) CheckTmpDir() error {
	if i.Conn.IsExists(filepath.ToSlash(i.TmpDir)) {
		if !i.Conn.IsDir(filepath.ToSlash(i.TmpDir)) {
			return fmt.Errorf("在机器: %s 上, 目标文件(%s)已经存在", i.Host, i.TmpDir)
		}
		b, err := i.Conn.IsEmpty(filepath.ToSlash(i.TmpDir))
		if err != nil {
			return fmt.Errorf("在机器: %s 上, 判断目录(%s)是否为空失败: %v", i.Host, i.TmpDir, err)
		}
		if !b {
			return fmt.Errorf("在机器: %s 上, 目标目录(%s)不为空", i.Host, i.TmpDir)
		}
	}
	return nil
}

func (i *Instance) Scp(source string) error {
	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "bin"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin")), err)
	}

	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "systemd"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "systemd")), err)
	}

	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "package/redis"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "bin", i.DbupCmd), filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}
	if err := i.Conn.Scp(path.Join(source, "package", "md5"), filepath.ToSlash(path.Join(i.TmpDir, "package", "md5"))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "md5")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "systemd", config.RedisServiceTemplateFile), filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.RedisServiceTemplateFile))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.RedisServiceTemplateFile)), err)
	}

	// TODO: 根据目标机器的操作系统类型选择包
	redisPackage := fmt.Sprintf("redis6_%s_%s.tar.gz", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	if err := i.Conn.Scp(path.Join(source, "package", "redis", redisPackage), filepath.ToSlash(path.Join(i.TmpDir, "package", "redis", redisPackage))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "pgsql", redisPackage)), err)
	}

	if err := i.Conn.Chmod(filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), 0755); err != nil {
		return fmt.Errorf("在机器: %s 上, chmod目录(%s)权限失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}
	return nil
}

func (i *Instance) DropTmpDir() error {
	cmd := fmt.Sprintf("cd %s; rm -rf *", filepath.ToSlash(i.TmpDir))
	if stdout, stderr, err := i.Conn.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s, 标准错误: %s", i.Host, cmd, err, stdout, stderr)
	}
	return nil
}

func (i *Instance) Install(cluster bool, onlyCheck bool, ipv6 bool) error {
	cmd := fmt.Sprintf("%s redis install --yes --port=%d --password='%s' --memory-size='%s' --dir='%s' --maxmemory-policy='%s' --module='%s' --master='%s' --system-user='%s' --system-group='%s' --resource-limit='%s' --log='%s'",
		i.DbupCmd,
		i.Inst.parameters.Port,
		i.Inst.parameters.Password,
		i.Inst.parameters.MemorySize,
		i.Inst.parameters.Dir,
		i.Inst.parameters.MaxmemoryPolicy,
		i.Inst.parameters.Module,
		i.Inst.parameters.Master,
		i.Inst.parameters.SystemUser,
		i.Inst.parameters.SystemGroup,
		i.Inst.parameters.ResourceLimit,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_redis_install.log")))

	if cluster {
		cmd = cmd + " --cluster"
	}

	if onlyCheck {
		cmd = cmd + " --only-check"
	}

	if ipv6 {
		cmd = cmd + " --ipv6"
	}
	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, stderr, err := i.Conn.Sudo(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s, 标准错误: %s", i.Host, cmd, err, stdout, stderr)
	}
	return nil
}

func (i *Instance) UNInstall() error {
	cmd := fmt.Sprintf("%s redis uninstall --yes --port='%d' --dir='%s' --log='%s'",
		i.DbupCmd,
		i.Inst.parameters.Port,
		i.Inst.parameters.Dir,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_redis_uninstall.log")))

	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, stderr, err := i.Conn.Sudo(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s, 标准错误: %s", i.Host, cmd, err, stdout, stderr)
	}
	return nil
}

func (i *Instance) SystemCtl(action string) error {
	cmd := fmt.Sprintf("systemctl %s %s", action, i.Inst.serviceFileName)
	if stdout, stderr, err := i.Conn.Sudo(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s, 标准错误: %s", i.Host, cmd, err, stdout, stderr)
	}
	return nil
}

func (i *Instance) ClusterCreate(nodes string, replica int) error {
	cli := fmt.Sprintf("%s/server/bin/redis-cli", i.Inst.parameters.Dir)
	cmd := fmt.Sprintf("%s -a '%s' --cluster create %s --cluster-replicas %d --cluster-yes", cli, i.Inst.parameters.Password, nodes, replica)
	if stdout, stderr, err := i.Conn.Sudo(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s, 标准错误: %s", i.Host, cmd, err, stdout, stderr)
	}
	return nil
}

func (i *Instance) ClusterAddNode(node, cluster, role, masterID string) error {
	cli := fmt.Sprintf("%s/server/bin/redis-cli", i.Inst.parameters.Dir)
	slaveCmd := ""
	if role == "slave" {
		slaveCmd = "--cluster-slave"
		if masterID != "" {
			slaveCmd = fmt.Sprintf("%s --cluster-master-id %s", slaveCmd, masterID)
		}
	}

	cmd := fmt.Sprintf("%s -a '%s' --cluster add-node %s %s %s --cluster-yes", cli, i.Inst.parameters.Password, node, cluster, slaveCmd)
	if stdout, stderr, err := i.Conn.Sudo(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s, 标准错误: %s", i.Host, cmd, err, stdout, stderr)
	}
	return nil
}

func (i *Instance) ClusterReBalance(cluster string) error {
	cli := fmt.Sprintf("%s/server/bin/redis-cli", i.Inst.parameters.Dir)
	cmd := fmt.Sprintf("%s -a '%s' --cluster rebalance %s --cluster-use-empty-masters --cluster-yes", cli, i.Inst.parameters.Password, cluster)
	if stdout, stderr, err := i.Conn.Sudo(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s, 标准错误: %s", i.Host, cmd, err, stdout, stderr)
	}
	return nil
}

//func (i *Instance) ClusterFix(cluster string) error {
//	cli := fmt.Sprintf("%s/server/bin/redis-cli", i.Inst.parameters.Dir)
//	cmd := fmt.Sprintf("%s -a %s --cluster fix %s --cluster-yes", cli, i.Inst.parameters.Password, cluster)
//	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
//		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
//	}
//	return nil
//}

func (i *Instance) Replication(master string, port int) error {
	conn, err := dao.NewRedisConn(i.Host, i.Inst.port, i.Inst.parameters.Password)
	if err != nil {
		return err
	}
	defer conn.Conn.Close()
	return conn.SlaveOf(master, port)
}

func (i *Instance) CheckSlaves() error {
	conn, err := dao.NewRedisConn(i.Host, i.Inst.port, i.Inst.parameters.Password)
	if err != nil {
		return err
	}
	defer conn.Conn.Close()
	status, err := conn.SlaveStatus()
	if err != nil {
		return err
	}
	if status != "up" {
		return fmt.Errorf("从库状态异常")
	}
	return nil
}
