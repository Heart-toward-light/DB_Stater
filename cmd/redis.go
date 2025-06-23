/*
@Author : WuWeiJian
@Date : 2020-12-02 17:42
*/

package cmd

import (
	"dbup/internal/environment"
	"dbup/internal/redis"
	"dbup/internal/redis/config"
	"dbup/internal/redis/services"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func redisCmd() *cobra.Command {
	// 定义二级命令: pgsql
	var cmd = &cobra.Command{
		Use:   "redis",
		Short: "redis相关操作",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	// 装载命令
	cmd.AddCommand(
		redisInstallCmd(),
		redisAddSlaveCmd(),
		redisUNInstallCmd(),
		redisDeployCmd(),
		redisRemoveDeployCmd(),
		redisBackupCmd(),
		redisBackupTaskCmd(),
		RedisUPgradeCmd(),
	)

	return cmd
}

// dbup redis install
func redisInstallCmd() *cobra.Command {
	var param config.Parameters
	var cfgFile string
	var onlyCheck bool
	//var packageName string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "redis 单机版安装",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return environment.MustRoot()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			rs := redis.NewRedis()
			return rs.Install(param, cfgFile, onlyCheck)
		},
	}
	cmd.Flags().StringVar(&param.SystemUser, "system-user", config.DefaultRedisSystemUser, "redis安装的操作系统用户")
	cmd.Flags().StringVar(&param.SystemGroup, "system-group", config.DefaultRedisSystemGroup, "redis安装的操作系统用户组")
	cmd.Flags().StringVarP(&param.Password, "password", "p", "", fmt.Sprintf("密码, 默认随机生成16位密码, 建议使用: %s", utils.GeneratePasswd(16)))
	cmd.Flags().StringVarP(&param.Dir, "dir", "d", "", "redis安装目录, 默认: /opt/redis$PORT")
	cmd.Flags().StringVarP(&param.MemorySize, "memory-size", "m", "", "redis 数据库内存大小, 默认操作系统最大内存的四分之一")
	cmd.Flags().StringVarP(&param.Module, "module", "M", "", "redis模块,多个模块以逗号分割, 目前仅支持[redisbloom,redisearch,redisgraph]")
	cmd.Flags().StringVar(&param.Master, "master", "", "要同步的主库节点<IP:PORT>, 为空则默认安装单机单实例主库")
	cmd.Flags().IntVarP(&param.Port, "port", "P", 0, "redis 数据库监听端口")
	cmd.Flags().BoolVar(&param.Ipv6, "ipv6", false, "是否开启IPV6功能,默认不开启")
	// cmd.Flags().StringVar(&param.Appendonly, "appendonly", "yes", "是否开启aof,可选值:<yes|no>")
	cmd.Flags().StringVar(&param.MaxmemoryPolicy, "maxmemory-policy", "noeviction", "key淘汰策略,可选值:<volatile-lru|allkeys-lru|volatile-random|allkeys-random|volatile-ttl|noeviction>")
	cmd.Flags().BoolVar(&onlyCheck, "only-check", false, "只检查配置和环境, 不进行实际安装操作")
	cmd.Flags().BoolVarP(&param.Yes, "yes", "y", false, "直接安装, 否则需要交互确认")
	cmd.Flags().BoolVar(&param.Cluster, "cluster", false, "是否为集群模式")
	cmd.Flags().StringVar(&param.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&param.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "安装配置文件, 默认不使用配置文件")
	return cmd
}

