/*
@Author : WuWeiJian
@Date : 2021-08-03 10:54
*/

package services

import (
	"dbup/internal/environment"
	"dbup/internal/redis/config"
	"dbup/internal/redis/dao"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 安装redis cluster的总控制逻辑
type RedisClusterManager struct {
}

func NewRedisClusterManager() *RedisClusterManager {
	return &RedisClusterManager{}
}

func (d *RedisClusterManager) AddNode(o config.RedisClusterAddNodeOption, role string) error {
	var masterID string

	logger.Infof("验证参数\n")
	if err := o.Validator(); err != nil {
		return err
	}

	o.Parameter.InitPortDir()

	if o.SSHConfig.Password == "" && o.SSHConfig.KeyFile == "" {
		o.SSHConfig.KeyFile = filepath.Join(environment.GlobalEnv().HomePath, ".ssh", "id_rsa")
	}

	logger.Infof("测试集群连接性\n")
	ClusterIpPort := strings.Split(o.Cluster, ":")
	ClusterPort, _ := strconv.Atoi(ClusterIpPort[1])
	if _, err := dao.NewRedisConn(ClusterIpPort[0], ClusterPort, o.Parameter.Password); err != nil {
		return fmt.Errorf("连接到 redis cluster 节点 %s 失败: %v", o.Cluster, err)
	}

	// 获取 master id
	if role == "slave" && o.Master != "" {
		logger.Infof("获取 master 节点 cluster id\n")
		MasterIpPort := strings.Split(o.Master, ":")
		MasterPort, _ := strconv.Atoi(MasterIpPort[1])
		conn, err := dao.NewRedisConn(MasterIpPort[0], MasterPort, o.Parameter.Password)
		if err != nil {
			return fmt.Errorf("连接到 redis master 节点 %s 失败: %v", o.Master, err)
		}
		if masterID, err = conn.ClusterID(); err != nil {
			return err
		}
	}

	var node *Instance
	var err error
	if o.SSHConfig.Password != "" {
		node, err = NewInstance(o.TmpDir,
			o.Host,
			o.SSHConfig.Username,
			o.SSHConfig.Password,
			o.SSHConfig.Port,
			o.Parameter)
		if err != nil {
			return err
		}
	} else {
		node, err = NewInstanceUseKeyFile(o.TmpDir,
			o.Host,
			o.SSHConfig.Username,
			o.SSHConfig.KeyFile,
			o.SSHConfig.Port,
			o.Parameter)
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

	if err := node.Install(true, true, o.IPV6); err != nil {
		return err
	}

	logger.Infof("开始安装redis实例\n")
	if o.IPV6 {
		o.Host = utils.Ipv6conversion(o.Host)
	}
	if err := d.Install(node, fmt.Sprintf("%s:%d", o.Host, o.Parameter.Port), o.Cluster, role, masterID, o.IPV6); err != nil {
		logger.Warningf("安装失败, 开始回滚\n")
		_ = node.UNInstall()
		return err
	}

	logger.Infof("新节点: %s:%d 添加成功\n", o.Host, o.Parameter.Port)

	return nil
}

func (d *RedisClusterManager) Install(node *Instance, n string, cluster string, role string, masterID string, ipv6 bool) error {
	if err := node.Install(true, false, ipv6); err != nil {
		return err
	}

	logger.Infof("实例安装成功, 开始加入集群\n")
	if err := node.ClusterAddNode(n, cluster, role, masterID); err != nil {
		return err
	}

	if role == "master" {
		logger.Infof("6 秒后开始 rebalance slot ...\n")
		time.Sleep(6 * time.Second)
		logger.Infof("开始 rebalance slot ...\n")
		if err := node.ClusterReBalance(cluster); err != nil {
			return err
		}
	}
	return nil
}

func (d *RedisClusterManager) RedisClusterFix(cli string, cluster string, password string) error {
	if cluster == "" {
		return fmt.Errorf("请指定 cluster 地址")
	}
	logger.Infof("测试集群连接性\n")
	ClusterIpPort := strings.Split(cluster, ":")
	ClusterPort, err := strconv.Atoi(ClusterIpPort[1])
	if err != nil {
		return fmt.Errorf("解析 cluster 地址(%s)失败: %v", cluster, err)
	}

	cmd := fmt.Sprintf("echo yes | %s -a %s --cluster fix %s --cluster-fix-with-unreachable-masters --cluster-yes", cli, password, cluster)
	l := command.Local{Timeout: 600}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("执行 fix 修复失败: %v, 标准错误输出: %s", err, stderr)
	}

	conn, err := dao.NewRedisConn(ClusterIpPort[0], ClusterPort, password)
	if err != nil {
		return fmt.Errorf("连接到 redis cluster 节点 %s 失败: %v", cluster, err)
	}
	nodes, err := conn.ClusterNodes()
	if err != nil {
		return err
	}

	OKNodes := dao.GetConnected(nodes)

	for _, node := range OKNodes {
		conn, err := dao.NewRedisConn(node.Host, node.Port, password)
		if err != nil {
			return fmt.Errorf("连接到 redis cluster 节点 %s:%d 失败: %v", node.Host, node.Port, err)
		}

		nodes, err := conn.ClusterNodes()
		if err != nil {
			return err
		}
		DisNodes := dao.GetDisConnected(nodes)
		for _, disNode := range DisNodes {
			if err := conn.ClusterForget(disNode.ClusterID); err != nil {
				return err
			}
		}
	}
	return nil
}
