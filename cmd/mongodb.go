/*
@Author : WuWeiJian
@Date : 2021-05-10 16:13
*/

package cmd

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/mongodb/config"
	"dbup/internal/mongodb/service"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator"
	"github.com/spf13/cobra"
)

func mongodbCmd() *cobra.Command {
	// 定义二级命令: pgsql
	var cmd = &cobra.Command{
		Use:   "mongodb",
		Short: "mongodb相关操作",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	// 装载命令
	cmd.AddCommand(
		mongodbInstallCmd(),
		mongodbAddSlaveCmd(),
		mongodbUNInstallCmd(),
		mongodbDeployCmd(),
		mongodbRemoveDeployCmd(),
		mongodbBackupCmd(),
		mongodbClusterDeployCmd(),
		mongoSinstallCmd(),
		mongoSUNInstallCmd(),
		mongodbRemoveClusterCmd(),
	)

	return cmd
}

// dbup mongodb install
func mongodbInstallCmd() *cobra.Command {
	var option config.MongodbOptions
	var onlyCheck bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "mongodb 单机版安装",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if option.Username == "" || option.Password == "" {
				return fmt.Errorf("请指定 MongoDB 的超级管理员用户名和密码")
			}
			if err := utils.CheckPasswordLever(option.Password); err != nil {
				return err
			}
			if err := option.CheckSpecialChar(); err != nil {
				return err
			}

			option.InitArgs()
			validate := validator.New()
			if err := validate.RegisterValidation("ipPort", config.ValidateIPPort); err != nil {
				return err
			}
			if err := validate.Struct(option); err != nil {
				return err
			}

			rs := service.NewMongoDBInstall(&option)
			if err := rs.CheckEnv(); err != nil {
				return err
			}

			if onlyCheck {
				return nil
			}

			return rs.Run()
		},
	}
	cmd.Flags().StringVar(&option.SystemUser, "system-user", config.DefaultMongoDBSystemUser, "mongodb安装的操作系统用户")
	cmd.Flags().StringVar(&option.SystemGroup, "system-group", config.DefaultMongoDBSystemGroup, "mongodb安装的操作系统用户组")
	cmd.Flags().IntVarP(&option.Port, "port", "P", 0, "mongodb 数据库监听端口")
	cmd.Flags().StringVarP(&option.Dir, "dir", "d", "", "mongodb安装目录, 默认: /opt/mongodb$PORT")
	cmd.Flags().StringVarP(&option.Username, "username", "u", "", "mongodb用户名")
	cmd.Flags().StringVarP(&option.Password, "password", "p", "", fmt.Sprintf("指定 MongoDB 用户密码, 建议使用: %s", utils.GeneratePasswd(16)))
	cmd.Flags().IntVarP(&option.Memory, "memory", "m", 1, "内存")
	cmd.Flags().StringVarP(&option.ReplSetName, "replSetName", "r", "", "副本集名")
	cmd.Flags().BoolVar(&option.Arbiter, "arbiter", false, "是否为仲裁节点")
	cmd.Flags().BoolVar(&option.Ipv6, "ipv6", false, "是否开启IPV6功能,默认不开启")
	cmd.Flags().StringVarP(&option.BindIP, "bind-ip", "b", "", "mongodb 数据库监听地址")
	cmd.Flags().StringVarP(&option.Join, "join", "j", "", "做为从库, 要加入的副本集群的任意一个节点<IP:PORT>")
	cmd.Flags().StringVarP(&option.Owner, "owner", "o", "", "当机器上有多个IP时, 指定以哪个IP创建实例")
	cmd.Flags().StringVar(&option.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&option.Yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	cmd.Flags().BoolVarP(&option.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	cmd.Flags().BoolVar(&onlyCheck, "only-check", false, "只检查配置和环境, 不进行实际安装操作")
	return cmd
}

// dbup mongos install
func mongoSinstallCmd() *cobra.Command {
	var option config.MongosOptions
	var onlyCheck bool
	cmd := &cobra.Command{
		Use:   "msinstall",
		Short: "mongos  单机版安装",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if option.Username == "" || option.Password == "" {
				return fmt.Errorf("请指定 MongoDB 已经授权的超级管理员用户名和密码")
			}

			option.InitArgs()

			if err := option.CheckConfigDB(); err != nil {
				return err
			}
			validate := validator.New()
			if err := validate.RegisterValidation("ipPort", config.ValidateIPPort); err != nil {
				return err
			}
			if err := validate.Struct(option); err != nil {
				return err
			}

			rs := service.NewMongoSInstall(&option)
			if err := rs.CheckEnv(); err != nil {
				return err
			}

			if onlyCheck {
				return nil
			}

			return rs.Run()
		},
	}

	cmd.Flags().StringVar(&option.SystemUser, "system-user", config.DefaultMongoDBSystemUser, "mongos安装的操作系统用户")
	cmd.Flags().StringVar(&option.SystemGroup, "system-group", config.DefaultMongoDBSystemGroup, "mongos安装的操作系统用户组")
	cmd.Flags().IntVarP(&option.Port, "port", "P", 0, "mongos 数据库监听端口")
	cmd.Flags().StringVarP(&option.ConfigDB, "ConfigDB", "C", "", "需要指定Config集群地址: 示例: 副本集名称/IP:PORT,IP:PORT,IP:PORT ")
	cmd.Flags().StringVarP(&option.Dir, "dir", "d", "", "mongos安装目录, 默认: /opt/mongodb$PORT")
	cmd.Flags().StringVarP(&option.Username, "username", "u", "", "mongodb 用户名")
	cmd.Flags().StringVarP(&option.Password, "password", "p", "", "Mongodb 用户密码")
	cmd.Flags().BoolVar(&option.Ipv6, "ipv6", false, "是否开启IPV6功能,默认不开启")
	cmd.Flags().StringVarP(&option.BindIP, "bind-ip", "b", "", "mongodb 数据库监听地址")
	cmd.Flags().StringVarP(&option.Owner, "owner", "o", "", "当机器上有多个IP时, 指定以哪个IP创建实例")
	cmd.Flags().StringVar(&option.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&option.Yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	cmd.Flags().BoolVarP(&option.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	cmd.Flags().BoolVar(&onlyCheck, "only-check", false, "只检查配置和环境, 不进行实际安装操作")
	return cmd
}

// dbup mongodb uninstall
func mongoSUNInstallCmd() *cobra.Command {
	var yes bool
	var uninst service.MongoSUNInstall
	cmd := &cobra.Command{
		Use:   "unmsinstall",
		Short: "mongos  单机版卸载",
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
			return uninst.MSuninstall()
		},
	}
	cmd.Flags().IntVarP(&uninst.Port, "port", "P", 0, "MongoS 数据库监听端口")
	cmd.Flags().StringVarP(&uninst.BasePath, "dir", "d", "", "MongoS 安装目录")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "是否确认卸载")
	return cmd
}

