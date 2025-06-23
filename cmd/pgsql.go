/*
@Author : WuWeiJian
@Date : 2020-12-02 17:42
*/

package cmd

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/pgsql"
	"dbup/internal/pgsql/config"
	"dbup/internal/pgsql/services"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func pgsqlCmd() *cobra.Command {
	// 定义二级命令: pgsql
	var cmd = &cobra.Command{
		Use:   "pgsql",
		Short: "pgsql相关操作",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	// 装载命令
	cmd.AddCommand(
		//pgsqlPrepareCmd(),
		pgsqlInstallCmd(),
		pgsqlInstallSlaveCmd(),
		pgsqlAddSlaveCmd(),
		pgsqlUNInstallCmd(),
		pgsqlBackupCmd(),
		pgsqlBackupTablesCmd(),
		pgsqlBackupTaskCmd(),
		pgsqlDeployCmd(),
		pgsqlRemoveDeployCmd(),
		// pgsqlPGPoolInstallCmd(),
		pgsqlUserCmd(),
		pgsqlDatabaseCmd(),
		pgsqlCheckSlavesCmd(),
		pgsqlCheckSelectCmd(),
		// pgpoolUNInstallCmd(),
		// pgsqlPGPoolClusterDeployCmd(),
		// pgsqlRemovePGPoolClusterDeployCmd(),
		pgsqlPromoteCmd(),
		PGsqlUPgradeCmd(),
	)

	return cmd
}

// dbup pgsql prepare 已经废弃了
func pgsqlPrepareCmd() *cobra.Command {
	var pre config.Prepare
	var cfgFile string
	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "pgsql 单机版安装之前, 生成安装前的部署配置信息",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return pre.MakeConfigFile(cfgFile)
		},
	}
	cmd.Flags().StringVarP(&pre.Username, "username", "u", "", "用户名, 2到63位小写字母,下划线,数字组成; 首位不能是数字")
	cmd.Flags().StringVarP(&pre.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&pre.Dir, "dir", "d", "", "数据目录")
	cmd.Flags().StringVarP(&pre.MemorySize, "memory-size", "m", "", "pgsql 数据库内存大小")
	cmd.Flags().IntVarP(&pre.Port, "port", "P", 0, "pgsql 数据库监听端口")
	cmd.Flags().StringVarP(&pre.Address, "address", "a", "", "pgsql 数据库授权IP列表")
	cmd.Flags().StringVarP(&pre.BindIP, "bind-ip", "b", "", "pgsql 数据库监听地址")
	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", fmt.Sprintf("安装配置文件,默认为:$HOME/%s", config.DefaultPGCfgFile))
	return cmd
}

// dbup pgsql install
func pgsqlInstallCmd() *cobra.Command {
	var pre config.Prepare
	var cfgFile string
	var packageName string
	var onlyCheck bool
	var onlyInstall bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "pgsql 单机版安装",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := pgsql.NewPgsql()
			return pg.Install(pre, cfgFile, packageName, onlyCheck, onlyInstall)
		},
	}
	cmd.Flags().StringVar(&pre.SystemUser, "system-user", config.DefaultPGAdminUser, "pgsql安装的操作系统用户")
	cmd.Flags().StringVar(&pre.SystemGroup, "system-group", config.DefaultPGAdminUser, "pgsql安装的操作系统用户组")
	cmd.Flags().StringVar(&pre.AdminPassword, "admin-password", "", fmt.Sprintf("请用 --admin-password 参数指定 超级管理员(postgres) 的密码, 建议使用: %s", utils.GeneratePasswd(16)))
	cmd.Flags().StringVar(&pre.AdminPasswordExpireAt, "admin-password-expire-at", "", "超级管理员(postgres) 密码的过期时间")
	cmd.Flags().StringVarP(&pre.Username, "username", "u", "", "用户名, 2到63位小写字母,下划线,数字组成; 首位不能是数字")
	cmd.Flags().StringVarP(&pre.Password, "password", "p", "", "密码, 默认随机生成16位密码")
	cmd.Flags().StringVarP(&pre.Dir, "dir", "d", "", "pgsql安装目录, 默认: /opt/pgsql$PORT")
	cmd.Flags().StringVarP(&pre.MemorySize, "memory-size", "m", "", "pgsql 数据库内存大小, 默认操作系统最大内存的50%")
	cmd.Flags().IntVarP(&pre.Port, "port", "P", 0, "pgsql 数据库监听端口, 默认: 5432")
	cmd.Flags().StringVar(&pre.AdminAddress, "admin-address", "", "pgsql 数据库授权IP列表, 默认只能本地登录")
	cmd.Flags().StringVarP(&pre.Address, "address", "a", "", "pgsql 数据库授权IP列表, 默认 0.0.0.0/0")
	cmd.Flags().StringVarP(&pre.BindIP, "bind-ip", "b", "", "pgsql 数据库监听地址, 默认: *")
	cmd.Flags().StringVar(&pre.Libraries, "libraries", "", "pgsql启用的插件,以逗号分割, 目前只支持 [timescaledb]")
	cmd.Flags().BoolVar(&pre.Ipv6, "ipv6", false, "是否开启IPV6功能,默认不开启")
	cmd.Flags().StringVar(&pre.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&pre.Yes, "yes", "y", false, "是否确认安装")
	cmd.Flags().BoolVarP(&pre.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "安装配置文件, 默认不使用配置文件")
	cmd.Flags().BoolVar(&onlyCheck, "only-check", false, "只检查配置和环境, 不进行实际安装操作")
	cmd.Flags().BoolVar(&onlyInstall, "only-install", false, "只解压pgsql二进制程序包和生成启动文件, 不初始化数据目录, 主从集群,初始化从库使用")
	//cmd.Flags().StringVar(&packageName, "package", "", "指定要安装的包文件, 默认为当前执行程序目录/../pgsql/pgsql12_linux_amd64.tar.gz")
	return cmd
}

