// Created by LiuSainan on 2021-12-09 18:07:04

package backupcmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

//const Version = "1.0.15"
var _version string

func versionCmd() *cobra.Command {
	// 定义二级命令: version
	var cmd = &cobra.Command{
		Use:   "version",
		Short: "dbupbackup 的版本",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dbupbackup version %s\n", _version)
		},
	}

	return cmd
}
