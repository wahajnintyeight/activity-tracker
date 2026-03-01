package idle

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	getLastInputInfo = user32.NewProc("GetLastInputInfo")
	getTickCount     = kernel32.NewProc("GetTickCount")
)

type LASTINPUTINFO struct {
	cbSize uint32
	dwTime uint32
}

// GetIdleTime returns how long the user has been idle
func GetIdleTime() (time.Duration, error) {
	var lastInput LASTINPUTINFO
	lastInput.cbSize = uint32(unsafe.Sizeof(lastInput))

	ret, _, err := getLastInputInfo.Call(uintptr(unsafe.Pointer(&lastInput)))
	if ret == 0 {
		return 0, err
	}

	// Get current tick count
	currentTick, _, _ := getTickCount.Call()

	// Calculate idle time in milliseconds
	idleMs := uint32(currentTick) - lastInput.dwTime

	return time.Duration(idleMs) * time.Millisecond, nil
}

// IsIdle checks if user has been idle for longer than the threshold
func IsIdle(threshold time.Duration) (bool, error) {
	idleTime, err := GetIdleTime()
	if err != nil {
		return false, err
	}

	return idleTime >= threshold, nil
}
