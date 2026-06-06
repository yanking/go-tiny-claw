//go:build !windows

package tools

import "syscall"

func newProcessGroupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
