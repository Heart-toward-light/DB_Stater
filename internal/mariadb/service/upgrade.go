package service

import (
	"bufio"
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/mariadb/config"
	"dbup/internal/mariadb/dao"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type UPgrade struct {
	Host        string
	Port        int
	Username    string
	Password    string
	SourceDir   string
	EmoloyDir   string
	OldVersion  string
	NewVersion  string
	Servicefile string
	TxIsolation string
	NoBackup    bool
	Default     bool
	Upsettings  []string
	Conn        *command.Connection
	Yes         bool
}

func NewUPgrade() *UPgrade {
	return &UPgrade{
		Upsettings: []string{"bin", "lib", "man", "scripts", "share", "support-files"},
		Default:    false,
	}
}

func (u *UPgrade) Validator() error {
	logger.Infof("验证参数\n")
	// Upsettings := []string{"bin", "lib", "man", "scripts", "share", "support-files"}
	// upgradeEmoloyfile := filepath.Join(u.EmoloyDir, "bin", config.DefaultMariaDBUPgradeBinFile)
	// u.EmoloyDir = config.TmpDir
	service := fmt.Sprintf(config.ServiceFileName, u.Port)
	servicePath := global.ServicePath
	serviceFileFullName := filepath.Join(servicePath, service)

	if !command.IsExists(serviceFileFullName) {
		return fmt.Errorf("实例的 service 启停文件 %s 不存在", service)
	}
	u.Servicefile = serviceFileFullName

	// if !command.IsExists(upgradeEmoloyfile) {
	// 	return fmt.Errorf("新版本实例升级文件 %s 不存在", upgradeEmoloyfile)
	// }

	if u.SourceDir == "" {
		return fmt.Errorf("请指定旧版本的安装主目录路径")
	}

	if u.EmoloyDir == "" {
		return fmt.Errorf("请指定新版本的安装主目录路径")
	}

	if err := command.VerifyDir(u.SourceDir, u.Upsettings); err != nil {
		return err
	}

	conn, err := dao.NewMariaDBConn(u.Host, u.Port, u.Username, u.Password, "")
	if err != nil {
		return fmt.Errorf("mariadb 指定的管理账号 %s 连接异常", u.Username)
	}

	// 检查是否存在表损坏等异常
	if err := conn.Parallel_check_table(); err != nil {
		return fmt.Errorf("检查表异常: %v", err)
	}

	if conn.Errornum > 0 {
		return fmt.Errorf("业务表出现异常,请先恢复表,再升级")
	}

	// 查询当前实例版本
	if err := conn.Select(); err != nil {
		return fmt.Errorf("账号 %s 连接失败: %v", u.Username, err)
	}
	defer conn.DB.Close()

	/* 1. 验证用户输入的事务隔离级别 */
	if u.TxIsolation == "" || u.TxIsolation == "RC" {
		u.TxIsolation = "READ-COMMITTED"
	} else if u.TxIsolation == "RR" {
		u.TxIsolation = "REPEATABLE-READ"
	} else {
		return fmt.Errorf("事务隔离级别(%s), 必须为 RR 或 RC ", u.TxIsolation)
	}

	// 同版本比较函数放到最后 return newVersion0,1 err info
	mariadbfile := filepath.Join(u.EmoloyDir, "bin", config.DefaultMariaDBBinFile)
	if !command.IsExists(mariadbfile) {
		u.Default = true
		if err := u.VersionComparison(); err != nil {
			return err
		}
	} else {
		upgradeEmoloyfile := filepath.Join(u.EmoloyDir, "bin", config.DefaultMariaDBUPgradeBinFile)
		if !command.IsExists(upgradeEmoloyfile) {
			return fmt.Errorf("新版本实例升级文件 %s 不存在", upgradeEmoloyfile)
		}

		if err := command.VerifyDir(u.EmoloyDir, u.Upsettings); err != nil {
			return err
		}

		if err := u.VersionComparison(); err != nil {
			return err
		}
	}

	return nil
}

func (u *UPgrade) Run() error {
	var newMariadbVersion string
	if err := u.Validator(); err != nil {
		// 不包含时 newVersion 返回 err
		if !strings.Contains(err.Error(), "newVersion") {
			return err
		}
		// 包含 newVersion 时, 不返回 err 继续
		newMariadbVersion = err.Error()
	}

	// 非 最新版本 需要升级到 最新版本
	if !strings.Contains(newMariadbVersion, "newVersion") {
		if !u.Yes {
			var yes string
			logger.Warningf("升级版本需要重启实例: %s:%d\n", u.Host, u.Port)
			// logger.Warningf("mariadb 从当前版本 %s 升级至 %s \n", u.OldVersion, u.NewVersion)
			logger.Warningf("是否确认重启进行升级[y|n]:")
			if _, err := fmt.Scanln(&yes); err != nil {
				return err
			}
			if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
				os.Exit(0)
			}
		}

		logger.Infof("开始升级\n")
		if err := u.UPgradeDB(); err != nil {
			u.RemoveTmp()
			return err
		}
		logger.Successf("升级完成\n")
	} else {
		// 已为 最新版本, 检测配置 jemalloc
		if !u.Yes {
			var yes string
			logger.Warningf("配置jemalloc需要重启实例: %s:%d\n", u.Host, u.Port)
			logger.Warningf("是否确认重启进行jemalloc配置[y|n]:")
			if _, err := fmt.Scanln(&yes); err != nil {
				return err
			}
			if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
				os.Exit(0)
			}
		}

		logger.Infof("开始配置 jemalloc, slave_parallel_threads\n")
		// copy libjemalloc文件 并 配置 systemctl service
		if err := u.UPConfig(); err != nil {
			u.RemoveTmp()
			return err
		}
		logger.Successf("配置 jemalloc,slave_parallel_threads 完成\n")
	}

	return nil
}

