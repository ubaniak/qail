//go:build darwin

package app

/*
#cgo LDFLAGS: -framework Carbon

#include <Carbon/Carbon.h>

extern void qailHotkeyFired(void);

static OSStatus qailHotkeyHandler(EventHandlerCallRef nextHandler, EventRef event, void *userData) {
    qailHotkeyFired();
    return noErr;
}

// qailRegisterHotkey installs a single global hotkey on the application
// event target. Returns the OSStatus from RegisterEventHotKey (0 = ok).
static int qailRegisterHotkey(unsigned int keyCode, unsigned int modifiers) {
    EventHotKeyID hotKeyID;
    hotKeyID.signature = 'qail';
    hotKeyID.id = 1;

    EventHotKeyRef hotKeyRef;
    EventTypeSpec evt;
    evt.eventClass = kEventClassKeyboard;
    evt.eventKind = kEventHotKeyPressed;

    InstallEventHandler(GetApplicationEventTarget(), &qailHotkeyHandler, 1, &evt, NULL, NULL);
    OSStatus s = RegisterEventHotKey(keyCode, modifiers, hotKeyID, GetApplicationEventTarget(), 0, &hotKeyRef);
    return (int)s;
}
*/
import "C"

import "fmt"

// Carbon modifier constants. See HIToolbox/Events.h.
const (
	carbonCmdKey     = 1 << 8  // 256
	carbonShiftKey   = 1 << 9  // 512
	carbonOptionKey  = 1 << 11 // 2048
	carbonControlKey = 1 << 12 // 4096
)

// Carbon virtual key codes. Q lives at 12. See HIToolbox/Events.h.
const keyCodeQ = 12

var hotkeyCh chan struct{}

//export qailHotkeyFired
func qailHotkeyFired() {
	// Non-blocking send — drop duplicate presses if a previous one is
	// still being handled. Carbon fires this on the main run loop, so
	// blocking here would stall the UI.
	select {
	case hotkeyCh <- struct{}{}:
	default:
	}
}

// RegisterToggleHotkey wires ⌃⌥Q as a process-global hotkey. Returns a
// channel that receives an empty struct on each press. Idempotent —
// subsequent calls return the same channel without re-registering.
func RegisterToggleHotkey() (<-chan struct{}, error) {
	if hotkeyCh != nil {
		return hotkeyCh, nil
	}
	hotkeyCh = make(chan struct{}, 1)
	if s := C.qailRegisterHotkey(C.uint(keyCodeQ), C.uint(carbonControlKey|carbonOptionKey)); s != 0 {
		return nil, fmt.Errorf("RegisterEventHotKey OSStatus=%d", int(s))
	}
	return hotkeyCh, nil
}
