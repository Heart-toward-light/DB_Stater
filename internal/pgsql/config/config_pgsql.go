/*
@Author : WuWeiJian
@Date : 2020-12-03 14:29
*/

package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// pgsql数据库程序的配置文件
type PgsqlConfig struct {
	ListenAddresses              string `ini:"listen_addresses"`
	Port                         int    `ini:"port"`
	UnixSocketDirectories        string `ini:"unix_socket_directories"`
	Timezone                     string `ini:"timezone"`
	Fsync                        string `ini:"fsync"`
	SharedBuffers                string `ini:"shared_buffers"`
	TempBuffers                  string `ini:"temp_buffers"`
	WorkMem                      string `ini:"work_mem"`
	HugePages                    string `ini:"huge_pages"`
	EffectiveCacheSize           string `ini:"effective_cache_size"`
	MaintenanceWorkMem           string `ini:"maintenance_work_mem"`
	MaxConnections               int    `ini:"max_connections"`
	MaxPreparedTransactions      int    `ini:"max_prepared_transactions"`
	SuperuserReservedConnections int    `ini:"superuser_reserved_connections"`
	TcpKeepalivesIdle            int    `ini:"tcp_keepalives_idle"`
	TcpKeepalivesInterval        int    `ini:"tcp_keepalives_interval"`
	TcpKeepalivesCount           int    `ini:"tcp_keepalives_count"`
	AuthenticationTimeout        string `ini:"authentication_timeout"`
	WalLevel                     string `ini:"wal_level"`
	WalBuffers                   string `ini:"wal_buffers"`
	//CheckpointSegments           int    `ini:"checkpoint_segments"`  // pgsql 9.5+ 之后不支持这个参数
	CheckpointCompletionTarget string `ini:"checkpoint_completion_target"`
	CommitDelay                string `ini:"commit_delay"`
	CommitSiblings             string `ini:"commit_siblings"`
	WalLogHints                string `ini:"wal_log_hints"`
	MaxWalSize                 string `ini:"max_wal_size"`
	MinWalSize                 string `ini:"min_wal_size"`
	// WalKeepSize                string `ini:"wal_keep_size"`
	WalKeepSegments         string `ini:"wal_keep_segments"`
	LoggingCollector        string `ini:"logging_collector"`
	LogDestination          string `ini:"log_destination"`
	LogDirectory            string `ini:"log_directory"`
	LogFilename             string `ini:"log_filename"`
	LogRotationAge          string `ini:"log_rotation_age"`
	LogDuration             string `ini:"log_duration"`
	LogTruncateOnRotation   string `ini:"log_truncate_on_rotation"`
	LogMinDurationStatement string `ini:"log_min_duration_statement"`
	LogCheckpoints          string `ini:"log_checkpoints"`
	LogConnections          string `ini:"log_connections"`
	LogDisconnections       string `ini:"log_disconnections"`
	LogLockWaits            string `ini:"log_lock_waits"`
	LogStatement            string `ini:"log_statement"`
	LogLinePrefix           string `ini:"log_line_prefix"`
	LogTimezone             string `ini:"log_timezone"`
	LogMinMessages          string `ini:"log_min_messages"`
	LogMinErrorStatement    string `ini:"log_min_error_statement"`
	ClientMinMessages       string `ini:"client_min_messages"`
	//LogStatementSampleRate       string `ini:"log_statement_sample_rate"`  // 好像pgsql13才开始支持这个参数
	SharedPreloadLibraries       string `ini:"shared_preload_libraries" comment:"# shared_preload_libraries       = 'timescaledb'"`
	PgStatStatementsMax          string `ini:"pg_stat_statements.max"`
	PgStatStatementsTrack        string `ini:"pg_stat_statements.track"`
	PgStatStatementsTrackUtility string `ini:"pg_stat_statements.track_utility"`
	PgStatStatementsSave         string `ini:"pg_stat_statements.save"`
}

func NewPgsqlConfig() *PgsqlConfig {
	return &PgsqlConfig{
		Timezone:                     "'Asia/Shanghai'",
		Fsync:                        "on",
		TempBuffers:                  "8MB",
		WorkMem:                      "8MB",
		HugePages:                    "off",
		EffectiveCacheSize:           "8GB",
		MaintenanceWorkMem:           "64MB",
		SuperuserReservedConnections: 2,
		TcpKeepalivesIdle:            120,
		TcpKeepalivesInterval:        10,
		TcpKeepalivesCount:           10,
		AuthenticationTimeout:        "10s",
		WalLevel:                     "replica",
		WalBuffers:                   "16MB",
		CheckpointCompletionTarget:   "0.9",
		CommitDelay:                  "10",
		CommitSiblings:               "4",
		WalLogHints:                  "on",
		MaxWalSize:                   "3GB",
		MinWalSize:                   "256MB",
		// WalKeepSize:                  "10240",
		WalKeepSegments:  "640",
		LoggingCollector: "on",
		LogDestination:   "'csvlog'",
		//LogDirectory:                 "",
		//LogFilename:                  "'postgresql-%Y-%m-%d_%H%M%S.log'",
		LogFilename:                  "'postgresql-%a.log'",
		LogRotationAge:               "1d",
		LogDuration:                  "off",
		LogTruncateOnRotation:        "on",
		LogMinDurationStatement:      "60000",
		LogCheckpoints:               "off",
		LogConnections:               "off",
		LogDisconnections:            "off",
		LogLockWaits:                 "off",
		LogStatement:                 "none",
		LogLinePrefix:                "'%t [%p]: user=%u,db=%d,client=%h '",
		LogTimezone:                  "'Asia/Shanghai'",
		LogMinMessages:               "'error'",
		LogMinErrorStatement:         "'error'",
		ClientMinMessages:            "'error'",
		PgStatStatementsMax:          "10000",
		PgStatStatementsTrack:        "all",
		PgStatStatementsTrackUtility: "off",
		PgStatStatementsSave:         "off",
	}
}

