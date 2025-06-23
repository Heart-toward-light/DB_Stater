/*
@Author : WuWeiJian
@Date : 2021-03-26 17:49
*/

package services

import (
	"bufio"
	"bytes"
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/pgsql/config"
	"dbup/internal/pgsql/dao"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-playground/validator"
)

type PGManager struct {
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	AdminDatabase string
	User          string
	Password      string
	DBName        string
	Address       string
	Role          string
	Ignore        bool
	Conn          *dao.PgConn
}

func NewPGManager() *PGManager {
	return &PGManager{}
}

func (p *PGManager) InitConn() error {
	if p.AdminDatabase == "" {
		p.AdminDatabase = p.AdminUser
	}

	conn, err := dao.NewPgConn(p.Host, p.Port, p.AdminUser, p.AdminPassword, p.AdminDatabase)
	if err != nil {
		return err
	}
	p.Conn = conn
	return nil
}

func (p *PGManager) ValidatorAddress() error {
	addrs := strings.Split(p.Address, ",")
	v := validator.New()
	for _, addr := range addrs {
		if addr == "localhost" || addr == "local" {
			continue
		}
		if err := utils.CheckAddressFormat(addr); err != nil {
			if e := v.Var(addr, "hostname"); e != nil {
				return fmt.Errorf("%s 不是有效IP地址, 也不是规范主机名", addr)
			}
		}
	}
	return nil
}

//func (p *PGManager) UserCreate() error {
//	if err := p.CreateUser(); err != nil {
//		return err
//	}
//
//	if p.DBName == "" {
//		return nil
//	}
//
//	if err := p.CreateDB(); err != nil {
//		return err
//	}
//
//	if p.Address == "" {
//		return nil
//	}
//
//	if err := p.Grant(); err != nil {
//		return err
//	}
//	return nil
//}

func (p *PGManager) DBUserCreate() error {
	exist, err := p.Conn.DBExist(p.DBName)
	if err != nil {
		return err
	}

	// if p.Role == "dbuser" {
	if exist {
		if err := p.Conn.GrantDBUser(p.User, p.DBName); err != nil {
			return err
		}
	} else {
		if err := p.Conn.CreateDBUser(p.User, p.DBName); err != nil {
			return err
		}
	}
	// }

	return nil
}

func (p *PGManager) DatabaseCreate() error {
	exist, err := p.Conn.DBExist(p.DBName)
	if err != nil {
		return err
	}

	if exist {
		if p.Ignore {
			return nil
		}
		return fmt.Errorf("库已经存在\n")
	}

	return p.Conn.CreateDB(p.DBName)
}

func (p *PGManager) UserCreate() error {
	exist, err := p.Conn.UserExist(p.User)
	if err != nil {
		return err
	}

	if exist {
		if p.Ignore {
			return nil
		}
		return fmt.Errorf("用户已经存在\n")
	}

	priv := ""
	switch p.Role {
	case "dbuser":
		priv = config.DefaultPGDBUserPriv
	case "normal":
		priv = config.DefaultPGUserPriv
	case "replication":
		priv = config.DefaultPGReplPriv
	case "admin":
		priv = config.DefaultPGHidePriv
	default:
		return fmt.Errorf("角色不正确")
	}

	return p.Conn.CreateUser(p.User, p.Password, priv)
}

func (p *PGManager) UserDelete() error {
	return nil
}

