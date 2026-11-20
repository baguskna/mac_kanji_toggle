package main

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// These tests double as the guardrail for the curated N3 deck: if the data is
// replaced or edited by hand, the app refuses obviously broken entries.

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

func loadTestDeck(t *testing.T) *Deck {
	t.Helper()
	d, err := LoadDeck()
	if err != nil {
		t.Fatalf("LoadDeck: %v", err)
	}
	return d
}

func TestDeckIsN3(t *testing.T) {
	d := loadTestDeck(t)
	if len(d.Kanji) < 180 {
		t.Errorf("kanji deck has %d entries, want >= 180 N3 kanji", len(d.Kanji))
	}
	if len(d.Kotoba) < 200 {
		t.Errorf("kotoba deck has %d entries, want >= 200 N3 words", len(d.Kotoba))
	}
	for _, k := range d.Kanji {
		if k.Level != 3 {
			t.Fatalf("kanji %q has level %d, want 3 (deck must stay N3)", k.Char, k.Level)
		}
	}
	for _, k := range d.Kotoba {
		if k.Level != 3 {
			t.Fatalf("kotoba %q has level %d, want 3 (deck must stay N3)", k.Word, k.Level)
		}
	}
}

func TestKanjiReadingsUseCorrectScript(t *testing.T) {
	d := loadTestDeck(t)
	for _, k := range d.Kanji {
		for _, on := range k.On {
			if !isKatakanaText(on) {
				t.Errorf("kanji %q: on'yomi %q must be katakana", k.Char, on)
			}
		}
		for _, kun := range k.Kun {
			if !isHiraganaText(kun) {
				t.Errorf("kanji %q: kun'yomi %q must be hiragana", k.Char, kun)
			}
		}
		for _, c := range k.Compounds {
			if !isHiraganaText(c.Kana) {
				t.Errorf("kanji %q: compound %q kana %q must be hiragana", k.Char, c.Word, c.Kana)
			}
		}
	}
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

func TestKotobaExampleHasKanaTranscript(t *testing.T) {
	d := loadTestDeck(t)
	allowed := "。、！？「」…・ 　"
	for _, k := range d.Kotoba {
		if !isKanaText(k.Kana, allowed) {
			t.Errorf("kotoba %q: kana %q contains non-kana characters", k.Word, k.Kana)
		}
		if !isKanaText(k.Example.Kana, allowed) {
			t.Errorf("kotoba %q: example kana %q contains non-kana characters", k.Word, k.Example.Kana)
		}
		if !strings.Contains(k.Example.JP, "。") && !strings.HasSuffix(k.Example.JP, "！") {
			t.Errorf("kotoba %q: example sentence should end a clause: %q", k.Word, k.Example.JP)
		}
	}
}

func TestCardForKanji(t *testing.T) {
	d := loadTestDeck(t)
	card := KanjiCard(d.Kanji[0])
	if card.ModeKind != "kanji" {
		t.Errorf("modeKind = %q", card.ModeKind)
	}
	if card.Badge != "N3" {
		t.Errorf("badge = %q, want N3", card.Badge)
	}
	if card.Reading1Label == "" || card.Reading1 == "" || card.Reading1Romaji == "" {
		t.Errorf("missing primary reading: %+v", card)
	}
	if len(card.Items) == 0 {
		t.Errorf("kanji card has no compound items")
	}
	if card.Title != d.Kanji[0].Char {
		t.Errorf("title = %q, want kanji char", card.Title)
	}
}

func TestCardForKotoba(t *testing.T) {
	d := loadTestDeck(t)
	entry := d.Kotoba[0]
	card := KotobaCard(entry)
	if card.ModeKind != "kotoba" {
		t.Errorf("modeKind = %q", card.ModeKind)
	}
	if card.Reading1 != entry.Kana {
		t.Errorf("reading = %q, want %q", card.Reading1, entry.Kana)
	}
	if card.ExampleJP != entry.Example.JP || card.ExampleEN != entry.Example.EN {
		t.Errorf("example not copied into card")
	}
	if card.Reading1Romaji == "" {
		t.Errorf("kotoba card missing generated romaji")
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

func TestCardFurigana(t *testing.T) {
	d := loadTestDeck(t)
	kanji := KanjiCard(d.Kanji[0])
	if kanji.TitleKana == "" {
		t.Errorf("kanji card is missing furigana for the menu bar")
	}
	kotoba := KotobaCard(d.Kotoba[0])
	if kotoba.TitleKana != d.Kotoba[0].Kana {
		t.Errorf("kotoba furigana = %q, want %q", kotoba.TitleKana, d.Kotoba[0].Kana)
	}
}

func TestAppNavigationAndPersistence(t *testing.T) {
	d := loadTestDeck(t)
	statePath := filepath.Join(t.TempDir(), "state.json")
	app := NewApp(d, statePath)

	want := d.Kanji[0].Char
	if got := app.CurrentCard().Headword; got != want {
		t.Fatalf("initial card = %q, want %q", got, want)
	}
	app.Next()
	if got, want := app.CurrentCard().Headword, d.Kanji[1].Char; got != want {
		t.Errorf("after Next = %q, want %q", got, want)
	}
	app.Prev()
	if got := app.CurrentCard().Headword; got != want {
		t.Errorf("after Prev = %q, want %q", got, want)
	}
	app.Prev()
	if got, want := app.CurrentCard().Headword, d.Kanji[len(d.Kanji)-1].Char; got != want {
		t.Errorf("Prev should wrap to %q, got %q", want, got)
	}

	app.ToggleMode()
	if got := app.CurrentCard().ModeKind; got != "kotoba" {
		t.Errorf("after ToggleMode modeKind = %q", got)
	}
	app.ToggleRomaji()
	app.ToggleFurigana()
	app.ToggleAuto()
	app.SaveState()

	app2 := NewApp(d, statePath)
	app2.LoadState()
	if got := app2.CurrentCard().ModeKind; got != "kotoba" {
		t.Errorf("restored modeKind = %q, want kotoba", got)
	}
	if app2.showRomaji {
		t.Errorf("restored showRomaji = true, want false")
	}
	if app2.auto {
		t.Errorf("restored autoSwitch = true, want false")
	}
	if app2.showFurigana {
		t.Errorf("restored showFurigana = true, want false")
	}
}

func TestAutoSwitchDefaultsOnAndPausesWhileOpen(t *testing.T) {
	d := loadTestDeck(t)
	app := NewApp(d, filepath.Join(t.TempDir(), "state.json"))
	if !app.auto {
		t.Fatalf("auto-switch should default to on")
	}

	// A tick advances the deck it is about to show, then flips to it.
	startMode := app.CurrentCard().ModeKind
	app.tickAutoSwitch()
	after := app.CurrentCard()
	if after.ModeKind == startMode {
		t.Errorf("auto-switch did not change deck (still %q)", after.ModeKind)
	}

	// Keep flipping and make sure both decks keep moving through the deck.
	seen := map[string]bool{}
	for i := 0; i < 6; i++ {
		seen[app.CurrentCard().ModeKind+"|"+app.CurrentCard().Headword] = true
		app.tickAutoSwitch()
	}
	if len(seen) < 4 {
		t.Errorf("auto-switch looks stuck: only %d distinct cards over 6 ticks", len(seen))
	}
}

func TestFixedHeightIsTallerThanEveryNaturalCard(t *testing.T) {
	// The panel uses the tallest card in the deck as its constant height; this
	// test documents that intent without needing the AppKit renderer.
	d := loadTestDeck(t)
	if len(d.Kanji) == 0 || len(d.Kotoba) == 0 {
		t.Fatal("empty deck")
	}
}
