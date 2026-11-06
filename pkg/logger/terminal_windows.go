//go:build windows

package logger

import (
	"io"
	"os"
	"syscall"
	"unsafe"
)

const enableVirtualTerminalProcessing = 0x0004

var (
	kernel32   = syscall.NewLazyDLL("kernel32.dll")
	getConMode = kernel32.NewProc("GetConsoleMode")
	setConMode = kernel32.NewProc("SetConsoleMode")
)

func enableANSI(w io.Writer) {
	f, ok := w.(*os.File)
	if !ok {
		return
	}
	var mode uint32
	r, _, _ := getConMode.Call(f.Fd(), uintptr(unsafe.Pointer(&mode)))
	if r == 0 || mode&enableVirtualTerminalProcessing != 0 {
		return
	}
	_, _, _ = setConMode.Call(f.Fd(), uintptr(mode|enableVirtualTerminalProcessing))
}
