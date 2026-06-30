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

type inputEvent struct {
	Sec   int64
	Usec  int64
	Type  uint16
	Code  uint16
	Value int32
}

var (
	hookFds    []int
	epollFd    int
	hookDone   chan struct{}
	keyHandler func(vkCode uint32, flags uint32, wParam uintptr)
	hookMu     sync.Mutex
)

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

	var fds []int
	for _, dev := range matches {
		fd, err := unix.Open(dev, unix.O_RDONLY|unix.O_NONBLOCK, 0)
		if err != nil {
			continue
		}
		if !isKeyboard(fd) {
			unix.Close(fd)
			continue
		}
		ev := unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(fd)}
		if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &ev); err != nil {
			unix.Close(fd)
			continue
		}
		fds = append(fds, fd)
	}
	if len(fds) == 0 {
		unix.Close(epfd)
		return fmt.Errorf("evdev: no keyboard devices found (user in 'input' group?)")
	}

	epollFd = epfd
	hookFds = fds
	keyHandler = handler

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
	for _, fd := range hookFds {
		unix.Close(fd)
	}
	hookFds = nil

	if hookDone != nil {
		<-hookDone
		hookDone = nil
	}
	keyHandler = nil
}

func isKeyboard(fd int) bool {
	var evBits [(unix.EV_CNT + 63) / 64]uint64
	cmd := unix.EVIOCGBIT(0, len(evBits)*8)
	if _, _, err := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(cmd), uintptr(unsafe.Pointer(&evBits))); err != 0 {
		return false
	}
	if evBits[unix.EV_KEY/64]&(1<<(unix.EV_KEY%64)) == 0 {
		return false
	}

	var keyBits [(unix.KEY_MAX + 63) / 64]uint64
	cmd = unix.EVIOCGBIT(unix.EV_KEY, len(keyBits)*8)
	if _, _, err := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(cmd), uintptr(unsafe.Pointer(&keyBits))); err != 0 {
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
				if ev.Type != 0x01 {
					continue
				}
				if ev.Value == 2 {
					continue
				}
				updateModifiers(ev.Code, ev.Value)

				hookMu.Lock()
				h := keyHandler
				hookMu.Unlock()

				if h != nil {
					h(uint32(ev.Code), 0, uintptr(ev.Value))
				}
			}
		}
	}
}
