// Created by LiuSainan on 2022-06-14 15:35:52

package dao

import (
	"database/sql"
	"dbup/internal/utils/logger"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MariaDBConn struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	Charset  string
	URI      string
	Errornum int
	DB       *sql.DB
}

func NewMariaDBConn(host string, port int, user, password, dbname string) (*MariaDBConn, error) {
	url := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local", user, password, host, port, dbname, "utf8mb4")
	stat := false
	for i := 1; i <= 20; i++ {
		conn, err := sql.Open("mysql", url)
		// 连接Mariadb数据库
		if err != nil {
			logger.Warningf("无法连接到Mariadb数据库：%v\n", err)
		} else {
			// 尝试执行一条查询语句来验证连接是否正常
			err := conn.Ping()
			if err != nil {
				logger.Warningf("Mariadb数据库连接异常：%v\n", err)
			} else {
				stat = true
				break
			}

			// 关闭数据库连接
			conn.Close()
		}

		// 等待一段时间后再进行下一次判断
		time.Sleep(6 * time.Second)
	}

	if !stat {
		return nil, fmt.Errorf("mariadb 数据库多次连接失败")
	}

	conn, err := sql.Open("mysql", url)
	connector := &MariaDBConn{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbname,
		Charset:  "utf8mb4",
		URI:      url,
		DB:       conn,
	}
	return connector, err
}

func (p *MariaDBConn) ChangePassword(user, host, password string) error {
	sql := fmt.Sprintf("set password for %s@%s = password('%s');", user, host, password)
	_, err := p.DB.Query(sql)
	return err
}

func (p *MariaDBConn) CreateUser(user, host, password string) error {
	sql := fmt.Sprintf("CREATE USER '%s'@'%s' identified by '%s';", user, host, password)
	_, err := p.DB.Query(sql)
	return err
}

func (p *MariaDBConn) Grant(user, host, privileges string) error {
	sql := fmt.Sprintf("GRANT %s ON *.* TO '%s'@'%s';", privileges, user, host)
	_, err := p.DB.Query(sql)
	return err
}

func (p *MariaDBConn) ChangeMasterTo(host string, port int, user, password string) error {
	sql := fmt.Sprintf("CHANGE MASTER TO MASTER_HOST='%s', MASTER_PORT=%d, MASTER_USER='%s', MASTER_PASSWORD='%s', MASTER_USE_GTID=slave_pos;",
		host,
		port,
		user,
		password)
	logger.Warningf(sql)
	_, err := p.DB.Query(sql)
	return err
}

func (p *MariaDBConn) StartSlave() error {
	sql := "start SLAVE ;"
	_, err := p.DB.Query(sql)
	return err
}

func (p *MariaDBConn) FlushUser() error {
	sql := "flush privileges;"
	_, err := p.DB.Query(sql)
	return err
}

func (p *MariaDBConn) CloseReadonly() error {
	sql := "set global read_only = 0 ;"
	_, err := p.DB.Query(sql)
	return err
}

func (p *MariaDBConn) Select() error {
	sql := "select 1;"
	_, err := p.DB.Query(sql)
	return err
}

func (p *MariaDBConn) ShowSlaveStatus() (status map[string]sql.NullString, err error) {
	sql := "SHOW SLAVE STATUS;"
	rows, err := p.DB.Query(sql)
	if err != nil {
		return status, err
	}

	return ScanMap(rows)
}

func (p *MariaDBConn) Version() (string, error) {
	var version string
	err := p.DB.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return "", err
	}
	myversion := strings.Split(version, "-")[0]
	return myversion, nil
}

func (p *MariaDBConn) Check_table(tableName string) error {
	var TableName, Op, Msg_type, Msg_text string
	checkQuery := fmt.Sprintf("CHECK TABLE %s", tableName)

	rows, err := p.DB.Query(checkQuery)
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&TableName, &Op, &Msg_type, &Msg_text)
		if err != nil {
			logger.Errorf("CHECKS 表结果异常: %s", err)
		}
		if Msg_text != "OK" {
			p.Errornum += 1
			logger.Warningf("表 %s 存在异常: %s\n", TableName, Msg_text)
		}
	}

	// defer wg.Done()
	return nil
}

func (p *MariaDBConn) Parallel_check_table() error {
	rows, err := p.DB.Query("select concat(TABLE_SCHEMA,'.',TABLE_NAME) from information_schema.tables where TABLE_SCHEMA not in ('information_schema','mysql','performance_schema','sys')")
	if err != nil {
		return err

	}
	var wg sync.WaitGroup
	concurrency := 10                             // 最大并发数量
	semaphore := make(chan struct{}, concurrency) // 控制并发的信号量

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return err
		}
		semaphore <- struct{}{} // 获取一个信号量
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量
			if err := p.Check_table(tableName); err != nil {
				logger.Errorf("CHECKS 表异常: %s", err)
			} // 调用处理函数
		}()

	}
	wg.Wait() // 等待所有goroutines完成
	return nil
}