func (p *PGManager) UserGrant() error {
	if p.Role == "dbuser" {
		if err := p.DBUserCreate(); err != nil {
			return err
		}
	}

	if err := p.ValidatorAddress(); err != nil {
		return err
	}

	hbaFile, err := p.Conn.PGHbaFilePath()
	if err != nil {
		return err
	}

	autofile := strings.Replace(hbaFile, config.PgHbaFileName, config.PGautofaile, -1)
	if utils.IsExists(autofile) {
		return fmt.Errorf(" pg_auto_failover 角色的节点,不能使用此授权方式")
	}

	hba := config.NewPgHba()
	if err := hba.Load(hbaFile); err != nil {
		return err
	}

	p.Address = "localhost," + p.Address
	addrs := strings.Split(p.Address, ",")
	for _, addr := range addrs {
		if addr == "localhost" || addr == "local" {
			find := hba.FindRecordByTypeAndUserAndDBAndAddr("local", p.User, p.DBName, "")
			if len(find) == 0 {
				hba.AddR("local", p.User, p.DBName, "")
			}
		} else {
			var ipm string
			if err := utils.CheckAddressFormat(addr); err == nil {
				ipm = utils.IpAddMaskIfNot(addr)
			} else {
				ipm = addr
			}
			find := hba.FindRecordByTypeAndUserAndDBAndAddr("host", p.User, p.DBName, ipm)
			if len(find) == 0 {
				hba.AddRecord(p.User, p.DBName, ipm)
			}
		}
	}

	if err := hba.SaveTo(hbaFile); err != nil {
		return err
	}

	return p.Conn.ReloadConfig()
}

func (p *PGManager) AlterUserExpireAt(username, expireAt string) error {
	return p.Conn.AlterUserExpireAt(username, expireAt)
}

func (p *PGManager) Revoke() error {
	return nil
}

func (p *PGManager) CheckSlaves(slaves string) error {
	slave := strings.Split(slaves, ",")
	repls, err := p.Conn.ReplicationIp()
	if err != nil {
		return err
	}

	// 支持域名后, 传入参数是域名, pgsql集群内部状态记录为IP地址, 导致匹配不到的问题, 所以改用从库数量对比方式
	//for _, s := range slave {
	//	if arrlib.InArray(s, repls) {
	//		logger.Successf("从库: %s OK\n", s)
	//	} else {
	//		return fmt.Errorf("未在集群中找到从库IP: %s\n", s)
	//	}
	//}

	if len(repls) == 0 || len(slave) != len(repls) {
		return fmt.Errorf("检查从库(%s)数量不正常", repls)
	}

	return nil
}

func (p *PGManager) CheckSelect() error {
	return p.Conn.Select()
}

func (p *PGManager) AddSlave(ssho global.SSHConfig, pre config.Prepare, master string) error {

	logger.Infof("验证参数\n")
	if err := ssho.Validator(); err != nil {
		return err
	}

	if ssho.TmpDir == "" {
		ssho.TmpDir = config.DeployTmpDir
	}

	if ssho.Password == "" && ssho.KeyFile == "" {
		ssho.KeyFile = filepath.Join(environment.GlobalEnv().HomePath, ".ssh", "id_rsa")
	}

	var node *Instance
	var err error
	if ssho.Password != "" {
		node, err = NewInstance(ssho.TmpDir,
			ssho.Host,
			ssho.Username,
			ssho.Password,
			ssho.Port,
			pre,
			0)
		if err != nil {
			return err
		}
	} else {
		node, err = NewInstanceUseKeyFile(ssho.TmpDir,
			ssho.Host,
			ssho.Username,
			ssho.KeyFile,
			ssho.Port,
			pre,
			0)
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

	logger.Infof("开始安装 PGSQL 从库实例\n")
	if err := node.InstallSlave(pre, master); err != nil {
		logger.Warningf("安装失败\n")
		// _ = node.UNInstall(pre)
		return err
	}

	logger.Infof("新从库节点: %s:%d 添加成功\n", ssho.Host, pre.Port)

	return nil
}

func (p *PGManager) Promote(wait string, seconds int) error {
	if err := p.InitConn(); err != nil {
		return err
	}
	defer p.Conn.DB.Close()

	t, err := p.Conn.Promote("true", 60)
	if err != nil {
		return err
	}

	if !t {
		return errors.New("提升为主库失败")
	}

	return nil
}

func (p *PGManager) ConfigChangeRepmgr() error {
	ConfigFile, err := p.Conn.PGFilePath()
	if err != nil {
		return err
	}

	input, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		return err
	}

	output := bytes.Replace(input, []byte("'pg_stat_statements'"), []byte("'pg_stat_statements,repmgr'"), -1)

	if err = ioutil.WriteFile(ConfigFile, output, 0666); err != nil {
		return fmt.Errorf("文件写入异常 %s", err)
	}

	return nil
}