// 根据系统判断时区
func (c *PgsqlConfig) ChangeTimeZone() error {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return fmt.Errorf("无法加载时区：%v", err)
	}
	currentTime := time.Now().In(loc)
	timezone, _ := currentTime.Zone()
	switch timezone {
	// 对埃及时区进行判断修改
	case "EEST":
		c.Timezone = "'EET'"
		c.LogTimezone = "'EET'"
	case "EET":
		c.Timezone = "'Africa/Cairo'"
		c.LogTimezone = "'Africa/Cairo'"
	}

	return nil
}

// HandleConfig 调整配置
func (c *PgsqlConfig) HandleConfig(pre *Prepare, logDir string) error {
	if err := c.ChangeTimeZone(); err != nil {
		return err
	}
	c.UnixSocketDirectories = fmt.Sprintf("'%s'", DefaultPGSocketPath)
	r, _ := regexp.Compile(RegexpMemorySuffix)
	index := r.FindStringIndex(pre.MemorySize)
	if index == nil {
		return fmt.Errorf("内存参数必须包含单位后缀(MB 或 GB)")
	}

	memory := pre.MemorySize[:index[0]]
	suffix := strings.ToUpper(pre.MemorySize[index[0]:])
	m, err := strconv.ParseFloat(memory, 64)
	if err != nil {
		return err
	}
	conn := int(m * 300)
	switch suffix {
	case "M", "MB":
		c.SharedBuffers = memory + "MB"
		conn = conn / 1000
	case "G", "GB":
		c.SharedBuffers = memory + "GB"
	default:
		return fmt.Errorf("不支持的内存后缀单位")
	}

	if conn < 100 {
		conn = 100
	}

	c.Port = pre.Port
	c.ListenAddresses = fmt.Sprintf("'%s'", pre.BindIP)
	c.MaxConnections = conn
	c.MaxPreparedTransactions = conn
	c.LogDirectory = fmt.Sprintf("'%s'", logDir)

	if pre.Libraries == "" {
		c.SharedPreloadLibraries = "'pg_stat_statements'"
	} else {
		c.SharedPreloadLibraries = "'pg_stat_statements," + pre.Libraries + "'"
	}

	return nil
}

func (c *PgsqlConfig) PGdataHandleConfig(pre *PGAutoFailoverPGNode, logDir string) error {
	if err := c.ChangeTimeZone(); err != nil {
		return err
	}
	c.UnixSocketDirectories = fmt.Sprintf("'%s'", DefaultPGSocketPath)
	r, _ := regexp.Compile(RegexpMemorySuffix)
	index := r.FindStringIndex(pre.MemorySize)
	if index == nil {
		return fmt.Errorf("内存参数必须包含单位后缀(MB 或 GB)")
	}

	memory := pre.MemorySize[:index[0]]
	suffix := strings.ToUpper(pre.MemorySize[index[0]:])
	m, err := strconv.ParseFloat(memory, 64)
	if err != nil {
		return err
	}
	conn := int(m * 300)
	switch suffix {
	case "M", "MB":
		c.SharedBuffers = memory + "MB"
		conn = conn / 1000
	case "G", "GB":
		c.SharedBuffers = memory + "GB"
	default:
		return fmt.Errorf("不支持的内存后缀单位")
	}

	if conn < 100 {
		conn = 100
	}

	c.Port = pre.Port
	c.ListenAddresses = fmt.Sprintf("'%s'", pre.BindIP)
	c.MaxConnections = conn
	c.MaxPreparedTransactions = conn
	c.LogDirectory = fmt.Sprintf("'%s'", logDir)

	if pre.Libraries == "" {
		c.SharedPreloadLibraries = "'pg_stat_statements'"
	} else {
		c.SharedPreloadLibraries = "'pg_stat_statements," + pre.Libraries + "'"
	}

	return nil
}

// SaveTo 将Prepare实例数据写入配置文件
func (c *PgsqlConfig) SaveTo(filename string) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true})
	if err := ini.ReflectFrom(cfg, c); err != nil {
		return fmt.Errorf("pgsql 配置文件 映射到(%s)文件错误: %v", filename, err)
	}
	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("pgsql 配置文件 保存到(%s)文件错误: %v", filename, err)
	}
	return nil
}
