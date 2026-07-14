//go:build windows

package main

import (
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ── RawInput constants ──

const (
	RIM_INPUT    = 0
	RIM_TYPEHID  = 2
	RID_INPUT    = 0x10000003
	RIDI_PREPARSEDDATA = 0x20000005

	WM_INPUT              = 0x00FF
	WM_INPUT_DEVICE_CHANGE = 0x00FE
	WM_CLOSE              = 0x0010
	WM_DESTROY            = 0x0002

	HID_USAGE_PAGE_GENERIC = 0x01
	HID_USAGE_PAGE_BUTTON  = 0x09
	HID_USAGE_GENERIC_GAMEPAD = 0x05
	HID_USAGE_GENERIC_JOYSTICK = 0x04

	HIDP_INPUT = 0
)

const (
	HID_USAGE_GENERIC_X    = 0x30
	HID_USAGE_GENERIC_Y    = 0x31
	HID_USAGE_GENERIC_Z    = 0x32
	HID_USAGE_GENERIC_RX   = 0x33
	HID_USAGE_GENERIC_RY   = 0x34
	HID_USAGE_GENERIC_RZ   = 0x35
	HID_USAGE_GENERIC_HAT  = 0x39
)

const hwndMessage = ^uintptr(2)

// ── Structs ──

type RAWINPUTHEADER struct {
	dwType  uint32
	dwSize  uint32
	hDevice uintptr
	wParam  uintptr
}

type RAWHID struct {
	dwSizeHid uint32
	dwCount   uint32
}

type RAWINPUTDEVICE struct {
	usUsagePage uint16
	usUsage     uint16
	dwFlags     uint32
	hwndTarget  uintptr
}

type WNDCLASSEXW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
}

// ── Public state ──

var (
	rawInputHWND   uintptr
	rawInputClass  atomWNDCLASS
	rawInputWndProcCb uintptr

	// Cached preparsed data handle per device handle
	preparsedHandles   map[uintptr]uintptr
	preparsedHandlesMu sync.Mutex
)

// ── DLLs and procedures ──

var (
	hidDll = windows.NewLazySystemDLL("hid.dll")

	procHidDGetPreparsedData  = hidDll.NewProc("HidD_GetPreparsedData")
	procHidDFreePreparsedData = hidDll.NewProc("HidD_FreePreparsedData")
	procHidPGetUsages         = hidDll.NewProc("HidP_GetUsages")
	procHidPGetUsageValue     = hidDll.NewProc("HidP_GetUsageValue")
)

var (
	procRegisterRawInputDevices = user32.NewProc("RegisterRawInputDevices")
	procGetRawInputData         = user32.NewProc("GetRawInputData")
	procRegisterClassExW        = user32.NewProc("RegisterClassExW")
	procCreateWindowExW         = user32.NewProc("CreateWindowExW")
	procDefWindowProcW          = user32.NewProc("DefWindowProcW")
	procDestroyWindow           = user32.NewProc("DestroyWindow")
	procPostQuitMessage         = user32.NewProc("PostQuitMessage")
	procGetModuleHandleW        = kernel32.NewProc("GetModuleHandleW")
)

// ── Window class atom ──

type atomWNDCLASS uintptr

func init() {
	rawInputWndProcCb = syscall.NewCallback(rawInputWndProc)
}

func rawInputWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_INPUT:
		parseRawInput(lParam)
		return 0
	case WM_INPUT_DEVICE_CHANGE:
		return 0
	case WM_CLOSE:
		procDestroyWindow.Call(hwnd)
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func registerWindowClass() (atomWNDCLASS, error) {
	className := utf16.Encode([]rune("KeylistenerRawInputWindow\x00"))

	hInst, _, _ := procGetModuleHandleW.Call(0)

	wc := WNDCLASSEXW{
		cbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		lpfnWndProc:   rawInputWndProcCb,
		hInstance:     hInst,
		lpszClassName: &className[0],
	}
	ret, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		return 0, windows.GetLastError()
	}
	return atomWNDCLASS(ret), nil
}