// dbup redis-cluster add-master
func redisAddSlaveCmd() *cobra.Command {
	var option config.RedisClusterAddNodeOption
	//var packageName string
	cmd := &cobra.Command{
		Use:   "add-slave",
		Short: "redis 添加从节库",
		RunE: func(cmd *cobra.Command, args []string) error {
			if option.Parameter.Port == 0 {
				return errors.New("请指定 --port 端口号")
			}

			rs := redis.NewRedis()
			return rs.RedisAddSlave(option)
		},
	}
	cmd.Flags().IntVar(&option.SSHConfig.Port, "ssh-port", 22, "ssh 端口号")
	cmd.Flags().StringVar(&option.SSHConfig.Username, "ssh-username", "", "ssh 用户名")
	cmd.Flags().StringVar(&option.SSHConfig.Password, "ssh-password", "", "ssh 密码")
	cmd.Flags().StringVar(&option.SSHConfig.KeyFile, "ssh-keyfile", "", "ssh 密钥")
	cmd.Flags().StringVar(&option.Host, "host", "", "新节点IP")
	cmd.Flags().BoolVar(&option.IPV6, "ipv6", false, "是否开启IPV6功能,默认不开启")
	cmd.Flags().StringVar(&option.TmpDir, "tmp-dir", config.RedisClusterDeployTmpDir, "远程机器的临时目录")
	cmd.Flags().StringVar(&option.Parameter.SystemUser, "system-user", config.DefaultRedisSystemUser, "redis安装的操作系统用户")
	cmd.Flags().StringVar(&option.Parameter.SystemGroup, "system-group", config.DefaultRedisSystemGroup, "redis安装的操作系统用户组")
	cmd.Flags().StringVar(&option.Parameter.Master, "master", "", "要加入的集群的主节点<IP:PORT>")
	cmd.Flags().StringVarP(&option.Parameter.Password, "password", "p", "", "redis密码, 需要与主库密码保持一直")
	cmd.Flags().StringVarP(&option.Parameter.Dir, "dir", "d", "", "redis安装目录, 默认: /opt/redis$PORT")
	cmd.Flags().StringVarP(&option.Parameter.MemorySize, "memory-size", "m", "", "redis 数据库内存大小, 默认操作系统最大内存的四分之一")
	cmd.Flags().IntVarP(&option.Parameter.Port, "port", "P", 0, "redis 数据库监听端口")
	cmd.Flags().StringVar(&option.Parameter.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&option.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	return cmd
}

// dbup redis uninstall
func redisUNInstallCmd() *cobra.Command {
	var yes bool
	var uninst services.UNInstall
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "redis 单机版卸载",
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

			rs := redis.NewRedis()
			return rs.UNInstall(&uninst)
		},
	}
	cmd.Flags().IntVarP(&uninst.Port, "port", "P", 0, "Redis 数据库监听端口")
	cmd.Flags().StringVarP(&uninst.BasePath, "dir", "d", "", "Redis 安装目录")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "是否确认卸载")
	return cmd
}

// dbup redis cluster-deploy
func redisDeployCmd() *cobra.Command {
	var config string
	cmd := &cobra.Command{
		Use:   "cluster-deploy",
		Short: "redis 主从集群部署",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			rs := redis.NewRedis()
			return rs.Deploy(config)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	return cmd
}

// dbup redis cluster-deploy
func redisRemoveDeployCmd() *cobra.Command {
	var yes bool
	var config string
	cmd := &cobra.Command{
		Use:   "remove-cluster-deploy",
		Short: "redis 主从集群删除",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			rs := redis.NewRedis()
			return rs.RemoveDeploy(config, yes)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "直接卸载, 否则需要交互确认")
	return cmd
}

// dbup redis backup
func redisBackupCmd() *cobra.Command {
	var backup = services.NewBackup()
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "redis 进行备份",
		RunE: func(cmd *cobra.Command, args []string) error {
			rs := redis.NewRedis()
			return rs.Backup(backup)
		},
	}
	cmd.Flags().StringVarP(&backup.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&backup.Host, "host", "H", "127.0.0.1", "redis 地址")
	cmd.Flags().IntVarP(&backup.Port, "port", "P", 5432, "redis 数据库监听端口")
	cmd.Flags().StringVarP(&backup.BackupCmd, "command", "c", "redis-cli", "redis 备份命令")
	cmd.Flags().StringVarP(&backup.BackupFile, "backupfile", "f", "", "redis 备份目录")
	return cmd
}

// dbup redis upgrade
func RedisUPgradeCmd() *cobra.Command {
	var upgrade = services.NewUPgrade()
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "redis 升级",
		RunE: func(cmd *cobra.Command, args []string) error {
			return upgrade.Run()
		},
	}
	cmd.Flags().StringVarP(&upgrade.Password, "password", "p", "", "旧版本实例密码")
	cmd.Flags().IntVarP(&upgrade.Port, "port", "P", 0, "redis 旧版本实例端口")
	cmd.Flags().StringVarP(&upgrade.SourceDir, "old", "o", "", "旧版本 redis 安装主目录")
	cmd.Flags().StringVarP(&upgrade.EmoloyDir, "new", "n", "", "新版本 redis 安装主目录")
	cmd.Flags().BoolVarP(&upgrade.Yes, "yes", "y", false, "直接升级, 否则需要交互确认")

	return cmd
}
