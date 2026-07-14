//go:build linux

package main

import (
	"fmt"
	"path/filepath"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

const sizeofInputEvent = 24

const KEY_MAX = 0x2ff

func eviocgbit(ev, size int) uintptr {
	return uintptr(2<<30) | uintptr('E'<<8) | uintptr(0x20+ev) | uintptr(size<<16)
}

type inputEvent struct {
	Sec   int64
	Usec  int64
	Type  uint16
	Code  uint16
	Value int32
}

type deviceInfo struct {
	fd        int
	isGamepad bool
}

var (
	devices    []deviceInfo
	epollFd    int
	hookDone   chan struct{}
	keyHandler func(vkCode uint32, flags uint32, wParam uintptr)
	hookMu     sync.Mutex
)

func isGamepadFD(fd int) bool {
	for _, d := range devices {
		if d.fd == fd {
			return d.isGamepad
		}
	}
	return false
}

func startHook(handler func(vkCode uint32, flags uint32, wParam uintptr)) error {
	hookMu.Lock()
	defer hookMu.Unlock()

	if epollFd > 0 {
		stopHook()
	}

	matches, err := filepath.Glob("/dev/input/event*")
	if err != nil || len(matches) == 0 {
		return fmt.Errorf("evdev: no input devices at /dev/input/event*")
	}

	epfd, err := unix.EpollCreate1(0)
	if err != nil {
		return fmt.Errorf("evdev: epoll_create1: %w", err)
	}

	var devs []deviceInfo
	for _, dev := range matches {
		fd, err := unix.Open(dev, unix.O_RDONLY|unix.O_NONBLOCK, 0)
		if err != nil {
			continue
		}
		isKB := isKeyboard(fd)
		isGP := isGamepad(fd)
		if !isKB && !isGP {
			unix.Close(fd)
			continue
		}
		ev := unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(fd)}
		if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &ev); err != nil {
			unix.Close(fd)
			continue
		}
		devs = append(devs, deviceInfo{fd, isGP})
	}
	if len(devs) == 0 {
		unix.Close(epfd)
		return fmt.Errorf("evdev: no input devices found (user in 'input' group?)")
	}

	epollFd = epfd
	devices = devs
	keyHandler = handler
	initGamepadStates()

	done := make(chan struct{})
	hookDone = done
	go eventLoop(done)
	return nil
}

func stopHook() {
	hookMu.Lock()
	defer hookMu.Unlock()

	if epollFd > 0 {
		unix.Close(epollFd)
		epollFd = 0
	}
	for _, d := range devices {
		unix.Close(d.fd)
	}
	devices = nil

	if hookDone != nil {
		<-hookDone
		hookDone = nil
	}
	keyHandler = nil
	clearGamepadStates()
}

func isKeyboard(fd int) bool {
	var evBits [(unix.EV_CNT + 63) / 64]uint64
	cmd := eviocgbit(0, len(evBits)*8)
	if _, _, err := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), cmd, uintptr(unsafe.Pointer(&evBits))); err != 0 {
		return false
	}
	if evBits[unix.EV_KEY/64]&(1<<(unix.EV_KEY%64)) == 0 {
		return false
	}

	var keyBits [(KEY_MAX + 63) / 64]uint64
	cmd = eviocgbit(unix.EV_KEY, len(keyBits)*8)
	if _, _, err := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), cmd, uintptr(unsafe.Pointer(&keyBits))); err != 0 {
		return false
	}
	const keyQ = 16
	const keyA = 30
	if keyBits[keyQ/64]&(1<<(keyQ%64)) != 0 {
		return true
	}
	return keyBits[keyA/64]&(1<<(keyA%64)) != 0
}

func eventLoop(done chan struct{}) {
	defer close(done)

	events := make([]unix.EpollEvent, 16)

	for {
		n, err := unix.EpollWait(epollFd, events, -1)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return
		}
		for i := 0; i < n; i++ {
			fd := int(events[i].Fd)
			var ev inputEvent
			buf := (*(*[sizeofInputEvent]byte)(unsafe.Pointer(&ev)))[:]
			for {
				m, err := unix.Read(fd, buf)
				if err != nil || m < sizeofInputEvent {
					break
				}

				switch ev.Type {
				case 0x01: // EV_KEY
					if ev.Value == 2 {
						continue
					}
					if isGamepadCode(ev.Code) {
						updateGamepadButton(fd, ev.Code, ev.Value)
						continue
					}
					updateModifiers(ev.Code, ev.Value)

					hookMu.Lock()
					h := keyHandler
					hookMu.Unlock()

					if h != nil {
						h(uint32(ev.Code), 0, uintptr(ev.Value))
					}

				case 0x03: // EV_ABS
					updateGamepadAxis(fd, ev.Code, ev.Value)

				case 0x00: // EV_SYN
					if isGamepadFD(fd) {
						emitGamepadIfChanged(fd)
					}

				default:
				}
			}
		}
	}
}