func (u *UPgrade) RemoveTmp() {
	if u.Default {
		logger.Infof("删除临时目标升级目录\n")
		os.RemoveAll(u.EmoloyDir)
	}
	// return nil
}

func (u *UPgrade) InitDefaultpkg() error {
	logger.Infof("开始检查默认升级版本 %s 的升级包\n", config.DefaultMariaDBVersion)
	PackageFullName := filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultMariaDBVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	u.EmoloyDir = filepath.Join(u.EmoloyDir, "mariadb_tmp_dir")
	if !command.IsExists(u.EmoloyDir) {
		if err := os.MkdirAll(u.EmoloyDir, 0755); err != nil {
			return err
		}
		// 解压安装包
		logger.Infof("解压升级包: %s 到 %s \n", PackageFullName, u.EmoloyDir)
		if err := utils.UntarGz(PackageFullName, u.EmoloyDir); err != nil {
			return err
		}
	} else {
		logger.Errorf("新版本的安装主目录的临时路径 %s 已存在\n", u.EmoloyDir)
		// return fmt.Errorf("新版本的安装主目录的临时路径 %s 已存在", u.EmoloyDir)
	}

	return nil
}

func (u *UPgrade) VersionComparison() error {
	var err error
	u.OldVersion, err = command.MariadbVersion(u.SourceDir)
	if err != nil {
		return err
	}

	if u.Default {
		u.NewVersion = config.DefaultMariaDBVersion
	} else {
		u.NewVersion, err = command.MariadbVersion(u.EmoloyDir)
		if err != nil {
			return err
		}
	}

	result := command.CompareVersion(u.OldVersion, u.NewVersion)

	switch result {
	case 1:
		return fmt.Errorf("newVersion%d", result)
		// logger.Errorf("Mariadb 老版本 %s 不能大于新版本 %s \n", u.OldVersion, u.NewVersion)
	case 0:
		return fmt.Errorf("newVersion%d", result)
		// logger.Errorf("Mariadb 老版本 %s 不能等于新版本 %s \n", u.OldVersion, u.NewVersion)
	}
	return nil
}

