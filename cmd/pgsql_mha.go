/*
@Author : WuWeiJian
@Date : 2020-12-02 17:42
*/

package cmd

import (
	"dbup/internal/environment"
	"dbup/internal/pgsql"
	"dbup/internal/pgsql/config"
	"dbup/internal/pgsql/services"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func pgsqlMHACmd() *cobra.Command {
	// 定义二级命令: pgsql
	var cmd = &cobra.Command{
		Use:   "pgsql-mha",
		Short: "pgsql 高可用集群相关操作",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	// 装载命令
	cmd.AddCommand(
		pgsqlMHADeployCmd(),
		pgsqlMHARemoveDeployCmd(),
		PgAutoFailoverMonitorAdd(),
		PgAutoFailoverPGdataAdd(),
		PgAutoFailoverUNInstallCmd(),
		pgsqlMHAUserAddCmd(),
		pgsqlMHAUserGrantCmd(),
	)

	return cmd
}

// dbup pg_auto_failover create monitor
func PgAutoFailoverMonitorAdd() *cobra.Command {
	var m config.PGAutoFailoverMonitor
	var onlyCheck bool
	cmd := &cobra.Command{
		Use:   "MonitorCreate",
		Short: "PgAutoFailover Monitor 监控节点创建",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			pg := pgsql.NewPgsql()

			return pg.MonitorInstall(m, onlyCheck)
		},
	}
	cmd.Flags().StringVar(&m.SystemUser, "system-user", config.DefaultPGAdminUser, "PG Monitor 安装的操作系统用户")
	cmd.Flags().StringVar(&m.SystemGroup, "system-group", config.DefaultPGAdminUser, "PG Monitor 安装的操作系统用户组")
	cmd.Flags().StringVarP(&m.Host, "host", "H", "", "指定 Monitor 监控节点的IP地址或主机名")
	cmd.Flags().IntVarP(&m.Port, "port", "P", 0, "指定 Monitor 监控节点的端口")
	cmd.Flags().StringVarP(&m.Dir, "dir", "d", "", "指定 Monitor 的安装主目录")
	// cmd.Flags().StringVar(&m.AdminPassword, "admin-password", "", fmt.Sprintf("请用 --admin-password 参数指定 超级管理员(postgres) 的密码, 建议使用: %s", utils.GeneratePasswd(16)))
	cmd.Flags().BoolVarP(&m.Yes, "yes", "y", false, "是否确认安装")
	cmd.Flags().BoolVarP(&m.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	cmd.Flags().BoolVar(&onlyCheck, "only-check", false, "只检查配置和环境, 不进行实际安装操作")
	return cmd

}

// dbup pg_auto_failover create pgdata
func PgAutoFailoverPGdataAdd() *cobra.Command {
	var d config.PGAutoFailoverPGNode
	var cfgFile string
	var onlyCheck bool
	var onlyflushpass bool
	cmd := &cobra.Command{
		Use:   "PGdataCreate",
		Short: "PgAutoFailover PGdata 数据节点创建",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			pg := pgsql.NewPgsql()

			return pg.PGdataInstall(d, onlyCheck, onlyflushpass, cfgFile)
		},
	}
	cmd.Flags().StringVar(&d.SystemUser, "system-user", config.DefaultPGAdminUser, "PGdata 安装的操作系统用户")
	cmd.Flags().StringVar(&d.SystemGroup, "system-group", config.DefaultPGAdminUser, "PGdata 安装的操作系统用户组")
	cmd.Flags().StringVar(&d.Mhost, "monitor-host", "", "指定 Monitor 监控节点的IP地址或域名")
	cmd.Flags().IntVarP(&d.Mport, "monitor-port", "", 0, "指定 Monitor 监控节点的端口")
	cmd.Flags().StringVar(&d.AllNode, "allnode", "", "指定所有数据节点进行弱密码加密配置 例: host:port,host:port")
	cmd.Flags().StringVar(&d.AdminPassword, "admin-password", "", fmt.Sprintf("请用 --admin-password 参数指定 超级管理员(postgres) 的密码, 建议使用: %s", utils.GeneratePasswd(16)))
	cmd.Flags().StringVar(&d.AdminPasswordExpireAt, "admin-password-expire-at", "", "超级管理员(postgres) 密码的过期时间")
	cmd.Flags().StringVarP(&d.Username, "username", "u", "", "用户名, 2到63位小写字母,下划线,数字组成; 首位不能是数字")
	cmd.Flags().StringVarP(&d.Password, "password", "p", "", "密码, 默认随机生成16位密码")
	cmd.Flags().StringVarP(&d.Dir, "dir", "d", "", "pgsql安装目录, 默认: /opt/pgsql$PORT")
	cmd.Flags().StringVarP(&d.MemorySize, "memory-size", "m", "", "pgsql 数据库内存大小, 默认操作系统最大内存的50%")
	cmd.Flags().StringVarP(&d.Host, "host", "H", "", "指定本机的IP或者域名")
	cmd.Flags().IntVarP(&d.Port, "port", "P", 0, "pgsql 数据库监听端口, 默认: 5432")
	cmd.Flags().StringVar(&d.AdminAddress, "admin-address", "", "pgsql 数据库授权IP列表, 默认只能本地登录")
	cmd.Flags().StringVarP(&d.Address, "address", "a", "", "pgsql 数据库授权IP列表, 默认 0.0.0.0/0")
	cmd.Flags().StringVarP(&d.BindIP, "bind-ip", "b", "", "pgsql 数据库监听地址, 默认: *")
	cmd.Flags().StringVar(&d.Libraries, "libraries", "", "pgsql启用的插件, 目前只支持 [timescaledb]")
	cmd.Flags().StringVar(&d.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "安装配置文件, 默认不使用配置文件")
	cmd.Flags().BoolVarP(&d.Onenode, "onenode", "o", false, "是否为安装的第一个节点,第一个数据节点必须指定此参数")
	cmd.Flags().BoolVarP(&d.Yes, "yes", "y", false, "是否确认安装")
	cmd.Flags().BoolVarP(&d.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	cmd.Flags().BoolVar(&onlyCheck, "only-check", false, "只检查配置和环境, 不进行实际安装操作")
	cmd.Flags().BoolVar(&onlyflushpass, "only-flushpass", false, "只在主机配置 pgpass 免密认证,不安装")
	return cmd

}

// dbup pgsql uninstall
func PgAutoFailoverUNInstallCmd() *cobra.Command {
	var yes bool
	var uninst services.UNInstall
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "PgAutoFailover 单点卸载",
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
			return pg.PGautoUNInstall(&uninst)
		},
	}
	cmd.Flags().StringVar(&uninst.SystemUser, "system-user", config.DefaultPGAdminUser, "pgautofailover 安装的操作系统用户")
	cmd.Flags().IntVarP(&uninst.Port, "port", "P", 0, "pgsql 数据库监听端口")
	cmd.Flags().StringVarP(&uninst.BasePath, "dir", "d", "", "pgsql安装目录")
	cmd.Flags().StringVarP(&uninst.AutopgRole, "pgautofailover-roles", "r", "", "monitor|pgdata")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "是否确认卸载")
	return cmd
}

