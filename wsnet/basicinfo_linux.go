// basicinfo_linux.go
//go:build !windows
// +build !windows

package wsnet

import (
	"os"
	"os/user"
	"runtime"
	"strconv"
)

func BuildGetInfoReply() (*WSNGetInfoReply, error) {
	info := &WSNGetInfoReply{
		pid: strconv.Itoa(os.Getpid()),
		os:  normalizePlatform(runtime.GOOS),
	}

	// Get the current user
	user, err := user.Current()
	if err != nil {
		info.username = "unknown"
	} else {
		info.username = user.Username
	}

	// Get the hostname
	hostname, err := os.Hostname()
	if err != nil {
		info.hostname = "unknown"
	} else {
		info.hostname = hostname
	}

	// CPU Architecture
	info.cpuarch = "X64"

	return info, nil
}