func (u *UPgrade) UPgradeDB() error {
	if u.Default {
		if err := u.InitDefaultpkg(); err != nil {
			return err
		}
	}
	service := fmt.Sprintf(config.ServiceFileName, u.Port)
	logger.Infof("开始关停老版本实例\n")
	if err := command.SystemCtl(service, "stop"); err != nil {
		return err
	}

	// 选择备份时
	if !u.NoBackup {
		logger.Infof("开始替换老版本实例依赖文件\n")
		for _, Libdir := range u.Upsettings {
			emoloyDir := filepath.Join(u.EmoloyDir, Libdir)
			serviceFileFullName := filepath.Join(u.SourceDir, Libdir)
			if err := command.MoveFile(serviceFileFullName); err != nil {
				return fmt.Errorf("修改依赖文件路径 %s 失败: %s", serviceFileFullName, err)
			}
			if err := command.MoveDir(emoloyDir, u.SourceDir); err != nil {
				return err
			}
			time.Sleep(3 * time.Second)
		}
	} else {
		logger.Infof("开始删除老版本实例依赖文件\n")
		for _, Libdir := range u.Upsettings {
			emoloyDir := filepath.Join(u.EmoloyDir, Libdir)
			serviceFileFullName := filepath.Join(u.SourceDir, Libdir)
			if err := os.RemoveAll(serviceFileFullName); err != nil {
				return err
			}
			if err := command.MoveDir(emoloyDir, u.SourceDir); err != nil {
				return err
			}
			time.Sleep(3 * time.Second)
		}
	}

	/* 1. 检测 mariadb my.cnf 文件是否包含 slave_parallel_threads 配置,如不包含添加*/
	logger.Infof("修改 my.cnf slave_parallel_threads\n")
	sourceConfigdir := filepath.Join(u.SourceDir, "config")
	if err := u.checkSlaveParallelCnf(sourceConfigdir); err != nil {
		return fmt.Errorf("修改 my.cnf slave_parallel_threads 发生错误: %v", err)
	}

	/* 2. 检测 libjemalloc是否存在并添加配置 */
	logger.Infof("修改 service libjemalloc\n")
	if err := u.checkjemallocService(u.Servicefile); err != nil {
		return fmt.Errorf("修改 service libjemalloc 发生错误: %v", err)
	}

	/* 3. 检测 transaction_isolation 并添加或修改配置 */
	logger.Infof("修改 my.cnf transaction_isolation\n")
	sourceConfigdir = filepath.Join(u.SourceDir, "config")
	targetLine := "transaction_isolation"
	replaceLine := fmt.Sprintf("transaction_isolation                                      = %s\n", u.TxIsolation)
	if err := u.checkTxIsolationCnf(sourceConfigdir, targetLine, replaceLine); err != nil {
		return fmt.Errorf("修改 my.cnf transaction_isolation 发生错误: %v", err)
	}

	// 修改启动模式
	newstartmode := "Type                 = simple"
	if err := command.ReplaceLineWithKeyword(u.Servicefile, "notify", newstartmode); err != nil {
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

	// service reload
	if err := command.SystemdReload(); err != nil {
		return err
	}

	logger.Infof("开始启动老版本实例\n")
	if err := command.SystemCtl(service, "start"); err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	logger.Infof("修改 service 文件 mariadb 版本号\n")
	if err := u.RewriteService(); err != nil {
		return err
	}

	// service reload
	if err := command.SystemdReload(); err != nil {
		return err
	}

	upgradefile := filepath.Join(u.SourceDir, "bin", config.DefaultMariaDBUPgradeBinFile)
	logger.Infof("开始升级老版本实例\n")
	cmd := fmt.Sprintf("%s --user=%s --password='%s' --host=%s --port=%d", upgradefile, u.Username, u.Password, u.Host, u.Port)
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("执行 mariadb 升级失败: %v, 标准错误输出: %s", err, stderr)
	}

	u.RemoveTmp()

	return nil
}

// mariadb >= 10.11.5 时 copy libjemalloc.so 目录文件 并 修改service libjemalloc 及 slave_parallel_threads
func (u *UPgrade) UPConfig() error {
	/*1. 判断 os/libjemalloc.so 是否存在，并 mv */
	if u.Default {
		if err := u.InitDefaultpkg(); err != nil {
			return err
		}
	}
	service := fmt.Sprintf(config.ServiceFileName, u.Port)
	logger.Infof("开始关停老版本实例\n")
	if err := command.SystemCtl(service, "stop"); err != nil {
		return err
	}

	// copy lib文件
	osinfo, _, _, err := command.GetOsArchInfo()
	if err != nil {
		return fmt.Errorf("获取操作系统,cpu架构信息发生错误: %v", err)
	}

	emoloyDir := filepath.Join(u.EmoloyDir, "lib", osinfo)
	serviceFileFullName := filepath.Join(u.SourceDir, "lib", osinfo)

	/* 1. 判断 os/libjemalloc.so 如不存在 mv /tmp /datadir/lib/ 下 */
	_, err = os.Stat(serviceFileFullName)
	if os.IsNotExist(err) {
		if err := command.MoveDir(emoloyDir, serviceFileFullName); err != nil {
			return err
		}
	}

	time.Sleep(3 * time.Second)

	/* 2. 检测 mariadb my.cnf 文件是否包含 slave_parallel_threads 配置,如不包含添加*/
	logger.Infof("修改 my.cnf slave_parallel_threads\n")
	sourceConfigdir := filepath.Join(u.SourceDir, "config")
	if err := u.checkSlaveParallelCnf(sourceConfigdir); err != nil {
		return fmt.Errorf("修改 my.cnf slave_parallel_threads 发生错误: %v", err)
	}

	/* 3. 检测 libjemalloc是否存在并添加配置*/
	logger.Infof("修改 service libjemalloc\n")
	if err := u.checkjemallocService(u.Servicefile); err != nil {
		return fmt.Errorf("修改 service libjemalloc 发生错误: %v", err)
	}

	/*4. 检测 transaction_isolation 并添加或修改配置 */
	logger.Infof("修改 my.cnf transaction_isolation\n")
	sourceConfigdir = filepath.Join(u.SourceDir, "config")
	targetLine := "transaction_isolation"
	replaceLine := fmt.Sprintf("transaction_isolation                                      = %s\n", u.TxIsolation)
	if err := u.checkTxIsolationCnf(sourceConfigdir, targetLine, replaceLine); err != nil {
		return fmt.Errorf("修改 my.cnf transaction_isolation 发生错误: %v", err)
	}

	// service reload
	if err := command.SystemdReload(); err != nil {
		return err
	}

	// 更改文件后,修改为原文件的 chown
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

	logger.Infof("开始启动老版本实例\n")
	if err := command.SystemCtl(service, "start"); err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	u.RemoveTmp()

	return nil
}

