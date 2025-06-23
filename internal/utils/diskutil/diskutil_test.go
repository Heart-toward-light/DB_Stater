package diskutil

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetFreeDiskGB(t *testing.T) {
	var path string
	if runtime.GOOS == "windows" {
		path = filepath.Clean("C:\\Program Files")
	} else {
		path = "/home"
	}

	size, err := GetFreeDiskGB(path)

	fmt.Printf("path:%s,size:%d,%v\n", path, size, err)
}
