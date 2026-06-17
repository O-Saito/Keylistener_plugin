//go:build windows

package main

import (
	"golang.org/x/sys/windows"
)

type KeyEvent struct {
	VkCode    uint32          `json:"vk_code"`
	VkName    string          `json:"vk_name"`
	Key       string          `json:"key"`
	IsDown    bool            `json:"is_down"`
	Modifiers map[string]bool `json:"modifiers"`
}

func buildKeyEvent(vkCode uint32, wParam uintptr) KeyEvent {
	isDown := wParam == WM_KEYDOWN || wParam == WM_SYSKEYDOWN

	mods := map[string]bool{
		"shift": getKeyState(windows.VK_SHIFT),
		"ctrl":  getKeyState(windows.VK_CONTROL),
		"alt":   getKeyState(windows.VK_MENU),
		"caps":  getCapsState(),
	}

	vkName := vkCodeToName(vkCode)
	key := vkCodeToChar(vkCode, mods["shift"], mods["caps"])

	return KeyEvent{
		VkCode:    vkCode,
		VkName:    vkName,
		Key:       key,
		IsDown:    isDown,
		Modifiers: mods,
	}
}

func getKeyState(vk int) bool {
	state, _, _ := procGetKeyState.Call(uintptr(vk))
	return state&0x8000 != 0
}

func getCapsState() bool {
	state, _, _ := procGetKeyState.Call(windows.VK_CAPITAL)
	return state&1 != 0
}

func vkCodeToName(code uint32) string {
	if name, ok := vkNames[code]; ok {
		return name
	}
	return ""
}

func vkCodeToChar(code uint32, shift, caps bool) string {
	if code >= '0' && code <= '9' {
		if shift {
			return shiftDigits[code]
		}
		return string(rune(code))
	}
	if code >= 'A' && code <= 'Z' {
		if shift != caps {
			return string(rune(code))
		}
		return string(rune(code + 32))
	}
	if ch, ok := vkChars[code]; ok {
		return ch
	}
	if name, ok := vkNames[code]; ok {
		return name
	}
	return ""
}

var shiftDigits = map[uint32]string{
	'0': ")", '1': "!", '2': "@", '3': "#", '4': "$",
	'5': "%", '6': "^", '7': "&", '8': "*", '9': "(",
}

var vkChars = map[uint32]string{
	windows.VK_OEM_1:      ";",
	windows.VK_OEM_PLUS:   "=",
	windows.VK_OEM_COMMA:  ",",
	windows.VK_OEM_MINUS:  "-",
	windows.VK_OEM_PERIOD: ".",
	windows.VK_OEM_2:      "/",
	windows.VK_OEM_3:      "`",
	windows.VK_OEM_4:      "[",
	windows.VK_OEM_5:      "\\",
	windows.VK_OEM_6:      "]",
	windows.VK_OEM_7:      "'",
	windows.VK_SPACE:      " ",
	windows.VK_RETURN:     "enter",
	windows.VK_BACK:       "backspace",
	windows.VK_TAB:        "tab",
	windows.VK_ESCAPE:     "escape",
	windows.VK_DELETE:     "delete",
	windows.VK_HOME:       "home",
	windows.VK_END:        "end",
	windows.VK_PRIOR:      "page_up",
	windows.VK_NEXT:       "page_down",
	windows.VK_INSERT:     "insert",
	windows.VK_LEFT:       "left",
	windows.VK_RIGHT:      "right",
	windows.VK_UP:         "up",
	windows.VK_DOWN:       "down",
	windows.VK_SNAPSHOT:   "print_screen",
	windows.VK_SCROLL:     "scroll_lock",
	windows.VK_PAUSE:      "pause",
	windows.VK_NUMLOCK:    "num_lock",
	windows.VK_CAPITAL:    "caps_lock",
	windows.VK_CLEAR:      "clear",
	windows.VK_APPS:       "apps",
	windows.VK_SELECT:     "select",
	windows.VK_EXECUTE:    "execute",
	windows.VK_HELP:       "help",
	windows.VK_LWIN:       "left_win",
	windows.VK_RWIN:       "right_win",
	windows.VK_SLEEP:      "sleep",
	windows.VK_ZOOM:       "zoom",
	windows.VK_SEPARATOR:  "separator",
	windows.VK_DIVIDE:     "divide",
	windows.VK_MULTIPLY:   "multiply",
	windows.VK_SUBTRACT:   "subtract",
	windows.VK_ADD:        "add",
	windows.VK_DECIMAL:    "decimal",
}

