# Keylistener Plugin — Lua Usage Guide

## Overview

The `plugin_keylistener` global lets Lua modules subscribe to **keyboard**
(Windows + Linux) and **gamepad/controller** (Linux only) input events.

Events are delivered as directed event types via `on_request(type, data)`.

## Availability guard

The `plugin_keylistener` table only exists if the plugin was loaded at bot
startup. Always check before calling:

```lua
if plugin_keylistener then
    plugin_keylistener.listen("my_module")
end
```

See `docs/plugin.md` § *Lua safety — plugin not loaded* for the full pattern.

---

## Keyboard API

### Actions

| Action | Args | Description |
|---|---|---|
| `listen(moduleName)` | `string` | Subscribe the named Lua module to keypress events |
| `stop_listen(moduleName)` | `string` | Unsubscribe from keypress events |

### Event: `"keypress"`

```lua
function on_request(type, data)
    if type == "keypress" then
        -- data.vk_code    = number (virtual key code, platform-native)
        -- data.vk_name    = string (e.g. "A", "SPACE", "F1", "LEFT")
        -- data.key        = string (printable character or name, e.g. "a", "enter")
        -- data.is_down    = boolean (true = key pressed, false = released)
        -- data.modifiers  = table
        --     data.modifiers.shift   = boolean
        --     data.modifiers.ctrl    = boolean
        --     data.modifiers.alt     = boolean
        --     data.modifiers.caps    = boolean
    end
end
```

### Lua example — keyboard logger

```lua
-- modules/customevents/my_module.lua
function on_start()
    ev.set_paused(false)
    if plugin_keylistener then
        plugin_keylistener.listen("my_module")
    end
end

function on_request(type, data)
    if type == "keypress" and data.is_down then
        g.print("Key pressed: " .. data.key)
        if data.modifiers.ctrl then
            g.print("  + CTRL")
        end
    end
end
```

### Lua example — key combo detection

```lua
function on_request(type, data)
    if type == "keypress" and data.is_down then
        if data.key == "f5" then
            g.send_message("twitch", "channel", "Someone pressed F5!")
        end
        if data.vk_name == "LEFT" then
            -- move left in game
        end
    end
end
```

---

## Gamepad / Controller API (Linux only)

### Actions

| Action | Args | Description |
|---|---|---|
| `listen_controller(moduleName)` | `string` | Subscribe to gamepad events |
| `stop_listen_controller(moduleName)` | `string` | Unsubscribe from gamepad events |

### Event: `"gamepad"`

```lua
function on_request(type, data)
    if type == "gamepad" then
        -- data.buttons = table (boolean per button)
        --     data.buttons.south       = boolean  -- A (Xbox) / Cross (PS)
        --     data.buttons.east        = boolean  -- B (Xbox) / Circle (PS)
        --     data.buttons.north       = boolean  -- Y (Xbox) / Triangle (PS)
        --     data.buttons.west        = boolean  -- X (Xbox) / Square (PS)
        --     data.buttons.l1          = boolean  -- LB / L1
        --     data.buttons.r1          = boolean  -- RB / R1
        --     data.buttons.l2          = boolean  -- LT digital / L2
        --     data.buttons.r2          = boolean  -- RT digital / R2
        --     data.buttons.select      = boolean  -- Back / Select
        --     data.buttons.start       = boolean
        --     data.buttons.guide       = boolean  -- Guide / PS button
        --     data.buttons.l3          = boolean  -- Left stick click
        --     data.buttons.r3          = boolean  -- Right stick click
        --     data.buttons.dpad_up     = boolean
        --     data.buttons.dpad_down   = boolean
        --     data.buttons.dpad_left   = boolean
        --     data.buttons.dpad_right  = boolean
        --
        -- data.axes = table (float, range depends on axis type)
        --     data.axes.lx  = float  -- left stick X  (-1.0 .. 1.0)
        --     data.axes.ly  = float  -- left stick Y  (-1.0 .. 1.0)
        --     data.axes.rx  = float  -- right stick X (-1.0 .. 1.0)
        --     data.axes.ry  = float  -- right stick Y (-1.0 .. 1.0)
        --     data.axes.lt  = float  -- left trigger  ( 0.0 .. 1.0)
        --     data.axes.rt  = float  -- right trigger ( 0.0 .. 1.0)
        --
        -- Axes use a 5% deadzone — tiny noise is filtered out.
    end
end
```

### Lua example — gamepad button press

```lua
function on_request(type, data)
    if type == "gamepad" then
        for name, down in pairs(data.buttons) do
            if down then
                g.print(name .. " pressed")
            end
        end

        if data.buttons.south then
            g.send_message("twitch", "channel", "A button pressed!")
        end
    end
end
```

### Lua example — analog stick threshold

```lua
function on_request(type, data)
    if type == "gamepad" then
        if data.axes.lx > 0.8 then
            g.print("Stick hard right")
        elseif data.axes.lx < -0.8 then
            g.print("Stick hard left")
        end

        if data.axes.rt > 0.5 then
            g.print("Right trigger pulled")
        end
    end
end
```

---

## Platform matrix

| Feature | Windows | Linux |
|---|---|---|
| Keyboard listen | ✅ `WH_KEYBOARD_LL` | ✅ evdev |
| Gamepad listen | ❌ (planned) | ✅ evdev |

---

## Best practices

- **Nil guard** every call — `plugin_keylistener` is nil if the DLL/.so
  wasn't loaded.
- **Don't spam** — `on_request` fires on every key event. Avoid heavy
  computation or chat messages on every keystroke.
- **Gamepad axes** have a built-in 5% deadzone. For deadzone-free raw
  values, check against the absolute value: `math.abs(data.axes.lx) > 0`.
- **Unsubscribe when done** — if your module stops needing input, call
  `stop_listen` / `stop_listen_controller` to avoid unnecessary event
  dispatch.
