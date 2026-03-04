package main

// Preamble must only have declarations (no definitions) when using //export.

/*
#include <stdlib.h>
*/
import "C"

var hotkeyChannel = make(chan struct{}, 1)

//export onHotkeyTriggered
func onHotkeyTriggered() {
	select {
	case hotkeyChannel <- struct{}{}:
	default:
	}
}