var vkNames = map[uint32]string{
	windows.VK_LBUTTON:             "VK_LBUTTON",
	windows.VK_RBUTTON:             "VK_RBUTTON",
	windows.VK_CANCEL:              "VK_CANCEL",
	windows.VK_MBUTTON:             "VK_MBUTTON",
	windows.VK_XBUTTON1:            "VK_XBUTTON1",
	windows.VK_XBUTTON2:            "VK_XBUTTON2",
	windows.VK_BACK:                "VK_BACK",
	windows.VK_TAB:                 "VK_TAB",
	windows.VK_CLEAR:               "VK_CLEAR",
	windows.VK_RETURN:              "VK_RETURN",
	windows.VK_SHIFT:               "VK_SHIFT",
	windows.VK_CONTROL:             "VK_CONTROL",
	windows.VK_MENU:                "VK_MENU",
	windows.VK_PAUSE:               "VK_PAUSE",
	windows.VK_CAPITAL:             "VK_CAPITAL",
	windows.VK_ESCAPE:              "VK_ESCAPE",
	windows.VK_SPACE:               "VK_SPACE",
	windows.VK_PRIOR:               "VK_PRIOR",
	windows.VK_NEXT:                "VK_NEXT",
	windows.VK_END:                 "VK_END",
	windows.VK_HOME:                "VK_HOME",
	windows.VK_LEFT:                "VK_LEFT",
	windows.VK_UP:                  "VK_UP",
	windows.VK_RIGHT:               "VK_RIGHT",
	windows.VK_DOWN:                "VK_DOWN",
	windows.VK_SELECT:              "VK_SELECT",
	windows.VK_PRINT:               "VK_PRINT",
	windows.VK_EXECUTE:             "VK_EXECUTE",
	windows.VK_SNAPSHOT:            "VK_SNAPSHOT",
	windows.VK_INSERT:              "VK_INSERT",
	windows.VK_DELETE:              "VK_DELETE",
	windows.VK_HELP:                "VK_HELP",
	windows.VK_LWIN:                "VK_LWIN",
	windows.VK_RWIN:                "VK_RWIN",
	windows.VK_APPS:                "VK_APPS",
	windows.VK_SLEEP:               "VK_SLEEP",
	windows.VK_NUMPAD0:             "VK_NUMPAD0",
	windows.VK_NUMPAD1:             "VK_NUMPAD1",
	windows.VK_NUMPAD2:             "VK_NUMPAD2",
	windows.VK_NUMPAD3:             "VK_NUMPAD3",
	windows.VK_NUMPAD4:             "VK_NUMPAD4",
	windows.VK_NUMPAD5:             "VK_NUMPAD5",
	windows.VK_NUMPAD6:             "VK_NUMPAD6",
	windows.VK_NUMPAD7:             "VK_NUMPAD7",
	windows.VK_NUMPAD8:             "VK_NUMPAD8",
	windows.VK_NUMPAD9:             "VK_NUMPAD9",
	windows.VK_MULTIPLY:            "VK_MULTIPLY",
	windows.VK_ADD:                 "VK_ADD",
	windows.VK_SEPARATOR:           "VK_SEPARATOR",
	windows.VK_SUBTRACT:            "VK_SUBTRACT",
	windows.VK_DECIMAL:             "VK_DECIMAL",
	windows.VK_DIVIDE:              "VK_DIVIDE",
	windows.VK_F1:                  "VK_F1",
	windows.VK_F2:                  "VK_F2",
	windows.VK_F3:                  "VK_F3",
	windows.VK_F4:                  "VK_F4",
	windows.VK_F5:                  "VK_F5",
	windows.VK_F6:                  "VK_F6",
	windows.VK_F7:                  "VK_F7",
	windows.VK_F8:                  "VK_F8",
	windows.VK_F9:                  "VK_F9",
	windows.VK_F10:                 "VK_F10",
	windows.VK_F11:                 "VK_F11",
	windows.VK_F12:                 "VK_F12",
	windows.VK_F13:                 "VK_F13",
	windows.VK_F14:                 "VK_F14",
	windows.VK_F15:                 "VK_F15",
	windows.VK_F16:                 "VK_F16",
	windows.VK_F17:                 "VK_F17",
	windows.VK_F18:                 "VK_F18",
	windows.VK_F19:                 "VK_F19",
	windows.VK_F20:                 "VK_F20",
	windows.VK_F21:                 "VK_F21",
	windows.VK_F22:                 "VK_F22",
	windows.VK_F23:                 "VK_F23",
	windows.VK_F24:                 "VK_F24",
	windows.VK_NUMLOCK:             "VK_NUMLOCK",
	windows.VK_SCROLL:              "VK_SCROLL",
	windows.VK_LSHIFT:              "VK_LSHIFT",
	windows.VK_RSHIFT:              "VK_RSHIFT",
	windows.VK_LCONTROL:            "VK_LCONTROL",
	windows.VK_RCONTROL:            "VK_RCONTROL",
	windows.VK_LMENU:               "VK_LMENU",
	windows.VK_RMENU:               "VK_RMENU",
	windows.VK_BROWSER_BACK:        "VK_BROWSER_BACK",
	windows.VK_BROWSER_FORWARD:     "VK_BROWSER_FORWARD",
	windows.VK_BROWSER_REFRESH:     "VK_BROWSER_REFRESH",
	windows.VK_BROWSER_STOP:        "VK_BROWSER_STOP",
	windows.VK_BROWSER_SEARCH:      "VK_BROWSER_SEARCH",
	windows.VK_BROWSER_FAVORITES:   "VK_BROWSER_FAVORITES",
	windows.VK_BROWSER_HOME:        "VK_BROWSER_HOME",
	windows.VK_VOLUME_MUTE:         "VK_VOLUME_MUTE",
	windows.VK_VOLUME_DOWN:         "VK_VOLUME_DOWN",
	windows.VK_VOLUME_UP:           "VK_VOLUME_UP",
	windows.VK_MEDIA_NEXT_TRACK:    "VK_MEDIA_NEXT_TRACK",
	windows.VK_MEDIA_PREV_TRACK:    "VK_MEDIA_PREV_TRACK",
	windows.VK_MEDIA_STOP:          "VK_MEDIA_STOP",
	windows.VK_MEDIA_PLAY_PAUSE:    "VK_MEDIA_PLAY_PAUSE",
	windows.VK_LAUNCH_MAIL:         "VK_LAUNCH_MAIL",
	windows.VK_LAUNCH_MEDIA_SELECT: "VK_LAUNCH_MEDIA_SELECT",
	windows.VK_LAUNCH_APP1:         "VK_LAUNCH_APP1",
	windows.VK_LAUNCH_APP2:         "VK_LAUNCH_APP2",
	windows.VK_OEM_1:               "VK_OEM_1",
	windows.VK_OEM_PLUS:            "VK_OEM_PLUS",
	windows.VK_OEM_COMMA:           "VK_OEM_COMMA",
	windows.VK_OEM_MINUS:           "VK_OEM_MINUS",
	windows.VK_OEM_PERIOD:          "VK_OEM_PERIOD",
	windows.VK_OEM_2:               "VK_OEM_2",
	windows.VK_OEM_3:               "VK_OEM_3",
	windows.VK_OEM_4:               "VK_OEM_4",
	windows.VK_OEM_5:               "VK_OEM_5",
	windows.VK_OEM_6:               "VK_OEM_6",
	windows.VK_OEM_7:               "VK_OEM_7",
	windows.VK_OEM_8:               "VK_OEM_8",
	windows.VK_OEM_102:             "VK_OEM_102",
	windows.VK_PROCESSKEY:          "VK_PROCESSKEY",
	0xE7:                           "VK_PACKET",
	windows.VK_ATTN:                "VK_ATTN",
	windows.VK_CRSEL:               "VK_CRSEL",
	windows.VK_EXSEL:               "VK_EXSEL",
	windows.VK_EREOF:               "VK_EREOF",
	windows.VK_PLAY:                "VK_PLAY",
	windows.VK_ZOOM:                "VK_ZOOM",
	windows.VK_NONAME:              "VK_NONAME",
	windows.VK_PA1:                 "VK_PA1",
	windows.VK_OEM_CLEAR:           "VK_OEM_CLEAR",
}

func init() {
	for code := uint32(0x41); code <= 0x5A; code++ {
		name := "VK_" + string(rune(code))
		vkNames[code] = name
	}
	for code := uint32('0'); code <= '9'; code++ {
		vkNames[code] = "VK_" + string(rune(code))
	}
}
