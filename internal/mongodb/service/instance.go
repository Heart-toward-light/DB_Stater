/*
@Author : WuWeiJian
@Date : 2021-05-13 15:49
*/

package service

import (
	"dbup/internal/environment"
	"dbup/internal/mongodb/config"
	"dbup/internal/utils/command"
	"fmt"
	"path"
	"path/filepath"
)

type MongoDBInstance struct {
	DbupCmd string
	Host    string
	TmpDir  string
	Inst    *MongoDBInstall
	Conn    *command.Connection
	//spool
}

func NewMongoDBInstance(tmp, host, user, password string, port int, option config.MongodbOptions) (*MongoDBInstance, error) {
	conn, err := command.NewConnection(host, user, password, port, 300)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewMongoDBInstall(&option)
	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &MongoDBInstance{DbupCmd: cmd, Host: host, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func NewMongoDBInstanceUseKeyFile(tmp, host, user, keyfile string, port int, option config.MongodbOptions) (*MongoDBInstance, error) {
	conn, err := command.NewConnectionUseKeyFile(host, user, keyfile, port, 300)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewMongoDBInstall(&option)
	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &MongoDBInstance{DbupCmd: cmd, Host: host, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func (i *MongoDBInstance) CheckTmpDir() error {
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

func (i *MongoDBInstance) CheckHosts(hostname []string) error {
	if err := i.Conn.Hostsanalysis(hostname); err != nil {
		return fmt.Errorf("%v", err)
	}
	return nil
}

func (i *MongoDBInstance) Scp(source string) error {
	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "bin"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin")), err)
	}

	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "systemd"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "systemd")), err)
	}

	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "package/mongodb"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package/mongodb")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "bin", i.DbupCmd), filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}
	if err := i.Conn.Scp(path.Join(source, "package", "md5"), filepath.ToSlash(path.Join(i.TmpDir, "package", "md5"))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "md5")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "systemd", config.MongoDBServiceTemplateFile), filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.MongoDBServiceTemplateFile))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.MongoDBServiceTemplateFile)), err)
	}

	// TODO: 根据目标机器的操作系统类型选择包
	mongodbPackage := fmt.Sprintf("mongodb%s-%s-%s.tar.gz", config.DefaultMongoDBVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	if err := i.Conn.Scp(path.Join(source, "package", "mongodb", mongodbPackage), filepath.ToSlash(path.Join(i.TmpDir, "package", "mongodb", mongodbPackage))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "mongodb", mongodbPackage)), err)
	}

	if err := i.Conn.Chmod(filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), 0755); err != nil {
		return fmt.Errorf("在机器: %s 上, chmod目录(%s)权限失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}
	return nil
}

func (i *MongoDBInstance) DropTmpDir() error {
	cmd := fmt.Sprintf("cd %s; rm -rf *", filepath.ToSlash(i.TmpDir))
	if stdout, err := i.Conn.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *MongoDBInstance) Install(onlyCheck bool, arbiter bool, noRollback bool, ipv6 bool) error {
	cmd := fmt.Sprintf("%s mongodb install --yes --port=%d --username='%s' --password='%s' --replSetName='%s' --memory=%d --dir='%s' --bind-ip='%s' --owner='%s' --join='%s' --system-user='%s' --system-group='%s' --resource-limit='%s' --log='%s'",
		i.DbupCmd,
		i.Inst.Option.Port,
		i.Inst.Option.Username,
		i.Inst.Option.Password,
		i.Inst.Option.ReplSetName,
		i.Inst.Option.Memory,
		i.Inst.Option.Dir,
		i.Inst.Option.BindIP,
		i.Host,
		i.Inst.Option.Join,
		i.Inst.Option.SystemUser,
		i.Inst.Option.SystemGroup,
		i.Inst.Option.ResourceLimit,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_mongodb_install.log")))

	if onlyCheck {
		cmd = cmd + " --only-check"
	}

	if arbiter {
		cmd = cmd + " --arbiter"
	}

	if noRollback {
		cmd = cmd + " --no-rollback"
	}

	if ipv6 {
		cmd = cmd + " --ipv6"
	}
	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *MongoDBInstance) UNInstall() error {
	cmd := fmt.Sprintf("%s mongodb uninstall --yes --port='%d' --dir='%s' --log='%s'",
		i.DbupCmd,
		i.Inst.Option.Port,
		i.Inst.Option.Dir,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_mongodb_uninstall.log")))

	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}
