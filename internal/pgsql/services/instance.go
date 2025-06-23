/*
@Author : WuWeiJian
@Date : 2021-02-28 17:48
*/

package services

import (
	"dbup/internal/environment"
	"dbup/internal/pgsql/config"
	"dbup/internal/utils/command"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

type Instance struct {
	DbupCmd string
	Host    string
	NodeID  int
	TmpDir  string
	Inst    *Install
	Conn    *command.Connection
	//spool
}

func NewInstance(tmp, host, user, password string, port int, pre config.Prepare, nodeID int) (*Instance, error) {
	conn, err := command.NewConnection(host, user, password, port, 30)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewInstall()
	if err := inst.HandlePrepareArgs(pre, ""); err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 初始化install失败: %v", host, err)
	}
	inst.HandleArgs("")

	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &Instance{DbupCmd: cmd, Host: host, NodeID: nodeID, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func NewInstanceUseKeyFile(tmp, host, user, keyfile string, port int, pre config.Prepare, nodeID int) (*Instance, error) {
	conn, err := command.NewConnectionUseKeyFile(host, user, keyfile, port, 30)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewInstall()
	if err := inst.HandlePrepareArgs(pre, ""); err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 初始化install失败: %v", host, err)
	}
	inst.HandleArgs("")

	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &Instance{DbupCmd: cmd, Host: host, NodeID: nodeID, TmpDir: tmp, Inst: inst, Conn: conn}, nil
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

	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "package/pgsql"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "pgsql")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "bin", i.DbupCmd), filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}
	if err := i.Conn.Scp(path.Join(source, "package", "md5"), filepath.ToSlash(path.Join(i.TmpDir, "package", "md5"))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "md5")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "systemd", config.PostgresServiceTemplateFile), filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.PostgresServiceTemplateFile))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.PostgresServiceTemplateFile)), err)
	}

	// TODO: 根据目标机器的操作系统类型选择包
	pgsqlPackage := fmt.Sprintf("pgsql12_%s_%s.tar.gz", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	if err := i.Conn.Scp(path.Join(source, "package", "pgsql", pgsqlPackage), filepath.ToSlash(path.Join(i.TmpDir, "package", "pgsql", pgsqlPackage))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "pgsql", pgsqlPackage)), err)
	}

	if err := i.Conn.Chmod(filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), 0755); err != nil {
		return fmt.Errorf("在机器: %s 上, chmod目录(%s)权限失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}
	return nil
}

func (i *Instance) DropTmpDir() error {
	cmd := fmt.Sprintf("cd %s; rm -rf *", filepath.ToSlash(i.TmpDir))
	if stdout, err := i.Conn.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *Instance) Install(p config.Prepare, onlyCheck, onlyInstall bool, ipv6 bool) error {
	cmd := fmt.Sprintf("%s pgsql install --yes --port=%d --admin-password='%s' --admin-password-expire-at='%s' --username='%s' --password='%s' --memory-size='%s' --dir='%s' --bind-ip='%s' --address='%s' --libraries='%s' --system-user='%s' --system-group='%s'  --resource-limit='%s' --log='%s'",
		i.DbupCmd,
		p.Port,
		p.AdminPassword,
		p.AdminPasswordExpireAt,
		p.Username,
		p.Password,
		p.MemorySize,
		p.Dir,
		p.BindIP,
		p.Address,
		p.Libraries,
		p.SystemUser,
		p.SystemGroup,
		p.ResourceLimit,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_install.log")))

	if onlyCheck {
		cmd = cmd + " --only-check"
	}

	if onlyInstall {
		cmd = cmd + " --only-install"
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

func (i *Instance) PrimaryInstall(p config.Prepare, onlyCheck bool) error {
	cmd := fmt.Sprintf("%s  pgsql-mha PrimaryInstall --yes --admin-password='%s' --dir='%s' --port=%d --log='%s'",
		i.DbupCmd,
		p.AdminPassword,
		p.Dir,
		p.Port,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_install.log")))

	if onlyCheck {
		cmd = cmd + " --only-check"
	}

	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}

	return nil
}

func (i *Instance) InstallSlave(p config.Prepare, master string) error {
	cmd := fmt.Sprintf("%s pgsql install-slave --yes --port=%d --username='%s' --password='%s' --dir='%s' --master='%s' --system-user='%s' --system-group='%s' --resource-limit='%s' --log='%s'",
		i.DbupCmd,
		p.Port,
		p.Username,
		p.Password,
		p.Dir,
		master,
		p.SystemUser,
		p.SystemGroup,
		p.ResourceLimit,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_install.log")))

	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *Instance) UNInstall(p config.Prepare) error {
	cmd := fmt.Sprintf("%s pgsql uninstall --yes --port='%d' --dir='%s' --log='%s'",
		i.DbupCmd,
		p.Port,
		p.Dir,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_uninstall.log")))

	if strings.Contains(p.Libraries, "repmgr") {
		cmd = cmd + " --repmgr"
	}

	cmd = path.Join(i.TmpDir, "bin", cmd)

	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

// func (i *Instance) UNRepmgrInstall(p config.Prepare) error {
// 	cmd := fmt.Sprintf("%s pgsql uninstall --yes --port='%d' --dir='%s' --log='%s'",
// 		i.DbupCmd,
// 		p.Port,
// 		p.Dir,
// 		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_uninstall.log")))

// 	if strings.Contains(p.Libraries, "repmgr") {
// 		cmd = cmd + " --repmgr"
// 	}

// 	cmd = path.Join(i.TmpDir, "bin", cmd)

// 	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
// 		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
// 	}
// 	return nil
// }

//没用了
//func (i *Instance) AddPgHba(ips []string) error {
//	for _, ip := range ips {
//		line := fmt.Sprintf(config.HbaFormat, "host", "replication", config.DefaultPGReplUser, ip+"/32", "md5")
//		cmd := fmt.Sprintf("echo '%s' >> %s", line, filepath.ToSlash(filepath.Join(i.Inst.dataPath, config.PgHbaFileName)))
//		if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
//			return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
//		}
//	}
//	if err := i.SystemCtl("reload"); err != nil {
//		return err
//	}
//	//cmd := fmt.Sprintf("\"sudo -S -H -u %s /bin/bash -l -c \"cd;%s reload -D %s\"\"", config.DefaultPGAdminUser, filepath.ToSlash(filepath.Join(i.Inst.serverBinPath, "pg_ctl")), filepath.ToSlash(i.Inst.dataPath))
//	//if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
//	//	return fmt.Errorf("执行(%s)失败: %v, 标准输出: %s", cmd, err, stdout)
//	//}
//	return nil
//}

func (i *Instance) CreateReplUser(slaves, PGReplPass string) error {
	cmd1 := fmt.Sprintf("%s pgsql user create --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --user='%s' --password='%s' --role='%s' --log='%s'",
		i.DbupCmd,
		config.DefaultPGSocketPath,
		i.Inst.port,
		i.Inst.prepare.SystemUser,
		i.Inst.adminPassword,
		config.DefaultPGAdminUser,
		config.DefaultPGReplUser,
		PGReplPass,
		"replication",
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_uninstall.log")))

	cmd1 = path.Join(i.TmpDir, "bin", cmd1)
	if stdout, err := i.Conn.Sudo(cmd1, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd1, err, stdout)
	}

	cmd2 := fmt.Sprintf("%s pgsql user grant --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --user='%s' --dbname='%s' --address='%s' --log='%s'",
		i.DbupCmd,
		config.DefaultPGSocketPath,
		i.Inst.port,
		i.Inst.prepare.SystemUser,
		i.Inst.adminPassword,
		config.DefaultPGAdminUser,
		config.DefaultPGReplUser,
		"replication",
		"0.0.0.0/0",
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_uninstall.log")))

	cmd2 = path.Join(i.TmpDir, "bin", cmd2)
	if stdout, err := i.Conn.Sudo(cmd2, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd2, err, stdout)
	}

	cmd3 := fmt.Sprintf("%s pgsql user grant --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --user='%s' --dbname='%s' --address='%s' --log='%s'",
		i.DbupCmd,
		config.DefaultPGSocketPath,
		i.Inst.port,
		i.Inst.prepare.SystemUser,
		i.Inst.adminPassword,
		config.DefaultPGAdminUser,
		config.DefaultPGReplUser,
		"replication",
		"::0/0",
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_uninstall.log")))

	cmd3 = path.Join(i.TmpDir, "bin", cmd3)
	if stdout, err := i.Conn.Sudo(cmd3, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd3, err, stdout)
	}
	return nil
}

func (i *Instance) UserGrant(user, dbname, ips string) error {
	cmd2 := fmt.Sprintf("%s pgsql user grant --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --user='%s' --dbname='%s' --address='%s' --log='%s'",
		i.DbupCmd,
		config.DefaultPGSocketPath,
		i.Inst.port,
		i.Inst.prepare.SystemUser,
		i.Inst.adminPassword,
		config.DefaultPGAdminUser,
		user,
		dbname,
		ips,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_uninstall.log")))

	cmd2 = path.Join(i.TmpDir, "bin", cmd2)
	if stdout, err := i.Conn.Sudo(cmd2, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd2, err, stdout)
	}

	return nil
}

func (i *Instance) CreateRepmgrUser(slaves string) error {

	cmd0 := fmt.Sprintf("%s pgsql database create --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --dbname='%s' --log='%s'",
		i.DbupCmd,
		config.DefaultPGSocketPath,
		i.Inst.port,
		i.Inst.prepare.SystemUser,
		i.Inst.adminPassword,
		config.DefaultPGAdminUser,
		i.Inst.prepare.RepmgrDBName,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_uninstall.log")))

	cmd0 = path.Join(i.TmpDir, "bin", cmd0)
	if stdout, err := i.Conn.Sudo(cmd0, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd0, err, stdout)
	}

	cmd1 := fmt.Sprintf("%s pgsql user create --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --user='%s' --password='%s' --role='%s' --log='%s'",
		i.DbupCmd,
		config.DefaultPGSocketPath,
		i.Inst.port,
		i.Inst.prepare.SystemUser,
		i.Inst.adminPassword,
		config.DefaultPGAdminUser,
		i.Inst.prepare.RepmgrUser,
		i.Inst.prepare.RepmgrPassword,
		"admin",
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_uninstall.log")))

	cmd1 = path.Join(i.TmpDir, "bin", cmd1)
	if stdout, err := i.Conn.Sudo(cmd1, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd1, err, stdout)
	}

	addr := "local,127.0.0.1," + i.Host + "," + slaves
	cmd2 := fmt.Sprintf("%s pgsql user grant --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --user='%s' --dbname='%s' --address='%s' --log='%s'",
		i.DbupCmd,
		config.DefaultPGSocketPath,
		i.Inst.port,
		i.Inst.prepare.SystemUser,
		i.Inst.adminPassword,
		config.DefaultPGAdminUser,
		i.Inst.prepare.RepmgrUser,
		"replication",
		addr,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_uninstall.log")))

	cmd2 = path.Join(i.TmpDir, "bin", cmd2)
	if stdout, err := i.Conn.Sudo(cmd2, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd2, err, stdout)
	}

	cmd3 := fmt.Sprintf("%s pgsql user grant --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --user='%s' --dbname='%s' --address='%s' --log='%s'",
		i.DbupCmd,
		config.DefaultPGSocketPath,
		i.Inst.port,
		i.Inst.prepare.SystemUser,
		i.Inst.adminPassword,
		config.DefaultPGAdminUser,
		i.Inst.prepare.RepmgrUser,
		i.Inst.prepare.RepmgrDBName,
		addr,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_uninstall.log")))

	cmd3 = path.Join(i.TmpDir, "bin", cmd3)
	if stdout, err := i.Conn.Sudo(cmd3, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd3, err, stdout)
	}

	return nil
}

func (i *Instance) GrantRepmgrSlaveUser(slaves string) error {
	cmd1 := fmt.Sprintf("%s pgsql user grant --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --user='%s' --dbname='%s' --address='%s' --log='%s'",
		i.DbupCmd,
		config.DefaultPGSocketPath,
		i.Inst.port,
		i.Inst.prepare.SystemUser,
		i.Inst.adminPassword,
		config.DefaultPGAdminUser,
		i.Inst.prepare.RepmgrUser,
		"replication",
		slaves,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_install.log")))

	cmd1 = path.Join(i.TmpDir, "bin", cmd1)
	if stdout, err := i.Conn.Sudo(cmd1, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd1, err, stdout)
	}

	cmd2 := fmt.Sprintf("%s pgsql user grant --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --user='%s' --dbname='%s' --address='%s' --log='%s'",
		i.DbupCmd,
		config.DefaultPGSocketPath,
		i.Inst.port,
		i.Inst.prepare.SystemUser,
		i.Inst.adminPassword,
		config.DefaultPGAdminUser,
		i.Inst.prepare.RepmgrUser,
		i.Inst.prepare.RepmgrDBName,
		slaves,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_install.log")))

	cmd2 = path.Join(i.TmpDir, "bin", cmd2)
	if stdout, err := i.Conn.Sudo(cmd2, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd2, err, stdout)
	}

	return nil
}

func (i *Instance) RepmgrPrimaryRegister(p config.Prepare) error {
	// sudo -u postgres /opt/pgsql5432/server/bin/repmgr -f /opt/pgsql5432/repmgr/repmgr.conf primary register
	cmd := fmt.Sprintf("sudo -u %s PGPASSWORD='%s' %s -f %s primary register",
		p.SystemUser,
		p.RepmgrPassword,
		filepath.Join(p.Dir, "server", "bin", "repmgr"),
		filepath.Join(p.Dir, "repmgr", "repmgr.conf"))

	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *Instance) RepmgrStandbyRegister(p config.Prepare) error {
	// sudo -u postgres /opt/pgsql5432/server/bin/repmgr -f /opt/pgsql5432/repmgr/repmgr.conf standby register
	cmd := fmt.Sprintf("sudo -u %s PGPASSWORD='%s' %s -f %s standby register",
		p.SystemUser,
		p.RepmgrPassword,
		filepath.Join(p.Dir, "server", "bin", "repmgr"),
		filepath.Join(p.Dir, "repmgr", "repmgr.conf"))

	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *Instance) RepmgrStandbyClone(p config.Prepare, master string) error {
	//  sudo -u postgres /opt/pgsql5432/server/bin/repmgr -f /opt/pgsql5432/repmgr/repmgr.conf -h 10.249.105.53 -p 5432 -U repmgr -d repmgr standby clone
	cmd := fmt.Sprintf("sudo -u %s PGPASSWORD='%s' %s -f %s -h %s -p %d -U %s -d %s standby clone",
		p.SystemUser,
		p.RepmgrPassword,
		filepath.Join(p.Dir, "server", "bin", "repmgr"),
		filepath.Join(p.Dir, "repmgr", "repmgr.conf"),
		master,
		p.Port,
		p.RepmgrUser,
		p.RepmgrDBName)

	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *Instance) RepmgrDaemon(p config.Prepare) error {
	// sudo -u postgres /opt/pgsql5432/server/bin/repmgrd -d -f /opt/pgsql5432/repmgr/repmgr.conf --pid-file /opt/pgsql5432/repmgr/repmgrd.pid
	cmd := fmt.Sprintf("sudo -u %s %s -d -f %s --pid-file %s",
		p.SystemUser,
		filepath.Join(p.Dir, "server", "bin", "repmgrd"),
		filepath.Join(p.Dir, "repmgr", "repmgr.conf"),
		filepath.Join(p.Dir, "repmgr", "repmgrd.pid"))

	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

// 好像没啥用
func (i *Instance) RepmgrClusterShow(p config.Prepare) error {
	// sudo -u postgres /opt/pgsql5432/server/bin/repmgr -f /opt/pgsql5432/repmgr/repmgr.conf cluster show
	cmd := fmt.Sprintf("sudo -u %s %s -f %s cluster show",
		p.SystemUser,
		filepath.Join(p.Dir, "server", "bin", "repmgr"),
		filepath.Join(p.Dir, "repmgr", "repmgr.conf"))

	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *Instance) RepmgrStartPostgreSql(p config.Prepare) error {
	// sudo -u postgres  /opt/pgsql5432/server/bin/pg_ctl start -D /opt/pgsql5432/data -l /opt/pgsql5432/logs/pgsql.log
	cmd := fmt.Sprintf("sudo -u %s %s start -D %s -l %s",
		p.SystemUser,
		filepath.Join(p.Dir, "server", "bin", "pg_ctl"),
		filepath.Join(p.Dir, "data"),
		filepath.Join(p.Dir, "logs", "postgres.log"))

	// fmt.Println(cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	} else {
		fmt.Println(string(stdout))
	}

	return nil
}

func (i *Instance) SystemCtl(action string) error {
	cmd := fmt.Sprintf("systemctl %s %s", action, i.Inst.serviceFileName)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

// 废弃了
func (i *Instance) RemoveData() error {
	cmd := fmt.Sprintf("rm -rf %s", i.Inst.dataPath)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *Instance) ChownData(user, group string) error {
	cmd := fmt.Sprintf("chown -R %s:%s %s", user, group, i.Inst.dataPath)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *Instance) Replication(master, PGReplPass string) error {
	cmd := fmt.Sprintf("PGPASSWORD=%s %s  -D %s -R -Fp -Xs -v  -p %d -h %s -U %s  -P",
		PGReplPass,
		filepath.ToSlash(filepath.Join(i.Inst.serverBinPath, "pg_basebackup")),
		i.Inst.dataPath,
		i.Inst.port,
		master,
		config.DefaultPGReplUser)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

//func (i *Instance) CheckOK(c int) (bool, error) {
//	conn, err := dao.NewPgConn(i.Host, i.Inst.port, i.Inst.prepare.Username, i.Inst.prepare.Password, i.Inst.prepare.Username)
//	if err != nil {
//		return false, err
//	}
//	count, err := conn.ReplCount()
//	if err != nil {
//		return false, err
//	}
//	if count != c {
//		return false, nil
//	}
//	return true, nil
//}

func (i *Instance) CheckSlaves(s string) error {
	cmd1 := fmt.Sprintf("%s pgsql check-slaves --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --log='%s' %s",
		i.DbupCmd,
		config.DefaultPGSocketPath,
		i.Inst.port,
		i.Inst.adminUser,
		i.Inst.adminPassword,
		config.DefaultPGAdminUser,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_manager.log")),
		s)

	cmd1 = path.Join(i.TmpDir, "bin", cmd1)
	if stdout, err := i.Conn.Sudo(cmd1, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd1, err, stdout)
	}
	return nil
}
