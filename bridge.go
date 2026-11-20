package main

/*
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore
#include <stdlib.h>
#include "appkit.h"
*/
import "C"

import (
	"encoding/json"
	"log"
	"unsafe"
)

// appRef is the single App instance the native UI talks to.
var appRef *App

// The following functions are called from Objective-C on the main thread.

//export GoNext
func GoNext() {
	if appRef != nil {
		appRef.Next()
	}
}

//export GoPrev
func GoPrev() {
	if appRef != nil {
		appRef.Prev()
	}
}

//export GoToggleMode
func GoToggleMode() {
	if appRef != nil {
		appRef.ToggleMode()
	}
}

//export GoToggleAuto
func GoToggleAuto() {
	if appRef != nil {
		appRef.ToggleAuto()
	}
}

//export GoToggleRomaji
func GoToggleRomaji() {
	if appRef != nil {
		appRef.ToggleRomaji()
	}
}

//export GoToggleFurigana
func GoToggleFurigana() {
	if appRef != nil {
		appRef.ToggleFurigana()
	}
}

// uiStart creates the status item, panel and popover-free card window.
func uiStart() {
	C.KBStart()
}

// uiRun runs the AppKit event loop; it only returns when the app quits.
func uiRun() {
	C.KBRun()
}

// uiShowCard hands the current card to the renderer as JSON.
func uiShowCard(card Card) {
	raw, err := json.Marshal(card)
	if err != nil {
		log.Printf("kanjibar: marshal card: %v", err)
		return
	}
	cs := C.CString(string(raw))
	defer C.free(unsafe.Pointer(cs))
	C.KBShowCard(cs)
}

// uiSetTitle updates the menu bar text.
func uiSetTitle(title string) {
	cs := C.CString(title)
	defer C.free(unsafe.Pointer(cs))
	C.KBSetTitle(cs)
}

// uiPanelVisible reports whether the card panel is currently open, so the
// 15-second auto-switch can pause while the user is reading.
func uiPanelVisible() bool {
	return C.KBIsPanelVisible() != 0
}

// uiMeasureCardHeight asks the renderer how tall a card naturally is.
func uiMeasureCardHeight(card Card) float64 {
	raw, err := json.Marshal(card)
	if err != nil {
		return 0
	}
	cs := C.CString(string(raw))
	defer C.free(unsafe.Pointer(cs))
	return float64(C.KBCardNaturalHeight(cs))
}

// uiSetFixedCardHeight makes every card render at the same height.
func uiSetFixedCardHeight(height float64) {
	C.KBSetFixedCardHeight(C.double(height))
}

// uiSetMenuState syncs the checkmarks in the context menu.
func uiSetMenuState(auto, romaji, furigana bool) {
	C.KBSetMenuState(cBool(auto), cBool(romaji), cBool(furigana))
}

// uiShowNow opens the panel shortly after launch (dev flag --show).
func uiShowNow() {
	C.KBShowNow()
}

// uiHideNow closes the panel (dev flag --hide-after).
func uiHideNow() {
	C.KBHideNow()
}

// uiQuit terminates the app (dev flag --exit-after).
func uiQuit() {
	C.KBQuit()
}

// uiSnapshot renders one card to a PNG file (used by --snapshot).
func uiSnapshot(card Card, path string, dark bool) {
	raw, err := json.Marshal(card)
	if err != nil {
		log.Fatalf("kanjibar: marshal card: %v", err)
	}
	cs := C.CString(string(raw))
	defer C.free(unsafe.Pointer(cs))
	cp := C.CString(path)
	defer C.free(unsafe.Pointer(cp))
	C.KBSnapshot(cs, cp, cBool(dark))
}

func cBool(v bool) C.int {
	if v {
		return 1
	}
	return 0
}
