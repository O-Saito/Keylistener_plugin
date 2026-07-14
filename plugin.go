package main

import (
	"strings"
	"sync"
)

type Plugin struct {
	subscribers map[string]struct{}
	mu          sync.RWMutex
}

var p = &Plugin{
	subscribers: make(map[string]struct{}),
}

type queuedKeyEv struct {
	evMap map[string]any
}

var (
	keyEvtCh      chan queuedKeyEv
	downKeys      map[uint32]struct{}
	downKeysMu    sync.Mutex
	processorStop chan struct{}
	processorDone chan struct{}
)

type queuedGamepadEv struct {
	evMap map[string]any
}

var (
	gamepadEvtCh          chan queuedGamepadEv
	gamepadSubs           map[string]struct{}
	gamepadSubsMu         sync.RWMutex
	gamepadProcessorStop  chan struct{}
	gamepadProcessorDone  chan struct{}
)

func onKeyEvent(vkCode uint32, flags uint32, wParam uintptr) {
	ev := buildKeyEvent(vkCode, wParam)

	if ev.IsDown {
		downKeysMu.Lock()
		if _, held := downKeys[vkCode]; held {
			downKeysMu.Unlock()
			return
		}
		downKeys[vkCode] = struct{}{}
		downKeysMu.Unlock()
	} else {
		downKeysMu.Lock()
		delete(downKeys, vkCode)
		downKeysMu.Unlock()
	}

	vkName := ev.VkName
	if strings.HasPrefix(vkName, "KEY_") || strings.HasPrefix(vkName, "VK_") {
		vkName = vkName[4:]
	}

	evMap := map[string]any{
		"vk_code": ev.VkCode,
		"vk_name": vkName,
		"key":     ev.Key,
		"is_down": ev.IsDown,
		"modifiers": map[string]any{
			"shift": ev.Modifiers["shift"],
			"ctrl":  ev.Modifiers["ctrl"],
			"alt":   ev.Modifiers["alt"],
			"caps":  ev.Modifiers["caps"],
		},
	}

	select {
	case keyEvtCh <- queuedKeyEv{evMap}:
	default:
	}
}

func startProcessor() {
	processorStop = make(chan struct{})
	processorDone = make(chan struct{})

	go func() {
		defer close(processorDone)
		for {
			select {
			case <-processorStop:
				return
			case qe := <-keyEvtCh:
				p.mu.RLock()
				modules := make([]string, 0, len(p.subscribers))
				for m := range p.subscribers {
					modules = append(modules, m)
				}
				p.mu.RUnlock()

				for _, module := range modules {
					hostEmitEvent("keypress", qe.evMap, module)
				}
			}
		}
	}()
}

func stopProcessor() {
	if processorStop != nil {
		close(processorStop)
		<-processorDone
	}
}

func queueGamepadEvent(data map[string]any) {
	if gamepadSubs == nil || gamepadEvtCh == nil {
		return
	}
	gamepadSubsMu.RLock()
	hasSubs := len(gamepadSubs) > 0
	gamepadSubsMu.RUnlock()
	if !hasSubs {
		return
	}
	select {
	case gamepadEvtCh <- queuedGamepadEv{data}:
	default:
	}
}

func startGamepadProcessor() {
	gamepadProcessorStop = make(chan struct{})
	gamepadProcessorDone = make(chan struct{})

	go func() {
		defer close(gamepadProcessorDone)
		for {
			select {
			case <-gamepadProcessorStop:
				return
			case qe := <-gamepadEvtCh:
				gamepadSubsMu.RLock()
				modules := make([]string, 0, len(gamepadSubs))
				for m := range gamepadSubs {
					modules = append(modules, m)
				}
				gamepadSubsMu.RUnlock()

				for _, module := range modules {
					hostEmitEvent("gamepad", qe.evMap, module)
				}
			}
		}
	}()
}

func stopGamepadProcessor() {
	if gamepadProcessorStop != nil {
		close(gamepadProcessorStop)
		<-gamepadProcessorDone
	}
}
