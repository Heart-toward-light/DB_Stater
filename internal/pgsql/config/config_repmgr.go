// Created by LiuSainan on 2022-05-18 16:01:41

package config

import (
	"fmt"

	"gopkg.in/ini.v1"
)

type RepmgrGlobal struct {
	SystemUser     string `ini:"system-user"`
	SystemGroup    string `ini:"system-group"`
	Dir            string `ini:"dir"`
	AdminUser      string `ini:"admin-user"`
	AdminPassword  string `ini:"admin-password"`
	RepmgrOwnerIP  string `ini:"repmgr-local-ip"`
	RepmgrNodeID   int    `ini:"repmgr-node-id"`
	RepmgrUser     string `ini:"repmgr-user"`
	RepmgrPassword string `ini:"repmgr-password"`
	RepmgrDBName   string `ini:"repmgr-dbname"`
	Yes            bool   `ini:"yes"`
	NoRollback     bool   `ini:"no-rollback"`
}

type RepmgrPrimary struct {
	SystemUser     string `ini:"system-user"`
	SystemGroup    string `ini:"system-group"`
	Dir            string `ini:"dir"`
	AdminUser      string `ini:"admin-user"`
	AdminPassword  string `ini:"admin-password"`
	Port           int    `ini:"port"`
	RepmgrOwnerIP  string `ini:"repmgr-owner-ip"`
	RepmgrNodeID   int    `ini:"repmgr-node-id"`
	RepmgrUser     string `ini:"repmgr-user"`
	RepmgrPassword string `ini:"repmgr-password"`
	RepmgrDBName   string `ini:"repmgr-dbname"`
	Yes            bool   `ini:"yes"`
	NoRollback     bool   `ini:"no-rollback"`
}

type RepmgrStandby struct {
	SystemUser     string `ini:"system-user"`
	SystemGroup    string `ini:"system-group"`
	Dir            string `ini:"dir" `
	AdminUser      string `ini:"admin-user"`
	AdminPassword  string `ini:"admin-password"`
	MasterPort     int    `ini:"port"`
	MasterIP       string `ini:"ip"`
	RepmgrOwnerIP  string `ini:"repmgr-owner-ip"`
	RepmgrNodeID   int    `ini:"repmgr-node-id"`
	RepmgrUser     string `ini:"repmgr-user"`
	RepmgrPassword string `ini:"repmgr-password"`
	RepmgrDBName   string `ini:"repmgr-dbname"`
	Yes            bool   `ini:"yes"`
	NoRollback     bool   `ini:"no-rollback"`
}

type RepmgrConfig struct {
	NodeId                    int    `ini:"node_id"`
	NodeName                  string `ini:"node_name"`
	Location                  string `ini:"location"`
	Conninfo                  string `ini:"conninfo"`
	PgBindir                  string `ini:"pg_bindir"`
	DataDirectory             string `ini:"data_directory"`
	LogFile                   string `ini:"log_file"`
	Failover                  string `ini:"failover"`
	ConnectionCheckType       string `ini:"connection_check_type"`
	PromoteCommand            string `ini:"promote_command"`
	FollowCommand             string `ini:"follow_command"`
	ReconnectAttempts         string `ini:"reconnect_attempts"`
	ReconnectInterval         string `ini:"reconnect_interval"`
	LogLevel                  string `ini:"log_level"`
	LogStatusInterval         int    `ini:"log_status_interval"`
	Priority                  string `ini:"priority"`
	DegradedMonitoringTimeout string `ini:"degraded_monitoring_timeout"`
	AsyncQueryTimeout         string `ini:"async_query_timeout"`
	UseReplicationSlots       bool   `ini:"use_replication_slots"`
	SshOptions                string `ini:"ssh_options"`
}

func NewRepmgrConfig() *RepmgrConfig {
	return &RepmgrConfig{
		Location:                  "'default'",
		Failover:                  "'automatic'",
		ConnectionCheckType:       "query",
		ReconnectAttempts:         "'10'",
		ReconnectInterval:         "'12'",
		LogLevel:                  "'NOTICE'",
		LogStatusInterval:         60,
		Priority:                  "'100'",
		DegradedMonitoringTimeout: "'5'",
		AsyncQueryTimeout:         "'20'",
		UseReplicationSlots:       true,
		SshOptions:                "'-o \"StrictHostKeyChecking no\" -v'",
	}
}

// SaveTo 将Prepare实例数据写入配置文件
func (r *RepmgrConfig) SaveTo(filename string) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true})
	if err := ini.ReflectFrom(cfg, r); err != nil {
		return fmt.Errorf("repmgr 配置文件 映射到(%s)文件错误: %v", filename, err)
	}
	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("repmgr 配置文件 保存到(%s)文件错误: %v", filename, err)
	}
	return nil
}
