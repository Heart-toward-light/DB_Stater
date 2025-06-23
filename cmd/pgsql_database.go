/*
@Author : WuWeiJian
@Date : 2021-03-29 17:00
*/

package cmd

import (
	"dbup/internal/pgsql"
	"dbup/internal/pgsql/config"
	"dbup/internal/pgsql/services"
	"fmt"

	"github.com/spf13/cobra"
)

// dbup pgsql backup-tasks
func pgsqlDatabaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "pgsql 数据库管理",
	}
	// 装载命令
	cmd.AddCommand(
		pgsqlDatabaseCreateCmd(),
	)
	return cmd
}

// dbup pgsql database create
func pgsqlDatabaseCreateCmd() *cobra.Command {
	var m = services.NewPGManager()
	cmd := &cobra.Command{
		Use:   "create",
		Short: "pgsql 创建库",
		RunE: func(cmd *cobra.Command, args []string) error {
			if m.DBName == "" {
				return fmt.Errorf("请指定要创建的库名\n")
			}

			pg := pgsql.NewPgsql()
			return pg.DatabaseCreate(m)
		},
	}
	cmd.Flags().StringVarP(&m.Host, "host", "H", config.DefaultPGSocketPath, "pgsql 地址")
	cmd.Flags().IntVarP(&m.Port, "port", "P", 5432, "pgsql 端口")
	cmd.Flags().StringVarP(&m.AdminUser, "admin-user", "u", config.DefaultPGAdminUser, "管理员用户")
	cmd.Flags().StringVarP(&m.AdminPassword, "admin-password", "p", "", "管理员密码")
	cmd.Flags().StringVarP(&m.AdminDatabase, "admin-database", "d", "", "管理员登录库, 默认与用户名同名")
	cmd.Flags().StringVar(&m.DBName, "dbname", "", "要创建的库")
	cmd.Flags().BoolVar(&m.Ignore, "ignore", false, "库已经存在则忽略")
	return cmd
}
