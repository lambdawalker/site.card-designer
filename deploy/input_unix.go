//go:build !windows

package main

import (
	"golang.org/x/sys/unix"
	"os"
	"time"
)

func readKey(f *os.File, d time.Duration) (byte, error) {
	p := []unix.PollFd{{Fd: int32(f.Fd()), Events: unix.POLLIN}}
	n, e := unix.Poll(p, int(d.Milliseconds()))
	if e != nil {
		return 0, e
	}
	if n == 0 {
		return 0, nil
	}
	b := make([]byte, 1)
	n, e = f.Read(b)
	if e != nil {
		return 0, e
	}
	if n == 0 {
		return 0, os.ErrClosed
	}
	return b[0], nil
}
