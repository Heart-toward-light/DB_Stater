/*
@Author : WuWeiJian
@Date : 2021-07-27 15:10
*/

package cmd

import (
	"dbup/internal/redis"
	"dbup/internal/redis/config"
	"dbup/internal/redis/services"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func redisClusterCmd() *cobra.Command {
	// 定义二级命令: redis-cluster
	var cmd = &cobra.Command{
		Use:   "redis-cluster",
		Short: "redis-cluster相关操作",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	// 装载命令
	cmd.AddCommand(
		redisClusterDeployCmd(),
		redisClusterRemoveDeployCmd(),
		redisClusterAddMasterCmd(),
		redisClusterAddSlaveCmd(),
		redisClusterFixCmd(),
		redisClusterBackupCmd(),
	)
	return cmd
}

// dbup redis-cluster deploy
func redisClusterDeployCmd() *cobra.Command {
	var config string
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "redis cluster 集群部署",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			rs := redis.NewRedis()
			return rs.RedisClusterDeploy(config)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	return cmd
}

func redisClusterRemoveDeployCmd() *cobra.Command {
	var yes bool
	var config string
	cmd := &cobra.Command{
		Use:   "remove-deploy",
		Short: "redis cluster 集群删除",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				return fmt.Errorf("请指定部署配置文件")
			}
			rs := redis.NewRedis()
			return rs.RedisClusterRemoveDeploy(config, yes)
		},
	}
	cmd.Flags().StringVarP(&config, "config", "c", "", "安装配置文件")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "直接卸载, 否则需要交互确认")
	return cmd
}