// dbup pgsql install
func pgsqlInstallSlaveCmd() *cobra.Command {
	var pre config.Prepare
	var master string
	cmd := &cobra.Command{
		Use:   "install-slave",
		Short: "pgsql 从库安装",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := pgsql.NewPgsql()
			return pg.InstallSlave(pre, master)
		},
	}
	cmd.Flags().StringVar(&pre.SystemUser, "system-user", config.DefaultPGAdminUser, "pgsql安装的操作系统用户")
	cmd.Flags().StringVar(&pre.SystemGroup, "system-group", config.DefaultPGAdminUser, "pgsql安装的操作系统用户组")
	cmd.Flags().StringVarP(&pre.Username, "username", "u", "", "要同步的主库上创建的主从同步用户")
	cmd.Flags().StringVarP(&pre.Password, "password", "p", "", "主从同步用户的密码")
	cmd.Flags().StringVarP(&pre.Dir, "dir", "d", "", "pgsql安装目录, 默认: /opt/pgsql$PORT")
	cmd.Flags().IntVarP(&pre.Port, "port", "P", 0, "pgsql 数据库监听端口, 默认: 5432")
	cmd.Flags().StringVarP(&master, "master", "m", "", "要同步的主库的<地址:端口>")
	cmd.Flags().StringVar(&pre.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&pre.Yes, "yes", "y", false, "是否确认安装")
	cmd.Flags().BoolVarP(&pre.NoRollback, "no-rollback", "n", false, "安装失败不回滚")

	return cmd
}

// dbup pgsql install
func pgsqlAddSlaveCmd() *cobra.Command {
	var sshOption global.SSHConfig
	var pre config.Prepare
	var master string
	cmd := &cobra.Command{
		Use:   "add-slave",
		Short: "pgsql 从库安装",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sshOption.Host == "" {
				return fmt.Errorf("请指定要创建从库的远程机器IP地址")
			}

			if master == "" {
				return fmt.Errorf("请指定要要同步的主库的节点信息<IP:PORT>")
			}

			if pre.Port == 0 {
				return errors.New("请指定 --port 端口号")
			}

			pg := pgsql.NewPgsql()
			return pg.AddSlave(sshOption, pre, master)
		},
	}
	cmd.Flags().StringVar(&sshOption.Host, "host", "", "新节点IP")
	cmd.Flags().IntVar(&sshOption.Port, "ssh-port", 22, "ssh 端口号")
	cmd.Flags().StringVar(&sshOption.Username, "ssh-username", "", "ssh 用户名")
	cmd.Flags().StringVar(&sshOption.Password, "ssh-password", "", "ssh 密码")
	cmd.Flags().StringVar(&sshOption.KeyFile, "ssh-keyfile", "", "ssh 密钥")
	cmd.Flags().StringVar(&pre.SystemUser, "system-user", config.DefaultPGAdminUser, "pgsql安装的操作系统用户")
	cmd.Flags().StringVar(&pre.SystemGroup, "system-group", config.DefaultPGAdminUser, "pgsql安装的操作系统用户组")
	cmd.Flags().StringVarP(&pre.Username, "username", "u", "", "要同步的主库上创建的主从同步用户")
	cmd.Flags().StringVarP(&pre.Password, "password", "p", "", "主从同步用户的密码")
	cmd.Flags().StringVarP(&pre.Dir, "dir", "d", "", "pgsql安装目录, 默认: /opt/pgsql$PORT")
	cmd.Flags().IntVarP(&pre.Port, "port", "P", 0, "pgsql 数据库监听端口, 默认: 5432")
	cmd.Flags().StringVarP(&master, "master", "m", "", "要同步的主库的<地址:端口>")
	cmd.Flags().StringVar(&pre.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&pre.Yes, "yes", "y", false, "是否确认安装")
	cmd.Flags().BoolVarP(&pre.NoRollback, "no-rollback", "n", false, "安装失败不回滚")

	return cmd
}

