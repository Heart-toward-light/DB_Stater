// Created by LiuSainan on 2022-06-09 15:46:48

package cmd

import (
	"dbup/internal/environment"
	"dbup/internal/mariadb/config"
	"dbup/internal/mariadb/service"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func mariadbCmd() *cobra.Command {
	// 定义二级命令: pgsql
	var cmd = &cobra.Command{
		Use:   "mariadb",
		Short: "mariadb相关操作",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	// 装载命令
	cmd.AddCommand(
		mariadbInstallCmd(),
		mariadbUNInstallCmd(),
		mariadbDeployCmd(),
		mariadbRemoveDeployCmd(),
		mariadbBackupCmd(),
		mariadbAddSlaveCmd(),
		mariadbGaleraDeployCmd(),
		MariadbUPgradeCmd(),
		Galera_startOnenode(),
	)

	return cmd
}

// dbup mariadb install
func mariadbInstallCmd() *cobra.Command {
	var option config.MariaDBOptions
	var onlyCheck bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "mariadb 单机版安装",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			option.Parameter()
			if err := option.Validator(); err != nil {
				return err
			}
			if err := option.Environment(); err != nil {
				return err
			}

			if onlyCheck {
				return nil
			}

			rs := service.NewMariaDBInstall(&option)
			return rs.Run()
		},
	}
	cmd.Flags().StringVar(&option.SystemUser, "system-user", config.DefaultMariaDBSystemUser, "mariadb安装的操作系统用户")
	cmd.Flags().StringVar(&option.SystemGroup, "system-group", config.DefaultMariaDBSystemGroup, "mariadb安装的操作系统用户组")
	cmd.Flags().StringVar(&option.Repluser, "repluser", "", "指定 mariadb 主从复制用户,初始化单实例无需指定")
	cmd.Flags().StringVar(&option.ReplPassword, "replpassword", "", fmt.Sprintf("指定 mariadb 主从复制用户密码, 建议使用: %s, 初始化单实例无需指定", utils.GeneratePasswd(16)))
	cmd.Flags().IntVarP(&option.Port, "port", "P", 0, "mariadb 数据库监听端口")
	cmd.Flags().StringVarP(&option.Dir, "dir", "d", "", "mariadb安装目录, 默认: /opt/mariadb$PORT")
	cmd.Flags().StringVarP(&option.Password, "password", "p", "", fmt.Sprintf("指定 mariadb root用户密码, 建议使用: %s", utils.GeneratePasswd(16)))
	cmd.Flags().StringVarP(&option.Memory, "memory", "m", "128M", "内存")
	cmd.Flags().StringVarP(&option.Join, "join", "j", "", "从库同步主库的主库地址<IP:PORT>, 单实例无需指定")
	cmd.Flags().StringVarP(&option.OwnerIP, "owner-ip", "o", "", "当机器上有多个IP时, 指定以哪个IP创建实例")
	cmd.Flags().StringVarP(&option.TxIsolation, "transaction_isolation", "i", "RC", "事务隔离级别(可选择 RR 或 RC)")
	cmd.Flags().StringVar(&option.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().StringVar(&option.Backupuser, "bakuser", "", "指定备份数据的用户名，添加从库时使用")
	cmd.Flags().StringVar(&option.BackupPassword, "bakpassword", "", "指定备份数据的用户密码，添加从库时使用")
	cmd.Flags().StringVar(&option.Wsrepclusteraddress, "cluster_address", "", "galera 集群所有节点ip, 例: ip1,ip2,ip3")
	cmd.Flags().BoolVarP(&option.Galera, "galera", "", false, "安装的实例是否为 galera 实例, 默认为否")
	cmd.Flags().BoolVarP(&option.Onenode, "onenode", "", false, "安装的实例是否为 galera 第一个实例, 默认为否")
	cmd.Flags().BoolVarP(&option.BackupData, "Backupdata", "b", false, "是否开启备份来添加从库, 默认关闭,单实例无需指定")
	cmd.Flags().BoolVarP(&option.AddSlave, "add-slave", "a", false, "初始化集群时使用参数,单实例无需指定")
	cmd.Flags().IntVar(&option.AutoIncrement, "autoincrement", 1, "配置双主节点同步自增步长策略(2/3),默认1")
	cmd.Flags().BoolVarP(&option.Yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	cmd.Flags().BoolVarP(&option.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	cmd.Flags().BoolVar(&onlyCheck, "only-check", false, "只检查配置和环境, 不进行实际安装操作")
	return cmd
}

// dbup mariadb uninstall
func mariadbUNInstallCmd() *cobra.Command {
	var yes bool
	var uninst service.MariaDBUNInstall
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "mariadb 单机版卸载",
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
			return uninst.Uninstall()
		},
	}
	cmd.Flags().IntVarP(&uninst.Port, "port", "P", 0, "MariaDB 数据库监听端口")
	cmd.Flags().StringVarP(&uninst.BasePath, "dir", "d", "", "MariaDB 安装目录")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "是否确认卸载")
	return cmd
}

