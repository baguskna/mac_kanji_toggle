package main

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// These tests double as the guardrail for the curated JLPT decks: if the data
// is replaced or edited by hand, the app refuses obviously broken entries.

func isHiragana(r rune) bool {
	return unicode.In(r, unicode.Hiragana) || r == 'ー'
}

func isKatakana(r rune) bool {
	return unicode.In(r, unicode.Katakana) || r == 'ー' || r == '・'
}

func isKana(r rune) bool {
	return isHiragana(r) || isKatakana(r)
}

// Kana fields may contain a little punctuation but never kanji or latin text.
func isKanaText(s string, allowPunct string) bool {
	for _, r := range s {
		switch {
		case isKana(r):
		case strings.ContainsRune(allowPunct, r):
		case unicode.IsSpace(r):
		default:
			return false
		}
	}
	return true
}

func isHiraganaText(s string) bool {
	for _, r := range s {
		if !isHiragana(r) {
			return false
		}
	}
	return s != ""
}

func isKatakanaText(s string) bool {
	for _, r := range s {
		if !isKatakana(r) {
			return false
		}
	}
	return s != ""
}

func loadTestDeck(t *testing.T) *Deck {
	t.Helper()
	d, err := LoadDeck()
	if err != nil {
		t.Fatalf("LoadDeck: %v", err)
	}
	return d
}

func TestDecksCoverN5N4N3(t *testing.T) {
	d := loadTestDeck(t)
	want := []int{5, 4, 3}
	if len(d.Levels) != len(want) {
		t.Fatalf("levels = %v, want %v", d.Levels, want)
	}
	for i, level := range want {
		if d.Levels[i] != level {
			t.Fatalf("levels = %v, want %v (easiest first)", d.Levels, want)
		}
		if !d.HasLevel(level) {
			t.Fatalf("level N%d is missing a deck", level)
		}
	}

	minKanji := map[int]int{5: 80, 4: 160, 3: 180}
	minKotoba := map[int]int{5: 200, 4: 200, 3: 200}
	for _, level := range want {
		if got := len(d.Kanji[level]); got < minKanji[level] {
			t.Errorf("N%d kanji deck has %d entries, want >= %d", level, got, minKanji[level])
		}
		if got := len(d.Kotoba[level]); got < minKotoba[level] {
			t.Errorf("N%d kotoba deck has %d entries, want >= %d", level, got, minKotoba[level])
		}
		for _, k := range d.Kanji[level] {
			if k.Level != level {
				t.Fatalf("kanji %q in N%d deck has level %d", k.Char, level, k.Level)
			}
		}
		for _, k := range d.Kotoba[level] {
			if k.Level != level {
				t.Fatalf("kotoba %q in N%d deck has level %d", k.Word, level, k.Level)
			}
		}
	}
}

func TestKanjiReadingsUseCorrectScript(t *testing.T) {
	d := loadTestDeck(t)
	for _, level := range d.Levels {
		for _, k := range d.Kanji[level] {
			for _, on := range k.On {
				if !isKatakanaText(on) {
					t.Errorf("N%d kanji %q: on'yomi %q must be katakana", level, k.Char, on)
				}
			}
			for _, kun := range k.Kun {
				if !isHiraganaText(kun) {
					t.Errorf("N%d kanji %q: kun'yomi %q must be hiragana", level, k.Char, kun)
				}
			}
			for _, c := range k.Compounds {
				if !isHiraganaText(c.Kana) {
					t.Errorf("N%d kanji %q: compound %q kana %q must be hiragana",
						level, k.Char, c.Word, c.Kana)
				}
			}
		}
	}
}

func TestKotobaExampleHasKanaTranscript(t *testing.T) {
	d := loadTestDeck(t)
	allowed := "。、！？「」…・ 　"
	for _, level := range d.Levels {
		for _, k := range d.Kotoba[level] {
			if !isKanaText(k.Kana, allowed) {
				t.Errorf("N%d kotoba %q: kana %q contains non-kana characters", level, k.Word, k.Kana)
			}
			if !isKanaText(k.Example.Kana, allowed) {
				t.Errorf("N%d kotoba %q: example kana %q contains non-kana characters",
					level, k.Word, k.Example.Kana)
			}
		}
	}
}

func TestCardsPerLevel(t *testing.T) {
	d := loadTestDeck(t)
	for _, level := range d.Levels {
		kanji := KanjiCard(d.Kanji[level][0])
		if kanji.ModeKind != "kanji" || kanji.Badge != LevelBadge(level) {
			t.Errorf("N%d kanji card: badge=%q modeKind=%q", level, kanji.Badge, kanji.ModeKind)
		}
		if kanji.Reading1 == "" || kanji.Reading1Romaji == "" || len(kanji.Items) == 0 {
			t.Errorf("N%d kanji card is incomplete", level)
		}
		kotoba := KotobaCard(d.Kotoba[level][0])
		if kotoba.ModeKind != "kotoba" || kotoba.Badge != LevelBadge(level) {
			t.Errorf("N%d kotoba card: badge=%q modeKind=%q", level, kotoba.Badge, kotoba.ModeKind)
		}
		if kotoba.Reading1Romaji == "" || kotoba.ExampleJP == "" {
			t.Errorf("N%d kotoba card is incomplete", level)
		}
	}
}

