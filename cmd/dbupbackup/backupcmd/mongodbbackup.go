// Created by LiuSainan on 2021-12-09 11:21:09

package backupcmd

import (
	"dbup/internal/mongodb/service"

	"github.com/spf13/cobra"
)

// dbup mongodb backup
func mongodbBackupCmd() *cobra.Command {
	var backup = service.NewBackup()
	cmd := &cobra.Command{
		Use:   "mongodb",
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
