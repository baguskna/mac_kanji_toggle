package main

import (
	"encoding/json"
	"fmt"
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

// autoSwitchInterval is how often the widget rotates between the 漢字 and
// 言葉 decks and between JLPT levels. The rotation pauses while the card panel
// is open so reading is not interrupted.
const autoSwitchInterval = 5 * time.Second

// defaultLevel is the JLPT level shown on a fresh install.
const defaultLevel = 3

// deckPos identifies a card position inside one level + mode deck.
type deckPos struct {
	level int
	mode  Mode
}

// App owns all study state. The Objective-C layer is a dumb renderer: it calls
// back into these methods and receives a fresh Card to display.
type App struct {
	deck *Deck

	mu      sync.Mutex
	mode    Mode
	level   int
	indices map[deckPos]int

	auto         bool
	autoInterval time.Duration
	showRomaji   bool
	showFurigana bool

	statePath string

	// uiEnabled is false in tests so that state transitions stay headless.
	uiEnabled bool
}

type savedState struct {
	Mode Mode `json:"mode"`
	// Level plus Indices are the current format (keys are "<level>:<mode>").
	Level   int            `json:"level,omitempty"`
	Indices map[string]int `json:"indices,omitempty"`
	// Legacy fields from the single-level (N3-only) version.
	KanjiIndex   int   `json:"kanjiIndex,omitempty"`
	KotobaIndex  int   `json:"kotobaIndex,omitempty"`
	AutoSwitch   *bool `json:"autoSwitch,omitempty"`
	ShowRomaji   *bool `json:"showRomaji,omitempty"`
	ShowFurigana *bool `json:"showFurigana,omitempty"`
}

func posKey(p deckPos) string {
	return fmt.Sprintf("%d:%d", p.level, p.mode)
}

// NewApp wires a deck into fresh state (romaji in the menu bar is on by default).
func NewApp(deck *Deck, statePath string) *App {
	level := defaultLevel
	if !deck.HasLevel(level) && len(deck.Levels) > 0 {
		level = deck.Levels[0]
	}
	return &App{
		deck:         deck,
		statePath:    statePath,
		level:        level,
		indices:      make(map[deckPos]int),
		showRomaji:   true,
		showFurigana: true,
		auto:         true,
		autoInterval: autoSwitchInterval,
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
	if a.deck.HasLevel(s.Level) {
		a.level = s.Level
	}
	for key, idx := range s.Indices {
		var level int
		var mode Mode
		if _, err := fmt.Sscanf(key, "%d:%d", &level, &mode); err != nil {
			continue
		}
		if (mode != ModeKanji && mode != ModeKotoba) || !a.deck.HasLevel(level) {
			continue
		}
		a.indices[deckPos{level, mode}] = clamp(idx, 0, a.deckLenLocked(level, mode))
	}
	// Migrate the old single-deck state file (N3 only).
	if len(s.Indices) == 0 && a.deck.HasLevel(3) {
		if s.KanjiIndex != 0 || s.KotobaIndex != 0 {
			a.indices[deckPos{3, ModeKanji}] = clamp(s.KanjiIndex, 0, a.deckLenLocked(3, ModeKanji))
			a.indices[deckPos{3, ModeKotoba}] = clamp(s.KotobaIndex, 0, a.deckLenLocked(3, ModeKotoba))
		}
	}
	// Older state files used "auto" for a card advance timer; a missing
	// "autoSwitch" means "keep the new default" (on).
	if s.AutoSwitch != nil {
		a.auto = *s.AutoSwitch
	}
	if s.ShowRomaji != nil {
		a.showRomaji = *s.ShowRomaji
	}
	if s.ShowFurigana != nil {
		a.showFurigana = *s.ShowFurigana
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

// SaveState persists mode, level, per-deck positions and preferences.
func (a *App) SaveState() {
	a.mu.Lock()
	showRomaji := a.showRomaji
	showFurigana := a.showFurigana
	autoSwitch := a.auto
	indices := make(map[string]int, len(a.indices))
	for pos, idx := range a.indices {
		indices[posKey(pos)] = idx
	}
	s := savedState{
		Mode:         a.mode,
		Level:        a.level,
		Indices:      indices,
		AutoSwitch:   &autoSwitch,
		ShowRomaji:   &showRomaji,
		ShowFurigana: &showFurigana,
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

// deckLenLocked returns the length of one deck. Callers must hold a.mu (or be
// in a context where the app is not shared yet).
func (a *App) deckLenLocked(level int, mode Mode) int {
	if mode == ModeKotoba {
		return len(a.deck.Kotoba[level])
	}
	return len(a.deck.Kanji[level])
}

// currentIndexLocked returns the clamped index for the current position.
func (a *App) currentIndexLocked() int {
	pos := deckPos{a.level, a.mode}
	return clamp(a.indices[pos], 0, a.deckLenLocked(a.level, a.mode))
}

// currentCardLocked returns the card for the current position. Callers must
// hold a.mu.
func (a *App) currentCardLocked() Card {
	idx := a.currentIndexLocked()
	if a.mode == ModeKotoba {
		return KotobaCard(a.deck.Kotoba[a.level][idx])
	}
	return KanjiCard(a.deck.Kanji[a.level][idx])
}

// CurrentCard returns the card for the current position.
func (a *App) CurrentCard() Card {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentCardLocked()
}

// Level returns the current JLPT level.
func (a *App) Level() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.level
}

// Levels returns the levels that have decks, easiest first.
func (a *App) Levels() []int {
	return a.deck.Levels
}

// Next advances the current deck, wrapping around at the end.
func (a *App) Next() {
	a.mu.Lock()
	pos := deckPos{a.level, a.mode}
	n := a.deckLenLocked(a.level, a.mode)
	a.indices[pos] = (a.currentIndexLocked() + 1) % n
	a.mu.Unlock()
	a.refresh()
}

// Prev steps back one card.
func (a *App) Prev() {
	a.mu.Lock()
	pos := deckPos{a.level, a.mode}
	n := a.deckLenLocked(a.level, a.mode)
	a.indices[pos] = (a.currentIndexLocked() - 1 + n) % n
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

// CycleLevel moves to the next JLPT level (N5 -> N4 -> N3 -> N5 ...).
func (a *App) CycleLevel() {
	a.mu.Lock()
	levels := a.deck.Levels
	for i, l := range levels {
		if l == a.level {
			a.level = levels[(i+1)%len(levels)]
			break
		}
	}
	a.mu.Unlock()
	a.refresh()
}

// SetLevel selects a level explicitly (menu item); returns false if the deck
// for that level is not available.
func (a *App) SetLevel(level int) bool {
	if !a.deck.HasLevel(level) {
		return false
	}
	a.mu.Lock()
	a.level = level
	a.mu.Unlock()
	a.refresh()
	return true
}

// setLevelSilent is used by the snapshot renderer (no UI push, no state write).
func (a *App) setLevelSilent(level int) bool {
	if !a.deck.HasLevel(level) {
		return false
	}
	a.mu.Lock()
	a.level = level
	a.mu.Unlock()
	return true
}

// ToggleAuto flips the 5-second deck/level auto-switch.
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

// ToggleFurigana flips whether the kana reading is shown next to the kanji in
// the menu bar.
func (a *App) ToggleFurigana() {
	a.mu.Lock()
	a.showFurigana = !a.showFurigana
	a.mu.Unlock()
	a.refresh()
}

// SetAutoInterval overrides the auto-switch interval (used by dev flags/tests).
func (a *App) SetAutoInterval(d time.Duration) {
	if d <= 0 {
		return
	}
	a.mu.Lock()
	a.autoInterval = d
	a.mu.Unlock()
}

// SetMode/SetIndex are used by the snapshot renderer.
func (a *App) SetMode(m Mode) {
	a.mu.Lock()
	a.mode = m
	a.mu.Unlock()
}

func (a *App) SetIndex(i int) {
	a.mu.Lock()
	pos := deckPos{a.level, a.mode}
	a.indices[pos] = clamp(i, 0, a.deckLenLocked(a.level, a.mode))
	a.mu.Unlock()
}

// refresh pushes the current state to the native UI and persists it.
func (a *App) refresh() {
	a.mu.Lock()
	card := a.currentCardLocked()
	showRomaji := a.showRomaji
	showFurigana := a.showFurigana
	auto := a.auto
	level := a.level
	a.mu.Unlock()

	if a.uiEnabled {
		uiShowCard(card)
		uiSetTitle(card.MenuTitle(showFurigana, showRomaji))
		uiSetMenuState(auto, showRomaji, showFurigana, level)
	}
	a.SaveState()
}

// StartAuto rotates decks and levels every autoInterval. A tick is skipped
// while the user has the card open, so the widget never changes underneath
// them.
func (a *App) StartAuto() {
	go func() {
		t := time.NewTicker(a.autoInterval)
		defer t.Stop()
		for range t.C {
			a.mu.Lock()
			on := a.auto
			a.mu.Unlock()
			if !on {
				continue
			}
			if uiPanelVisible() {
				continue
			}
			a.tickAutoSwitch()
		}
	}()
}

// nextLevelLocked returns the next JLPT level in the deck's cycle (N5 -> N4 ->
// N3 -> N5). It skips levels without a complete pair of decks.
func (a *App) nextLevelLocked() int {
	levels := a.deck.Levels
	if len(levels) == 0 {
		return a.level
	}
	start := 0
	for i, level := range levels {
		if level == a.level {
			start = i
			break
		}
	}
	for step := 1; step <= len(levels); step++ {
		level := levels[(start+step)%len(levels)]
		if a.deck.HasLevel(level) {
			return level
		}
	}
	return a.level
}

// tickAutoSwitch advances the deck it is about to show, then flips to it, so
// the widget rotates through new material instead of alternating the same pair.
// The rotation runs kanji -> kotoba within a level, then moves to the next
// level: N5 kanji -> N5 kotoba -> N4 kanji -> N4 kotoba -> N3 ...
func (a *App) tickAutoSwitch() {
	a.mu.Lock()
	if a.mode == ModeKanji {
		pos := deckPos{a.level, ModeKotoba}
		n := a.deckLenLocked(a.level, ModeKotoba)
		if n > 0 {
			a.indices[pos] = (clamp(a.indices[pos], 0, n) + 1) % n
		}
		a.mode = ModeKotoba
	} else {
		a.level = a.nextLevelLocked()
		pos := deckPos{a.level, ModeKanji}
		n := a.deckLenLocked(a.level, ModeKanji)
		if n > 0 {
			a.indices[pos] = (clamp(a.indices[pos], 0, n) + 1) % n
		}
		a.mode = ModeKanji
	}
	a.mu.Unlock()
	a.refresh()
}
