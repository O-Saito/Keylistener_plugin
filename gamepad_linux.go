//go:build linux

package main

import (
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	BTN_SOUTH  = 0x130
	BTN_EAST   = 0x131
	BTN_C      = 0x132
	BTN_NORTH  = 0x133
	BTN_WEST   = 0x134
	BTN_TL     = 0x136
	BTN_TR     = 0x137
	BTN_TL2    = 0x138
	BTN_TR2    = 0x139
	BTN_SELECT = 0x13a
	BTN_START  = 0x13b
	BTN_MODE   = 0x13c
	BTN_THUMBL = 0x13d
	BTN_THUMBR = 0x13e
	BTN_DPAD_UP    = 0x220
	BTN_DPAD_DOWN  = 0x222
	BTN_DPAD_LEFT  = 0x221
	BTN_DPAD_RIGHT = 0x223

	ABS_X    = 0x00
	ABS_Y    = 0x01
	ABS_Z    = 0x02
	ABS_RX   = 0x03
	ABS_RY   = 0x04
	ABS_RZ   = 0x05
	ABS_HAT0X = 0x10
	ABS_HAT0Y = 0x11
)

const btnMax = 0x13f

const (
	deadzone      = 0.05
	emitThrottle  = 33 * time.Millisecond
)

var btnNames = map[uint16]string{
	BTN_SOUTH:  "south",
	BTN_EAST:   "east",
	BTN_NORTH:  "north",
	BTN_WEST:   "west",
	BTN_TL:     "l1",
	BTN_TR:     "r1",
	BTN_TL2:    "l2",
	BTN_TR2:    "r2",
	BTN_SELECT: "select",
	BTN_START:  "start",
	BTN_MODE:   "guide",
	BTN_THUMBL: "l3",
	BTN_THUMBR: "r3",
}

type axisInfo struct {
	min int32
	max int32
}

type gamepadState struct {
	buttons     map[uint16]bool
	rawAxes     map[uint16]int32
	lastButtons map[uint16]bool
	lastAxes    map[uint16]float64
	axisRanges  map[uint16]axisInfo
	lastEmit    time.Time
}

var (
	gpStates   map[int]*gamepadState
	gpStatesMu sync.Mutex
)

func initGamepadStates() {
	gpStatesMu.Lock()
	gpStates = make(map[int]*gamepadState)
	gpStatesMu.Unlock()
}

func clearGamepadStates() {
	gpStatesMu.Lock()
	gpStates = nil
	gpStatesMu.Unlock()
}

func getOrCreateState(fd int) *gamepadState {
	gpStatesMu.Lock()
	defer gpStatesMu.Unlock()
	if s, ok := gpStates[fd]; ok {
		return s
	}
	s := &gamepadState{
		buttons:     make(map[uint16]bool),
		rawAxes:     make(map[uint16]int32),
		lastButtons: make(map[uint16]bool),
		lastAxes:    make(map[uint16]float64),
		axisRanges:  queryAxisRanges(fd),
	}
	gpStates[fd] = s
	return s
}

type inputAbsInfo struct {
	Value      int32
	Minimum    int32
	Maximum    int32
	Fuzz       int32
	Flat       int32
	Resolution int32
}

func queryAxisRanges(fd int) map[uint16]axisInfo {
	ranges := make(map[uint16]axisInfo)
	for _, code := range []uint16{ABS_X, ABS_Y, ABS_Z, ABS_RX, ABS_RY, ABS_RZ, ABS_HAT0X, ABS_HAT0Y} {
		var info inputAbsInfo
		cmd := uintptr(2<<30) | uintptr('E'<<8) | uintptr(0x40+code) | uintptr(unsafe.Sizeof(info)<<16)
		if _, _, err := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), cmd, uintptr(unsafe.Pointer(&info))); err == 0 {
			ranges[code] = axisInfo{min: info.Minimum, max: info.Maximum}
		}
	}
	return ranges
}

