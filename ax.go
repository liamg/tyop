package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation -framework CoreGraphics -framework Foundation -framework AppKit
#include <ApplicationServices/ApplicationServices.h>
#include <CoreGraphics/CoreGraphics.h>
#include <Foundation/Foundation.h>
#include <AppKit/AppKit.h>
#include <stdlib.h>

static int isAccessibilityGranted() {
	return AXIsProcessTrusted() ? 1 : 0;
}

static int requestAccessibility() {
	// Prompt the user if not already trusted. Returns 1 if trusted.
	NSDictionary *options = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @YES};
	return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options) ? 1 : 0;
}

static void openAccessibilityPrefs() {
	CFURLRef url = CFURLCreateWithString(NULL,
		CFSTR("x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility"),
		NULL);
	if (url) {
		LSOpenCFURLRef(url, NULL);
		CFRelease(url);
	}
}

// Track PIDs we've already enabled AXEnhancedUserInterface for.
static pid_t enhancedPIDs[256] = {0};
static int enhancedPIDCount = 0;

static int setEnhancedUserInterface(pid_t pid) {
	for (int i = 0; i < enhancedPIDCount; i++) {
		if (enhancedPIDs[i] == pid) return 0; // already done, no wait needed
	}
	AXUIElementRef appEl = AXUIElementCreateApplication(pid);
	if (appEl) {
		AXUIElementSetAttributeValue(appEl, CFSTR("AXEnhancedUserInterface"), kCFBooleanTrue);
		CFRelease(appEl);
	}
	if (enhancedPIDCount < 256) enhancedPIDs[enhancedPIDCount++] = pid;
	usleep(500000); // 500ms for Chromium to build the accessibility tree
	return 1; // first time — caller may want to retry
}

// Get the PID of the frontmost app via window list (no AppKit required).
static pid_t frontmostAppPID() {
	CFArrayRef list = CGWindowListCopyWindowInfo(
		kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
		kCGNullWindowID);
	if (!list) return -1;
	pid_t pid = -1;
	CFIndex n = CFArrayGetCount(list);
	for (CFIndex i = 0; i < n && pid == -1; i++) {
		CFDictionaryRef w = CFArrayGetValueAtIndex(list, i);
		CFNumberRef layerNum = CFDictionaryGetValue(w, kCGWindowLayer);
		int layer = 0;
		if (layerNum) CFNumberGetValue(layerNum, kCFNumberIntType, &layer);
		if (layer == 0) {
			CFNumberRef pidNum = CFDictionaryGetValue(w, kCGWindowOwnerPID);
			if (pidNum) CFNumberGetValue(pidNum, kCFNumberSInt32Type, &pid);
		}
	}
	CFRelease(list);
	return pid;
}

static AXUIElementRef copyFocusedElement() {
	pid_t pid = frontmostAppPID();
	if (pid < 0) return NULL;

	// Try 1: focused element directly on the app element.
	AXUIElementRef appEl = AXUIElementCreateApplication(pid);
	if (appEl) {
		// Enable Chromium/Electron accessibility tree. Waits 500ms on first encounter.
		setEnhancedUserInterface(pid);

		AXUIElementRef focused = NULL;
		if (AXUIElementCopyAttributeValue(appEl, kAXFocusedUIElementAttribute, (CFTypeRef *)&focused) == kAXErrorSuccess && focused) {
			CFRelease(appEl);
			return focused;
		}
		// Try 2: go via focused window (helps with some apps).
		AXUIElementRef window = NULL;
		if (AXUIElementCopyAttributeValue(appEl, kAXFocusedWindowAttribute, (CFTypeRef *)&window) == kAXErrorSuccess && window) {
			AXUIElementRef focused2 = NULL;
			AXUIElementCopyAttributeValue(window, kAXFocusedUIElementAttribute, (CFTypeRef *)&focused2);
			CFRelease(window);
			if (focused2) { CFRelease(appEl); return focused2; }
		}
		CFRelease(appEl);
	}

	// Try 3: system-wide element (last resort).
	AXUIElementRef sysWide = AXUIElementCreateSystemWide();
	AXUIElementRef focused3 = NULL;
	AXUIElementCopyAttributeValue(sysWide, kAXFocusedUIElementAttribute, (CFTypeRef *)&focused3);
	CFRelease(sysWide);
	return focused3;
}

static char *copyElementText(AXUIElementRef el) {
	CFTypeRef value = NULL;
	if (AXUIElementCopyAttributeValue(el, kAXValueAttribute, &value) != kAXErrorSuccess) {
		return NULL;
	}
	if (CFGetTypeID(value) != CFStringGetTypeID()) {
		CFRelease(value);
		return NULL;
	}
	CFStringRef str = (CFStringRef)value;
	CFIndex maxLen = CFStringGetMaximumSizeForEncoding(CFStringGetLength(str), kCFStringEncodingUTF8) + 1;
	char *buf = malloc(maxLen);
	if (!CFStringGetCString(str, buf, maxLen, kCFStringEncodingUTF8)) {
		free(buf);
		CFRelease(value);
		return NULL;
	}
	CFRelease(value);
	return buf;
}

static int setElementText(AXUIElementRef el, const char *text) {
	CFStringRef str = CFStringCreateWithCString(NULL, text, kCFStringEncodingUTF8);
	if (!str) return 0;
	AXError err = AXUIElementSetAttributeValue(el, kAXValueAttribute, str);
	if (err == kAXErrorSuccess) {
		usleep(50000); // wait for app to process value change before moving cursor
		CFIndex len = CFStringGetLength(str);
		CFRange range = CFRangeMake(len, 0);
		AXValueRef axRange = AXValueCreate(kAXValueCFRangeType, &range);
		AXUIElementSetAttributeValue(el, kAXSelectedTextRangeAttribute, axRange);
		CFRelease(axRange);
	}
	CFRelease(str);
	return err == kAXErrorSuccess ? 1 : 0;
}

