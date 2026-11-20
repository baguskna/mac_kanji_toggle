package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Mode selects which deck is being studied.
type Mode int

const (
	ModeKanji Mode = iota
	ModeKotoba
)

// autoInterval is how often the menu bar pick changes when auto-advance is on.
const autoInterval = 15 * time.Minute

// App owns all study state. The Objective-C layer is a dumb renderer: it calls
// back into these methods and receives a fresh Card to display.
type App struct {
	deck *Deck

	mu         sync.Mutex
	mode       Mode
	kanjiIdx   int
	kotobaIdx  int
	auto       bool
	showRomaji bool

	statePath string
}

type savedState struct {
	Mode        Mode  `json:"mode"`
	KanjiIndex  int   `json:"kanjiIndex"`
	KotobaIndex int   `json:"kotobaIndex"`
	Auto        bool  `json:"auto"`
	ShowRomaji  *bool `json:"showRomaji,omitempty"`
}

// NewApp wires a deck into fresh state (romaji in the menu bar is on by default).
func NewApp(deck *Deck, statePath string) *App {
	return &App{
		deck:       deck,
		statePath:  statePath,
		showRomaji: true,
	}
}

// LoadState restores the previous session, if any. Failures are non-fatal:
// a corrupt state file simply means starting from the first card.
func (a *App) LoadState() {
	raw, err := os.ReadFile(a.statePath)
	if err != nil {
		return
	}
	var s savedState
	if err := json.Unmarshal(raw, &s); err != nil {
		log.Printf("kanjibar: ignoring unreadable state: %v", err)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if s.Mode == ModeKanji || s.Mode == ModeKotoba {
		a.mode = s.Mode
	}
	a.kanjiIdx = clamp(s.KanjiIndex, 0, len(a.deck.Kanji))
	a.kotobaIdx = clamp(s.KotobaIndex, 0, len(a.deck.Kotoba))
	a.auto = s.Auto
	if s.ShowRomaji != nil {
		a.showRomaji = *s.ShowRomaji
	}
}

func clamp(v, lo, hi int) int {
	if hi <= 0 {
		return 0
	}
	if v < lo {
		return lo
	}
	if v >= hi {
		return hi - 1
	}
	return v
}

// SaveState persists mode, per-deck positions and preferences.
func (a *App) SaveState() {
	a.mu.Lock()
	showRomaji := a.showRomaji
	s := savedState{
		Mode:        a.mode,
		KanjiIndex:  a.kanjiIdx,
		KotobaIndex: a.kotobaIdx,
		Auto:        a.auto,
		ShowRomaji:  &showRomaji,
	}
	a.mu.Unlock()

	if a.statePath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(a.statePath), 0o755); err != nil {
		log.Printf("kanjibar: cannot create state dir: %v", err)
		return
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(a.statePath, raw, 0o644); err != nil {
		log.Printf("kanjibar: cannot write state: %v", err)
	}
}

// currentCardLocked returns the card for the current position. Callers must
// hold a.mu.
func (a *App) currentCardLocked() Card {
	if a.mode == ModeKotoba {
		return KotobaCard(a.deck.Kotoba[a.kotobaIdx])
	}
	return KanjiCard(a.deck.Kanji[a.kanjiIdx])
}

// CurrentCard returns the card for the current position.
func (a *App) CurrentCard() Card {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentCardLocked()
}

// Next advances the current deck, wrapping around at the end.
func (a *App) Next() {
	a.mu.Lock()
	if a.mode == ModeKotoba {
		a.kotobaIdx = (a.kotobaIdx + 1) % len(a.deck.Kotoba)
	} else {
		a.kanjiIdx = (a.kanjiIdx + 1) % len(a.deck.Kanji)
	}
	a.mu.Unlock()
	a.refresh()
}

// Prev steps back one card.
func (a *App) Prev() {
	a.mu.Lock()
	if a.mode == ModeKotoba {
		a.kotobaIdx = (a.kotobaIdx - 1 + len(a.deck.Kotoba)) % len(a.deck.Kotoba)
	} else {
		a.kanjiIdx = (a.kanjiIdx - 1 + len(a.deck.Kanji)) % len(a.deck.Kanji)
	}
	a.mu.Unlock()
	a.refresh()
}

// ToggleMode switches between the kanji deck and the vocabulary deck.
func (a *App) ToggleMode() {
	a.mu.Lock()
	if a.mode == ModeKanji {
		a.mode = ModeKotoba
	} else {
		a.mode = ModeKanji
	}
	a.mu.Unlock()
	a.refresh()
}

// ToggleAuto flips the 15-minute auto-advance.
func (a *App) ToggleAuto() {
	a.mu.Lock()
	a.auto = !a.auto
	a.mu.Unlock()
	a.refresh()
}

// ToggleRomaji flips whether romaji is appended in the menu bar title.
func (a *App) ToggleRomaji() {
	a.mu.Lock()
	a.showRomaji = !a.showRomaji
	a.mu.Unlock()
	a.refresh()
}

// SetMode/SetIndex are used by the snapshot renderer.
func (a *App) SetMode(m Mode) {
	a.mu.Lock()
	a.mode = m
	a.mu.Unlock()
}

func (a *App) SetIndex(i int) {
	a.mu.Lock()
	if a.mode == ModeKotoba {
		a.kotobaIdx = clamp(i, 0, len(a.deck.Kotoba))
	} else {
		a.kanjiIdx = clamp(i, 0, len(a.deck.Kanji))
	}
	a.mu.Unlock()
}

// refresh pushes the current state to the native UI and persists it.
func (a *App) refresh() {
	a.mu.Lock()
	card := a.currentCardLocked()
	showRomaji := a.showRomaji
	auto := a.auto
	a.mu.Unlock()

	uiShowCard(card)
	uiSetTitle(card.MenuTitle(showRomaji))
	uiSetMenuState(auto, showRomaji)
	a.SaveState()
}

// StartAuto runs the auto-advance loop. It is cheap: a tick simply checks the
// flag and advances only when enabled.
func (a *App) StartAuto() {
	go func() {
		t := time.NewTicker(autoInterval)
		defer t.Stop()
		for range t.C {
			a.mu.Lock()
			on := a.auto
			a.mu.Unlock()
			if on {
				a.Next()
			}
		}
	}()
}
