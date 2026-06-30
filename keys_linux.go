//go:build linux

package main

import "fmt"

type KeyEvent struct {
	VkCode    uint32          `json:"vk_code"`
	VkName    string          `json:"vk_name"`
	Key       string          `json:"key"`
	IsDown    bool            `json:"is_down"`
	Modifiers map[string]bool `json:"modifiers"`
}

const (
	KEY_RESERVED     = 0
	KEY_ESC          = 1
	KEY_1            = 2
	KEY_2            = 3
	KEY_3            = 4
	KEY_4            = 5
	KEY_5            = 6
	KEY_6            = 7
	KEY_7            = 8
	KEY_8            = 9
	KEY_9            = 10
	KEY_0            = 11
	KEY_MINUS        = 12
	KEY_EQUAL        = 13
	KEY_BACKSPACE    = 14
	KEY_TAB          = 15
	KEY_Q            = 16
	KEY_W            = 17
	KEY_E            = 18
	KEY_R            = 19
	KEY_T            = 20
	KEY_Y            = 21
	KEY_U            = 22
	KEY_I            = 23
	KEY_O            = 24
	KEY_P            = 25
	KEY_LEFTBRACE    = 26
	KEY_RIGHTBRACE   = 27
	KEY_ENTER        = 28
	KEY_LEFTCTRL     = 29
	KEY_A            = 30
	KEY_S            = 31
	KEY_D            = 32
	KEY_F            = 33
	KEY_G            = 34
	KEY_H            = 35
	KEY_J            = 36
	KEY_K            = 37
	KEY_L            = 38
	KEY_SEMICOLON    = 39
	KEY_APOSTROPHE   = 40
	KEY_GRAVE        = 41
	KEY_LEFTSHIFT    = 42
	KEY_BACKSLASH    = 43
	KEY_Z            = 44
	KEY_X            = 45
	KEY_C            = 46
	KEY_V            = 47
	KEY_B            = 48
	KEY_N            = 49
	KEY_M            = 50
	KEY_COMMA        = 51
	KEY_DOT          = 52
	KEY_SLASH        = 53
	KEY_RIGHTSHIFT   = 54
	KEY_KPASTERISK   = 55
	KEY_LEFTALT      = 56
	KEY_SPACE        = 57
	KEY_CAPSLOCK     = 58
	KEY_F1           = 59
	KEY_F2           = 60
	KEY_F3           = 61
	KEY_F4           = 62
	KEY_F5           = 63
	KEY_F6           = 64
	KEY_F7           = 65
	KEY_F8           = 66
	KEY_F9           = 67
	KEY_F10          = 68
	KEY_NUMLOCK      = 69
	KEY_SCROLLLOCK   = 70
	KEY_KP7          = 71
	KEY_KP8          = 72
	KEY_KP9          = 73
	KEY_KPMINUS      = 74
	KEY_KP4          = 75
	KEY_KP5          = 76
	KEY_KP6          = 77
	KEY_KPPLUS       = 78
	KEY_KP1          = 79
	KEY_KP2          = 80
	KEY_KP3          = 81
	KEY_KP0          = 82
	KEY_KPDOT        = 83
	KEY_F11          = 87
	KEY_F12          = 88
	KEY_KPENTER      = 96
	KEY_RIGHTCTRL    = 97
	KEY_KPSLASH      = 98
	KEY_SYSRQ        = 99
	KEY_RIGHTALT     = 100
	KEY_HOME         = 102
	KEY_UP           = 103
	KEY_PAGEUP       = 104
	KEY_LEFT         = 105
	KEY_RIGHT        = 106
	KEY_END          = 107
	KEY_DOWN         = 108
	KEY_PAGEDOWN     = 109
	KEY_INSERT       = 110
	KEY_DELETE       = 111
	KEY_MUTE         = 113
	KEY_VOLUMEDOWN   = 114
	KEY_VOLUMEUP     = 115
	KEY_PAUSE        = 119
	KEY_LEFTMETA     = 125
	KEY_RIGHTMETA    = 126
	KEY_MENU         = 127
	KEY_STOP         = 128
	KEY_AGAIN        = 129
	KEY_UNDO         = 130
	KEY_COPY         = 133
	KEY_CUT          = 137
	KEY_FIND         = 136
	KEY_PASTE        = 135
	KEY_HELP         = 138
	KEY_CALC         = 140
	KEY_SLEEP        = 142
	KEY_WWW          = 150
	KEY_MAIL         = 155
	KEY_BOOKMARKS    = 156
	KEY_COMPUTER     = 157
	KEY_BACK         = 158
	KEY_FORWARD      = 159
	KEY_PLAYPAUSE    = 164
	KEY_NEXTSONG     = 163
	KEY_PREVIOUSSONG = 165
	KEY_HOMEPAGE     = 172
	KEY_REFRESH      = 173
	KEY_SEARCH       = 217
	KEY_F13          = 183
	KEY_F14          = 184
	KEY_F15          = 185
	KEY_F16          = 186
	KEY_F17          = 187
	KEY_F18          = 188
	KEY_F19          = 189
	KEY_F20          = 190
	KEY_F21          = 191
	KEY_F22          = 192
	KEY_F23          = 193
	KEY_F24          = 194
	KEY_MICMUTE      = 248
)

