// basicinfo_unix.go
//go:build !windows
// +build !windows

package wsnet

import (
    "os"
    "os/user"
    "runtime"
    "strconv"
)

// BuildGetInfoReply gathers basic system information for non-Windows platforms.
func BuildGetInfoReply() (*WSNGetInfoReply, error) {
    info := &WSNGetInfoReply{
        pid: strconv.Itoa(os.Getpid()),
        os:  normalizePlatform(runtime.GOOS),
    }

    // Current user
    u, err := user.Current()
    if err != nil {
        info.username = "unknown"
    } else {
        info.username = u.Username
    }

    // Hostname
    hostname, err := os.Hostname()
    if err != nil {
        info.hostname = "unknown"
    } else {
        info.hostname = hostname
    }

    info.cpuarch = "X64"
    return info, nil
} 