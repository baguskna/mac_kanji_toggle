package main

import "testing"

func TestRomaji(t *testing.T) {
	cases := []struct{ kana, want string }{
		// plain gojūon
		{"あいうえお", "aiueo"},
		{"かきくけこ", "kakikukeko"},
		{"さしすせそ", "sashisuseso"},
		{"たちつてと", "tachitsuteto"},
		{"なにぬねの", "naninuneno"},
		{"はひふへほ", "hahifuheho"},
		{"まみむめも", "mamimumemo"},
		{"やゆよ", "yayuyo"},
		{"らりるれろ", "rarirurero"},
		{"わをん", "waon"},
		// words
		{"たべる", "taberu"},
		{"がっこう", "gakkou"},
		{"にほんご", "nihongo"},
		{"とうきょう", "toukyou"},
		{"きょう", "kyou"},
		{"しんぶん", "shinbun"},
		{"きっぷ", "kippu"},
		{"まっちゃ", "matcha"},
		{"ちょっと", "chotto"},
		{"ぎゅうにゅう", "gyuunyuu"},
		{"せんせい", "sensei"},
		{"おおさか", "oosaka"},
		{"でんしゃ", "densha"},
		{"しゃしん", "shashin"},
		{"ふぁん", "fan"},
		{"ぱん", "pan"},
		{"ありがとう", "arigatou"},
		{"おはようございます", "ohayougozaimasu"},
		// katakana
		{"コーヒー", "koohii"},
		{"パン", "pan"},
		{"ヴァイオリン", "vaiorin"},
		{"チョコレート", "chokoreeto"},
		// n disambiguation
		{"んあ", "n'a"},
		{"しんよう", "shin'you"},
		{"かんい", "kan'i"},
		// digraphs for ぢ/づ
		{"ぢゃ", "ja"},
		{"づつう", "zutsuu"},
		// particle spelling
		{"こんにちは", "konnichiwa"},
		{"こんばんは", "konbanwa"},
	}
	for _, c := range cases {
		if got := Romaji(c.kana); got != c.want {
			t.Errorf("Romaji(%q) = %q, want %q", c.kana, got, c.want)
		}
	}
}

func TestRomajiEmpty(t *testing.T) {
	for _, s := range []string{"", "  ", "・"} {
		if got := Romaji(s); got != "" && s != "・" {
			t.Errorf("Romaji(%q) = %q, want empty", s, got)
		}
	}
}

func TestRomajiList(t *testing.T) {
	if got := RomajiList([]string{"ニチ", "ジツ"}); got != "nichi / jitsu" {
		t.Errorf("RomajiList = %q", got)
	}
	if got := RomajiList(nil); got != "" {
		t.Errorf("RomajiList(nil) = %q", got)
	}
}