var (
	shiftDown bool
	ctrlDown  bool
	altDown   bool
	capsLock  bool
)

func updateModifiers(code uint16, value int32) {
	isDown := value == 1
	switch code {
	case KEY_LEFTSHIFT, KEY_RIGHTSHIFT:
		shiftDown = isDown
	case KEY_LEFTCTRL, KEY_RIGHTCTRL:
		ctrlDown = isDown
	case KEY_LEFTALT, KEY_RIGHTALT:
		altDown = isDown
	case KEY_CAPSLOCK:
		if isDown {
			capsLock = !capsLock
		}
	}
}

func buildKeyEvent(vkCode uint32, wParam uintptr) KeyEvent {
	isDown := wParam == 1
	mods := map[string]bool{
		"shift": shiftDown,
		"ctrl":  ctrlDown,
		"alt":   altDown,
		"caps":  capsLock,
	}
	code := uint16(vkCode)
	vkName := vkCodeToName(code)
	key := vkCodeToChar(code, mods["shift"], mods["caps"])
	return KeyEvent{
		VkCode:    vkCode,
		VkName:    vkName,
		Key:       key,
		IsDown:    isDown,
		Modifiers: mods,
	}
}

func vkCodeToName(code uint16) string {
	if name, ok := keyNames[code]; ok {
		return name
	}
	return fmt.Sprintf("KEY_%d", code)
}

func vkCodeToChar(code uint16, shift, caps bool) string {
	if code >= KEY_1 && code <= KEY_0 {
		if shift {
			return shiftDigits[code]
		}
		return string(rune('0' + (code-KEY_1+1)%10))
	}

	letter := letterAtCode(code)
	if letter != 0 {
		if shift != caps {
			return string(letter)
		}
		return string(letter + 32)
	}

	if ch, ok := keyChars[code]; ok {
		return ch
	}
	if name, ok := keyNames[code]; ok {
		return name
	}
	return fmt.Sprintf("KEY_%d", code)
}

