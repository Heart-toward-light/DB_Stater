/*
@Author : WuWeiJian
@Date : 2021-03-29 10:18
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
func pgsqlUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "pgsql 用户管理",
	}
	// 装载命令
	cmd.AddCommand(
		pgsqlUserAddCmd(),
		pgsqlUserGrantCmd(),
	)
	return cmd
}

// dbup pgsql user add
func pgsqlUserAddCmd() *cobra.Command {
	var m = services.NewPGManager()
	cmd := &cobra.Command{
		Use:   "create",
		Short: "pgsql 添加用户",
		RunE: func(cmd *cobra.Command, args []string) error {
			if m.User == "" || m.Password == "" {
				return fmt.Errorf("请指定要创建的用户名和密码")
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
func pgsqlUserGrantCmd() *cobra.Command {
	var m = services.NewPGManager()
	cmd := &cobra.Command{
		Use:   "grant",
		Short: "pgsql 用户授权",
		RunE: func(cmd *cobra.Command, args []string) error {
			if m.User == "" || m.DBName == "" || m.Address == "" {
				return fmt.Errorf("请指定要授权的用户名,库名,IP地址")
			}

			pg := pgsql.NewPgsql()
			return pg.UserGrant(m)
		},
	}
	cmd.Flags().StringVarP(&m.Host, "host", "H", config.DefaultPGSocketPath, "pgsql 地址")
	cmd.Flags().IntVarP(&m.Port, "port", "P", 5432, "pgsql 端口")
	cmd.Flags().StringVarP(&m.AdminUser, "admin-user", "u", config.DefaultPGAdminUser, "管理员用户")
	cmd.Flags().StringVarP(&m.AdminPassword, "admin-password", "p", "", "管理员密码")
	cmd.Flags().StringVarP(&m.AdminDatabase, "admin-database", "d", "", "管理员登录库")
	cmd.Flags().StringVar(&m.User, "user", "", "要授权的用户")
	//cmd.Flags().StringVar(&m.Password, "password","", "要创建的用户密码")
	cmd.Flags().StringVar(&m.DBName, "dbname", "", "要授权用户的登录库")
	cmd.Flags().StringVarP(&m.Address, "address", "a", "", "要授权用户的授权IP列表")
	//cmd.Flags().BoolVar(&m.Ignore, "ignore", false, "用户已经存在则忽略")
	return cmd
}