func createRawInputWindow() (uintptr, error) {
	className := utf16.Encode([]rune("KeylistenerRawInputWindow\x00"))
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(&className[0])),
		0,
		0,
		0, 0, 0, 0,
		hwndMessage,
		0,
		0,
		0,
	)
	if hwnd == 0 {
		return 0, windows.GetLastError()
	}
	return hwnd, nil
}

func registerRawInputDevice(hwnd uintptr) error {
	dev := RAWINPUTDEVICE{
		usUsagePage: HID_USAGE_PAGE_GENERIC,
		usUsage:     HID_USAGE_GENERIC_GAMEPAD,
		dwFlags:     0x00000100, // RIDEV_INPUTSINK — receive even when background
		hwndTarget:  hwnd,
	}
	ret, _, _ := procRegisterRawInputDevices.Call(
		uintptr(unsafe.Pointer(&dev)),
		1,
		uintptr(unsafe.Sizeof(dev)),
	)
	if ret == 0 {
		return windows.GetLastError()
	}
	return nil
}

// ── HID parsing ──

func getPreparsedData(hDevice uintptr) (uintptr, bool) {
	preparsedHandlesMu.Lock()
	defer preparsedHandlesMu.Unlock()

	if h, ok := preparsedHandles[hDevice]; ok {
		return h, true
	}

	var prep uintptr
	ret, _, _ := procHidDGetPreparsedData.Call(hDevice, uintptr(unsafe.Pointer(&prep)))
	if ret == 0 {
		return 0, false
	}

	preparsedHandles[hDevice] = prep
	return prep, true
}

// ── Button names ──

var btnUsageNames = map[uint16]string{
	1:  "south",
	2:  "east",
	3:  "west",
	4:  "north",
	5:  "l1",
	6:  "r1",
	7:  "select",
	8:  "start",
	9:  "guide",
	10: "l3",
	11: "r3",
	12: "l2",
	13: "r2",
}

// ── State tracking ──

type windowsGamepadState struct {
	buttons     map[uint16]bool
	axes        map[uint16]float64
	lastButtons map[uint16]bool
	lastAxes    map[uint16]float64
	axisRanges  map[uint16]axisRangeInfo
}

type axisRangeInfo struct {
	min float64
	max float64
}

const deadzone = 0.05

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

var (
	windowsGpState   *windowsGamepadState
	windowsGpStateMu sync.Mutex
)

func initWindowsGamepad() {
	windowsGpStateMu.Lock()
	windowsGpState = &windowsGamepadState{
		buttons:     make(map[uint16]bool),
		axes:        make(map[uint16]float64),
		lastButtons: make(map[uint16]bool),
		lastAxes:    make(map[uint16]float64),
		axisRanges:  make(map[uint16]axisRangeInfo),
	}
	windowsGpStateMu.Unlock()
}

