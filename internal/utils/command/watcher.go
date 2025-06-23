/*
@Author : WuWeiJian
@Date : 2020-12-02 21:39
*/

package command

import (
	"bufio"
	"io"
	"strings"
)

func watcher(in io.WriteCloser, out io.Reader, output *[]byte, password string) {
	var line string
	var r = bufio.NewReader(out)
	for {
		b, err := r.ReadByte()
		if err != nil {
			break
		}

		*output = append(*output, b)

		if b == byte('\n') {
			line = ""
			continue
		}

		line += string(b)

		if strings.HasPrefix(line, "[sudo] password for ") && strings.HasSuffix(line, ": ") {
			_, err = in.Write([]byte(password + "\n"))
			if err != nil {
				break
			}
		}
	}
}
