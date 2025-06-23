// Created by LiuSainan on 2021-12-09 18:08:21

package backupcmd

import (
	"dbup/internal/redis"
	"dbup/internal/redis/services"

	"github.com/spf13/cobra"
)

// dbup pgsql backup
func redisBackupCmd() *cobra.Command {
	var backup = services.NewBackup()
	cmd := &cobra.Command{
		Use:   "redis",
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