// dbup pgsql uninstall
func pgsqlUNInstallCmd() *cobra.Command {
	var yes bool
	var uninst services.UNInstall
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "pgsql 卸载",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if uninst.Port == 0 || uninst.BasePath == "" {
				return fmt.Errorf("必须手动指定 --port and --dir 参数")
			}

			if !yes {
				var s string
				logger.Successf("端口: %d\n", uninst.Port)
				logger.Successf("数据路径: %s\n", uninst.BasePath)
				logger.Successf("是否确认卸载[y|n]:")
				if _, err := fmt.Scanln(&s); err != nil {
					return err
				}
				if strings.ToUpper(s) != "Y" && strings.ToUpper(s) != "YES" {
					os.Exit(0)
				}
			}

			pg := pgsql.NewPgsql()
			return pg.UNInstall(&uninst)
		},
	}
	cmd.Flags().IntVarP(&uninst.Port, "port", "P", 0, "pgsql 数据库监听端口")
	cmd.Flags().StringVarP(&uninst.BasePath, "dir", "d", "", "pgsql安装目录")
	// cmd.Flags().BoolVar(&uninst.Repmgr, "repmgr", false, "是否为repmgr集群")
	// cmd.Flags().StringVarP(&uninst.RepmgrRole, "repmgr-role", "", "", "选择Repmgr角色 Primary|Standby")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "是否确认卸载")
	return cmd
}

// dbup pgsql backup
func pgsqlBackupCmd() *cobra.Command {
	var backup = services.NewBackup()
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "pgsql 进行备份",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := pgsql.NewPgsql()
			return pg.Backup(backup)
		},
	}
	cmd.Flags().StringVarP(&backup.Username, "username", "u", "pguser", "用户名")
	cmd.Flags().StringVarP(&backup.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&backup.Host, "host", "H", "127.0.0.1", "pgsql 地址")
	cmd.Flags().IntVarP(&backup.Port, "port", "P", 5432, "pgsql 数据库监听端口")
	cmd.Flags().StringVarP(&backup.BackupCmd, "command", "c", "pg_basebackup", "pgsql 备份命令")
	cmd.Flags().StringVarP(&backup.BackupDir, "backupdir", "d", "", "pgsql 备份目录")
	return cmd
}

// dbup pgsql backup-tables
func pgsqlBackupTablesCmd() *cobra.Command {
	var tables string
	var list string
	var backup = services.NewBackupTables()
	cmd := &cobra.Command{
		Use:   "backup-tables",
		Short: "pgsql 表备份",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := pgsql.NewPgsql()
			return pg.BackupTables(backup, tables, list)
		},
	}
	cmd.Flags().StringVarP(&backup.Username, "username", "u", "pguser", "用户名")
	cmd.Flags().StringVarP(&backup.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&backup.Host, "host", "H", "127.0.0.1", "pgsql 地址")
	cmd.Flags().IntVarP(&backup.Port, "port", "P", 5432, "pgsql 数据库监听端口")
	cmd.Flags().StringVarP(&backup.BackupCmd, "command", "c", "pg_dump", "pgsql 备份命令")
	cmd.Flags().StringVarP(&backup.BackupFile, "backup-file", "f", "", "pgsql 备份文件名全路径")
	cmd.Flags().StringVarP(&backup.Database, "database", "D", "", "pgsql 库, 必须指定")
	cmd.Flags().StringVarP(&tables, "tables", "T", "", "pgsql 要备份的表名, 用逗号分割, 如:  tbname1,tbname2,tbname3")
	cmd.Flags().StringVarP(&list, "list-file", "l", "", "pgsql 列表文件, 一行一个表名")
	cmd.Flags().StringVarP(&backup.Format, "format", "F", "c", "导出的文件格式, 默认为二进制格式")
	return cmd
}

