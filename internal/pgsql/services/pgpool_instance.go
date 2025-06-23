/*
@Author : WuWeiJian
@Date : 2021-04-25 11:52
*/

package services

import (
	"dbup/internal/environment"
	"dbup/internal/pgsql/config"
	"dbup/internal/utils/command"
	"fmt"
	"path"
	"path/filepath"
)

type PGPoolInstance struct {
	DbupCmd string
	Host    string
	TmpDir  string
	Inst    *PgPoolInstall
	Conn    *command.Connection
	//spool
}

func NewPGPoolInstance(tmp, host, user, password string, port int, param config.PgPoolParameter) (*PGPoolInstance, error) {
	conn, err := command.NewConnection(host, user, password, port, 30)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewPgPoolInstall()
	if err := inst.HandlePrepareArgs(param, ""); err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 初始化install失败: %v", host, err)
	}
	inst.HandleArgs("")
	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &PGPoolInstance{DbupCmd: cmd, Host: host, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func NewPGPoolInstanceUseKeyFile(tmp, host, user, keyfile string, port int, param config.PgPoolParameter) (*PGPoolInstance, error) {
	conn, err := command.NewConnectionUseKeyFile(host, user, keyfile, port, 30)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewPgPoolInstall()
	if err := inst.HandlePrepareArgs(param, ""); err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 初始化install失败: %v", host, err)
	}
	inst.HandleArgs("")
	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &PGPoolInstance{DbupCmd: cmd, Host: host, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func (i *PGPoolInstance) CheckTmpDir() error {
	if i.Conn.IsExists(filepath.ToSlash(i.TmpDir)) {
		if !i.Conn.IsDir(filepath.ToSlash(i.TmpDir)) {
			return fmt.Errorf("在机器: %s 上, 目标文件已经存在", i.Host)
		}
		b, err := i.Conn.IsEmpty(filepath.ToSlash(i.TmpDir))
		if err != nil {
			return fmt.Errorf("在机器: %s 上, 判断目录(%s)是否为空失败: %v", i.Host, i.TmpDir, err)
		}
		if !b {
			return fmt.Errorf("在机器: %s 上, 目标目录不为空", i.Host)
		}
	}
	return nil
}

func (i *PGPoolInstance) Scp(source string) error {
	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "bin"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin")), err)
	}

	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "systemd"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "systemd")), err)
	}

	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "package/pgpool"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "pgpool")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "bin", i.DbupCmd), filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}
	if err := i.Conn.Scp(path.Join(source, "package", "md5"), filepath.ToSlash(path.Join(i.TmpDir, "package", "md5"))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "md5")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "systemd", config.PGPoolServiceTemplateFile), filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.PGPoolServiceTemplateFile))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.PGPoolServiceTemplateFile)), err)
	}

	// TODO: 根据目标机器的操作系统类型选择包
	pgpoolPackage := fmt.Sprintf("pgpool4.2_%s_%s.tar.gz", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	if err := i.Conn.Scp(path.Join(source, "package", "pgpool", pgpoolPackage), filepath.ToSlash(path.Join(i.TmpDir, "package", "pgpool", pgpoolPackage))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "pgpool", pgpoolPackage)), err)
	}

	if err := i.Conn.Chmod(filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), 0755); err != nil {
		return fmt.Errorf("在机器: %s 上, chmod目录(%s)权限失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}
	return nil
}

func (i *PGPoolInstance) DropTmpDir() error {
	cmd := fmt.Sprintf("cd %s; rm -rf *", filepath.ToSlash(i.TmpDir))
	if stdout, err := i.Conn.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *PGPoolInstance) Install(onlyCheck bool) error {
	cmd := fmt.Sprintf("%s pgsql pgpool-install --yes --port=%d --pcp-port=%d --wd-port=%d --heart-port=%d --bind-ip='%s' --pcp-bind-ip='%s' --address='%s' --username='%s' --password='%s' --dir='%s' --pgpool-ip='%s' --pg-port=%d --pg-dir='%s' --pg-master='%s' --pg-slave='%s' --node-id=%d --resource-limit='%s' --log='%s'",
		i.DbupCmd,
		i.Inst.parameter.Port,
		i.Inst.parameter.PcpPort,
		i.Inst.parameter.WDPort,
		i.Inst.parameter.HeartPort,
		i.Inst.parameter.BindIP,
		i.Inst.parameter.PcpBindIP,
		i.Inst.parameter.Address,
		i.Inst.parameter.Username,
		i.Inst.parameter.Password,
		i.Inst.parameter.Dir,
		i.Inst.parameter.PGPoolIP,
		i.Inst.parameter.PGPort,
		i.Inst.parameter.PGDir,
		i.Inst.parameter.PGMaster,
		i.Inst.parameter.PGSlave,
		i.Inst.parameter.NodeID,
		i.Inst.parameter.ResourceLimit,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgpool_install.log")))

	if onlyCheck {
		cmd = cmd + " --only-check"
	}

	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *PGPoolInstance) UNInstall() error {
	cmd := fmt.Sprintf("%s pgsql pgpool-uninstall --yes --port='%d' --dir='%s' --log='%s'",
		i.DbupCmd,
		i.Inst.parameter.Port,
		i.Inst.parameter.Dir,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgpool_uninstall.log")))

	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *PGPoolInstance) CheckSelect(port int, username, password, dbname string) error {
	cmd1 := fmt.Sprintf("%s pgsql check-select --host='%s' --port=%d --admin-user='%s' --admin-password='%s' --admin-database='%s' --log='%s'",
		i.DbupCmd,
		"127.0.0.1",
		port,
		username,
		password,
		dbname,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pgsql_manager.log")))

	cmd1 = path.Join(i.TmpDir, "bin", cmd1)
	if stdout, err := i.Conn.Sudo(cmd1, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd1, err, stdout)
	}
	return nil
}

//func (i *PGPoolInstance) SystemCtl(action string) error {
//	cmd := fmt.Sprintf("systemctl %s %s", action, i.Inst.serviceFileName)
//	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
//		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
//	}
//	return nil
//}
//
//func (i *PGPoolInstance) ChownData() error {
//	cmd := fmt.Sprintf("chown -R %s:%s %s", config.DefaultPGAdminUser, config.DefaultPGAdminUser, i.Inst.basePath)
//	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
//		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
//	}
//	return nil
//}
