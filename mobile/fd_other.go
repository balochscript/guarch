//go:build !android

package mobile

import "fmt"

func dupFD(fd int) (int, error) {
	return 0, fmt.Errorf("dup not supported on this platform")
}