func parseRawInput(lParam uintptr) {
	windowsGpStateMu.Lock()
	st := windowsGpState
	windowsGpStateMu.Unlock()
	if st == nil {
		return
	}

	// Get header to find data size
	var header RAWINPUTHEADER
	var size uint32
	procGetRawInputData.Call(
		lParam, RID_INPUT, 0,
		uintptr(unsafe.Pointer(&size)),
		uintptr(unsafe.Sizeof(header)),
	)
	if size == 0 {
		return
	}

	buf := make([]byte, size)
	procGetRawInputData.Call(
		lParam, RID_INPUT,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		uintptr(unsafe.Sizeof(header)),
	)

	// Parse RAWINPUT
	rawHeader := (*RAWINPUTHEADER)(unsafe.Pointer(&buf[0]))
	hDevice := rawHeader.hDevice

	// HID data starts after header + RAWHID
	hidOffset := unsafe.Sizeof(RAWINPUTHEADER{}) + unsafe.Sizeof(RAWHID{})
	if uintptr(len(buf)) <= hidOffset {
		return
	}
	rawHid := (*RAWHID)(unsafe.Pointer(&buf[unsafe.Sizeof(RAWINPUTHEADER{})]))
	report := buf[hidOffset : hidOffset+uintptr(rawHid.dwSizeHid)]

	// Get cached preparsed data handle
	prepPtr, hasPrep := getPreparsedData(hDevice)
	if !hasPrep {
		return
	}

	// Parse buttons
	// We know the preparsed data handles all our 13 buttons.
	// HidP_GetUsages returns the usages that are ON (pressed).
	for code := uint16(1); code <= 13; code++ {
		st.buttons[code] = false
	}

	var btnUsages [32]uint16
	var usageLen uint32 = 32
	status, _, _ := procHidPGetUsages.Call(
		HIDP_INPUT, HID_USAGE_PAGE_BUTTON, 0,
		uintptr(unsafe.Pointer(&btnUsages[0])),
		uintptr(unsafe.Pointer(&usageLen)),
		prepPtr,
		uintptr(unsafe.Pointer(&report[0])),
		uintptr(len(report)),
	)
	if status == 0x00110000 {
		for i := uint32(0); i < usageLen; i++ {
			code := btnUsages[i]
			if code >= 1 && code <= 13 {
				st.buttons[code] = true
			}
		}
	}

	// Parse axes
	axisUsages := []uint16{
		HID_USAGE_GENERIC_X, HID_USAGE_GENERIC_Y,
		HID_USAGE_GENERIC_Z, HID_USAGE_GENERIC_RX,
		HID_USAGE_GENERIC_RY, HID_USAGE_GENERIC_RZ,
		HID_USAGE_GENERIC_HAT,
	}

	for _, usage := range axisUsages {
		var rawValue uint32
		status, _, _ := procHidPGetUsageValue.Call(
			HIDP_INPUT, HID_USAGE_PAGE_GENERIC, 0,
			uintptr(usage),
			uintptr(unsafe.Pointer(&rawValue)),
			prepPtr,
			uintptr(unsafe.Pointer(&report[0])),
			uintptr(len(report)),
		)
		if status != 0x00110000 {
			continue
		}

		// Get range from value caps cache
		rng, hasRange := st.axisRanges[usage]
		if !hasRange {
			// Guess range from the raw value
			rng = guessAxisRange(usage, rawValue)
			st.axisRanges[usage] = rng
		}

		var normalized float64
		if usage == HID_USAGE_GENERIC_HAT {
			// Hat switch: -1 center, 0 up, 2 right, 4 down, 6 left, etc.
			normalized = float64(rawValue)
		} else if rng.max-rng.min > 0 {
			normalized = (float64(rawValue) - rng.min) / (rng.max - rng.min)
		}
		st.axes[usage] = normalized
	}

	// Check for change and emit
	if shouldEmitWindowsGamepad(st) {
		ev := buildWindowsGamepadEvent(st)
		// Update last state
		for code := range st.buttons {
			st.lastButtons[code] = st.buttons[code]
		}
		for usage := range st.axes {
			st.lastAxes[usage] = st.axes[usage]
		}
		queueGamepadEvent(ev)
	}
}

func guessAxisRange(usage uint16, rawValue uint32) axisRangeInfo {
	// For HID gamepads, typical axis ranges:
	// Triggers (Z, Rz): 0-255 or 0-1023
	// Sticks (X, Y, Rx, Ry): 0-65535 or -32768 to 32767 (as twos complement → uint32)
	// Hat: -1 to 7

	if usage == HID_USAGE_GENERIC_HAT {
		return axisRangeInfo{min: -1, max: 7}
	}

	if usage == HID_USAGE_GENERIC_Z || usage == HID_USAGE_GENERIC_RZ {
		// Triggers: if value is small (< 256) assume 0-255, else 0-1023
		if rawValue < 256 {
			return axisRangeInfo{min: 0, max: 255}
		}
		return axisRangeInfo{min: 0, max: 1023}
	}

	// Sticks: check if value pattern looks signed (< 32768 centered at 0) or unsigned
	if rawValue > 65535 {
		return axisRangeInfo{min: 0, max: 65535}
	}
	// Assume unsigned 0-65535 (most common for Xbox/Windows controllers)
	return axisRangeInfo{min: 0, max: 65535}
}

