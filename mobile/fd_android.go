//go:build android

package mobile

import "syscall"

func dupFD(fd int) (int, error) {
	return syscall.Dup(fd)
}