func isGamepad(fd int) bool {
	var evBits [(unix.EV_CNT + 63) / 64]uint64
	cmd := eviocgbit(0, len(evBits)*8)
	if _, _, err := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), cmd, uintptr(unsafe.Pointer(&evBits))); err != 0 {
		return false
	}
	if evBits[unix.EV_KEY/64]&(1<<(unix.EV_KEY%64)) == 0 {
		return false
	}
	if evBits[unix.EV_ABS/64]&(1<<(unix.EV_ABS%64)) == 0 {
		return false
	}

	var keyBits [(btnMax + 63) / 64]uint64
	cmd = eviocgbit(unix.EV_KEY, len(keyBits)*8)
	if _, _, err := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), cmd, uintptr(unsafe.Pointer(&keyBits))); err != 0 {
		return false
	}
	for code := BTN_SOUTH; code <= BTN_THUMBR; code++ {
		if keyBits[code/64]&(1<<(code%64)) != 0 {
			return true
		}
	}
	return false
}

func isGamepadCode(code uint16) bool {
	return code >= BTN_SOUTH && code <= BTN_THUMBR
}

func updateGamepadButton(fd int, code uint16, value int32) {
	st := getOrCreateState(fd)
	st.buttons[code] = value == 1
}

func updateGamepadAxis(fd int, code uint16, value int32) {
	st := getOrCreateState(fd)
	st.rawAxes[code] = value
}

func normalizeAxis(code uint16, raw int32, r axisInfo) float64 {
	if r.max == r.min {
		return 0
	}
	// Triggers (ABS_Z, ABS_RZ) are 0-based
	if code == ABS_Z || code == ABS_RZ {
		return float64(raw-r.min) / float64(r.max-r.min)
	}
	// Sticks and HAT are signed centered
	centered := float64(raw - (r.min+r.max)/2)
	half := float64(r.max-r.min) / 2
	if half == 0 {
		return 0
	}
	return centered / half
}

func shouldEmitGamepad(st *gamepadState) bool {
	for code := range st.buttons {
		if st.buttons[code] != st.lastButtons[code] {
			return true
		}
	}
	for code, r := range st.axisRanges {
		cur := normalizeAxis(code, st.rawAxes[code], r)
		prev := st.lastAxes[code]
		if absDiff(cur, prev) > deadzone {
			return true
		}
	}
	return false
}

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

func buildGamepadEvent(st *gamepadState) map[string]any {
	buttons := make(map[string]bool)
	for code, name := range btnNames {
		buttons[name] = st.buttons[code]
	}

	// D-pad from hat
	if r, ok := st.axisRanges[ABS_HAT0X]; ok {
		v := normalizeAxis(ABS_HAT0X, st.rawAxes[ABS_HAT0X], r)
		buttons["dpad_left"] = v < -0.5
		buttons["dpad_right"] = v > 0.5
	}
	if r, ok := st.axisRanges[ABS_HAT0Y]; ok {
		v := normalizeAxis(ABS_HAT0Y, st.rawAxes[ABS_HAT0Y], r)
		buttons["dpad_up"] = v < -0.5
		buttons["dpad_down"] = v > 0.5
	}

	axes := make(map[string]float64)
	for code, r := range st.axisRanges {
		name := axisName(code)
		if name != "" {
			axes[name] = normalizeAxis(code, st.rawAxes[code], r)
		}
	}

	return map[string]any{
		"buttons": buttons,
		"axes":    axes,
	}
}

func axisName(code uint16) string {
	switch code {
	case ABS_X:
		return "lx"
	case ABS_Y:
		return "ly"
	case ABS_RX:
		return "rx"
	case ABS_RY:
		return "ry"
	case ABS_Z:
		return "lt"
	case ABS_RZ:
		return "rt"
	}
	return ""
}

func emitGamepadIfChanged(fd int) {
	gpStatesMu.Lock()
	st := gpStates[fd]
	gpStatesMu.Unlock()
	if st == nil {
		return
	}

	if !shouldEmitGamepad(st) {
		return
	}

	ev := buildGamepadEvent(st)

	// Update last known state
	for code := range st.buttons {
		st.lastButtons[code] = st.buttons[code]
	}
	for code, r := range st.axisRanges {
		st.lastAxes[code] = normalizeAxis(code, st.rawAxes[code], r)
	}
	st.lastEmit = time.Now()

	queueGamepadEvent(ev)
}
