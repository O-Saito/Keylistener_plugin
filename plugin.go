//go:build windows

package main

import "sync"

type Plugin struct {
	subscribers map[string]struct{}
	mu          sync.RWMutex
}

var p = &Plugin{
	subscribers: make(map[string]struct{}),
}

func onKeyEvent(vkCode uint32, flags uint32, wParam uintptr) {
	if wParam != WM_KEYDOWN && wParam != WM_SYSKEYDOWN {
		return
	}

	ev := buildKeyEvent(vkCode, wParam)

	evMap := map[string]any{
		"vk_code": ev.VkCode,
		"vk_name": ev.VkName,
		"key":     ev.Key,
		"is_down": ev.IsDown,
		"modifiers": map[string]any{
			"shift": ev.Modifiers["shift"],
			"ctrl":  ev.Modifiers["ctrl"],
			"alt":   ev.Modifiers["alt"],
			"caps":  ev.Modifiers["caps"],
		},
	}

	p.mu.RLock()
	modules := make([]string, 0, len(p.subscribers))
	for m := range p.subscribers {
		modules = append(modules, m)
	}
	p.mu.RUnlock()

	for _, module := range modules {
		hostEmitEvent("keypress", evMap, module)
	}
}