// Keyboard key codes (Carbon)
#define kVK_ANSI_A 0x00
#define kVK_ANSI_V 0x09

static void simulateCmdKey(CGKeyCode key) {
	CGEventRef down = CGEventCreateKeyboardEvent(NULL, key, true);
	CGEventRef up   = CGEventCreateKeyboardEvent(NULL, key, false);
	CGEventSetFlags(down, kCGEventFlagMaskCommand);
	CGEventSetFlags(up,   kCGEventFlagMaskCommand);
	CGEventPost(kCGHIDEventTap, down);
	usleep(20000);
	CGEventPost(kCGHIDEventTap, up);
	CFRelease(down);
	CFRelease(up);
	usleep(50000);
}

// Fallback for apps (e.g. Zed) that don't support kAXValueAttribute write.
// Saves clipboard, sets corrected text, Cmd+A selects all, Cmd+V pastes, then
// restores the original clipboard asynchronously.
static void setTextViaClipboard(const char *text) {
	NSPasteboard *pb = [NSPasteboard generalPasteboard];
	NSString *old = [[pb stringForType:NSPasteboardTypeString] copy];
	[pb clearContents];
	[pb setString:[NSString stringWithUTF8String:text] forType:NSPasteboardTypeString];
	simulateCmdKey(kVK_ANSI_A);
	simulateCmdKey(kVK_ANSI_V);
	dispatch_after(dispatch_time(DISPATCH_TIME_NOW, 600 * NSEC_PER_MSEC),
		dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT, 0), ^{
		[pb clearContents];
		if (old) [pb setString:old forType:NSPasteboardTypeString];
	});
}

// Read text via clipboard: Cmd+A to select all, Cmd+C to copy, read clipboard.
// Restores original clipboard content after reading.
// Returns a malloc'd C string — caller must free.
static char *readTextViaClipboard() {
	NSPasteboard *pb = [NSPasteboard generalPasteboard];
	NSString *old = [[pb stringForType:NSPasteboardTypeString] copy];
	NSString *text = nil;
	char *result = NULL;
	const char *utf8 = NULL;
	CGEventRef down = NULL;
	CGEventRef up = NULL;

	[pb clearContents];

	simulateCmdKey(kVK_ANSI_A);

	// Cmd+C (kVK_ANSI_C = 0x08)
	down = CGEventCreateKeyboardEvent(NULL, 0x08, true);
	up   = CGEventCreateKeyboardEvent(NULL, 0x08, false);
	CGEventSetFlags(down, kCGEventFlagMaskCommand);
	CGEventSetFlags(up,   kCGEventFlagMaskCommand);
	CGEventPost(kCGHIDEventTap, down);
	usleep(20000);
	CGEventPost(kCGHIDEventTap, up);
	CFRelease(down);
	CFRelease(up);
	usleep(150000); // wait for clipboard to be populated

	text = [pb stringForType:NSPasteboardTypeString];
	if (text) {
		utf8 = [text UTF8String];
		if (utf8) result = strdup(utf8);
	}

	// Restore original clipboard
	[pb clearContents];
	if (old) [pb setString:old forType:NSPasteboardTypeString];

	return result;
}

static void moveCursorToEnd(AXUIElementRef el) {
	CFTypeRef lenVal = NULL;
	if (AXUIElementCopyAttributeValue(el, kAXNumberOfCharactersAttribute, &lenVal) != kAXErrorSuccess) return;
	CFIndex len = 0;
	CFNumberGetValue((CFNumberRef)lenVal, kCFNumberCFIndexType, &len);
	CFRelease(lenVal);
	CFRange range = CFRangeMake(len, 0);
	AXValueRef axRange = AXValueCreate(kAXValueCFRangeType, &range);
	AXUIElementSetAttributeValue(el, kAXSelectedTextRangeAttribute, axRange);
	CFRelease(axRange);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func accessibilityGranted() bool {
	return C.isAccessibilityGranted() != 0
}

func openAccessibilityPrefs() {
	C.openAccessibilityPrefs()
}

func readFocusedText(clipboardFallback bool) (text string, usedClipboard bool, err error) {
	el := C.copyFocusedElement()
	if unsafe.Pointer(el) != nil {
		defer C.CFRelease(C.CFTypeRef(unsafe.Pointer(el)))
		cstr := C.copyElementText(el)
		if cstr != nil {
			defer C.free(unsafe.Pointer(cstr))
			return C.GoString(cstr), false, nil
		}
	}
	if !clipboardFallback {
		return "", false, fmt.Errorf("no readable text element")
	}
	// AX read failed — try clipboard (Cmd+A, Cmd+C).
	cstr := C.readTextViaClipboard()
	if cstr == nil {
		return "", false, fmt.Errorf("clipboard read failed")
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr), true, nil
}

func writeFocusedText(text string, useClipboard bool) error {
	if useClipboard {
		cstr := C.CString(text)
		defer C.free(unsafe.Pointer(cstr))
		C.setTextViaClipboard(cstr)
		return nil
	}
	el := C.copyFocusedElement()
	if unsafe.Pointer(el) == nil {
		return fmt.Errorf("no focused element")
	}
	defer C.CFRelease(C.CFTypeRef(unsafe.Pointer(el)))
	cstr := C.CString(text)
	defer C.free(unsafe.Pointer(cstr))
	if C.setElementText(el, cstr) == 0 {
		return fmt.Errorf("AX write failed (app may not support it)")
	}
	return nil
}