// 检查用户授权规范
func (p *PGManager) CheckUserChar() error {
	r, _ := regexp.Compile(config.RegexpUsername)
	if ok := r.MatchString(p.User); !ok {
		return fmt.Errorf("用户名(%s)不符合规则: 2到63位小写字母,数字,下划线; 不能以数字开头", p.User)
	}

	ps, _ := regexp.Compile(config.RegexpSpecialChar)
	index := ps.FindStringIndex(p.Password)
	if index != nil {
		return fmt.Errorf("密码不能包含特殊字符(: / + @ ? & =), 随机示例: %s", utils.GeneratePasswd(16))
	}

	return nil
}

// 检查DB名称规范
func (p *PGManager) CheckDBChar() error {
	r, _ := regexp.Compile(config.RegexpSpecialChar)
	index := r.FindStringIndex(p.DBName)
	if index != nil {
		return fmt.Errorf("数据库名不能包含特殊字符(: / + @ ? & =)")
	}

	return nil
}

func (p *PGManager) AutofailoverGrantuser() error {
	if err := p.DBUserCreate(); err != nil {
		return err
	}

	if err := p.ValidatorAddress(); err != nil {
		return err
	}

	hbaFile, err := p.Conn.PGHbaFilePath()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(hbaFile, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("hba 文件打开失败 %s\n", err)
	}

	defer file.Close()

	// 验证授权的地址是否重复
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		for _, addr := range strings.Split(p.Address, ",") {
			localhbatest := fmt.Sprintf("local   %s     %s                          md5", p.DBName, p.User)
			if addr == "localhost" || addr == "local" {
				if strings.Contains(scanner.Text(), localhbatest) {
					return fmt.Errorf("授权文件 %s 已经包含用户 %s 的 local 授权", hbaFile, p.User)
				}
			} else {
				var ipm string
				if err := utils.CheckAddressFormat(addr); err == nil {
					ipm = utils.IpAddMaskIfNot(addr)
				} else {
					ipm = addr
				}
				hosthbatest := fmt.Sprintf("host    %s     %s   %s               md5", p.DBName, p.User, ipm)
				if strings.Contains(scanner.Text(), hosthbatest) {
					return fmt.Errorf("授权文件 %s 已经包含用户 %s 的 %s 授权", hbaFile, p.User, addr)
				}
			}
		}
	}

	reader := bufio.NewReader(file)
	for {
		_, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("读取文件失败: %s\n", err)
		}
	}

	var localhba string
	var hosthba string

	addrs := strings.Split(p.Address, ",")

	for _, addr := range addrs {
		// logger.Warningf("addrs 的值: %s ", addr)
		write := bufio.NewWriter(file)
		// scanner := bufio.NewScanner(file)
		if addr == "localhost" || addr == "local" {
			localhba = fmt.Sprintf("\nlocal   %s     %s                          md5\n", p.DBName, p.User)
			write.WriteString(localhba)
		} else {
			var ipm string
			if err := utils.CheckAddressFormat(addr); err == nil {
				ipm = utils.IpAddMaskIfNot(addr)
			} else {
				ipm = addr
			}
			hosthba = fmt.Sprintf("host    %s     %s   %s               md5\n", p.DBName, p.User, ipm)
			write.WriteString(hosthba)
		}
		write.Flush()
	}

	// 修改 hba 文件所有内容 trust 为 md5
	if err = p.Changetrust(hbaFile); err != nil {
		return fmt.Errorf("修改所有trust为md5失败: %s", err)
	}

	return p.Conn.ReloadConfig()
}

func (p *PGManager) Changetrust(hbaFile string) error {
	content, err := ioutil.ReadFile(hbaFile)
	if err != nil {
		return fmt.Errorf("读取文件失败: %s\n", err)
	}

	// 将文件内容转换为字符串
	fileContent := string(content)

	// 替换指定内容
	newContent := strings.Replace(fileContent, "trust", "md5", -1)

	// 将修改后的内容写回文件
	err = ioutil.WriteFile(hbaFile, []byte(newContent), 0666)
	if err != nil {
		return fmt.Errorf("写入文件失败: %s\n", err)
	}

	return nil
}
