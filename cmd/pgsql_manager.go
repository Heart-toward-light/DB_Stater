/*
@Author : WuWeiJian
@Date : 2021-04-02 10:24
*/

package cmd

import (
	"dbup/internal/pgsql"
	"dbup/internal/pgsql/config"
	"dbup/internal/pgsql/services"
	"fmt"

	"github.com/spf13/cobra"
)

// dbup pgsql check-slaves
func pgsqlCheckSlavesCmd() *cobra.Command {
	var m = services.NewPGManager()
	cmd := &cobra.Command{
		Use:   "check-slaves",
		Short: "pgsql 检查从库信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请输入从库IP")
			}

			pg := pgsql.NewPgsql()
			return pg.CheckSlaves(m, args[0])
		},
	}

	cmd.Flags().StringVarP(&m.Host, "host", "H", config.DefaultPGSocketPath, "pgsql 地址")
	cmd.Flags().IntVarP(&m.Port, "port", "P", 5432, "pgsql 端口")
	cmd.Flags().StringVarP(&m.AdminUser, "admin-user", "u", config.DefaultPGAdminUser, "管理员用户")
	cmd.Flags().StringVarP(&m.AdminPassword, "admin-password", "p", "", "管理员密码")
	cmd.Flags().StringVarP(&m.AdminDatabase, "admin-database", "d", "", "管理员登录库, 默认与用户名同名")
	return cmd
}

// dbup pgsql check-slaves
func pgsqlCheckSelectCmd() *cobra.Command {
	var m = services.NewPGManager()
	cmd := &cobra.Command{
		Use:   "check-select",
		Short: "pgsql 实例存活状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := pgsql.NewPgsql()
			return pg.CheckSelect(m)
		},
	}

	cmd.Flags().StringVarP(&m.Host, "host", "H", "127.0.0.1", "pgsql 地址")
	cmd.Flags().IntVarP(&m.Port, "port", "P", 5432, "pgsql 端口")
	cmd.Flags().StringVarP(&m.AdminUser, "admin-user", "u", config.DefaultPGUser, "用户名")
	cmd.Flags().StringVarP(&m.AdminPassword, "admin-password", "p", "", "用户密码")
	cmd.Flags().StringVarP(&m.AdminDatabase, "admin-database", "d", "", "用户登录库, 默认与用户名同名")
	return cmd
}
