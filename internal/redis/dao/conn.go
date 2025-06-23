/*
@Author : WuWeiJian
@Date : 2021-04-13 16:34
*/

package dao

import (
	"dbup/internal/utils/logger"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

type RedisClient struct {
	Host     string
	Port     int
	Password string
	Conn     redis.Conn
}

func NewRedisConn(host string, port int, password string) (*RedisClient, error) {
	var err error
	c := &RedisClient{
		Host:     host,
		Port:     port,
		Password: password,
	}
	if c.Conn, err = redis.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), redis.DialPassword(c.Password)); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *RedisClient) SlaveOf(master string, port int) error {
	if _, err := c.Conn.Do("slaveof", master, port); err != nil {
		return err
	}
	if _, err := c.Conn.Do("CONFIG", "REWRITE"); err != nil {
		return err
	}

	return nil
}

func (c *RedisClient) ClusterID() (string, error) {
	return redis.String(c.Conn.Do("cluster", "myid"))
}

func (c *RedisClient) SlaveStatus() (string, error) {
	s, err := redis.String(c.Conn.Do("info", "Replication"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(s, "\r\n") {
		if strings.Index(line, ":") != -1 {
			kv := strings.Split(line, ":")
			key := strings.Trim(kv[0], " ")
			value := strings.Trim(kv[1], " ")
			if key == "master_link_status" {
				return value, nil
			}
		}
	}
	return "", nil
}

func (c *RedisClient) SlaveIPs() []string {
	var ips []string
	r, _ := regexp.Compile("^slave[0-9]+$")

	s, err := redis.String(c.Conn.Do("info", "Replication"))
	if err != nil {
		return ips
	}
	for _, line := range strings.Split(s, "\r\n") {
		if strings.Index(line, ":") != -1 {
			kv := strings.Split(line, ":")
			key := strings.Trim(kv[0], " ")
			value := strings.Trim(kv[1], " ")
			if ok := r.MatchString(key); !ok {
				continue
			}
			slaveinfo := strings.Split(value, ",")
			var ip string
			var port string
			ipinfo := strings.Split(slaveinfo[0], "=")
			if len(ipinfo) < 2 {
				continue
			}
			ip = ipinfo[1]
			portinfo := strings.Split(slaveinfo[1], "=")
			if len(portinfo) < 2 {
				continue
			}
			port = portinfo[1]
			ips = append(ips, ip+":"+port)
		}
	}
	return ips
}

func (c *RedisClient) ClusterNodes() ([]ClusterNode, error) {
	var nodes []ClusterNode
	s, err := redis.String(c.Conn.Do("cluster", "nodes"))
	if err != nil {
		return nodes, err
	}

	for _, line := range strings.Split(s, "\n") {
		if line == "" {
			continue
		}
		nodeInfo := strings.Split(line, " ")
		hostPort := strings.Split(strings.Split(nodeInfo[1], "@")[0], ":")
		port, err := strconv.Atoi(hostPort[1])
		if err != nil {
			return nodes, err
		}

		// role := nodeInfo[2]
		// if strings.Contains(role, ",") {
		// 	roles := strings.Split(role, ",")
		// 	if len(roles) >= 2 && roles[0] == "myself" {
		// 		role = roles[1]
		// 	} else {
		// 		role = roles[0]
		// 	}
		// }

		var role string = "master"
		var fail bool = false
		if strings.Contains(nodeInfo[2], "slave") {
			role = "slave"
		}

		if strings.Contains(nodeInfo[2], "fail") {
			fail = true
		}

		nodes = append(nodes, ClusterNode{
			ClusterID: nodeInfo[0],
			Host:      hostPort[0],
			Port:      port,
			Role:      role,
			Fail:      fail,
			MasterID:  nodeInfo[3],
			Connected: nodeInfo[7],
		})
	}
	return nodes, nil
}

func (c *RedisClient) ClusterForget(id string) error {
	if _, err := c.Conn.Do("cluster", "forget", id); err != nil {
		return err
	}
	return nil
}

func (c *RedisClient) GetMaster() (bool, error) {
	s, err := redis.String(c.Conn.Do("info", "Replication"))
	if err != nil {
		return false, err
	}

	for _, line := range strings.Split(s, "\r\n") {
		if strings.Contains(line, ":master") {
			return true, nil
		}
	}

	return false, nil

}

func (c *RedisClient) GetAOFStatus() (bool, error) {
	s, err := redis.String(c.Conn.Do("info", "Persistence"))
	if err != nil {
		return false, err
	}

	for _, line := range strings.Split(s, "\r\n") {
		if strings.Contains(line, "aof_rewrite_in_progress") {
			kv := strings.Split(line, ":")
			value := strings.Trim(kv[1], " ")
			if value == "0" {
				return true, nil
			}
		}
	}

	return false, nil

}

func (c *RedisClient) FlushAOF() error {
	stat := false
	if _, err := redis.String(c.Conn.Do("BGREWRITEAOF")); err != nil {
		return err
	}

	for i := 1; i <= 30; i++ {
		time.Sleep(3 * time.Second)
		if f, err := c.GetAOFStatus(); err != nil {
			return err
		} else {
			if f {
				stat = true
				break
			} else {
				logger.Warningf("等到AOF文件持久化完成...\n")
			}
		}
	}

	if !stat {
		return fmt.Errorf("AOF 一直未持久化完成,请检查")
	}

	return nil
}
