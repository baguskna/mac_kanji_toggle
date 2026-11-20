# KanjiBar

A small macOS **menu bar widget** that shows a JLPT **N3** kanji or vocabulary
word with its kana reading, **romaji** and English meaning — one click, no app
window, no Dock icon.

Built with **Go** (state, data, romaji, timer) and **AppKit/Objective-C** (the
native status item and popover-style card). It compiles with the Command Line
Tools — Xcode is not required.

| Menu bar | Kanji card | Kotoba card |
| --- | --- | --- |
| ![Menu bar](docs/menubar.png) | ![Kanji card](docs/card-kanji.png) | ![Kotoba card](docs/card-kotoba.png) |

![Live panel](docs/panel-live.png)

## Install

```bash
./build.sh --install      # builds, ad-hoc signs, copies to ~/Applications, launches
```

Or build only and run it from anywhere:

```bash
./build.sh
open dist/KanjiBar.app
```

The app is a 2 MB bundle, ad-hoc signed, `LSUIElement` (no Dock icon), and keeps
its state in `~/Library/Application Support/KanjiBar/state.json`.

## Use

- **Left-click the menu bar item** — opens the card.
- **Right-click** — context menu (next / previous / toggle deck / auto-advance /
  menu bar romaji / copy / quit).
- **← →** or **space** — previous / next card while the card is open.
- **T** — switch between the kanji deck (漢字) and the vocabulary deck (言葉).
- **A** — toggle auto-advance (a new card every 15 minutes while the app sits in
  the menu bar).
- **R** — toggle romaji in the menu bar title (`連  ren` ⇄ `連`).
- **C** — copy the current card as text.
- **Esc** — close the card.

Clicking anywhere outside the card dismisses it and returns focus to the app you
were using. The current card, deck and preferences survive restarts.

## The decks

| Deck | File | Entries | Contents |
| --- | --- | --- | --- |
| 漢字 Kanji | `data/kanji.json` | 200 N3 kanji | on'yomi (katakana), kun'yomi (hiragana), English gloss, 3 common compounds with kana + meaning |
| 言葉 Kotoba | `data/kotoba.json` | 210 N3 words | kana reading, part of speech, English gloss, example sentence with kana transcript + translation |

Both decks are **N3 only** (`"level": 3`); the mode pill and the orange `N3`
badge reflect that. To study another level, replace the JSON files — the schema
is validated at startup and by `go test`:

```jsonc
// data/kanji.json (array of entries)
{
  "char": "連", "level": 3, "strokes": 10,
  "on": ["レン"], "kun": ["つら"],
  "meaning": "connect; take along",
  "compounds": [
    {"word": "連絡", "kana": "れんらく", "meaning": "contact"},
    {"word": "連続", "kana": "れんぞく", "meaning": "continuation"},
    {"word": "関連", "kana": "かんれん", "meaning": "relation"}
  ]
}

// data/kotoba.json (array of entries)
{
  "word": "経験", "kana": "けいけん", "level": 3, "pos": "noun",
  "meaning": "experience",
  "example": {
    "jp": "いい経験になった。",
    "kana": "いいけいけんになった。",
    "en": "It became a good experience."
  }
}
```

### Where romaji comes from

Romaji is **generated at runtime** from the kana rather than stored, so the
dataset stays small and the transliteration stays consistent. `romaji.go`
implements Hepburn-ish rules: yōon (`しゃ` → sha), small tsu gemination
(`まっちゃ` → matcha), katakana long vowels (`コーヒー` → koohii), `ん`
disambiguation, and a small override table for words where `は` is the particle
(`こんにちは` → konnichiwa). `romaji_test.go` covers the tricky cases.

## Architecture

```
main.go        flags, bootstrap: load deck → start AppKit → run loop
app.go         study state: mode, per-deck position, auto-advance, persistence
card.go        builds the Card view model (one JSON object per screen)
data.go        embedded JSON decks + validation
romaji.go      kana → romaji
bridge.go      cgo bridge; Go exports GoNext/GoPrev/GoToggle* for ObjC
appkit.h/.m    AppKit: NSStatusItem, borderless status-level panel, custom drawing
instance.go    single-instance flock
build.sh       go vet + go test + go build + Info.plist + ad-hoc sign
```

Data flows one way for rendering and the other way for input:

- Go owns the state and pushes a `Card` to ObjC as JSON (`KBShowCard`). The
  Objective-C card view measures and draws every string itself — no XIBs, no
  Auto Layout, full control over the look.
- Button clicks and key presses call back into Go (`GoNext`, `GoToggleMode`, …),
  which updates state and pushes a fresh card. That keeps the UI a dumb renderer
  and the study logic testable (`go test` runs it headlessly).

### Why Go, and the one AppKit gotcha

Go gives a tiny, dependency-free binary that starts instantly; AppKit gives a
real `NSStatusItem` instead of a floating window. The whole UI sits behind a
~100-line cgo surface.

The cgo/Objective-C code is compiled **without ARC**, so long-lived AppKit
objects must be retained manually. The status item in particular is
autoreleased — it has to be `retain`ed or later title updates crash. That is
also why the panel is a plain borderless `NSPanel` rather than `NSPopover`
(popovers proved unstable when hosted by a Go main loop on macOS 26).

## Development

```bash
go test ./...                     # deck validation, romaji, state transitions
./build.sh                        # vet + test + bundle
```

Useful dev flags (run the binary directly, not through `open`):

```bash
dist/KanjiBar.app/Contents/MacOS/KanjiBar --show --exit-after 8   # open card, quit after 8s
dist/KanjiBar.app/Contents/MacOS/KanjiBar --cycle 2               # advance every 2s
dist/KanjiBar.app/Contents/MacOS/KanjiBar --snapshot out.png --mode kotoba --index 7 [--dark]
dist/KanjiBar.app/Contents/MacOS/KanjiBar --logfile /tmp/kb.log
```

`--snapshot` renders one card to PNG headlessly, which is handy for design
iterations and for regenerating the images in `docs/`.

## Notes / limitations

- Ad-hoc signed, not notarized. Locally built apps run fine; distributing it to
  another Mac would need a Developer ID signature and notarization.
- No "launch at login" yet (add `SMAppService` if you want it).
- The menu bar title is the kanji/word itself, so very long words widen the
  item; turn off "Show Romaji in Menu Bar" for a tighter fit.
- macOS 26 hosts status items through Control Center scenes; the app deliberately
  keeps every status item mutation on the main thread and retains the item.
