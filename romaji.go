package main

import (
	"strings"
	"unicode/utf8"
)

// Romaji is generated from kana at runtime instead of being stored in the
// dataset, so the JSON stays small and the transliteration stays consistent
// between the two decks.
//
// The rules are Hepburn-ish but literal: long "o"/"e" vowels are written as
// ou/ei (toukyou, sensei) and katakana long-vowel marks repeat the vowel
// (koohii). That is the spelling most learners use when typing Japanese.

var kanaBase = map[rune]string{
	// vowels
	'あ': "a", 'い': "i", 'う': "u", 'え': "e", 'お': "o",
	'ぁ': "a", 'ぃ': "i", 'ぅ': "u", 'ぇ': "e", 'ぉ': "o",
	// k
	'か': "ka", 'き': "ki", 'く': "ku", 'け': "ke", 'こ': "ko",
	'が': "ga", 'ぎ': "gi", 'ぐ': "gu", 'げ': "ge", 'ご': "go",
	// s
	'さ': "sa", 'し': "shi", 'す': "su", 'せ': "se", 'そ': "so",
	'ざ': "za", 'じ': "ji", 'ず': "zu", 'ぜ': "ze", 'ぞ': "zo",
	// t
	'た': "ta", 'ち': "chi", 'つ': "tsu", 'て': "te", 'と': "to",
	'だ': "da", 'ぢ': "ji", 'づ': "zu", 'で': "de", 'ど': "do",
	// n
	'な': "na", 'に': "ni", 'ぬ': "nu", 'ね': "ne", 'の': "no",
	// h
	'は': "ha", 'ひ': "hi", 'ふ': "fu", 'へ': "he", 'ほ': "ho",
	'ば': "ba", 'び': "bi", 'ぶ': "bu", 'べ': "be", 'ぼ': "bo",
	'ぱ': "pa", 'ぴ': "pi", 'ぷ': "pu", 'ぺ': "pe", 'ぽ': "po",
	// m
	'ま': "ma", 'み': "mi", 'む': "mu", 'め': "me", 'も': "mo",
	// y
	'や': "ya", 'ゆ': "yu", 'よ': "yo", 'ゃ': "ya", 'ゅ': "yu", 'ょ': "yo",
	// r
	'ら': "ra", 'り': "ri", 'る': "ru", 'れ': "re", 'ろ': "ro",
	// w
	'わ': "wa", 'ゐ': "wi", 'ゑ': "we", 'を': "o", 'ゎ': "wa",
	// n / v
	'ん': "n", 'ゔ': "vu",
}

// digraphs are two-kana combinations (yōon and common loanword sounds).
var digraph = map[[2]rune]string{
	{'き', 'ゃ'}: "kya", {'き', 'ゅ'}: "kyu", {'き', 'ょ'}: "kyo",
	{'ぎ', 'ゃ'}: "gya", {'ぎ', 'ゅ'}: "gyu", {'ぎ', 'ょ'}: "gyo",
	{'し', 'ゃ'}: "sha", {'し', 'ゅ'}: "shu", {'し', 'ょ'}: "sho",
	{'じ', 'ゃ'}: "ja", {'じ', 'ゅ'}: "ju", {'じ', 'ょ'}: "jo",
	{'ち', 'ゃ'}: "cha", {'ち', 'ゅ'}: "chu", {'ち', 'ょ'}: "cho",
	{'ぢ', 'ゃ'}: "ja", {'ぢ', 'ゅ'}: "ju", {'ぢ', 'ょ'}: "jo",
	{'に', 'ゃ'}: "nya", {'に', 'ゅ'}: "nyu", {'に', 'ょ'}: "nyo",
	{'ひ', 'ゃ'}: "hya", {'ひ', 'ゅ'}: "hyu", {'ひ', 'ょ'}: "hyo",
	{'び', 'ゃ'}: "bya", {'び', 'ゅ'}: "byu", {'び', 'ょ'}: "byo",
	{'ぴ', 'ゃ'}: "pya", {'ぴ', 'ゅ'}: "pyu", {'ぴ', 'ょ'}: "pyo",
	{'み', 'ゃ'}: "mya", {'み', 'ゅ'}: "myu", {'み', 'ょ'}: "myo",
	{'り', 'ゃ'}: "rya", {'り', 'ゅ'}: "ryu", {'り', 'ょ'}: "ryo",
	// loanword combinations
	{'ふ', 'ぁ'}: "fa", {'ふ', 'ぃ'}: "fi", {'ふ', 'ぇ'}: "fe", {'ふ', 'ぉ'}: "fo",
	{'て', 'ぃ'}: "ti", {'で', 'ぃ'}: "di",
	{'と', 'ぅ'}: "tu", {'ど', 'ぅ'}: "du",
	{'う', 'ぃ'}: "wi", {'う', 'ぇ'}: "we", {'う', 'ぉ'}: "wo",
	{'ゔ', 'ぁ'}: "va", {'ゔ', 'ぃ'}: "vi", {'ゔ', 'ぇ'}: "ve", {'ゔ', 'ぉ'}: "vo",
	{'し', 'ぇ'}: "she", {'じ', 'ぇ'}: "je", {'ち', 'ぇ'}: "che",
	{'つ', 'ぁ'}: "tsa", {'つ', 'ぃ'}: "tsi", {'つ', 'ぇ'}: "tse", {'つ', 'ぉ'}: "tso",
}

