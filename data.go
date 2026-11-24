package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// The decks are embedded so the built .app is a single self-contained binary.
// One file per JLPT level; the level is also stored on every entry and checked
// against the file it came from.
//
//go:embed data/kanji_n5.json
var kanjiN5Raw []byte

//go:embed data/kanji_n4.json
var kanjiN4Raw []byte

//go:embed data/kanji.json
var kanjiN3Raw []byte

//go:embed data/kotoba_n5.json
var kotobaN5Raw []byte

//go:embed data/kotoba_n4.json
var kotobaN4Raw []byte

//go:embed data/kotoba.json
var kotobaN3Raw []byte

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

// Deck holds both study modes, indexed by JLPT level (5 = easiest, 1 = hardest).
type Deck struct {
	Kanji  map[int][]Kanji
	Kotoba map[int][]Kotoba
	Levels []int // descending, e.g. [5 4 3]
}

type deckSource struct {
	path  string
	level int
	raw   []byte
}

// LoadDeck parses and validates the embedded per-level decks.
func LoadDeck() (*Deck, error) {
	kanjiSources := []deckSource{
		{"data/kanji_n5.json", 5, kanjiN5Raw},
		{"data/kanji_n4.json", 4, kanjiN4Raw},
		{"data/kanji.json", 3, kanjiN3Raw},
	}
	kotobaSources := []deckSource{
		{"data/kotoba_n5.json", 5, kotobaN5Raw},
		{"data/kotoba_n4.json", 4, kotobaN4Raw},
		{"data/kotoba.json", 3, kotobaN3Raw},
	}

	d := &Deck{
		Kanji:  make(map[int][]Kanji, len(kanjiSources)),
		Kotoba: make(map[int][]Kotoba, len(kotobaSources)),
	}

	for _, src := range kanjiSources {
		var entries []Kanji
		if err := decodeStrict(src.raw, &entries); err != nil {
			return nil, fmt.Errorf("%s: %w", src.path, err)
		}
		for i := range entries {
			if entries[i].Level != src.level {
				return nil, fmt.Errorf("%s: entry %q has level %d, want %d",
					src.path, entries[i].Char, entries[i].Level, src.level)
			}
		}
		d.Kanji[src.level] = entries
	}

	for _, src := range kotobaSources {
		var entries []Kotoba
		if err := decodeStrict(src.raw, &entries); err != nil {
			return nil, fmt.Errorf("%s: %w", src.path, err)
		}
		for i := range entries {
			if entries[i].Level != src.level {
				return nil, fmt.Errorf("%s: entry %q has level %d, want %d",
					src.path, entries[i].Word, entries[i].Level, src.level)
			}
		}
		d.Kotoba[src.level] = entries
	}

	for level := range d.Kanji {
		d.Levels = append(d.Levels, level)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(d.Levels)))

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

// Validate checks structural integrity of every level's decks.
func (d *Deck) Validate() error {
	if len(d.Levels) == 0 {
		return fmt.Errorf("no decks loaded")
	}
	for _, level := range d.Levels {
		if level < 1 || level > 5 {
			return fmt.Errorf("deck level %d out of range 1..5", level)
		}
		if err := validateKanji(d.Kanji[level], level); err != nil {
			return err
		}
		if err := validateKotoba(d.Kotoba[level], level); err != nil {
			return err
		}
	}
	return nil
}

func validateKanji(entries []Kanji, level int) error {
	if len(entries) == 0 {
		return fmt.Errorf("kanji deck N%d is empty", level)
	}
	seen := make(map[string]bool, len(entries))
	for i, k := range entries {
		if utf8.RuneCountInString(k.Char) != 1 {
			return fmt.Errorf("kanji[%d] %q: char must be exactly one rune", i, k.Char)
		}
		if seen[k.Char] {
			return fmt.Errorf("kanji[%d]: duplicate char %q", i, k.Char)
		}
		seen[k.Char] = true
		if strings.TrimSpace(k.Meaning) == "" {
			return fmt.Errorf("kanji %q: missing meaning", k.Char)
		}
		if k.Level != level {
			return fmt.Errorf("kanji %q: level %d, want %d", k.Char, k.Level, level)
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
	return nil
}

func validateKotoba(entries []Kotoba, level int) error {
	if len(entries) == 0 {
		return fmt.Errorf("kotoba deck N%d is empty", level)
	}
	seen := make(map[string]bool, len(entries))
	for i, k := range entries {
		if strings.TrimSpace(k.Word) == "" {
			return fmt.Errorf("kotoba[%d]: missing word", i)
		}
		if seen[k.Word] {
			return fmt.Errorf("kotoba[%d]: duplicate word %q", i, k.Word)
		}
		seen[k.Word] = true
		if strings.TrimSpace(k.Kana) == "" {
			return fmt.Errorf("kotoba %q: missing kana", k.Word)
		}
		if strings.TrimSpace(k.Meaning) == "" {
			return fmt.Errorf("kotoba %q: missing meaning", k.Word)
		}
		if strings.TrimSpace(k.Example.JP) == "" || strings.TrimSpace(k.Example.EN) == "" {
			return fmt.Errorf("kotoba %q: incomplete example", k.Word)
		}
		if k.Level != level {
			return fmt.Errorf("kotoba %q: level %d, want %d", k.Word, k.Level, level)
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

// HasLevel reports whether both decks exist for a level.
func (d *Deck) HasLevel(level int) bool {
	return len(d.Kanji[level]) > 0 && len(d.Kotoba[level]) > 0
}

// LevelCounts summarizes the loaded decks, e.g. for startup logging.
func (d *Deck) LevelCounts() string {
	parts := make([]string, 0, len(d.Levels))
	for _, level := range d.Levels {
		parts = append(parts, fmt.Sprintf("N%d %d/%d", level, len(d.Kanji[level]), len(d.Kotoba[level])))
	}
	return strings.Join(parts, ", ")
}