// dbup pgsql cluster-deploy
func pgsqlDeployCmd() *cobra.Command {
	var config string
	cmd := &cobra.Command{
		Use:   "cluster-deploy",
		Short: "pgsql 主从部署",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			pg := pgsql.NewPgsql()
			return pg.Deploy(config)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	return cmd
}

// dbup pgsql cluster-deploy
func pgsqlRemoveDeployCmd() *cobra.Command {
	var yes bool
	var config string
	cmd := &cobra.Command{
		Use:   "remove-cluster-deploy",
		Short: "pgsql 集群删除",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			pg := pgsql.NewPgsql()
			return pg.RemoveDeploy(config, yes)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	return cmd
}

// dbup pgsql pgpool-install
func pgsqlPGPoolInstallCmd() *cobra.Command {
	var param config.PgPoolParameter
	var cfgFile string
	var onlyCheck bool
	cmd := &cobra.Command{
		Use:   "pgpool-install",
		Short: "pgpool 安装",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := pgsql.NewPgsql()
			return pg.PGPoolInstall(param, cfgFile, onlyCheck)
		},
	}
	cmd.Flags().IntVar(&param.Port, "port", 0, "pgpool监听端口, 默认: 9999")
	cmd.Flags().IntVar(&param.PcpPort, "pcp-port", 0, "pcp监听端口, 默认: 9898")
	cmd.Flags().IntVar(&param.WDPort, "wd-port", 0, "wd监听端口, 默认: 9000")
	cmd.Flags().IntVar(&param.HeartPort, "heart-port", 0, "heart监听端口, 默认: 9694")
	cmd.Flags().IntVar(&param.NodeID, "node-id", 0, "pgpool_node_id值")
	cmd.Flags().StringVar(&param.BindIP, "bind-ip", "", "pgpool监听地址")
	cmd.Flags().StringVar(&param.PcpBindIP, "pcp-bind-ip", "", "pcp监听地址")
	cmd.Flags().StringVar(&param.PGPoolIP, "pgpool-ip", "", "pgpool安装地址")
	cmd.Flags().StringVarP(&param.Dir, "dir", "d", "", "pgpool安装目录, 默认: /opt/pgpool$PORT")
	cmd.Flags().StringVarP(&param.Username, "username", "u", "", "用户名, 2到63位小写字母,下划线,数字组成; 首位不能是数字")
	cmd.Flags().StringVarP(&param.Password, "password", "p", "", "密码")
	cmd.Flags().StringVar(&param.Address, "address", "", "pgpool 数据库授权IP列表")
	cmd.Flags().StringVar(&param.PGMaster, "pg-master", "", "pgsql主库IP")
	cmd.Flags().StringVar(&param.PGSlave, "pg-slave", "", "pgsql从库IP")
	cmd.Flags().IntVar(&param.PGPort, "pg-port", 0, "pgsql 数据库监听端口, 默认: 5432")
	cmd.Flags().StringVar(&param.PGDir, "pg-dir", "", "pgsql 数据目录")
	cmd.Flags().StringVar(&param.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&param.Yes, "yes", "y", false, "是否确认安装")
	cmd.Flags().BoolVarP(&param.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	cmd.Flags().BoolVar(&onlyCheck, "only-check", false, "只检查配置和环境, 不进行实际安装操作")
	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "安装配置文件, 默认不使用配置文件")
	return cmd
}

// dbup pgsql uninstall
func pgpoolUNInstallCmd() *cobra.Command {
	var yes bool
	var uninst services.PGPoolUNInstall
	cmd := &cobra.Command{
		Use:   "pgpool-uninstall",
		Short: "pgsql 卸载",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if uninst.Port == 0 || uninst.BasePath == "" {
				return fmt.Errorf("必须手动指定 --port and --dir 参数")
			}

			if !yes {
				var s string
				logger.Successf("端口: %d\n", uninst.Port)
				logger.Successf("数据路径: %s\n", uninst.BasePath)
				logger.Successf("是否确认卸载[y|n]:")
				if _, err := fmt.Scanln(&s); err != nil {
					return err
				}
				if strings.ToUpper(s) != "Y" && strings.ToUpper(s) != "YES" {
					os.Exit(0)
				}
			}

			pg := pgsql.NewPgsql()
			return pg.PGPoolUNInstall(&uninst)
		},
	}
	cmd.Flags().IntVarP(&uninst.Port, "port", "P", 0, "pgsql 数据库监听端口")
	cmd.Flags().StringVarP(&uninst.BasePath, "dir", "d", "", "pgsql安装目录")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "是否确认卸载")
	return cmd
}