func TestMenuTitle(t *testing.T) {
	card := Card{Title: "食べる", TitleKana: "たべる", TitleSub: "taberu"}
	if got := card.MenuTitle(false, false); got != "食べる" {
		t.Errorf("plain title = %q", got)
	}
	if got := card.MenuTitle(true, false); got != "食べる  たべる" {
		t.Errorf("furigana title = %q", got)
	}
	if got := card.MenuTitle(true, true); got != "食べる  たべる  taberu" {
		t.Errorf("furigana+romaji title = %q", got)
	}
	if got := card.MenuTitle(false, true); got != "食べる  taberu" {
		t.Errorf("romaji-only title = %q", got)
	}

	// Kana-only words must not repeat themselves as "furigana".
	kanaOnly := Card{Title: "ありがとう", TitleKana: "ありがとう", TitleSub: "arigatou"}
	if got := kanaOnly.MenuTitle(true, true); got != "ありがとう  arigatou" {
		t.Errorf("kana-only title = %q", got)
	}
}

func TestAppNavigationAndPersistence(t *testing.T) {
	d := loadTestDeck(t)
	statePath := filepath.Join(t.TempDir(), "state.json")
	app := NewApp(d, statePath)

	if app.Level() != 3 {
		t.Fatalf("default level = %d, want 3", app.Level())
	}
	want := d.Kanji[3][0].Char
	if got := app.CurrentCard().Headword; got != want {
		t.Fatalf("initial card = %q, want %q", got, want)
	}
	app.Next()
	if got, want := app.CurrentCard().Headword, d.Kanji[3][1].Char; got != want {
		t.Errorf("after Next = %q, want %q", got, want)
	}
	app.Prev()
	if got := app.CurrentCard().Headword; got != want {
		t.Errorf("after Prev = %q, want %q", got, want)
	}
	app.Prev()
	if got, want := app.CurrentCard().Headword, d.Kanji[3][len(d.Kanji[3])-1].Char; got != want {
		t.Errorf("Prev should wrap to %q, got %q", want, got)
	}

	app.ToggleMode()
	if got := app.CurrentCard().ModeKind; got != "kotoba" {
		t.Errorf("after ToggleMode modeKind = %q", got)
	}

	if !app.SetLevel(5) {
		t.Fatalf("SetLevel(5) failed")
	}
	if got := app.CurrentCard().Badge; got != "N5" {
		t.Errorf("after SetLevel(5) badge = %q", got)
	}
	if app.SetLevel(1) {
		t.Errorf("SetLevel(1) should fail when no N1 deck exists")
	}

	app.ToggleRomaji()
	app.ToggleFurigana()
	app.ToggleAuto()
	app.SaveState()

	app2 := NewApp(d, statePath)
	app2.LoadState()
	if got := app2.Level(); got != 5 {
		t.Errorf("restored level = %d, want 5", got)
	}
	if got := app2.CurrentCard().Badge; got != "N5" {
		t.Errorf("restored badge = %q, want N5", got)
	}
	if app2.showRomaji {
		t.Errorf("restored showRomaji = true, want false")
	}
	if app2.showFurigana {
		t.Errorf("restored showFurigana = true, want false")
	}
	if app2.auto {
		t.Errorf("restored autoSwitch = true, want false")
	}
}

func TestCycleLevelWraps(t *testing.T) {
	d := loadTestDeck(t)
	app := NewApp(d, filepath.Join(t.TempDir(), "state.json"))
	seen := []int{app.Level()}
	for i := 0; i < len(d.Levels); i++ {
		app.CycleLevel()
		seen = append(seen, app.Level())
	}
	// N3 -> N5 -> N4 -> N3 with Levels = [5 4 3].
	want := []int{3, 5, 4, 3}
	if len(seen) != len(want) {
		t.Fatalf("level cycle = %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("level cycle = %v, want %v", seen, want)
		}
	}
}

func TestAutoSwitchDefaultsOnAndMovesCards(t *testing.T) {
	d := loadTestDeck(t)
	app := NewApp(d, filepath.Join(t.TempDir(), "state.json"))
	if !app.auto {
		t.Fatalf("auto-switch should default to on")
	}

	startMode := app.CurrentCard().ModeKind
	app.tickAutoSwitch()
	after := app.CurrentCard()
	if after.ModeKind == startMode {
		t.Errorf("auto-switch did not change deck (still %q)", after.ModeKind)
	}
	if after.Badge != "N3" {
		t.Errorf("auto-switch changed level unexpectedly: badge=%q", after.Badge)
	}

	seen := map[string]bool{}
	for i := 0; i < 6; i++ {
		seen[app.CurrentCard().ModeKind+"|"+app.CurrentCard().Headword] = true
		app.tickAutoSwitch()
	}
	if len(seen) < 4 {
		t.Errorf("auto-switch looks stuck: only %d distinct cards over 6 ticks", len(seen))
	}
}
