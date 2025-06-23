/*
@Author : WuWeiJian
@Date : 2020-12-04 16:24
*/

package utils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"regexp"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// 生成随机密码
func GeneratePasswd(length int) string {
	rand.Seed(time.Now().UnixNano())
	digits := "0123456789"
	specials := "#!"
	letters := "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + digits + specials
	buf := make([]byte, length)
	buf[0] = digits[rand.Intn(len(digits))]
	buf[1] = specials[rand.Intn(len(specials))]
	for i := 2; i < length; i++ {
		buf[i] = letters[rand.Intn(len(letters))]
	}
	rand.Shuffle(len(buf), func(i, j int) {
		buf[i], buf[j] = buf[j], buf[i]
	})
	return string(buf)
}

// 生成随机字符串
func GenerateString(length int) string {
	rand.Seed(time.Now().UnixNano())
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		buf[i] = letters[rand.Intn(len(letters))]
	}
	rand.Shuffle(len(buf), func(i, j int) {
		buf[i], buf[j] = buf[j], buf[i]
	})
	return string(buf)
}

// CheckPasswordLever 校验密码复杂度
func CheckPasswordLever(ps string) error {
	if len(ps) < 16 {
		return fmt.Errorf("超级管理员密码必须大于16位,且包含大小写字母,数字,特殊字符: 小于16位; 随机示例: %s\n", GeneratePasswd(16))
	}
	digital := `[0-9]{1}`
	lower := `[a-z]{1}`
	upper := `[A-Z]{1}`
	symbol := `[!@#~$%^&*()+|_\=\-,./\\\[\]?]{1}`
	if b, err := regexp.MatchString(digital, ps); !b || err != nil {
		return fmt.Errorf("超级管理员密码必须大于16位,且包含大小写字母,数字,特殊字符: %v; 随机示例: %s\n", err, GeneratePasswd(16))
	}
	if b, err := regexp.MatchString(lower, ps); !b || err != nil {
		return fmt.Errorf("超级管理员密码必须大于16位,且包含大小写字母,数字,特殊字符: %v; 随机示例: %s\n", err, GeneratePasswd(16))
	}
	if b, err := regexp.MatchString(upper, ps); !b || err != nil {
		return fmt.Errorf("超级管理员密码必须大于16位,且包含大小写字母,数字,特殊字符: %v; 随机示例: %s\n", err, GeneratePasswd(16))
	}
	if b, err := regexp.MatchString(symbol, ps); !b || err != nil {
		return fmt.Errorf("超级管理员密码必须大于16位,且包含大小写字母,数字,特殊字符: %v; 随机示例: %s\n", err, GeneratePasswd(16))
	}
	return nil
}

func GbkToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}

// 定义一个函数来检查一个字符串切片是否包含某一个特定的字符串
func ContainsString(slice []string, str string) bool {
	for _, elem := range slice {
		if elem == str {
			return true
		}
	}
	return false
}