// A few words spell the topic particle は as "wa"; a pure kana->romaji pass
// would get these wrong.
var romajiOverrides = map[string]string{
	"こんにちは": "konnichiwa",
	"こんばんは": "konbanwa",
	"こんばんわ": "konbanwa",
}

// Romaji transliterates a kana reading. Spaces and unknown characters pass
// through unchanged.
func Romaji(kana string) string {
	kana = strings.TrimSpace(kana)
	if kana == "" {
		return ""
	}
	if v, ok := romajiOverrides[kana]; ok {
		return v
	}

	rs := normaliseKana([]rune(kana))
	var b strings.Builder

	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch r {
		case 'っ':
			// Small tsu doubles the consonant of the next syllable.
			if i+1 < len(rs) {
				b.WriteString(geminate(syllable(rs, i+1)))
			}
		case 'ー':
			// Katakana long vowel: repeat the previous vowel.
			if s := b.String(); s != "" {
				last := rune(s[len(s)-1])
				if strings.ContainsRune("aeiou", last) {
					b.WriteRune(last)
				}
			}
		case 'ん':
			if i+1 < len(rs) && isVowelOrY(rs[i+1]) {
				b.WriteString("n'")
			} else {
				b.WriteString("n")
			}
		default:
			if i+1 < len(rs) {
				if v, ok := digraph[[2]rune{r, rs[i+1]}]; ok {
					b.WriteString(v)
					i++
					continue
				}
			}
			if v, ok := kanaBase[r]; ok {
				b.WriteString(v)
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

// RomajiList transliterates several readings and joins them, e.g.
// ["ニチ","ジツ"] -> "nichi / jitsu".
func RomajiList(readings []string) string {
	parts := make([]string, 0, len(readings))
	for _, r := range readings {
		if s := Romaji(r); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " / ")
}

func normaliseKana(rs []rune) []rune {
	out := make([]rune, len(rs))
	for i, r := range rs {
		// ァ..ヶ (0x30A1..0x30F6) -> hiragana by subtracting 0x60.
		if r >= 0x30A1 && r <= 0x30F6 {
			r -= 0x60
		}
		out[i] = r
	}
	return out
}

// syllable returns the romaji for the syllable starting at index i.
func syllable(rs []rune, i int) string {
	if i >= len(rs) {
		return ""
	}
	if i+1 < len(rs) {
		if v, ok := digraph[[2]rune{rs[i], rs[i+1]}]; ok {
			return v
		}
	}
	if v, ok := kanaBase[rs[i]]; ok {
		return v
	}
	return ""
}

// geminate returns the consonant to double before a syllable.
func geminate(next string) string {
	if next == "" {
		return ""
	}
	if strings.HasPrefix(next, "ch") {
		return "t" // matcha: ま + t + cha
	}
	r, _ := utf8.DecodeRuneInString(next)
	if strings.ContainsRune("aeiou", r) {
		return ""
	}
	return string(r)
}

func isVowelOrY(r rune) bool {
	switch r {
	case 'あ', 'い', 'う', 'え', 'お', 'ぁ', 'ぃ', 'ぅ', 'ぇ', 'ぉ',
		'や', 'ゆ', 'よ', 'ゃ', 'ゅ', 'ょ':
		return true
	}
	return false
}
