package backupcmd

import (
	"dbup/internal/mongodb/service"

	"github.com/spf13/cobra"
)

// dbup mariadb backup
func mariadbBackupCmd() *cobra.Command {
	var backup = service.NewBackup()
	cmd := &cobra.Command{
		Use:   "mariadb",
		Short: "mariadb 进行备份",
		RunE: func(cmd *cobra.Command, args []string) error {
			return backup.Run()
		},
	}
	cmd.Flags().StringVarP(&backup.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&backup.Host, "host", "H", "127.0.0.1", "mariadb 地址")
	cmd.Flags().IntVarP(&backup.Port, "port", "P", 3306, "mariadb 数据库监听端口")
	cmd.Flags().StringVarP(&backup.Username, "username", "u", "", "用户名")
	cmd.Flags().StringVarP(&backup.BackupCmd, "command", "c", "mariadb-dump", "mariadb 备份命令")
	cmd.Flags().StringVarP(&backup.BackupFile, "backupfile", "f", "", "mariadb 备份目录")
	return cmd
}
