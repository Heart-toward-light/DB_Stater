// Created by LiuSainan on 2021-12-09 18:12:46

package backupcmd

import (
	"dbup/internal/pgsql"
	"dbup/internal/pgsql/services"

	"github.com/spf13/cobra"
)

// dbup pgsql backup
func pgsqlBackupCmd() *cobra.Command {
	var backup = services.NewBackup()
	cmd := &cobra.Command{
		Use:   "pgsql",
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
		Use:   "pgsql-backup-tables",
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
