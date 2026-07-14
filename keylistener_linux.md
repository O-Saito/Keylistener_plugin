# Keylistener Plugin

Captures keyboard input via evdev (`/dev/input/event*`) on Linux, and via
Win32 API on Windows. Built as a C shared library (`-buildmode=c-shared`) and
loaded at runtime by MyStreamBot.

## Building

```bash
cd plugins_projects/keylistener
go build -buildmode=c-shared -o ../../plugins/keylistener.so .
```

Produces `keylistener.so` and `keylistener.h`.

## Setup on Bazzite / Fedora Atomic

The plugin opens `/dev/input/event*` devices with `O_RDONLY`. On Fedora
Atomic desktops (Bazzite, Silverblue, Kinoite) these are owned by
`root:input` with `660` permissions.

### 1. Add your user to the input group

```bash
sudo usermod -aG input $USER
```

### 2. Test without logging out

```bash
newgrp input
python3 -c "open('/dev/input/event0', 'rb').close(); print('OK')"
```

`newgrp` spawns a shell with the new group — the change is temporary and
drops when you exit that shell. A full logout/login makes it permanent in
your desktop session.

### 3. Verify after re-login

```bash
ls -l /dev/input/event0       # should show crw-rw---- root:input
python3 -c "open('/dev/input/event0', 'rb').close(); print('OK')"
```

### udev fallback

If group membership alone isn't enough, create a udev rule to ensure the
correct permissions:

```bash
echo 'KERNEL=="event*", SUBSYSTEM=="input", MODE="0660", GROUP="input"' \
  | sudo tee /etc/udev/rules.d/99-keylistener-input.rules
sudo udevadm control --reload-rules
sudo udevadm trigger --subsystem-match=input
```

`/etc` is writable at runtime on Fedora Atomic — this persists across
reboots.

### Distrobox extra step

If running MyStreamBot inside a Distrobox container, Podman remaps UID/GID
inside the container so group-based permissions don't work. Run this on the
**host** instead of the rule above:

```bash
echo 'KERNEL=="event*", SUBSYSTEM=="input", MODE="0664", TAG+="uaccess"' \
  | sudo tee /etc/udev/rules.d/99-distrobox-input.rules
sudo udevadm control --reload-rules
sudo udevadm trigger --subsystem-match=input
```

### SELinux

If devices open but no events arrive, check for SELinux denials:

```bash
sudo ausearch -m avc --start recent
```

## Troubleshooting

- **"evdev: no keyboard devices found (user in 'input' group?)"** — the
  plugin prints this if it can't open any evdev device. Follow the steps
  above.
- **"evdev: no input devices at /dev/input/event\*"** — no glob match.
  Check that `/dev/input/` exists and has `event*` entries.
