//go:build windows

package main

import (
	"golang.org/x/sys/windows"
	"os"
	"time"
)

func readKey(f *os.File, d time.Duration) (byte, error) {
	r, e := windows.WaitForSingleObject(windows.Handle(f.Fd()), uint32(d.Milliseconds()))
	if e != nil {
		return 0, e
	}
	if r == uint32(windows.WAIT_TIMEOUT) {
		return 0, nil
	}
	b := make([]byte, 1)
	_, e = f.Read(b)
	return b[0], e
}