// // 配置 jemalloc
// func (u *UPgrade) ServiceConfigjemalloc() error {
// 	service := fmt.Sprintf(config.ServiceFileName, u.Port)
// 	// 检测 libjemalloc是否存在并添加配置
// 	logger.Infof("修改 service libjemalloc\n")
// 	if err := u.checkjemallocService(u.Servicefile); err != nil {
// 		return fmt.Errorf("修改 service libjemalloc 发生错误: %v", err)
// 	}
// 	// service reload
// 	if err := command.SystemdReload(); err != nil {
// 		return err
// 	}
// 	logger.Infof("重启实例\n")
// 	if err := command.SystemCtl(service, "restart"); err != nil {
// 		return err
// 	}

// 	return nil
// }

func (u *UPgrade) RewriteService() error {
	conn, err := dao.NewMariaDBConn(u.Host, u.Port, u.Username, u.Password, "")
	if err != nil {
		return fmt.Errorf("mariadb 指定的管理账号 %s 连接异常", u.Username)
	}

	if version, err := conn.Version(); err != nil {
		return fmt.Errorf("查询版本号失败: %v", err)
	} else {
		newContent := fmt.Sprintf("Description   = MariaDB %s database server", version)
		newstartmode := "Type                 = simple"
		if err := command.ReplaceLineWithKeyword(u.Servicefile, "Description", newContent); err != nil {
			return err
		}
		if err := command.ReplaceLineWithKeyword(u.Servicefile, "notify", newstartmode); err != nil {
			return err
		}
	}

	defer conn.DB.Close()

	return nil
}

// 检测 mariadb service 文件是否包含 jemalloc配置,如不包含添加
func (u *UPgrade) checkjemallocService(filePath string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("无法打开文件: %v", err)
	}
	defer file.Close()

	// 逐行读取文件内容
	scanner := bufio.NewScanner(file)
	var fileContent string
	for scanner.Scan() {
		line := scanner.Text()
		fileContent += line + "\n"
		if strings.Contains(line, "libjemalloc") {
			// 文件已包含 libjemalloc,跳过添加
			return nil
		}
		// 在TasksMax 行后添加 libjemalloc 配置
		if strings.Contains(line, "TasksMax") {
			osinfo, arch, jemalloc, err := command.GetOsArchInfo()
			if err != nil {
				return fmt.Errorf("获取操作系统,cpu架构信息发生错误: %v", err)
			}
			jemallocService := fmt.Sprintf("Environment          = LD_PRELOAD=%s", filepath.Join(u.SourceDir, "lib", osinfo, arch, jemalloc))
			fileContent += fmt.Sprintf("%s\n", jemallocService)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取文件时发生错误: %v", err)
	}

	// 将修改后的内容写入临时文件
	tmpfile, err := os.CreateTemp("/usr/lib/systemd/system/", "tempfile")
	if err != nil {
		return fmt.Errorf("无法创建临时文件: %v", err)
	}
	defer tmpfile.Close()

	if _, err := tmpfile.Write([]byte(fileContent)); err != nil {
		return fmt.Errorf("写入临时文件时发生错误: %v", err)
	}

	// 替换原始文件内容
	if err := os.Rename(tmpfile.Name(), filePath); err != nil {
		return fmt.Errorf("替换文件内容时发生错误: %v", err)
	}

	return nil
}

