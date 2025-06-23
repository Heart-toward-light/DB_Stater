/*
@Author : WuWeiJian
@Date : 2020-12-24 11:59
*/

package dao

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type PgConn struct {
	Host     string
	Port     int
	User     string
	Password string
	Dbname   string
	Info     string
	DB       *sql.DB
}

func NewPgConn(host string, port int, user, password, dbname string) (*PgConn, error) {
	var err error
	c := &PgConn{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Dbname:   dbname,
	}
	c.Info = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", c.Host, c.Port, c.User, c.Password, c.Dbname)
	c.DB, err = sql.Open("postgres", c.Info)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (p *PgConn) ReloadConfig() error {
	sql := "select pg_reload_conf();"
	_, err := p.DB.Query(sql)
	return err
}

func (p *PgConn) Promote(wait string, seconds int) (bool, error) {
	var t bool
	sql := fmt.Sprintf("select pg_promote(%s,%d);", wait, seconds)
	err := p.DB.QueryRow(sql).Scan(&t)
	return t, err
}

func (p *PgConn) DBExist(dbname string) (bool, error) {
	var n int
	sql := fmt.Sprintf("select count(*) from pg_catalog.pg_database where datname = '%s';", dbname)
	err := p.DB.QueryRow(sql).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("获取数据库列表失败: %v", err)
	}
	if n == 0 {
		return false, nil
	}
	return true, nil
}

func (p *PgConn) UserExist(username string) (bool, error) {
	var n int
	sql := fmt.Sprintf("select count(*) from pg_catalog.pg_user where usename = '%s';", username)
	err := p.DB.QueryRow(sql).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("获取用户列表失败: %v", err)
	}
	if n == 0 {
		return false, nil
	}
	return true, nil
}

func (p *PgConn) IsReplicationGrant(username string) bool {
	var repl bool
	sql := fmt.Sprintf("select userepl from pg_catalog.pg_user where usename='%s';", username)
	err := p.DB.QueryRow(sql).Scan(&repl)
	if err != nil {
		return false
	}
	return repl
}

func (p *PgConn) CreateDB(dbname string) error {
	sql := fmt.Sprintf("CREATE DATABASE \"%s\";", dbname)
	_, err := p.DB.Query(sql)
	return err
}

func (p *PgConn) CreateDBUser(username, dbname string) error {
	// var sql string
	sql := fmt.Sprintf("CREATE DATABASE \"%s\"  OWNER  %s;", dbname, username)

	_, err := p.DB.Query(sql)
	return err
}

func (p *PgConn) GrantDBUser(username, dbname string) error {
	// var sql string
	sql := fmt.Sprintf("ALTER DATABASE \"%s\"  OWNER TO  %s;", dbname, username)

	_, err := p.DB.Query(sql)
	return err
}

func (p *PgConn) CreateUser(username, password, privileges string) error {
	var sql string
	if privileges == "DBUSER" {
		sql = fmt.Sprintf("CREATE USER \"%s\"  ENCRYPTED password '%s';", username, password)
	} else {
		sql = fmt.Sprintf("CREATE USER \"%s\" WITH %s ENCRYPTED password '%s';", username, privileges, password)
	}
	_, err := p.DB.Query(sql)
	return err
}

func (p *PgConn) AlterUserExpireAt(username, expireAt string) error {
	sql := fmt.Sprintf("alter user %s with valid until '%s';", username, expireAt)
	_, err := p.DB.Query(sql)
	return err
}

func (p *PgConn) AlterPassword(username, password string) error {
	sql := fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s';", username, password)
	_, err := p.DB.Query(sql)
	return err
}

func (p *PgConn) DBSize() (uint64, error) {
	var size uint64
	sql := "select sum(pg_database_size(datname)) from pg_catalog.pg_database;"
	err := p.DB.QueryRow(sql).Scan(&size)
	if err != nil {
		return 0, fmt.Errorf("获取PG大小失败: %v", err)
	}
	return size, nil
}

func (p *PgConn) ReplicationIp() ([]string, error) {
	var repl []string
	sql := "select client_addr from pg_catalog.pg_stat_replication;"
	rows, err := p.DB.Query(sql)
	if err != nil {
		return repl, fmt.Errorf("获取从库数量失败: %v", err)
	}
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return repl, err
		}
		repl = append(repl, ip)
	}
	return repl, nil
}

func (p *PgConn) Select() error {
	var r int
	return p.DB.QueryRow("select 1;").Scan(&r)
}

func (p *PgConn) PGHbaFilePath() (string, error) {
	var path string
	sql := "show hba_file;"
	err := p.DB.QueryRow(sql).Scan(&path)
	if err != nil {
		return "", fmt.Errorf("获取 pg_hba.conf 文件位置失败: %v", err)
	}
	return path, nil
}

func (p *PgConn) PGFilePath() (string, error) {
	var path string
	sql := "show config_file;"
	err := p.DB.QueryRow(sql).Scan(&path)
	if err != nil {
		return "", fmt.Errorf("获取 postgresql.conf 文件位置失败: %v", err)
	}
	return path, nil
}