func letterAtCode(code uint16) rune {
	switch code {
	case KEY_Q:
		return 'Q'
	case KEY_W:
		return 'W'
	case KEY_E:
		return 'E'
	case KEY_R:
		return 'R'
	case KEY_T:
		return 'T'
	case KEY_Y:
		return 'Y'
	case KEY_U:
		return 'U'
	case KEY_I:
		return 'I'
	case KEY_O:
		return 'O'
	case KEY_P:
		return 'P'
	case KEY_A:
		return 'A'
	case KEY_S:
		return 'S'
	case KEY_D:
		return 'D'
	case KEY_F:
		return 'F'
	case KEY_G:
		return 'G'
	case KEY_H:
		return 'H'
	case KEY_J:
		return 'J'
	case KEY_K:
		return 'K'
	case KEY_L:
		return 'L'
	case KEY_Z:
		return 'Z'
	case KEY_X:
		return 'X'
	case KEY_C:
		return 'C'
	case KEY_V:
		return 'V'
	case KEY_B:
		return 'B'
	case KEY_N:
		return 'N'
	case KEY_M:
		return 'M'
	}
	return 0
}

var shiftDigits = map[uint16]string{
	KEY_1: "!", KEY_2: "@", KEY_3: "#", KEY_4: "$", KEY_5: "%",
	KEY_6: "^", KEY_7: "&", KEY_8: "*", KEY_9: "(", KEY_0: ")",
}

var keyChars = map[uint16]string{
	KEY_MINUS:      "-",
	KEY_EQUAL:      "=",
	KEY_LEFTBRACE:  "[",
	KEY_RIGHTBRACE: "]",
	KEY_SEMICOLON:  ";",
	KEY_APOSTROPHE: "'",
	KEY_GRAVE:      "`",
	KEY_BACKSLASH:  "\\",
	KEY_COMMA:      ",",
	KEY_DOT:        ".",
	KEY_SLASH:      "/",
	KEY_SPACE:      " ",
	KEY_ENTER:      "enter",
	KEY_TAB:        "tab",
	KEY_BACKSPACE:  "backspace",
	KEY_ESC:        "escape",
	KEY_DELETE:     "delete",
	KEY_HOME:       "home",
	KEY_END:        "end",
	KEY_PAGEUP:     "page_up",
	KEY_PAGEDOWN:   "page_down",
	KEY_INSERT:     "insert",
	KEY_LEFT:       "left",
	KEY_RIGHT:      "right",
	KEY_UP:         "up",
	KEY_DOWN:       "down",
	KEY_CAPSLOCK:   "caps_lock",
	KEY_NUMLOCK:    "num_lock",
	KEY_SCROLLLOCK: "scroll_lock",
	KEY_PAUSE:      "pause",
	KEY_SYSRQ:      "print_screen",
	KEY_MENU:       "menu",
	KEY_LEFTMETA:   "left_meta",
	KEY_RIGHTMETA:  "right_meta",
	KEY_HELP:       "help",
	KEY_SLEEP:      "sleep",
	KEY_CALC:       "calc",
	KEY_F1:         "f1",
	KEY_F2:         "f2",
	KEY_F3:         "f3",
	KEY_F4:         "f4",
	KEY_F5:         "f5",
	KEY_F6:         "f6",
	KEY_F7:         "f7",
	KEY_F8:         "f8",
	KEY_F9:         "f9",
	KEY_F10:        "f10",
	KEY_F11:        "f11",
	KEY_F12:        "f12",
	KEY_F13:        "f13",
	KEY_F14:        "f14",
	KEY_F15:        "f15",
	KEY_F16:        "f16",
	KEY_F17:        "f17",
	KEY_F18:        "f18",
	KEY_F19:        "f19",
	KEY_F20:        "f20",
	KEY_F21:        "f21",
	KEY_F22:        "f22",
	KEY_F23:        "f23",
	KEY_F24:        "f24",
	KEY_KP0:        "numpad_0",
	KEY_KP1:        "numpad_1",
	KEY_KP2:        "numpad_2",
	KEY_KP3:        "numpad_3",
	KEY_KP4:        "numpad_4",
	KEY_KP5:        "numpad_5",
	KEY_KP6:        "numpad_6",
	KEY_KP7:        "numpad_7",
	KEY_KP8:        "numpad_8",
	KEY_KP9:        "numpad_9",
	KEY_KPDOT:      "decimal",
	KEY_KPSLASH:    "divide",
	KEY_KPASTERISK: "multiply",
	KEY_KPMINUS:    "subtract",
	KEY_KPPLUS:     "add",
	KEY_KPENTER:    "numpad_enter",
	KEY_LEFTSHIFT:  "left_shift",
	KEY_RIGHTSHIFT: "right_shift",
	KEY_LEFTCTRL:   "left_ctrl",
	KEY_RIGHTCTRL:  "right_ctrl",
	KEY_LEFTALT:    "left_alt",
	KEY_RIGHTALT:   "right_alt",
}

