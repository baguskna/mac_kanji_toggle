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

	if *snapshot != "" {
		switch strings.ToLower(*modeFlag) {
		case "kanji":
			app.SetMode(ModeKanji)
		case "kotoba":
			app.SetMode(ModeKotoba)
		case "":
		default:
			log.Fatalf("%s: unknown --mode %q (want kanji or kotoba)", appName, *modeFlag)
		}
		app.SetIndex(*indexFlag)
		uiSnapshot(app.CurrentCard(), *snapshot, *dark)
		return
	}

	log.Printf("%s: %d kanji / %d kotoba loaded", appName, len(deck.Kanji), len(deck.Kotoba))

	uiStart()
	app.refresh()
	app.StartAuto()

	if *showNow {
		uiShowNow()
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

// stateFilePath is ~/Library/Application Support/KanjiBar/state.json.
func stateFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, appName, "state.json")
}