// dbup mongodb install
func mongodbAddSlaveCmd() *cobra.Command {
	var sshOption global.SSHConfig
	var option config.MongodbOptions
	var onlyCheck bool
	cmd := &cobra.Command{
		Use:   "add-slave",
		Short: "mongodb 添加从库",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sshOption.Host == "" {
				return fmt.Errorf("请指定要创建从库的远程机器IP地址")
			}

			if option.Port == 0 {
				return fmt.Errorf("请指定要创建从库的端口号")
			}

			if option.Join == "" {
				return fmt.Errorf("请指定要加入的主库实例信息:<IP:PORT>")
			}

			if option.Username == "" || option.Password == "" {
				return fmt.Errorf("请指定 MongoDB 的超级管理员用户名和密码")
			}

			option.InitArgs()
			validate := validator.New()
			if err := validate.RegisterValidation("ipPort", config.ValidateIPPort); err != nil {
				return err
			}
			if err := validate.Struct(option); err != nil {
				return err
			}

			mm := service.NewMongoDBManager()
			return mm.AddSlaveNode(sshOption, option)
		},
	}
	cmd.Flags().StringVar(&sshOption.Host, "host", "", "新节点IP")
	cmd.Flags().IntVar(&sshOption.Port, "ssh-port", 22, "ssh 端口号")
	cmd.Flags().StringVar(&sshOption.Username, "ssh-username", "", "ssh 用户名")
	cmd.Flags().StringVar(&sshOption.Password, "ssh-password", "", "ssh 密码")
	cmd.Flags().StringVar(&sshOption.KeyFile, "ssh-keyfile", "", "ssh 密钥")
	cmd.Flags().StringVar(&option.SystemUser, "system-user", config.DefaultMongoDBSystemUser, "mongodb安装的操作系统用户")
	cmd.Flags().StringVar(&option.SystemGroup, "system-group", config.DefaultMongoDBSystemGroup, "mongodb安装的操作系统用户组")
	cmd.Flags().IntVarP(&option.Port, "port", "P", 0, "mongodb 数据库监听端口")
	cmd.Flags().StringVarP(&option.Dir, "dir", "d", "", "mongodb安装目录, 默认: /opt/mongodb$PORT")
	cmd.Flags().StringVarP(&option.Username, "username", "u", "", "mongodb用户名")
	cmd.Flags().StringVarP(&option.Password, "password", "p", "", "mongodb密码")
	cmd.Flags().IntVarP(&option.Memory, "memory", "m", 1, "内存")
	cmd.Flags().BoolVar(&option.Arbiter, "arbiter", false, "是否为仲裁节点")
	cmd.Flags().StringVarP(&option.BindIP, "bind-ip", "b", "", "mongodb 数据库监听地址")
	cmd.Flags().StringVarP(&option.Join, "join", "j", "", "做为从库, 要加入的副本集群的任意一个节点<IP:PORT>")
	cmd.Flags().StringVarP(&option.Owner, "owner", "o", "", "当机器上有多个IP时, 指定以哪个IP创建实例")
	cmd.Flags().StringVar(&option.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&option.Yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	cmd.Flags().BoolVarP(&option.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	cmd.Flags().BoolVar(&onlyCheck, "only-check", false, "只检查配置和环境, 不进行实际安装操作")
	return cmd
}

// dbup mongodb uninstall
func mongodbUNInstallCmd() *cobra.Command {
	var yes bool
	var uninst service.MongoDBUNInstall
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "mongodb 单机版卸载",
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
	cmd.Flags().IntVarP(&uninst.Port, "port", "P", 0, "MongoDB 数据库监听端口")
	cmd.Flags().StringVarP(&uninst.BasePath, "dir", "d", "", "MongoDB 安装目录")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "是否确认卸载")
	return cmd
}

