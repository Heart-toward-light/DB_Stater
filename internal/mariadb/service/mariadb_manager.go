package service

import (
	"dbup/internal/environment"
	"dbup/internal/mariadb/config"
	"dbup/internal/mariadb/dao"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type MariaDBManager struct {
}

func NewMariaDBManager() *MariaDBManager {
	return &MariaDBManager{}
}

func (d *MariaDBManager) AddSlaveNode(ssh config.Server, m config.MariaDBOptions) error {
	logger.Infof("验证参数\n")
	if err := ssh.Checkport(); err != nil {
		return err
	}

	if err := d.ParameterCheck(m); err != nil {
		return err
	}

	if ssh.TmpDir == "" {
		ssh.TmpDir = config.DeployTmpDir
	}

	if ssh.Password == "" && ssh.KeyFile == "" {
		ssh.KeyFile = filepath.Join(environment.GlobalEnv().HomePath, ".ssh", "id_rsa")
	}

	var node *MariaDBInstance
	var err error

	if ssh.Password != "" {
		node, err = NewmariaDBInstance(ssh.TmpDir,
			ssh.Address,
			ssh.User,
			ssh.Password,
			ssh.SshPort,
			m)
		if err != nil {
			return err
		}
	} else {
		node, err = NewmariaDBInstanceUseKeyFile(ssh.TmpDir,
			ssh.Address,
			ssh.User,
			ssh.KeyFile,
			ssh.SshPort,
			m)
		if err != nil {
			return err
		}
	}

	if err := node.CheckTmpDir(); err != nil {
		return err
	}

	defer node.DropTmpDir()

	logger.Infof("将安装包复制到目标机器\n")
	source := path.Join(environment.GlobalEnv().ProgramPath, "..")
	if err := node.Scp(source); err != nil {
		return err
	}

	if err := node.InstallSlave(true, true); err != nil {
		return err
	}

	logger.Infof("开始安装 Mariadb 从库并同步备份数据, 数据量大时间会比较长\n")
	if err := node.InstallSlave(false, true); err != nil {
		logger.Warningf("安装失败\n")
		return err
	}

	time.Sleep(3 * time.Second)
	logger.Infof("开始检查 Mariadb 从库同步状态\n")
	if err := d.CheckSlavestatus(ssh, m); err != nil {
		return err
	}

	logger.Successf("MariaDB 从库新增 [完成]\n")

	// logger.Infof("开始同步备份数据,数据量大时间会比较长\n")

	return nil
}

func (d *MariaDBManager) ParameterCheck(m config.MariaDBOptions) error {

	if m.Dir == "" {
		return fmt.Errorf("必须指定安装主路经, 如无特殊需求需要和主库保持一致,例: /opt/mariadb%d", m.Port)
	}

	if m.Join != "" {
		ipPort := strings.Split(m.Join, ":")
		if len(ipPort) != 2 {
			return fmt.Errorf("--join 指定同步主库的地址格式不对, 请参考 <IP:PORT>")
		}
		if err := utils.IsIPv4(ipPort[0]); err != nil {
			if !utils.IsHostName(ipPort[0]) {
				return fmt.Errorf("--join 参数的地址部分即不是IP地址, 也不是有效的主机名")
			}
		}
		ok, _ := utils.TcpGather(m.Join)
		if !ok {
			return fmt.Errorf("mariadb 指定的主库ip与端口服务 %s 连接异常", m.Join)
		}

		masterip := ipPort[0]
		masterport, _ := strconv.Atoi(ipPort[1])

		replconn, err := dao.NewMariaDBConn(masterip, masterport, m.Repluser, m.ReplPassword, "")
		if err != nil {
			return fmt.Errorf("mariadb 指定的复制账号 %s 连接异常", m.Repluser)
		}

		if err := replconn.Select(); err != nil {
			return fmt.Errorf("账号 %s 连接失败: %v", m.Repluser, err)
		}
		replconn.DB.Close()

		bakconn, err := dao.NewMariaDBConn(masterip, masterport, m.Backupuser, m.BackupPassword, "")
		if err != nil {
			return fmt.Errorf("mariadb 指定的备份账号 %s 连接异常", m.Backupuser)
		}
		if err := bakconn.Select(); err != nil {
			return fmt.Errorf("账号 %s 连接失败: %v", m.Backupuser, err)
		}
		bakconn.DB.Close()

	} else {
		return fmt.Errorf("--join 必须指定同步主库的地址 <IP:PORT>")
	}

	// dumpfile := filepath.Join(m.Dir, "/bin/mariadb-dump")

	// if !utils.IsExists(dumpfile) {
	// 	return fmt.Errorf("无法找到备份二进制文件,请确认主路径 %s 是否填写正确",m.Dir)
	// }

	return nil
}

func (d *MariaDBManager) CheckSlavestatus(ssh config.Server, m config.MariaDBOptions) error {
	conn, err := dao.NewMariaDBConn(ssh.Address, m.Port, m.Repluser, m.ReplPassword, "")
	if err != nil {
		return err
	}
	defer conn.DB.Close()

	status, err := conn.ShowSlaveStatus()
	if err != nil {
		return fmt.Errorf("从库 %s:%d 同步状态异常: %s ", ssh.Address, m.Port, err)
	}

	for k, v := range status {
		if v.Valid {

			if k == "Slave_IO_Running" && v.String != "Yes" {
				return fmt.Errorf("从库 %s:%d IO线程同步状态异常", ssh.Address, m.Port)
			}

			if k == "Slave_SQL_Running" && v.String != "Yes" {
				return fmt.Errorf("从库 %s:%d SQL线程同步状态异常", ssh.Address, m.Port)
			}

		}
	}

	return nil
}
