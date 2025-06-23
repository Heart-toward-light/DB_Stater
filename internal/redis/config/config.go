/*
@Author : WuWeiJian
@Date : 2021-01-06 11:34
*/

package config

import (
	"fmt"
	"io/ioutil"
)

// redis数据库程序的配置文件
type RedisConfig struct {
	Body            string
	PidFile         string
	Port            int
	Socket          string
	Logfile         string
	DbFilename      string
	Dir             string
	MaxMemory       int
	Appendonly      string
	MaxmemoryPolicy string
	RequirePass     string
	MasterAuth      string
	Modules         []string
	Cluster         string
	Save            string
}

func NewRedisConfig() *RedisConfig {
	var body = `#QAX Redis configuration
#Verredis60
daemonize yes
pidfile %s       
port %d                                    
unixsocket %s               
unixsocketperm 755
timeout 86400
loglevel notice
logfile %s          
databases 16
repl-backlog-size %dmb
repl-backlog-ttl 0
%s
stop-writes-on-bgsave-error no
rdbcompression yes
rdbchecksum yes
dbfilename %s                       
dir %s                        
aof-use-rdb-preamble yes
maxmemory-policy %s
maxclients 10000
maxmemory %dmb                                  
maxmemory-samples 3
appendonly %s
appendfsync everysec
no-appendfsync-on-rewrite yes
auto-aof-rewrite-percentage 300
auto-aof-rewrite-min-size 1G
slowlog-log-slower-than 1000
slowlog-max-len 1024
hash-max-ziplist-entries 512
hash-max-ziplist-value 64
list-max-ziplist-entries 512
list-max-ziplist-value 64
set-max-intset-entries 512
zset-max-ziplist-entries 128
zset-max-ziplist-value 64
activerehashing yes
client-output-buffer-limit normal 0 0 0
client-output-buffer-limit pubsub 0 0 0
client-output-buffer-limit slave 0 0 0
requirepass %s                 
masterauth %s                  
hz 50
cluster-enabled %s
cluster-config-file nodes.conf
cluster-node-timeout 60000
activedefrag yes
active-defrag-ignore-bytes 500mb
active-defrag-threshold-lower 30
active-defrag-threshold-upper 100
active-defrag-cycle-min 15
active-defrag-cycle-max 30`
	return &RedisConfig{Body: body, Save: "save \"\"", Cluster: "no"}
}

func (c RedisConfig) HandleConfig() error {
	return nil
}

func (c *RedisConfig) FormatBody() {
	c.Body = fmt.Sprintf(c.Body,
		c.PidFile,
		c.Port,
		c.Socket,
		c.Logfile,
		ReplBacklogSizeMB,
		c.Save,
		c.DbFilename,
		c.Dir,
		c.MaxmemoryPolicy,
		c.MaxMemory,
		c.Appendonly,
		c.RequirePass,
		c.MasterAuth,
		c.Cluster)
	for _, module := range c.Modules {
		c.Body = c.Body + fmt.Sprintf("\nloadmodule %s", module)
	}
}

func (c *RedisConfig) SaveTo(filename string) error {
	return ioutil.WriteFile(filename, []byte(c.Body), 0755)
}
