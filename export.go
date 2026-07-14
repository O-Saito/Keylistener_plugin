package main

/*
#include <stdlib.h>
#include "hostapi.h"

static inline void host_log(bot_host_api_t* api, int level, char* msg) {
	if (api && api->log) api->log(level, msg);
}

static inline char* host_emit_event(bot_host_api_t* api, char* type, char* data, char* target) {
	if (api && api->emit_event) return api->emit_event(type, data, target);
	return NULL;
}

static inline void host_free(bot_host_api_t* api, void* ptr) {
	if (api && api->free) api->free(ptr);
}
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"
)

var hostAPI *C.bot_host_api_t

// ── host API helpers ──

func hostLog(level int, format string, args ...any) {
	if hostAPI == nil {
		return
	}
	msg := C.CString(fmt.Sprintf(format, args...))
	defer C.free(unsafe.Pointer(msg))
	C.host_log(hostAPI, C.int(level), msg)
}

func hostEmitEvent(eventType string, data map[string]any, target string) {
	if hostAPI == nil {
		return
	}
	jsonData, _ := json.Marshal(data)
	ctype := C.CString(eventType)
	cdata := C.CString(string(jsonData))
	ctarget := C.CString(target)
	defer C.free(unsafe.Pointer(ctype))
	defer C.free(unsafe.Pointer(cdata))
	defer C.free(unsafe.Pointer(ctarget))

	result := C.host_emit_event(hostAPI, ctype, cdata, ctarget)
	if result != nil {
		C.host_free(hostAPI, unsafe.Pointer(result))
	}
}

// ── C exports ──

//export plugin_name
func plugin_name() *C.char {
	return C.CString("keylistener")
}

//export plugin_api_version
func plugin_api_version() C.uint32_t {
	return 0x00020000
}

//export plugin_set_host
func plugin_set_host(api *C.bot_host_api_t) {
	hostAPI = api
}

//export plugin_free
func plugin_free(ptr unsafe.Pointer) {
	C.free(ptr)
}

//export plugin_init
func plugin_init(configJSON *C.char) *C.char {
	if configJSON == nil || C.GoString(configJSON) == "" {
		return nil
	}
	hostLog(0, "keylistener initialized")
	return nil
}

//export plugin_start
func plugin_start() {
	keyEvtCh = make(chan queuedKeyEv, 256)
	downKeys = make(map[uint32]struct{})
	startProcessor()
	gamepadEvtCh = make(chan queuedGamepadEv, 64)
	gamepadSubs = make(map[string]struct{})
	startGamepadProcessor()
	if err := startHook(onKeyEvent); err != nil {
		hostLog(3, "keylistener: failed to start evdev hook: %v", err)
	}
}

//export plugin_stop
func plugin_stop() {
	stopHook()
	stopProcessor()
	stopGamepadProcessor()
	p.mu.Lock()
	p.subscribers = make(map[string]struct{})
	p.mu.Unlock()
	gamepadSubsMu.Lock()
	gamepadSubs = make(map[string]struct{})
	gamepadSubsMu.Unlock()
}

//export plugin_get_actions
func plugin_get_actions() *C.char {
	actions, _ := json.Marshal([]map[string]string{
		{"name": "listen"},
		{"name": "stop_listen"},
		{"name": "listen_controller"},
		{"name": "stop_listen_controller"},
	})
	return C.CString(string(actions))
}

//export plugin_call_action
func plugin_call_action(name, jsonArgs, meta *C.char) *C.char {
	goName := C.GoString(name)
	goArgs := C.GoString(jsonArgs)

	switch goName {
	case "listen":
		var moduleName string
		if err := json.Unmarshal([]byte(goArgs), &moduleName); err != nil || moduleName == "" {
			return nil
		}
		p.mu.Lock()
		p.subscribers[moduleName] = struct{}{}
		p.mu.Unlock()
		hostLog(0, "%s subscribed to key events", moduleName)
		return nil
	case "stop_listen":
		var moduleName string
		if err := json.Unmarshal([]byte(goArgs), &moduleName); err != nil || moduleName == "" {
			return nil
		}
		p.mu.Lock()
		delete(p.subscribers, moduleName)
		p.mu.Unlock()
		hostLog(0, "%s unsubscribed from key events", moduleName)
		return nil
	case "listen_controller":
		var moduleName string
		if err := json.Unmarshal([]byte(goArgs), &moduleName); err != nil || moduleName == "" {
			return nil
		}
		gamepadSubsMu.Lock()
		gamepadSubs[moduleName] = struct{}{}
		gamepadSubsMu.Unlock()
		hostLog(0, "%s subscribed to gamepad events", moduleName)
		return nil
	case "stop_listen_controller":
		var moduleName string
		if err := json.Unmarshal([]byte(goArgs), &moduleName); err != nil || moduleName == "" {
			return nil
		}
		gamepadSubsMu.Lock()
		delete(gamepadSubs, moduleName)
		gamepadSubsMu.Unlock()
		hostLog(0, "%s unsubscribed from gamepad events", moduleName)
		return nil
	default:
		return nil
	}
}

func main() {}