// dbup redis-cluster add-master
func redisClusterAddMasterCmd() *cobra.Command {
	var option config.RedisClusterAddNodeOption
	//var packageName string
	cmd := &cobra.Command{
		Use:   "add-master",
		Short: "redis cluster 添加主节点",
		RunE: func(cmd *cobra.Command, args []string) error {
			if option.Parameter.Port == 0 {
				return errors.New("请指定 --port 端口号")
			}
			rs := redis.NewRedis()
			return rs.RedisClusterAddMaster(option)
		},
	}
	cmd.Flags().IntVar(&option.SSHConfig.Port, "ssh-port", 22, "ssh 端口号")
	cmd.Flags().StringVar(&option.SSHConfig.Username, "ssh-username", "", "ssh 用户名")
	cmd.Flags().StringVar(&option.SSHConfig.Password, "ssh-password", "", "ssh 密码")
	cmd.Flags().StringVar(&option.SSHConfig.KeyFile, "ssh-keyfile", "", "ssh 密钥")
	cmd.Flags().StringVar(&option.Host, "host", "", "新节点IP")
	cmd.Flags().BoolVar(&option.IPV6, "ipv6", false, "是否开启IPV6功能,默认不开启")
	cmd.Flags().StringVar(&option.Cluster, "cluster", "", "要加入的集群任意一节点的<IP:PORT>")
	cmd.Flags().StringVar(&option.TmpDir, "tmp-dir", config.RedisClusterDeployTmpDir, "远程机器的临时目录")
	cmd.Flags().StringVar(&option.Parameter.SystemUser, "system-user", config.DefaultRedisSystemUser, "redis安装的操作系统用户")
	cmd.Flags().StringVar(&option.Parameter.SystemGroup, "system-group", config.DefaultRedisSystemGroup, "redis安装的操作系统用户组")
	cmd.Flags().StringVarP(&option.Parameter.Password, "password", "p", "", "redis密码, 需要与集群密码保持一直")
	cmd.Flags().StringVarP(&option.Parameter.Dir, "dir", "d", "", "redis安装目录, 默认: /opt/redis$PORT")
	cmd.Flags().StringVarP(&option.Parameter.MemorySize, "memory-size", "m", "", "redis 数据库内存大小, 默认操作系统最大内存的四分之一")
	cmd.Flags().IntVarP(&option.Parameter.Port, "port", "P", 0, "redis 数据库监听端口")
	cmd.Flags().StringVar(&option.Parameter.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&option.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	return cmd
}

// dbup redis-cluster add-master
func redisClusterAddSlaveCmd() *cobra.Command {
	var option config.RedisClusterAddNodeOption
	//var packageName string
	cmd := &cobra.Command{
		Use:   "add-slave",
		Short: "redis cluster 添加从节点",
		RunE: func(cmd *cobra.Command, args []string) error {
			if option.Parameter.Port == 0 {
				return errors.New("请指定 --port 端口号")
			}
			rs := redis.NewRedis()
			return rs.RedisClusterAddSlave(option)
		},
	}
	cmd.Flags().IntVar(&option.SSHConfig.Port, "ssh-port", 22, "ssh 端口号")
	cmd.Flags().StringVar(&option.SSHConfig.Username, "ssh-username", "", "ssh 用户名")
	cmd.Flags().StringVar(&option.SSHConfig.Password, "ssh-password", "", "ssh 密码")
	cmd.Flags().StringVar(&option.SSHConfig.KeyFile, "ssh-keyfile", "", "ssh 密码")
	cmd.Flags().StringVar(&option.Host, "host", "", "新节点IP")
	cmd.Flags().BoolVar(&option.IPV6, "ipv6", false, "是否开启IPV6功能,默认不开启")
	cmd.Flags().StringVar(&option.Cluster, "cluster", "", "要加入的集群任意一节点的<IP:PORT>")
	cmd.Flags().StringVar(&option.TmpDir, "tmp-dir", config.RedisClusterDeployTmpDir, "远程机器的临时目录")
	cmd.Flags().StringVar(&option.Master, "master", "", "加入集群后作为哪个<IP:PORT>的从库, 如果为空, 则默认找最少从库的节点添加")
	cmd.Flags().StringVar(&option.Parameter.SystemUser, "system-user", config.DefaultRedisSystemUser, "redis安装的操作系统用户")
	cmd.Flags().StringVar(&option.Parameter.SystemGroup, "system-group", config.DefaultRedisSystemGroup, "redis安装的操作系统用户组")
	cmd.Flags().StringVarP(&option.Parameter.Password, "password", "p", "", "redis密码, 需要与集群密码保持一直")
	cmd.Flags().StringVarP(&option.Parameter.Dir, "dir", "d", "", "redis安装目录, 默认: /opt/redis$PORT")
	cmd.Flags().StringVarP(&option.Parameter.MemorySize, "memory-size", "m", "", "redis 数据库内存大小, 默认操作系统最大内存的四分之一")
	cmd.Flags().IntVarP(&option.Parameter.Port, "port", "P", 0, "redis 数据库监听端口")
	cmd.Flags().StringVar(&option.Parameter.ResourceLimit, "resource-limit", "", "资源限制清单, 通过执行 systemctl set-property 实现. 例: --resource-limit='MemoryLimit=512M CPUShares=500'")
	cmd.Flags().BoolVarP(&option.NoRollback, "no-rollback", "n", false, "安装失败不回滚")
	return cmd
}

// dbup redis-cluster add-master
func redisClusterFixCmd() *cobra.Command {
	var (
		command  string
		cluster  string
		password string
	)
	//var packageName string
	cmd := &cobra.Command{
		Use:   "fix",
		Short: "redis cluster 修复集群, 重新分配slot, 并将坏节点从集群中踢出",
		RunE: func(cmd *cobra.Command, args []string) error {
			rs := redis.NewRedis()
			return rs.RedisClusterFix(command, cluster, password)
		},
	}
	cmd.Flags().StringVarP(&command, "command", "c", "redis-cli", "redis 备份命令")
	cmd.Flags().StringVar(&cluster, "cluster", "", "要修复的集群的任意一节点的<IP:PORT>")
	cmd.Flags().StringVarP(&password, "password", "p", "", "redis密码, 需要与集群密码保持一直")
	return cmd
}

// dbup redis-cluster backup
func redisClusterBackupCmd() *cobra.Command {
	var backup = services.NewRedisClusterBackup()
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "redis cluster 进行备份",
		RunE: func(cmd *cobra.Command, args []string) error {
			rs := redis.NewRedis()
			return rs.RedisClusterBackup(backup)
		},
	}
	cmd.Flags().StringVarP(&backup.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&backup.Host, "host", "H", "127.0.0.1", "redis 地址")
	cmd.Flags().IntVarP(&backup.Port, "port", "P", 6379, "redis 数据库监听端口")
	cmd.Flags().StringVarP(&backup.BackupCmd, "command", "c", "redis-cli", "redis 备份命令")
	cmd.Flags().StringVarP(&backup.BackupBasePath, "backupdir", "d", "", "redis 备份目录")
	cmd.Flags().IntVarP(&backup.Expire, "expire", "e", 0, "过期删除多少天之前的备份, 0表示永不删除")
	cmd.Flags().BoolVar(&backup.BackupToS3, "backupToS3", false, "是否要备份到S3, 备份到S3会直接将本地备份删除")
	cmd.Flags().StringVar(&backup.EndPoint, "endpoint", "", "S3地址")
	cmd.Flags().StringVar(&backup.AccessKey, "accesskey", "", "S3 accesskey")
	cmd.Flags().StringVar(&backup.SecretKey, "secretkey", "", "S3 secretkey")
	cmd.Flags().StringVar(&backup.Bucket, "bucket", "", "S3 bucket")
	cmd.Flags().StringVar(&backup.Mode, "mode", "normal", "S3 连接模式, <normal|SkipVerify>")
	cmd.Flags().StringVar(&backup.S3BasePath, "s3path", "", "S3 存放key前缀, 默认使用 -d | --backupdir 参数值")
	return cmd
}
