package cmd

import (
	"syscall"
	"unsafe"
)

// detachConsole hides the console window Windows opens for qail.exe when
// `qail tray` is started from a shortcut or Explorer rather than a
// terminal. It only frees the console when no other process (a shell)
// shares it, so `qail tray` typed in a terminal keeps logging there.
func detachConsole() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	var pids [2]uint32
	n, _, _ := kernel32.NewProc("GetConsoleProcessList").Call(uintptr(unsafe.Pointer(&pids[0])), 2)
	if n == 1 {
		_, _, _ = kernel32.NewProc("FreeConsole").Call()
	}
}