var keyNames = map[uint16]string{
	KEY_ESC:      "KEY_ESC",
	KEY_1:        "KEY_1",
	KEY_2:        "KEY_2",
	KEY_3:        "KEY_3",
	KEY_4:        "KEY_4",
	KEY_5:        "KEY_5",
	KEY_6:        "KEY_6",
	KEY_7:        "KEY_7",
	KEY_8:        "KEY_8",
	KEY_9:        "KEY_9",
	KEY_0:        "KEY_0",
	KEY_MINUS:    "KEY_MINUS",
	KEY_EQUAL:    "KEY_EQUAL",
	KEY_BACKSPACE: "KEY_BACKSPACE",
	KEY_TAB:      "KEY_TAB",
	KEY_Q:        "KEY_Q",
	KEY_W:        "KEY_W",
	KEY_E:        "KEY_E",
	KEY_R:        "KEY_R",
	KEY_T:        "KEY_T",
	KEY_Y:        "KEY_Y",
	KEY_U:        "KEY_U",
	KEY_I:        "KEY_I",
	KEY_O:        "KEY_O",
	KEY_P:        "KEY_P",
	KEY_LEFTBRACE:  "KEY_LEFTBRACE",
	KEY_RIGHTBRACE: "KEY_RIGHTBRACE",
	KEY_ENTER:      "KEY_ENTER",
	KEY_LEFTCTRL:   "KEY_LEFTCTRL",
	KEY_A:          "KEY_A",
	KEY_S:          "KEY_S",
	KEY_D:          "KEY_D",
	KEY_F:          "KEY_F",
	KEY_G:          "KEY_G",
	KEY_H:          "KEY_H",
	KEY_J:          "KEY_J",
	KEY_K:          "KEY_K",
	KEY_L:          "KEY_L",
	KEY_SEMICOLON:  "KEY_SEMICOLON",
	KEY_APOSTROPHE: "KEY_APOSTROPHE",
	KEY_GRAVE:      "KEY_GRAVE",
	KEY_LEFTSHIFT:  "KEY_LEFTSHIFT",
	KEY_BACKSLASH:  "KEY_BACKSLASH",
	KEY_Z:          "KEY_Z",
	KEY_X:          "KEY_X",
	KEY_C:          "KEY_C",
	KEY_V:          "KEY_V",
	KEY_B:          "KEY_B",
	KEY_N:          "KEY_N",
	KEY_M:          "KEY_M",
	KEY_COMMA:      "KEY_COMMA",
	KEY_DOT:        "KEY_DOT",
	KEY_SLASH:      "KEY_SLASH",
	KEY_RIGHTSHIFT: "KEY_RIGHTSHIFT",
	KEY_LEFTALT:    "KEY_LEFTALT",
	KEY_SPACE:      "KEY_SPACE",
	KEY_CAPSLOCK:   "KEY_CAPSLOCK",
	KEY_F1:         "KEY_F1",
	KEY_F2:         "KEY_F2",
	KEY_F3:         "KEY_F3",
	KEY_F4:         "KEY_F4",
	KEY_F5:         "KEY_F5",
	KEY_F6:         "KEY_F6",
	KEY_F7:         "KEY_F7",
	KEY_F8:         "KEY_F8",
	KEY_F9:         "KEY_F9",
	KEY_F10:        "KEY_F10",
	KEY_F11:        "KEY_F11",
	KEY_F12:        "KEY_F12",
	KEY_F13:        "KEY_F13",
	KEY_F14:        "KEY_F14",
	KEY_F15:        "KEY_F15",
	KEY_F16:        "KEY_F16",
	KEY_F17:        "KEY_F17",
	KEY_F18:        "KEY_F18",
	KEY_F19:        "KEY_F19",
	KEY_F20:        "KEY_F20",
	KEY_F21:        "KEY_F21",
	KEY_F22:        "KEY_F22",
	KEY_F23:        "KEY_F23",
	KEY_F24:        "KEY_F24",
	KEY_RIGHTCTRL:  "KEY_RIGHTCTRL",
	KEY_RIGHTALT:   "KEY_RIGHTALT",
	KEY_HOME:       "KEY_HOME",
	KEY_UP:         "KEY_UP",
	KEY_PAGEUP:     "KEY_PAGEUP",
	KEY_LEFT:       "KEY_LEFT",
	KEY_RIGHT:      "KEY_RIGHT",
	KEY_END:        "KEY_END",
	KEY_DOWN:       "KEY_DOWN",
	KEY_PAGEDOWN:   "KEY_PAGEDOWN",
	KEY_INSERT:     "KEY_INSERT",
	KEY_DELETE:     "KEY_DELETE",
	KEY_MUTE:       "KEY_MUTE",
	KEY_VOLUMEDOWN: "KEY_VOLUMEDOWN",
	KEY_VOLUMEUP:   "KEY_VOLUMEUP",
	KEY_PAUSE:      "KEY_PAUSE",
	KEY_SYSRQ:      "KEY_SYSRQ",
	KEY_LEFTMETA:   "KEY_LEFTMETA",
	KEY_RIGHTMETA:  "KEY_RIGHTMETA",
	KEY_MENU:       "KEY_MENU",
	KEY_HELP:       "KEY_HELP",
	KEY_SLEEP:      "KEY_SLEEP",
	KEY_STOP:       "KEY_STOP",
	KEY_AGAIN:      "KEY_AGAIN",
	KEY_UNDO:       "KEY_UNDO",
	KEY_COPY:       "KEY_COPY",
	KEY_CUT:        "KEY_CUT",
	KEY_PASTE:      "KEY_PASTE",
	KEY_FIND:       "KEY_FIND",
	KEY_CALC:       "KEY_CALC",
	KEY_WWW:        "KEY_WWW",
	KEY_MAIL:       "KEY_MAIL",
	KEY_BOOKMARKS:  "KEY_BOOKMARKS",
	KEY_COMPUTER:   "KEY_COMPUTER",
	KEY_BACK:       "KEY_BACK",
	KEY_FORWARD:    "KEY_FORWARD",
	KEY_PLAYPAUSE:  "KEY_PLAYPAUSE",
	KEY_NEXTSONG:   "KEY_NEXTSONG",
	KEY_PREVIOUSSONG: "KEY_PREVIOUSSONG",
	KEY_HOMEPAGE:   "KEY_HOMEPAGE",
	KEY_REFRESH:    "KEY_REFRESH",
	KEY_SEARCH:     "KEY_SEARCH",
	KEY_SCROLLLOCK: "KEY_SCROLLLOCK",
	KEY_NUMLOCK:    "KEY_NUMLOCK",
	KEY_MICMUTE:    "KEY_MICMUTE",
	KEY_KP0: "KEY_KP0", KEY_KP1: "KEY_KP1",
	KEY_KP2: "KEY_KP2", KEY_KP3: "KEY_KP3",
	KEY_KP4: "KEY_KP4", KEY_KP5: "KEY_KP5",
	KEY_KP6: "KEY_KP6", KEY_KP7: "KEY_KP7",
	KEY_KP8: "KEY_KP8", KEY_KP9: "KEY_KP9",
}