// dbup mariadb replication-deploy
func mariadbDeployCmd() *cobra.Command {
	var config string
	cmd := &cobra.Command{
		Use:   "replication-deploy",
		Short: "mariadb 主从或主主集群部署",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			rs := service.NewmariadbDeploy()
			return rs.Run(config)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	return cmd
}

// dbup mariadb galera-deploy
func mariadbGaleraDeployCmd() *cobra.Command {
	var config string
	cmd := &cobra.Command{
		Use:   "galera",
		Short: "mariadb Galera 集群部署",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			rs := service.NewGaleraDeploy()
			return rs.Run(config)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	return cmd
}

// dbup mariadb replication-deploy
func mariadbRemoveDeployCmd() *cobra.Command {
	var yes bool
	var config string
	cmd := &cobra.Command{
		Use:   "remove-replication-deploy",
		Short: "mariadb 移除集群部署(主从/galera)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			rs := service.NewmariadbDeploy()
			return rs.RemoveCluster(config, yes)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	return cmd
}

func mariadbAddSlaveCmd() *cobra.Command {
	var sshOption config.Server
	var option config.MariaDBOptions
	cmd := &cobra.Command{
		Use:   "add-slave",
		Short: "mariadb 添加从库",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sshOption.Address == "" {
				return fmt.Errorf("请指定要创建从库的远程机器IP地址")
			}

			if option.Port == 0 {
				return fmt.Errorf("请指定要创建从库的端口号")
			}

			if option.Join == "" {
				return fmt.Errorf("请指定要加入的主库实例信息:<IP:PORT>")
			}

			if option.Password == "" {
				return fmt.Errorf("请指定 Mariadb 的超级管理员 root 用户密码")
			}

			if option.Repluser == "" {
				return fmt.Errorf("请指定 Mariadb 复制用户名")
			}

			if option.ReplPassword == "" {
				return fmt.Errorf("请指定 Mariadb 复制用户密码")
			}

			if option.Backupuser == "" {
				return fmt.Errorf("请指定 Mariadb 备份数据的用户名")
			}

			if option.BackupPassword == "" {
				return fmt.Errorf("请指定 Mariadb 备份数据的用户密码")
			}

			mm := service.NewMariaDBManager()
			return mm.AddSlaveNode(sshOption, option)
		},
	}

	cmd.Flags().IntVar(&sshOption.SshPort, "ssh-port", 22, "ssh 端口号")
	cmd.Flags().StringVar(&sshOption.User, "ssh-username", "", "ssh 用户名")
	cmd.Flags().StringVar(&sshOption.Password, "ssh-password", "", "ssh 密码")
	cmd.Flags().StringVar(&sshOption.KeyFile, "ssh-keyfile", "", "ssh 密钥")
	cmd.Flags().StringVar(&option.SystemUser, "system-user", config.DefaultMariaDBSystemUser, "mariadb安装的操作系统用户")
	cmd.Flags().StringVar(&option.SystemGroup, "system-group", config.DefaultMariaDBSystemGroup, "mariadb 安装的操作系统用户组")
	cmd.Flags().StringVarP(&sshOption.Address, "host", "H", "", "新增从节点IP地址, 必填项")
	cmd.Flags().IntVarP(&option.Port, "port", "P", 0, "新增从节点端口, 必填项")
	cmd.Flags().StringVarP(&option.Dir, "dir", "d", "", "mariadb 安装目录, 必填项")
	cmd.Flags().StringVarP(&option.Password, "password", "p", "", "root 密码")
	cmd.Flags().StringVar(&option.Repluser, "repluser", "", "指定 mariadb 主从复制用户")
	cmd.Flags().StringVar(&option.ReplPassword, "replpassword", "", "指定 mariadb 主从复制用户密码")
	cmd.Flags().StringVar(&option.Backupuser, "bakuser", "", "指定备份数据的用户名")
	cmd.Flags().StringVar(&option.BackupPassword, "bakpassword", "", "指定备份数据的用户密码")
	cmd.Flags().StringVarP(&option.Memory, "memory", "m", "1G", "内存")
	cmd.Flags().StringVarP(&option.Join, "join", "j", "", "从库同步主库的主库地址<IP:PORT>")
	cmd.Flags().StringVar(&option.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&option.Yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	return cmd
}

// dbup mariadb backup
func mariadbBackupCmd() *cobra.Command {
	var backup = service.NewBackup()
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "mariadb 进行备份",
		RunE: func(cmd *cobra.Command, args []string) error {
			return backup.Run()
		},
	}
	cmd.Flags().StringVarP(&backup.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&backup.Host, "host", "H", "", "mariadb 地址")
	cmd.Flags().IntVarP(&backup.Port, "port", "P", 3306, "mariadb 数据库监听端口")
	cmd.Flags().StringVarP(&backup.Username, "username", "u", "", "用户名")
	cmd.Flags().StringVarP(&backup.BackupCmd, "command", "c", "mariadb-dump", "mariadb 备份命令")
	cmd.Flags().StringVarP(&backup.BackupFile, "backupfile", "f", "", "mariadb 备份目录")
	return cmd
}

// dbup galera start Onenode
func Galera_startOnenode() *cobra.Command {
	var galera = service.NewGaleraNode()
	cmd := &cobra.Command{
		Use:   "galera-start-onenode",
		Short: "galera 启动集群的第一个节点",
		Long:  `1、如果是新集群,可随意指定其中一个节点作为第一个节点启动  2、如果是老集群,需要指定最后一个停止的节点为启动的第一个节点`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if galera.Port == 0 || galera.BasePath == "" {
				return fmt.Errorf("必须手动指定 --port and --dir 参数")
			}

			return galera.Start_Onenode()
		},
	}

	cmd.Flags().IntVarP(&galera.Port, "port", "P", 0, "Galera 集群第一个节点的端口")
	cmd.Flags().StringVarP(&galera.BasePath, "dir", "d", "", "Galera 集群第一个节点的主目录")
	return cmd
}

// dbup mariadb upgrade
func MariadbUPgradeCmd() *cobra.Command {
	var upgrade = service.NewUPgrade()
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "mariadb 升级",
		RunE: func(cmd *cobra.Command, args []string) error {
			return upgrade.Run()
		},
	}
	cmd.Flags().StringVarP(&upgrade.Password, "password", "p", "", "旧版本实例密码")
	cmd.Flags().StringVarP(&upgrade.Host, "host", "H", "127.0.0.1", "mariadb 旧版本实例本地地址")
	cmd.Flags().IntVarP(&upgrade.Port, "port", "P", 3306, "mariadb 旧版本实例端口")
	cmd.Flags().StringVarP(&upgrade.Username, "username", "u", "", "旧版本实例管理用户名")
	cmd.Flags().StringVarP(&upgrade.SourceDir, "old", "o", "", "旧版本 mariadb 安装主目录")
	cmd.Flags().StringVarP(&upgrade.EmoloyDir, "new", "n", "/tmp", "新版本 mariadb 临时解压目录")
	cmd.Flags().StringVarP(&upgrade.TxIsolation, "transaction_isolation", "i", "RC", "事务隔离级别(可选择 RR 或 RC)")
	cmd.Flags().BoolVarP(&upgrade.NoBackup, "nobak", "", false, "不进行基础文件备份")
	cmd.Flags().BoolVarP(&upgrade.Yes, "yes", "y", false, "直接安装, 否则需要交互确认")

	return cmd
}
