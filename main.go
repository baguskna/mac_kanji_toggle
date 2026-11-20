package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const appName = "KanjiBar"

func main() {
	// AppKit wants the event loop on the process's main thread.
	runtime.LockOSThread()

	snapshot := flag.String("snapshot", "", "render the current card to a PNG file and exit")
	dark := flag.Bool("dark", false, "use the dark appearance with --snapshot")
	modeFlag := flag.String("mode", "", "deck for --snapshot: kanji or kotoba")
	indexFlag := flag.Int("index", 0, "deck index for --snapshot")
	version := flag.Bool("version", false, "print the version and exit")
	showNow := flag.Bool("show", false, "dev: open the card right after launch")
	logFile := flag.String("logfile", "", "dev: also write logs to this file")
	exitAfter := flag.Int("exit-after", 0, "dev: quit after N seconds")
	cycle := flag.Int("cycle", 0, "dev: advance to the next card every N seconds")
	hideAfter := flag.Int("hide-after", 0, "dev: close the panel N seconds after launch")
	autoSwitch := flag.Int("auto-switch", 0, "dev: deck auto-switch interval in seconds (default 15)")
	flag.Parse()

	if *version {
		fmt.Printf("%s 1.0.0\n", appName)
		return
	}

	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			log.SetOutput(f)
		}
	}

	deck, err := LoadDeck()
	if err != nil {
		log.Fatalf("%s: invalid data: %v", appName, err)
	}

	app := NewApp(deck, stateFilePath())
	appRef = app
	app.LoadState()

	if *modeFlag != "" {
		switch strings.ToLower(*modeFlag) {
		case "kanji":
			app.SetMode(ModeKanji)
		case "kotoba":
			app.SetMode(ModeKotoba)
		default:
			log.Fatalf("%s: unknown --mode %q (want kanji or kotoba)", appName, *modeFlag)
		}
	}

	if *snapshot != "" {
		app.SetIndex(*indexFlag)
		uiSnapshot(app.CurrentCard(), *snapshot, *dark)
		return
	}

	if err := acquireSingleInstance(); err != nil {
		log.Fatalf("%s: %v", appName, err)
	}

	if *autoSwitch > 0 {
		app.SetAutoInterval(time.Duration(*autoSwitch) * time.Second)
	}

	log.Printf("%s: %d kanji / %d kotoba loaded", appName, len(deck.Kanji), len(deck.Kotoba))

	uiStart()
	app.uiEnabled = true

	// Measure every card once so the panel can keep one constant size.
	uiSetFixedCardHeight(measureFixedCardHeight(deck))

	app.refresh()
	app.StartAuto()

	if *showNow {
		uiShowNow()
	}
	if *hideAfter > 0 {
		go func() {
			time.Sleep(time.Duration(*hideAfter) * time.Second)
			uiHideNow()
		}()
	}
	if *cycle > 0 {
		go func() {
			t := time.NewTicker(time.Duration(*cycle) * time.Second)
			for range t.C {
				app.Next()
			}
		}()
	}
	if *exitAfter > 0 {
		go func() {
			time.Sleep(time.Duration(*exitAfter) * time.Second)
			uiQuit()
		}()
	}

	uiRun()
}

// measureFixedCardHeight returns the tallest card in the deck. The panel uses
// it as a fixed height so switching cards never resizes the widget.
func measureFixedCardHeight(deck *Deck) float64 {
	start := time.Now()
	maxHeight := 0.0
	for i := range deck.Kanji {
		if h := uiMeasureCardHeight(KanjiCard(deck.Kanji[i])); h > maxHeight {
			maxHeight = h
		}
	}
	for i := range deck.Kotoba {
		if h := uiMeasureCardHeight(KotobaCard(deck.Kotoba[i])); h > maxHeight {
			maxHeight = h
		}
	}
	log.Printf("%s: card height %.0f (measured %d cards in %v)",
		appName, maxHeight, len(deck.Kanji)+len(deck.Kotoba), time.Since(start).Round(time.Millisecond))
	return maxHeight
}

// stateFilePath is ~/Library/Application Support/KanjiBar/state.json.
func stateFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, appName, "state.json")
}
