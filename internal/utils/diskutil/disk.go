package diskutil

import (
	"dbup/internal/utils/arrlib"
	"fmt"
	"github.com/shirou/gopsutil/disk"
	"path/filepath"
	"strings"
)

const (
	BYTE = 1 << (10 * iota)
	KILOBYTE
	MEGABYTE
	GIGABYTE
	TERABYTE
	PETABYTE
	EXABYTE
)

func GetFreeDiskGB(path string) (d uint64, err error) {
	size,err := GetFreeDiskByte(path)
	if err != nil {
		return
	}

	d = size / GIGABYTE
	return
}

func GetFreeDiskMB(path string) (d uint64, err error) {
	size,err := GetFreeDiskByte(path)
	if err != nil {
		return
	}

	d = size /MEGABYTE
	return
}

func GetFreeDiskByte(path string) (d uint64, err error) {
	parts, err := disk.Partitions(true)
	if err != nil {
		return
	}

	paths := strings.Split(filepath.Clean(path), string(filepath.Separator))

	var maxLen int
	var maxMp string

	for _, part := range parts {
		fstype := strings.ToLower(part.Fstype)
		if !arrlib.InArray(fstype, []string{"ext4", "xfs", "apfs", "ntfs"}) {
			continue
		}

		mpPaths := strings.Split(part.Mountpoint, string(filepath.Separator))

		// 获取两个路径的最长匹配路径段
		max := maxMatchCnt(paths, mpPaths)

		if max > maxLen {
			maxLen = max
			maxMp = part.Mountpoint
		}

	}

	if maxLen == 0 {
		err = fmt.Errorf("未找到 %s 对应的挂载点", path)
		return
	}

	diskInfo, err := disk.Usage(maxMp)
	if err != nil {
		return
	}

	return diskInfo.Free, nil
}

func maxMatchCnt(p1, p2 []string) (max int) {
	pLen := len(p1)
	mpLen := len(p2)
	if pLen > mpLen {
		max = mpLen
		for i, p := range p2 {
			if p != p1[i] {
				max = i
				break
			}
		}
	} else {
		max = pLen
		for i, p := range p1 {
			if p != p2[i] {
				max = i
				break
			}
		}
	}
	return
}
