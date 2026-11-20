package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// The decks are embedded so the built .app is a single self-contained binary.
//
//go:embed data/kanji.json
var kanjiRaw []byte

//go:embed data/kotoba.json
var kotobaRaw []byte

// Compound is a common word built around a kanji (e.g. 日本 for 日).
type Compound struct {
	Word    string `json:"word"`
	Kana    string `json:"kana"`
	Meaning string `json:"meaning"`
}

// Kanji is one entry of the kanji deck.
type Kanji struct {
	Char      string     `json:"char"`
	Level     int        `json:"level"`
	Strokes   int        `json:"strokes"`
	On        []string   `json:"on"`
	Kun       []string   `json:"kun"`
	Meaning   string     `json:"meaning"`
	Compounds []Compound `json:"compounds"`
}

// Example is a sample sentence for a vocabulary entry.
type Example struct {
	JP   string `json:"jp"`
	Kana string `json:"kana"`
	EN   string `json:"en"`
}

// Kotoba is one entry of the vocabulary deck.
type Kotoba struct {
	Word    string  `json:"word"`
	Kana    string  `json:"kana"`
	Level   int     `json:"level"`
	Pos     string  `json:"pos"`
	Meaning string  `json:"meaning"`
	Example Example `json:"example"`
}

// Deck holds both study modes.
type Deck struct {
	Kanji  []Kanji  `json:"kanji"`
	Kotoba []Kotoba `json:"kotoba"`
}

// LoadDeck parses and validates the embedded decks.
func LoadDeck() (*Deck, error) {
	var kanji []Kanji
	if err := decodeStrict(kanjiRaw, &kanji); err != nil {
		return nil, fmt.Errorf("data/kanji.json: %w", err)
	}
	var kotoba []Kotoba
	if err := decodeStrict(kotobaRaw, &kotoba); err != nil {
		return nil, fmt.Errorf("data/kotoba.json: %w", err)
	}

	d := &Deck{Kanji: kanji, Kotoba: kotoba}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return d, nil
}

func decodeStrict(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("trailing JSON content")
	}
	return nil
}

// Validate checks structural integrity of the deck. It deliberately does not
// enforce a JLPT level so a different-level deck can be dropped in.
func (d *Deck) Validate() error {
	if len(d.Kanji) == 0 {
		return fmt.Errorf("kanji deck is empty")
	}
	if len(d.Kotoba) == 0 {
		return fmt.Errorf("kotoba deck is empty")
	}

	seenKanji := make(map[string]bool, len(d.Kanji))
	for i, k := range d.Kanji {
		if utf8.RuneCountInString(k.Char) != 1 {
			return fmt.Errorf("kanji[%d] %q: char must be exactly one rune", i, k.Char)
		}
		if seenKanji[k.Char] {
			return fmt.Errorf("kanji[%d]: duplicate char %q", i, k.Char)
		}
		seenKanji[k.Char] = true
		if strings.TrimSpace(k.Meaning) == "" {
			return fmt.Errorf("kanji %q: missing meaning", k.Char)
		}
		if k.Level < 1 || k.Level > 5 {
			return fmt.Errorf("kanji %q: level %d out of range 1..5", k.Char, k.Level)
		}
		if k.Strokes < 1 || k.Strokes > 30 {
			return fmt.Errorf("kanji %q: implausible stroke count %d", k.Char, k.Strokes)
		}
		if len(k.On) == 0 && len(k.Kun) == 0 {
			return fmt.Errorf("kanji %q: no reading", k.Char)
		}
		if len(k.Compounds) < 2 {
			return fmt.Errorf("kanji %q: expected at least 2 compounds, got %d", k.Char, len(k.Compounds))
		}
		for j, c := range k.Compounds {
			if strings.TrimSpace(c.Word) == "" || strings.TrimSpace(c.Kana) == "" || strings.TrimSpace(c.Meaning) == "" {
				return fmt.Errorf("kanji %q compound %d: empty field", k.Char, j)
			}
			if !strings.Contains(c.Word, k.Char) {
				return fmt.Errorf("kanji %q compound %q does not contain the kanji", k.Char, c.Word)
			}
		}
	}

	seenWord := make(map[string]bool, len(d.Kotoba))
	for i, k := range d.Kotoba {
		if strings.TrimSpace(k.Word) == "" {
			return fmt.Errorf("kotoba[%d]: missing word", i)
		}
		if seenWord[k.Word] {
			return fmt.Errorf("kotoba[%d]: duplicate word %q", i, k.Word)
		}
		seenWord[k.Word] = true
		if strings.TrimSpace(k.Kana) == "" {
			return fmt.Errorf("kotoba %q: missing kana", k.Word)
		}
		if strings.TrimSpace(k.Meaning) == "" {
			return fmt.Errorf("kotoba %q: missing meaning", k.Word)
		}
		if strings.TrimSpace(k.Example.JP) == "" || strings.TrimSpace(k.Example.EN) == "" {
			return fmt.Errorf("kotoba %q: incomplete example", k.Word)
		}
		if k.Level < 1 || k.Level > 5 {
			return fmt.Errorf("kotoba %q: level %d out of range 1..5", k.Word, k.Level)
		}
	}

	return nil
}

// LevelBadge turns a JLPT level number into its label (3 -> "N3").
func LevelBadge(level int) string {
	if level < 1 || level > 5 {
		return "N?"
	}
	return fmt.Sprintf("N%d", level)
}