// 检测 mariadb my.cnf 文件是否包含 slave_parallel_threads 配置,如不包含添加
func (u *UPgrade) checkSlaveParallelCnf(sourceConfigdir string) error {
	// 打开文件
	filePath := filepath.Join(sourceConfigdir, "my.cnf")
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("无法打开文件: %v", err)
	}
	defer file.Close()

	// 逐行读取文件内容
	scanner := bufio.NewScanner(file)
	var fileContent string
	for scanner.Scan() {
		line := scanner.Text()
		fileContent += line + "\n"
		if strings.Contains(line, "slave_parallel_threads") {
			// 文件已包含 slave_parallel_threads,跳过添加
			return nil
		}
		// 在slave_transaction_retries 行后添加 slave_parallel_threads 配置
		if strings.Contains(line, "slave_transaction_retries") {
			slavePara := fmt.Sprintf("slave_parallel_threads                                     = %d", 4)
			fileContent += fmt.Sprintf("%s\n", slavePara)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取文件时发生错误: %v", err)
	}

	// 将修改后的内容写入临时文件
	tmpfile, err := os.CreateTemp(sourceConfigdir, "tempfile")
	if err != nil {
		return fmt.Errorf("无法创建临时文件: %v", err)
	}
	defer tmpfile.Close()

	if _, err := tmpfile.Write([]byte(fileContent)); err != nil {
		return fmt.Errorf("写入临时文件时发生错误: %v", err)
	}

	// 替换原始文件内容
	if err := os.Rename(tmpfile.Name(), filePath); err != nil {
		return fmt.Errorf("替换文件内容时发生错误: %v", err)
	}

	return nil
}

// 检测 mariadb my.cnf 文件是否包含 transaction_isolation 配置,包含替换,不包含则添加
func (u *UPgrade) checkTxIsolationCnf(sourceConfigdir string, targetLine string, replaceLine string) error {
	filePath := filepath.Join(sourceConfigdir, "my.cnf")
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件出错: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	found := false

	// 寻找目标行
	for i, line := range lines {
		if strings.Contains(line, targetLine) {
			lines[i] = replaceLine
			found = true
			break
		}
	}

	// 如果未找到目标行，则添加新行
	if !found {
		lines = append(lines, replaceLine)
	}

	// 重写文件
	var newContent strings.Builder
	for i, line := range lines {
		if i > 0 && i != len(lines)-1 {
			newContent.WriteString("\n")
		}
		newContent.WriteString(line)
	}
	err = os.WriteFile(filePath, []byte(newContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("写入文件出错: %v", err)
	}

	return nil

}

// 验证后即可删除此函数
// func (u *UPgrade) checkTxIsolationCnf(sourceConfigdir string) error {
// 	// 打开文件
// 	filePath := filepath.Join(sourceConfigdir, "my.cnf")
// 	file, err := os.Open(filePath)
// 	if err != nil {
// 		return fmt.Errorf("无法打开文件: %v", err)
// 	}
// 	defer file.Close()

// 	// 逐行读取文件内容
// 	scanner := bufio.NewScanner(file)
// 	var fileContent string
// 	for scanner.Scan() {
// 		line := scanner.Text()

// 		// 文件已包含 transaction_isolation,替换整行
// 		if strings.Contains(line, "transaction_isolation") {
// 			line = fmt.Sprintf("transaction_isolation                                     = %s", u.TxIsolation)
// 			fileContent += line + "\n"
// 			// return nil
// 		} else {
// 			// 文件不包含 transaction_isolation, 在innodb_buffer_pool_dump_pct行后添加transaction_isolation配置
// 			if strings.Contains(line, "innodb_buffer_pool_dump_pct") {
// 				fileContent += line + "\n"
// 				txIsoPara := fmt.Sprintf("transaction_isolation                                     = %s", u.TxIsolation)
// 				fileContent += fmt.Sprintf("%s\n", txIsoPara)
// 			}

// 		}
// 	}

// 	if err := scanner.Err(); err != nil {
// 		return fmt.Errorf("读取文件时发生错误: %v", err)
// 	}

// 	// 将修改后的内容写入临时文件
// 	tmpfile, err := os.CreateTemp(sourceConfigdir, "tempfile")
// 	if err != nil {
// 		return fmt.Errorf("无法创建临时文件: %v", err)
// 	}
// 	defer tmpfile.Close()

// 	if _, err := tmpfile.Write([]byte(fileContent)); err != nil {
// 		return fmt.Errorf("写入临时文件时发生错误: %v", err)
// 	}

// 	// 替换原始文件内容
// 	if err := os.Rename(tmpfile.Name(), filePath); err != nil {
// 		return fmt.Errorf("替换文件内容时发生错误: %v", err)
// 	}

// 	return nil
// }
