package service

import (
	"bytes"
	"dbup/internal/global"
	"dbup/internal/mariadb/config"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"
)

type Galera_node struct {
	Port     int
	BasePath string
}

func NewGaleraNode() *Galera_node {
	return &Galera_node{}
}

func (g *Galera_node) Validator(serviceFileFullName string) error {

	logger.Infof("启动前验证参数\n")
	if utils.PortInUse(g.Port) {
		return fmt.Errorf("galera 第一个节点端口号 %d 被占用", g.Port)
	}

	if !utils.IsExists(serviceFileFullName) {
		return fmt.Errorf("启动文件(%s)不存在, 停止启动 galera 第一个节点", serviceFileFullName)
	}

	if !utils.IsDir(g.BasePath) {
		return fmt.Errorf("指定的数据目录 %s 不存在", g.BasePath)
	}

	return nil
}

func (g *Galera_node) Start_Onenode() error {
	service := fmt.Sprintf(config.ServiceFileName, g.Port)
	servicePath := global.ServicePath
	serviceFileFullName := filepath.Join(servicePath, service)
	portname := fmt.Sprintf("port=%d", g.Port)
	newname := fmt.Sprintf("%s --wsrep-new-cluster", portname)

	if err := g.Validator(serviceFileFullName); err != nil {
		return err
	}

	input, err := ioutil.ReadFile(serviceFileFullName)
	if err != nil {
		return err
	}

	output := bytes.Replace(input, []byte(portname), []byte(newname), -1)

	if err = ioutil.WriteFile(serviceFileFullName, output, 0666); err != nil {
		return fmt.Errorf("%s 文件写入异常 %s", serviceFileFullName, err)
	}

	if err := command.SystemdReload(); err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	if err := command.SystemCtl(service, "start"); err != nil {
		if err := g.Remove_service(serviceFileFullName); err != nil {
			return err
		}
		return err
	}

	if err := g.Remove_service(serviceFileFullName); err != nil {
		return err
	}

	logger.Successf("Galera 集群第一个节点启动完成\n")
	return nil
}

func (g *Galera_node) Remove_service(serviceFileFullName string) error {
	input, err := ioutil.ReadFile(serviceFileFullName)
	if err != nil {
		return err
	}

	outputs := bytes.Replace(input, []byte("--wsrep-new-cluster"), []byte(" "), -1)

	if err = ioutil.WriteFile(serviceFileFullName, outputs, 0666); err != nil {
		return fmt.Errorf("%s 文件写入异常 %s", serviceFileFullName, err)
	}

	if err := command.SystemdReload(); err != nil {
		return err
	}

	return nil
}
