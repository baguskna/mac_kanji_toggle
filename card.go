package main

import (
	"fmt"
	"strings"
)

// Card is the view model handed to the Objective-C layer. It is serialised to
// JSON and parsed there, which keeps the cgo boundary to plain UTF-8 strings.
type Card struct {
	Mode       string `json:"mode"`      // 漢字 / 言葉, shown in the mode pill
	ModeLabel  string `json:"modeLabel"` // KANJI / KOTOBA
	ModeKind   string `json:"modeKind"`  // "kanji" or "kotoba"
	Badge      string `json:"badge"`     // N3
	Headword   string `json:"headword"`
	Meta       string `json:"meta"` // strokes count or part of speech

	Reading1Label  string `json:"reading1Label"`
	Reading1       string `json:"reading1"`
	Reading1Romaji string `json:"reading1Romaji"`

	Reading2Label  string `json:"reading2Label"`
	Reading2       string `json:"reading2"`
	Reading2Romaji string `json:"reading2Romaji"`

	Meaning string `json:"meaning"`

	Section string     `json:"section"`
	Items   []CardItem `json:"items"`

	ExampleJP   string `json:"exampleJP"`
	ExampleKana string `json:"exampleKana"`
	ExampleEN   string `json:"exampleEN"`

	// Title, TitleKana and TitleSub are used for the menu bar item, e.g.
	// "対  タイ  tai".
	Title     string `json:"title"`
	TitleKana string `json:"titleKana"`
	TitleSub  string `json:"titleSub"`
}

// CardItem is one row in the "compounds" list.
type CardItem struct {
	Head string `json:"head"`
	Sub  string `json:"sub"`
	Tail string `json:"tail"`
}

// KanjiCard builds the card for a kanji entry.
func KanjiCard(k Kanji) Card {
	c := Card{
		Mode:      "漢字",
		ModeLabel: "KANJI",
		ModeKind:  "kanji",
		Badge:     LevelBadge(k.Level),
		Headword:  k.Char,
		Meta:      fmt.Sprintf("%d strokes", k.Strokes),
		Meaning:   k.Meaning,
		Section:   "Words with " + k.Char,
		Title:     k.Char,
		TitleKana: firstReading(k.On, k.Kun),
	}

	// On'yomi is the primary reading, kun'yomi secondary.
	if len(k.On) > 0 {
		c.Reading1Label = "On'yomi"
		c.Reading1 = strings.Join(k.On, "・")
		c.Reading1Romaji = RomajiList(k.On)
		c.TitleSub = firstRomaji(k.On)
	}
	if len(k.Kun) > 0 {
		if c.Reading1Label == "" {
			c.Reading1Label = "Kun'yomi"
			c.Reading1 = strings.Join(k.Kun, "・")
			c.Reading1Romaji = RomajiList(k.Kun)
			c.TitleSub = firstRomaji(k.Kun)
		} else {
			c.Reading2Label = "Kun'yomi"
			c.Reading2 = strings.Join(k.Kun, "・")
			c.Reading2Romaji = RomajiList(k.Kun)
		}
	}

	for i, comp := range k.Compounds {
		if i >= 3 {
			break
		}
		c.Items = append(c.Items, CardItem{Head: comp.Word, Sub: comp.Kana, Tail: comp.Meaning})
	}
	return c
}

// KotobaCard builds the card for a vocabulary entry.
func KotobaCard(k Kotoba) Card {
	c := Card{
		Mode:           "言葉",
		ModeLabel:      "KOTOBA",
		ModeKind:       "kotoba",
		Badge:          LevelBadge(k.Level),
		Headword:       k.Word,
		Meta:           k.Pos,
		Meaning:        k.Meaning,
		Reading1Label:  "Reading",
		Reading1:       k.Kana,
		Reading1Romaji: Romaji(k.Kana),
		Section:        "Example",
		TitleKana:      k.Kana,
		ExampleJP:      k.Example.JP,
		ExampleKana:    k.Example.Kana,
		ExampleEN:      k.Example.EN,
		Title:          k.Word,
		TitleSub:       Romaji(k.Kana),
	}
	return c
}

// firstReading returns the reading shown as furigana in the menu bar: the
// first on'yomi, falling back to the first kun'yomi.
func firstReading(on, kun []string) string {
	if len(on) > 0 {
		return on[0]
	}
	if len(kun) > 0 {
		return kun[0]
	}
	return ""
}

func firstRomaji(readings []string) string {
	for _, r := range readings {
		if s := Romaji(r); s != "" {
			return s
		}
	}
	return ""
}

// MenuTitle composes the status item title. With everything enabled it looks
// like "対  タイ  tai" (kanji + furigana + romaji) or "経験  けいけん  keiken".
// Kana-only words skip the furigana part because it would just repeat the word.
func (c Card) MenuTitle(showFurigana, showRomaji bool) string {
	parts := []string{c.Title}
	if showFurigana && c.TitleKana != "" && c.TitleKana != c.Title {
		parts = append(parts, c.TitleKana)
	}
	if showRomaji && c.TitleSub != "" {
		parts = append(parts, c.TitleSub)
	}
	return strings.Join(parts, "  ")
}
