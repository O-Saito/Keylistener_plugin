//go:build windows

package main

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procSetWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessage          = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procGetKeyState         = user32.NewProc("GetKeyState")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procPostThreadMessageW  = user32.NewProc("PostThreadMessageW")
	procGetCurrentThreadId  = kernel32.NewProc("GetCurrentThreadId")
)

const (
	WH_KEYBOARD_LL = 13

	WM_KEYDOWN    = 0x0100
	WM_KEYUP      = 0x0101
	WM_SYSKEYDOWN = 0x0104
	WM_SYSKEYUP   = 0x0105
	WM_QUIT       = 0x0012
)

type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	PtX     int32
	PtY     int32
}

var (
	hookHandle   uintptr
	hookThreadID uintptr
	hookDone     chan struct{}
	keyHandler   func(vkCode uint32, flags uint32, wParam uintptr)

	hookCallbackOnce sync.Once
	hookCallbackFn   uintptr
)

func getHookCallback() uintptr {
	hookCallbackOnce.Do(func() {
		hookCallbackFn = syscall.NewCallback(func(nCode int, wParam, lParam uintptr) uintptr {
			if nCode >= 0 && keyHandler != nil {
				kbd := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
				keyHandler(kbd.VkCode, kbd.Flags, wParam)
			}
			ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
			return ret
		})
	})
	return hookCallbackFn
}

func startHook(handler func(vkCode uint32, flags uint32, wParam uintptr)) error {
	if hookHandle != 0 {
		stopHook()
	}

	keyHandler = handler

	cb := getHookCallback()

	ready := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		// Initialize raw input window for gamepad (best-effort)
		gpHwnd, gpErr := initRawInput()
		if gpErr != nil {
			hostLog(2, "keylistener: rawinput init failed, gamepad unavailable: %v", gpErr)
		}

		done := make(chan struct{})
		defer close(done)

		h, _, _ := procSetWindowsHookEx.Call(
			WH_KEYBOARD_LL,
			cb,
			0,
			0,
		)
		if h == 0 {
			if gpHwnd != 0 {
				cleanupRawInput()
			}
			ready <- fmt.Errorf("SetWindowsHookEx failed")
			return
		}

		tid, _, _ := procGetCurrentThreadId.Call()
		hookHandle = h
		hookThreadID = tid
		hookDone = done
		ready <- nil

		var msg MSG
		for {
			ret, _, _ := procGetMessage.Call(
				uintptr(unsafe.Pointer(&msg)),
				0,
				0,
				0,
			)
			if ret == 0 {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}

		if gpHwnd != 0 {
			cleanupRawInput()
		}
		hookHandle = 0
		hookThreadID = 0
	}()

	return <-ready
}

func stopHook() {
	h := hookHandle
	tid := hookThreadID
	hookHandle = 0
	hookThreadID = 0

	if h != 0 {
		procUnhookWindowsHookEx.Call(h)
	}

	// Post WM_CLOSE to the raw input hidden window if available;
	// falls back to PostThreadMessageW with WM_QUIT.
	if rawInputHWND != 0 {
		procPostMessageW.Call(rawInputHWND, WM_CLOSE, 0, 0)
	} else if tid != 0 {
		procPostThreadMessageW.Call(tid, WM_QUIT, 0, 0)
	}

	if hookDone != nil {
		<-hookDone
		hookDone = nil
	}

	keyHandler = nil
}
