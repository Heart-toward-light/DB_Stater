/*
@Author : WuWeiJian
@Date : 2020-12-03 21:01
*/

package utils

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

// 判断文件是否存在
func IsExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

// 判断所给路径是否为目录, 是否为文件可以用 !IsDir
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// 目录是否为空
func IsEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// 验证数据目录, 存在看是否为空
func ValidatorDir(dir string) error {
	if !IsExists(dir) {
		//if err := os.MkdirAll(dir, 0755); err != nil {
		//	return err
		//}
		return nil
	}

	if !IsDir(dir) {
		return fmt.Errorf("数据目录(%s)异常", dir)
	}

	empty, err := IsEmpty(dir)
	if err != nil {
		return err
	}
	if !empty {
		return fmt.Errorf("数据目录(%s)不为空", dir)
	}

	return nil
}

// 生成指定文件的md5效验码
func CheckMd5sumByFile(file string) (string, error) {
	tarball, err := os.OpenFile(file, os.O_RDONLY, 0)
	if err != nil {
		return "", err
	}
	defer tarball.Close()
	m := md5.New()
	if _, err := io.Copy(m, tarball); err != nil {
		return "", err
	}

	checksum := hex.EncodeToString(m.Sum(nil))
	return checksum, nil
}

// 生成指定字符数组的md5效验码
func CheckMd5sumByByte(b []byte) string {
	m := md5.New()
	m.Write(b)
	return hex.EncodeToString(m.Sum(nil))
}

func decFile(hdr *tar.Header, tr *tar.Reader, to string) error {
	file := path.Join(to, hdr.Name)
	if dir := filepath.Dir(file); !IsExists(dir) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	switch hdr.Typeflag {
	case tar.TypeSymlink:
		if err := os.Symlink(hdr.Linkname, file); err != nil {
			return fmt.Errorf("解压包中的软链接文件失败: %v", err)
		}
	default:
		fw, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}
		defer fw.Close()

		_, err = io.Copy(fw, tr)
		return err
	}
	return nil
}

// 解压tar.gz文件
func UntarGz(from string, to string) error {
	fr, err := os.Open(from)
	if err != nil {
		return err
	}
	defer fr.Close()

	gr, err := gzip.NewReader(fr)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(path.Join(to, hdr.Name), hdr.FileInfo().Mode()); err != nil {
				return err
			}
		} else {
			if err := decFile(hdr, tr, to); err != nil {
				return err
			}
		}
	}
	return nil
}

// 开机自动创建 /var/run 目录下的文件或路径
func CreateRunDir(filename, dir, user, group string) error {
	filename = filepath.Join("/usr/lib/tmpfiles.d", filename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	title := fmt.Sprintf("d /var/run/%s 0755 %s %s", dir, user, group)
	if _, err := fmt.Fprintln(w, title); err != nil {
		return err
	}
	return w.Flush()
}

func WriteToFile(filename string, content string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if _, err = w.WriteString(content); err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	return w.Flush()
}

func ReadLineFromFile(filename string) ([]string, error) {
	var content []string
	f, err := os.Open(filename)
	if err != nil {
		return content, fmt.Errorf("打开文件失败: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		content = append(content, scanner.Text())
	}

	return content, nil
}
