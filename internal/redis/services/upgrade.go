package services

import (
	"dbup/internal/global"
	"dbup/internal/redis/config"
	"dbup/internal/redis/dao"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type UPgrade struct {
	Port        int
	Password    string
	SourceDir   string
	EmoloyDir   string
	NewVersion  string
	Servicefile string
	Yes         bool
}

func NewUPgrade() *UPgrade {
	return &UPgrade{}
}

func (u *UPgrade) Validator() error {
	logger.Infof("验证参数\n")

	if u.Port == 0 {
		return fmt.Errorf("请指定准备升级的redis实例端口")
	}

	if u.SourceDir == "" {
		return fmt.Errorf("请指定旧版本的安装主目录路径")
	}

	if u.EmoloyDir == "" {
		return fmt.Errorf("请指定新版本的安装主目录路径")
	}

	OldVersion, err := command.RedisVersion(u.SourceDir)
	if err != nil {
		return err
	}

	u.NewVersion, err = command.RedisVersion(u.EmoloyDir)
	if err != nil {
		return err
	}

	result := command.CompareVersion(OldVersion, u.NewVersion)

	switch result {
	case 1:
		logger.Errorf("Redis 老版本 %s 不能大于新版本 %s \n", OldVersion, u.NewVersion)
	case 0:
		logger.Errorf("Redis 老版本 %s 不能等于新版本 %s \n", OldVersion, u.NewVersion)
	}

	return nil
}

func (u *UPgrade) Run() error {
	if err := u.Validator(); err != nil {
		return err
	}

	if !u.Yes {
		var yes string
		logger.Warningf("升级版本需要备份主节点AOF和重启实例: 127.0.0.1:%d\n", u.Port)
		logger.Warningf("是否确认升级[y|n]:")
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	logger.Infof("开始升级\n")
	if err := u.UPgradeDB(); err != nil {
		return err
	}

	logger.Successf("升级完成\n")

	return nil
}

func (u *UPgrade) UPgradeDB() error {
	conn, err := dao.NewRedisConn("127.0.0.1", u.Port, u.Password)
	if err != nil {
		return err
	}
	defer conn.Conn.Close()

	master, err := conn.GetMaster()
	if err != nil {
		return err
	}
	if master {
		logger.Infof("开始持久化主节点AOF\n")
		if err := conn.FlushAOF(); err != nil {
			return err
		}
	}

	service := fmt.Sprintf(config.ServiceFileName, u.Port)
	u.Servicefile = filepath.Join(global.ServicePath, service)
	logger.Infof("开始关停老版本实例\n")
	if err := command.SystemCtl(service, "stop"); err != nil {
		return err
	}

	logger.Infof("开始替换老版本实例依赖文件\n")
	SourceServer := filepath.Join(u.SourceDir, "server")
	NewServer := filepath.Join(u.EmoloyDir, "server")
	if err := command.MoveFile(SourceServer); err != nil {
		return fmt.Errorf("修改旧版本依赖文件路径 %s 失败: %s", SourceServer, err)
	}

	if err := command.CopyDir(NewServer, u.SourceDir); err != nil {
		return err
	}

	sourcedata := filepath.Join(u.SourceDir, "data")

	l := command.Local{Timeout: 259200}
	if user, group, err := command.GetUserInfo(sourcedata); err != nil {
		return err
	} else {
		chowncmd := fmt.Sprintf("chown -R %s:%s %s", user, group, u.SourceDir)
		if _, stderr, err := l.Run(chowncmd); err != nil {
			return fmt.Errorf("执行修改路径所属权限失败: %v, 标准错误输出: %s", err, stderr)
		}
	}

	logger.Infof("修改 Service 文件版本号为: %s\n", u.NewVersion)
	newContent := fmt.Sprintf("Description = Redis %s database server", u.NewVersion)
	if err := command.ReplaceLineWithKeyword(u.Servicefile, "Description", newContent); err != nil {
		return err
	}

	// service reload
	if err := command.SystemdReload(); err != nil {
		return err
	}

	logger.Infof("开始启动实例\n")
	if err := command.SystemCtl(service, "start"); err != nil {
		return err
	}

	return nil
}