// dbup pgsql cluster-deploy
func pgsqlPGPoolClusterDeployCmd() *cobra.Command {
	var config string
	cmd := &cobra.Command{
		Use:   "pgpool-cluster-deploy",
		Short: "pgsql 的pgpool 集群部署",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件\n")
			}
			pg := pgsql.NewPgsql()
			return pg.PGPoolClusterDeploy(config)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	return cmd
}

// dbup pgsql cluster-deploy
func pgsqlRemovePGPoolClusterDeployCmd() *cobra.Command {
	var yes bool
	var config string
	cmd := &cobra.Command{
		Use:   "remove-pgpool-cluster-deploy",
		Short: "pgsql 的pgpool 集群删除",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件\n")
			}
			pg := pgsql.NewPgsql()
			return pg.RemovePGPoolClusterDeploy(config, yes)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	return cmd
}

// dbup pgsql promote
func pgsqlPromoteCmd() *cobra.Command {
	var pgm = services.NewPGManager()
	var wait string
	var waitseconds int
	cmd := &cobra.Command{
		Use:   "promote",
		Short: "pgsql 从库提升为主库",
		RunE: func(cmd *cobra.Command, args []string) error {
			if wait != "true" && wait != "false" {
				return errors.New("wait 只能是 true 和 false")
			}
			return pgm.Promote(wait, waitseconds)
		},
	}
	cmd.Flags().StringVarP(&pgm.AdminUser, "username", "u", "postgres", "用户名")
	cmd.Flags().StringVarP(&pgm.AdminPassword, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&pgm.AdminDatabase, "database", "d", "postgres", "库名")
	cmd.Flags().StringVarP(&pgm.Host, "host", "H", "127.0.0.1", "pgsql 地址")
	cmd.Flags().IntVarP(&pgm.Port, "port", "P", 5432, "pgsql 数据库监听端口")
	cmd.Flags().StringVarP(&wait, "wait", "w", "true", "是否等待操作成功 true|false")
	cmd.Flags().IntVarP(&waitseconds, "wait-seconds", "s", 60, "等待多少秒")
	return cmd
}

// dbup pgsql upgrade
func PGsqlUPgradeCmd() *cobra.Command {
	var upgrade = services.NewUPgrade()
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "pgsql 升级",
		RunE: func(cmd *cobra.Command, args []string) error {
			return upgrade.Run()
		},
	}

	cmd.Flags().StringVarP(&upgrade.Dir, "dir", "d", "", "pgsql 安装主目录")
	cmd.Flags().IntVarP(&upgrade.Port, "port", "P", 0, "pgsql 端口")
	cmd.Flags().BoolVarP(&upgrade.Yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	// cmd.Flags().StringVarP(&upgrade.EmoloyDir, "new", "n", "/tmp", "新版本pgsql临时解压目录")

	// cmd.Flags().StringVarP(&upgrade.Password, "password", "p", "", "旧版本实例密码")
	// cmd.Flags().StringVarP(&upgrade.Host, "host", "H", "127.0.0.1", "mariadb 旧版本实例本地地址")
	// cmd.Flags().IntVarP(&upgrade.Port, "port", "P", 3306, "mariadb 旧版本实例端口")
	// cmd.Flags().StringVarP(&upgrade.Username, "username", "u", "", "旧版本实例管理用户名")
	// cmd.Flags().StringVarP(&upgrade.SourceDir, "old", "o", "", "旧版本 mariadb 安装主目录")
	// cmd.Flags().StringVarP(&upgrade.EmoloyDir, "new", "n", "/tmp", "新版本 mariadb 临时解压目录")
	// cmd.Flags().StringVarP(&upgrade.TxIsolation, "transaction_isolation", "i", "RC", "事务隔离级别(可选择 RR 或 RC)")
	// cmd.Flags().BoolVarP(&upgrade.NoBackup, "nobak", "", false, "不进行基础文件备份")
	// cmd.Flags().BoolVarP(&upgrade.Yes, "yes", "y", false, "直接安装, 否则需要交互确认")

	return cmd
}