// dbup pgsql cluster-deploy
func pgsqlMHADeployCmd() *cobra.Command {
	var config string
	var NoRollback bool
	var yes bool
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "PgAutoFailover HA 集群部署",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			pg := pgsql.NewPgsql()
			return pg.MHADeploy(config, NoRollback, yes)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	cmd.Flags().BoolVarP(&NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "交互确认,可以在新增从节点时使用")
	return cmd
}

// dbup pgsql cluster-deploy
func pgsqlMHARemoveDeployCmd() *cobra.Command {
	var yes bool
	var config string
	cmd := &cobra.Command{
		Use:   "remove-deploy",
		Short: "PgAutoFailover HA 集群删除",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			pg := pgsql.NewPgsql()
			return pg.MHARemoveDeploy(config, yes)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "直接卸载, 否则需要交互确认")
	return cmd
}

// dbup pgdata user add
func pgsqlMHAUserAddCmd() *cobra.Command {
	var m = services.NewPGManager()
	cmd := &cobra.Command{
		Use:   "create-user",
		Short: "PgAutoFailover pgdata 添加用户",
		RunE: func(cmd *cobra.Command, args []string) error {
			if m.User == "" || m.Password == "" {
				return fmt.Errorf("请指定要创建的用户名和密码")
			}

			if err := m.CheckUserChar(); err != nil {
				return err
			}

			pg := pgsql.NewPgsql()
			return pg.UserCreate(m)
		},
	}
	cmd.Flags().StringVarP(&m.Host, "host", "H", config.DefaultPGSocketPath, "pgsql 地址")
	cmd.Flags().IntVarP(&m.Port, "port", "P", 5432, "pgsql 端口")
	cmd.Flags().StringVarP(&m.AdminUser, "admin-user", "u", config.DefaultPGAdminUser, "管理员用户")
	cmd.Flags().StringVarP(&m.AdminPassword, "admin-password", "p", "", "管理员密码")
	cmd.Flags().StringVarP(&m.AdminDatabase, "admin-database", "d", "", "管理员登录库, 默认与用户名同名")
	cmd.Flags().StringVar(&m.User, "user", "", "要创建的用户")
	cmd.Flags().StringVar(&m.Password, "password", "", "要创建的用户密码")
	cmd.Flags().StringVar(&m.Role, "role", "normal", "要创建的用户角色<'dbuser'|'normal'|'replication'|'admin'>")
	cmd.Flags().BoolVar(&m.Ignore, "ignore", false, "用户已经存在则忽略")
	return cmd
}

// dbup pgsql user grant
func pgsqlMHAUserGrantCmd() *cobra.Command {
	var m = services.NewPGManager()
	cmd := &cobra.Command{
		Use:   "grant-user",
		Short: "PgAutoFailover pgdata 用户授权",
		RunE: func(cmd *cobra.Command, args []string) error {
			if m.User == "" || m.DBName == "" || m.Address == "" {
				return fmt.Errorf("请指定要授权的用户名,库名,IP地址")
			}

			if err := m.CheckDBChar(); err != nil {
				return err
			}

			pg := pgsql.NewPgsql()
			return pg.UserGrantPGdata(m)
		},
	}
	cmd.Flags().StringVarP(&m.Host, "host", "H", config.DefaultPGSocketPath, "pgsql 地址")
	cmd.Flags().IntVarP(&m.Port, "port", "P", 5432, "pgsql 端口")
	cmd.Flags().StringVarP(&m.AdminUser, "admin-user", "u", config.DefaultPGAdminUser, "管理员用户")
	cmd.Flags().StringVarP(&m.AdminPassword, "admin-password", "p", "", "管理员密码")
	cmd.Flags().StringVarP(&m.AdminDatabase, "admin-database", "d", "", "管理员登录库")
	cmd.Flags().StringVar(&m.User, "user", "", "要授权的用户")
	cmd.Flags().StringVar(&m.DBName, "dbname", "", "要授权用户的登录库")
	cmd.Flags().StringVarP(&m.Address, "address", "a", "", "要授权用户的授权IP列表")
	return cmd
}
