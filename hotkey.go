package main

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>

extern void onHotkeyTriggered(void);

static CGKeyCode    gHotkeyCode  = 47;       // kVK_ANSI_Period
static CGEventFlags gHotkeyMods  = 0x40000;  // kCGEventFlagMaskControl
static CFRunLoopRef gTapRunLoop  = NULL;

static CGEventRef eventTapCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *refcon) {
	if (type == kCGEventKeyDown) {
		CGKeyCode   keyCode = (CGKeyCode)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
		CGEventFlags flags  = CGEventGetFlags(event);
		if (keyCode == gHotkeyCode && (flags & gHotkeyMods)) {
			onHotkeyTriggered();
			return NULL;
		}
	}
	return event;
}

static void runEventTap(CGKeyCode code, CGEventFlags mods) {
	gHotkeyCode = code;
	gHotkeyMods = mods;
	CGEventMask mask = CGEventMaskBit(kCGEventKeyDown);
	CFMachPortRef tap = CGEventTapCreate(
		kCGSessionEventTap,
		kCGHeadInsertEventTap,
		0,
		mask,
		eventTapCallback,
		NULL
	);
	if (!tap) {
		fprintf(stderr, "tyop: CGEventTapCreate failed — check Accessibility and Input Monitoring permissions\n");
		return;
	}
	CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(NULL, tap, 0);
	gTapRunLoop = CFRunLoopGetCurrent();
	CFRunLoopAddSource(gTapRunLoop, source, kCFRunLoopCommonModes);
	CFRelease(source);
	CFRelease(tap);
	CFRunLoopRun();
	gTapRunLoop = NULL;
}

static void stopEventTap(void) {
	if (gTapRunLoop) {
		CFRunLoopStop(gTapRunLoop);
	}
}
*/
import "C"
import "runtime"

// hotkeyDefs maps friendly name to (keycode, modifier).
var hotkeyDefs = map[string][2]uint{
	"Ctrl+.": {47, 0x40000},
	"Ctrl+,": {43, 0x40000},
	"Ctrl+;": {41, 0x40000},
}

func startHotkeyListener(hotkey string) {
	def, ok := hotkeyDefs[hotkey]
	if !ok {
		def = hotkeyDefs["Ctrl+."]
	}
	go func() {
		runtime.LockOSThread()
		C.runEventTap(C.CGKeyCode(def[0]), C.CGEventFlags(def[1]))
	}()
}

func stopHotkeyListener() {
	C.stopEventTap()
}