func shouldEmitWindowsGamepad(st *windowsGamepadState) bool {
	for code := uint16(1); code <= 13; code++ {
		if st.buttons[code] != st.lastButtons[code] {
			return true
		}
	}
	for usage, val := range st.axes {
		prev := st.lastAxes[usage]
		if absDiff(val, prev) > deadzone {
			return true
		}
	}
	return false
}

func buildWindowsGamepadEvent(st *windowsGamepadState) map[string]any {
	buttons := make(map[string]bool)
	for code, name := range btnUsageNames {
		buttons[name] = st.buttons[code]
	}

	// D-pad from hat
	if hatVal, ok := st.axes[HID_USAGE_GENERIC_HAT]; ok {
		hat := int(hatVal)
		buttons["dpad_up"] = hat == 0 || hat == 1 || hat == 7
		buttons["dpad_right"] = hat == 1 || hat == 2 || hat == 3
		buttons["dpad_down"] = hat == 3 || hat == 4 || hat == 5
		buttons["dpad_left"] = hat == 5 || hat == 6 || hat == 7
	}

	axes := make(map[string]float64)
	for usage, val := range st.axes {
		if usage == HID_USAGE_GENERIC_HAT {
			continue
		}
		name := axisUsageName(usage)
		if name == "" {
			continue
		}

		// Normalize: val is in 0.0-1.0 range
		// Triggers stay 0.0-1.0, sticks shift to -1.0-1.0
		switch usage {
		case HID_USAGE_GENERIC_X, HID_USAGE_GENERIC_Y,
			HID_USAGE_GENERIC_RX, HID_USAGE_GENERIC_RY:
			axes[name] = val*2.0 - 1.0
		case HID_USAGE_GENERIC_Z, HID_USAGE_GENERIC_RZ:
			axes[name] = val
		default:
			axes[name] = val*2.0 - 1.0
		}
	}

	return map[string]any{
		"buttons": buttons,
		"axes":    axes,
	}
}

func axisUsageName(usage uint16) string {
	switch usage {
	case HID_USAGE_GENERIC_X:
		return "lx"
	case HID_USAGE_GENERIC_Y:
		return "ly"
	case HID_USAGE_GENERIC_RX:
		return "rx"
	case HID_USAGE_GENERIC_RY:
		return "ry"
	case HID_USAGE_GENERIC_Z:
		return "lt"
	case HID_USAGE_GENERIC_RZ:
		return "rt"
	}
	return ""
}

// ── Init / Cleanup ──

func initRawInput() (uintptr, error) {
	preparsedHandles = make(map[uintptr]uintptr)

	classAtom, err := registerWindowClass()
	if err != nil {
		return 0, err
	}
	rawInputClass = classAtom

	hwnd, err := createRawInputWindow()
	if err != nil {
		return 0, err
	}
	rawInputHWND = hwnd

	if err := registerRawInputDevice(hwnd); err != nil {
		// Non-fatal: log and continue without gamepad
		hostLog(2, "rawinput: RegisterRawInputDevices failed: %v", err)
		rawInputHWND = 0
		return 0, err
	}

	initWindowsGamepad()
	hostLog(0, "rawinput: initialized (hwnd=%d)", hwnd)
	return hwnd, nil
}

func cleanupRawInput() {
	preparsedHandlesMu.Lock()
	for hDevice, prep := range preparsedHandles {
		procHidDFreePreparsedData.Call(prep)
		_ = hDevice
	}
	preparsedHandles = nil
	preparsedHandlesMu.Unlock()

	if rawInputHWND != 0 {
		procDestroyWindow.Call(rawInputHWND)
		rawInputHWND = 0
	}
}
