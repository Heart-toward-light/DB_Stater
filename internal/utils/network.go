/*
@Author : WuWeiJian
@Date : 2020-12-04 19:48
*/

package utils

import (
	"dbup/internal/utils/logger"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func IsHostName(hostname string) bool {
	if _, err := net.LookupHost(hostname); err != nil {
		logger.Warningf("解析主机名(%s)失败: %v\n", hostname, err)
		return false
	}
	return true
}

// PortInUse 检测端口是否被占用
func PortInUse(port int) bool {
	c, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return true
	}
	defer c.Close()
	return false
}

// 验证是否为IP地址
func IsIPv4(addr string) error {
	if address := net.ParseIP(addr); address == nil {
		return fmt.Errorf("IP地址格式不正确: %s", addr)
	}
	if strings.Index(addr, ":") != -1 {
		return fmt.Errorf("IP地址为IPv6格式: %s", addr)
	}
	return nil
}

// 验证是否为IP地址
func IsIP(addr string) error {
	if address := net.ParseIP(addr); address == nil {
		return fmt.Errorf("IP地址(%s)格式不正确", addr)
	}
	return nil
}

// 验证是否为IPv6地址, 目前没用
func IsIPv6(addr string) error {
	if address := net.ParseIP(addr); address == nil {
		return fmt.Errorf("IP地址(%s)格式不正确", addr)
	}
	if strings.Index(addr, ":") == -1 {
		return fmt.Errorf("IP地址为IPv4格式: %s", addr)
	}
	return nil
}

// 验证是否为IPv4+掩码格式
func IsIPv4Mask(addr string) error {
	ipMask := strings.Split(addr, "/")
	if err := IsIPv4(ipMask[0]); err != nil {
		return err
	}
	mask, err := strconv.Atoi(ipMask[1])
	if err != nil {
		return err
	}
	if mask < 0 || mask > 32 {
		return fmt.Errorf("子网掩码必须为 1 ~ 32 之间")
	}
	return nil
}

func CheckAddressFormat(s string) error {
	ipMask := strings.Split(s, "/")
	switch len(ipMask) {
	case 0:
		return fmt.Errorf("授权IP字符串不符合规则")
	case 1:
		return IsIP(s)
	case 2:
		return IsIPMask(s)
	default:
		return fmt.Errorf("授权IP字符串不符合规则")
	}
}

// 验证是否为IP+掩码格式
func IsIPMask(addr string) error {
	ipMask := strings.Split(addr, "/")
	mask, err := strconv.Atoi(ipMask[1])
	if err != nil {
		return err
	}

	if address := net.ParseIP(ipMask[0]); address == nil {
		return fmt.Errorf("IP地址(%s)格式不正确", addr)
	}

	if strings.Index(ipMask[0], ":") == -1 {
		if mask < 0 || mask > 32 {
			return fmt.Errorf("IPV4子网掩码必须为 1 ~ 32 之间")
		}
	} else {
		if mask < 0 || mask > 128 {
			return fmt.Errorf("IPV6子网掩码必须为 1 ~ 128 之间")
		}
	}
	return nil
}

// Ipv4AddMask 给IP增加掩码
func Ipv4AddMask(addr string) string {
	var ipMask string
	switch {
	case addr == "0.0.0.0":
		ipMask = "0.0.0.0/0"
	case addr[len(addr)-6:] == ".0.0.0":
		ipMask = addr + "/8"
	case addr[len(addr)-4:] == ".0.0":
		ipMask = addr + "/16"
	case addr[len(addr)-2:] == ".0":
		ipMask = addr + "/24"
	default:
		ipMask = addr + "/32"
	}
	return ipMask
}

// Ipv4AddMask 给IP增加掩码
func IpAddMask(addr string) string {
	if strings.Index(addr, ":") == -1 {
		return Ipv4AddMask(addr)
	} else {
		return addr + "/128"
	}
}

// Ipv4AddMaskIfNot 给IP增加掩码
func Ipv4AddMaskIfNot(addr string) string {
	var ipmsk string
	ipMask := strings.Split(addr, "/")
	switch len(ipMask) {
	case 1:
		ipmsk = Ipv4AddMask(addr)
	case 2:
		ipmsk = addr
	default:
		ipmsk = addr
	}
	return ipmsk
}

// Ipv4AddMaskIfNot 给IP增加掩码
func IpAddMaskIfNot(addr string) string {
	var ipmsk string
	ipMask := strings.Split(addr, "/")
	switch len(ipMask) {
	case 1:
		ipmsk = IpAddMask(addr)
	case 2:
		ipmsk = addr
	default:
		ipmsk = addr
	}
	return ipmsk
}

// 随机选择一个可用的端口号, 到3万还没选出来,就还是用默认吧
func RandomPort(port int) int {
	for i := port; i <= 30000; i++ {
		if !PortInUse(i) {
			port = i
			break
		}
	}
	return port
}

func LocalIP() ([]string, error) {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips, err
	}
	for _, address := range addrs {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			ips = append(ips, ipNet.IP.String())
		}
	}
	return ips, nil
}

// 检查端口是否存在
func TcpGather(ipport string) (results bool, err error) {

	results = false
	// 3 秒超时
	conn, err := net.DialTimeout("tcp", ipport, 3*time.Second)
	if err != nil {
		return results, err
		// todo log handler
	} else {
		if conn != nil {
			results = true
			_ = conn.Close()
		}
	}

	return results, nil
}

// 转换IPV6域名地址为IPV6地址
func Ipv6conversion(domain string) string {
	ips, _ := net.LookupIP(domain)
	if ips[0].To4() == nil {
		ipv6 := ips[0].String()
		return ipv6
	}
	return ""
}

// 检查 IPV6 环境
func Ipv6Check() error {
	domain, _ := os.Hostname()
	ips, err := net.LookupIP(domain)
	if err != nil {
		return fmt.Errorf("获取地址 %s 失败: %v\n", domain, err)
	}

	if ips[0].To4() == nil {
		ifaces, err := net.Interfaces()
		if err != nil {
			return fmt.Errorf("获取网络接口失败：%v\n", err)
		}

		// 判断是否与本地IPV6地址匹配
		found := false
		for _, iface := range ifaces {
			addrs, err := iface.Addrs()
			if err != nil {
				// fmt.Println("获取地址失败：", err)
				continue
			}

			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() == nil {
					if ipnet.IP.Equal(ips[0]) {
						found = true
						break
					}
				}
			}
		}
		if !found {
			return fmt.Errorf("%s 域名解析的IPV6地址与本地地址不匹配,请检查\n", domain)
		}

	}
	return nil
}
