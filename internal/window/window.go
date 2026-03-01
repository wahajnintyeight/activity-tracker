package window

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	getForegroundWin = user32.NewProc("GetForegroundWindow")
	getWindowText    = user32.NewProc("GetWindowTextW")
	getWindowThread  = user32.NewProc("GetWindowThreadProcessId")
	openProcess      = kernel32.NewProc("OpenProcess")
	queryFullName    = kernel32.NewProc("QueryFullProcessImageNameW")
)

type ActiveWindow struct {
	Title       string
	ProcessName string
	ProcessID   uint32
}

// GetActive returns information about the currently focused window
func GetActive() (*ActiveWindow, error) {
	hwnd, _, _ := getForegroundWin.Call()
	if hwnd == 0 {
		return nil, fmt.Errorf("no active window")
	}

	// Get window title
	textLen := 256
	buf := make([]uint16, textLen)
	getWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(textLen))
	title := syscall.UTF16ToString(buf)

	// Get process ID
	var processID uint32
	getWindowThread.Call(hwnd, uintptr(unsafe.Pointer(&processID)))

	// Get process name
	processName := ""
	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	hProcess, _, _ := openProcess.Call(PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(processID))
	if hProcess != 0 {
		defer windows.CloseHandle(windows.Handle(hProcess))

		var size uint32 = 260
		nameBuf := make([]uint16, size)
		queryFullName.Call(
			hProcess,
			0,
			uintptr(unsafe.Pointer(&nameBuf[0])),
			uintptr(unsafe.Pointer(&size)),
		)
		fullPath := syscall.UTF16ToString(nameBuf)

		// Extract just the executable name from full path
		for i := len(fullPath) - 1; i >= 0; i-- {
			if fullPath[i] == '\\' {
				processName = fullPath[i+1:]
				break
			}
		}
		if processName == "" {
			processName = fullPath
		}
	}

	return &ActiveWindow{
		Title:       title,
		ProcessName: processName,
		ProcessID:   processID,
	}, nil
}