// dbup mongodb cluster-deploy
func mongodbClusterDeployCmd() *cobra.Command {
	var config string
	cmd := &cobra.Command{
		Use:   "cluster-deploy",
		Short: "mongodb 分片集群部署",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			rs := service.NewMongoClusterDeploy()
			return rs.Run(config)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	return cmd
}

// dbup mongodb cluster-deploy
func mongodbRemoveClusterCmd() *cobra.Command {
	var yes bool
	var config string
	cmd := &cobra.Command{
		Use:   "remove-cluster-deploy",
		Short: "mongodb 移除分片集群部署",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			rs := service.NewMongoClusterDeploy()
			return rs.RemoveCluster(config, yes)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	return cmd
}

// dbup mongodb replication-deploy
func mongodbDeployCmd() *cobra.Command {
	var config string
	var noRollback bool
	cmd := &cobra.Command{
		Use:   "replication-deploy",
		Short: "mongodb 集群部署",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			rs := service.NewMongoDBDeploy()
			return rs.Run(config, noRollback)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	cmd.Flags().BoolVarP(&noRollback, "no-rollback", "n", false, "安装失败不回滚")
	return cmd
}

// dbup mongodb replication-deploy
func mongodbRemoveDeployCmd() *cobra.Command {
	var yes bool
	var config string
	cmd := &cobra.Command{
		Use:   "remove-replication-deploy",
		Short: "mongodb 移除副本集部署",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			rs := service.NewMongoDBDeploy()
			return rs.RemoveCluster(config, yes)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	return cmd
}

// dbup mongodb backup
func mongodbBackupCmd() *cobra.Command {
	var backup = service.NewBackup()
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "mongodb 进行备份",
		RunE: func(cmd *cobra.Command, args []string) error {
			return backup.Run()
		},
	}
	cmd.Flags().StringVarP(&backup.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&backup.Host, "host", "H", "127.0.0.1", "mongodb 地址")
	cmd.Flags().IntVarP(&backup.Port, "port", "P", 5432, "mongodb 数据库监听端口")
	cmd.Flags().StringVarP(&backup.AuthDB, "auth-db", "d", "", "认证库名")
	cmd.Flags().StringVarP(&backup.Username, "username", "u", "", "用户名")
	cmd.Flags().StringVarP(&backup.BackupCmd, "command", "c", "mongodump", "redis 备份命令")
	cmd.Flags().StringVarP(&backup.BackupFile, "backupfile", "f", "", "redis 备份目录")
	return cmd
}
