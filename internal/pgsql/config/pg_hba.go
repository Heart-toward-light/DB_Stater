/*
@Author : WuWeiJian
@Date : 2020-12-05 16:59
*/

package config

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
)

type PgHbaConfig struct {
	Type     string
	Database string
	User     string
	Address  string
	Method   string
}

type PgHba struct {
	Header []string
	Config []*PgHbaConfig
}

func NewPgHba() *PgHba {
	return &PgHba{
		Header: []string{"# TYPE", "DATABASE", "USER", "ADDRESS", "METHOD"},
	}
}

// Init 调整 PG_AUTO_FAILOVER 配置
func (p *PgHba) Trust_Init(user string) {
	p.Config = append(p.Config,
		&PgHbaConfig{
			Type:     "local",
			Database: "all",
			User:     user,
			Address:  "",
			Method:   "trust",
		},
	)
}

// Init 调整配置
func (p *PgHba) Init(user string) {
	p.Config = append(p.Config,
		&PgHbaConfig{
			Type:     "local",
			Database: "all",
			User:     user,
			Address:  "",
			Method:   "md5",
		},
		&PgHbaConfig{
			Type:     "host",
			Database: "all",
			User:     user,
			Address:  "127.0.0.1/32",
			Method:   "md5",
		},
	)
}

// Load 从磁盘加载
func (p *PgHba) Load(filename string) error {
	fs, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("打开文件失败: %s\n", err)
	}
	defer fs.Close()
	r := csv.NewReader(fs)
	r.Comma = '\t'
	r.Comment = '#'
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	for {
		var d PgHbaConfig
		row, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("读取文件失败: %s\n", err)
		}

		// logger.Warningf(" database 是: %T ", row)

		d.Type = strings.TrimSpace(row[0])
		d.Database = strings.TrimSpace(row[1])
		d.User = strings.TrimSpace(row[2])
		d.Address = strings.TrimSpace(row[3])
		d.Method = strings.TrimSpace(row[4])
		p.Config = append(p.Config, &d)
	}
	return nil
}

// SaveTo 保存到磁盘
func (p *PgHba) SaveTo(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	table := tablewriter.NewWriter(w)
	table.SetHeader(p.Header)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("\t")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)
	for _, v := range p.Config {
		d := []string{v.Type, v.Database, v.User, v.Address, v.Method}
		table.Append(d)
	}
	table.Render()
	return w.Flush()
}

// AddRecord 增加一条记录
func (p *PgHba) AddRecord(user, database, address string) {
	p.AddR("host", user, database, address)
}

// Add 增加一条记录
func (p *PgHba) AddR(t, user, database, address string) {
	p.Config = append(p.Config,
		&PgHbaConfig{
			Type:     t,
			Database: database,
			User:     user,
			Address:  address,
			Method:   "md5",
		},
	)
}

// DelRecord 删除记录
func (p *PgHba) DelRecord(user, database, address string) {
	switch {
	case user == "":
		return
	case database == "":
		p.DelRecordByUser(user)
	case address == "":
		p.DelRecordByUserAndDB(user, database)
	default:
		p.DelRecordByUserAndDBAndAddr(user, database, address)
	}
}

// DelRecord 删除记录
func (p *PgHba) DelRecordByUser(user string) {
	var hba []*PgHbaConfig
	for _, config := range p.Config {
		if config.User != user {
			hba = append(hba, config)
		}
	}
	p.Config = hba
}

// DelRecord 删除记录
func (p *PgHba) DelRecordByUserAndDB(user, database string) {
	var hba []*PgHbaConfig
	for _, config := range p.Config {
		if config.User != user && config.Database != database {
			hba = append(hba, config)
		}
	}
	p.Config = hba
}

// DelRecord 删除记录
func (p *PgHba) DelRecordByUserAndDBAndAddr(user, database, address string) {
	var hba []*PgHbaConfig
	for _, config := range p.Config {
		if config.User != user && config.Database != database && config.Address != address {
			hba = append(hba, config)
		}
	}
	p.Config = hba
}

// FindRecordByTypeAndUserAndDBAndAddr 查找记录
func (p *PgHba) FindRecordByTypeAndUserAndDBAndAddr(t, user, database, address string) []*PgHbaConfig {
	var hba []*PgHbaConfig
	for _, config := range p.Config {
		if config.Type == t && config.User == user && config.Database == database && config.Address == address {
			hba = append(hba, config)
		}
	}
	return hba
}

//// ModifyRecord 修改记录
//func (p *PgHba) ModifyRecord(user, database, address string) {
//	switch {
//	case user == "":
//		return
//	case database == "":
//		return
//	case address == "":
//		return
//	default:
//		return
//	}
//}
//
//// ModifyRecord 修改记录
//func (p *PgHba) ModifyRecordByUser(user string) {
//	var hba []*PgHbaConfig
//	for _, config := range p.Config {
//		if config.User != user {
//			hba = append(hba, config)
//		}
//	}
//	p.Config = hba
//}
//
//// ModifyRecord 修改记录
//func (p *PgHba) ModifyRecordByUserAndDB(user, database string) {
//	var hba []*PgHbaConfig
//	for _, config := range p.Config {
//		if config.User != user {
//			hba = append(hba, config)
//		}
//	}
//	p.Config = hba
//}
//
//// ModifyRecord 修改记录
//func (p *PgHba) ModifyRecordByUserAndDBAndAddr(user, database, address string) {
//	var hba []*PgHbaConfig
//	for _, config := range p.Config {
//		if config.User != user {
//			hba = append(hba, config)
//		}
//	}
//	p.Config = hba
//}
