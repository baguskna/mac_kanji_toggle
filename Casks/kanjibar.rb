cask "kanjibar" do
  version "1.0.3"
  sha256 "348535fecfbc3c15cfd446476c7da265eb0eaaab9b0ec6edf2fd497c0081fe5b"

  url "https://github.com/baguskna/mac_kanji_toggle/releases/download/v#{version}/KanjiBar-#{version}.zip"
  name "KanjiBar"
  desc "JLPT N5–N3 kanji and vocabulary widget for the macOS menu bar"
  homepage "https://github.com/baguskna/mac_kanji_toggle"

  livecheck do
    url "https://github.com/baguskna/mac_kanji_toggle"
    strategy :github_latest
  end

  depends_on macos: :big_sur

  app "KanjiBar.app"

  uninstall quit: "com.baguskna.kanjibar"

  # Releases are ad-hoc signed (no Developer ID / notarization yet). Homebrew
  # downloads over curl, which leaves a quarantine flag that blocks the first
  # launch, so clear it after the app lands in /Applications. Delete this once
  # releases are Developer-ID signed and notarized.
  postflight_steps do
    run "/usr/bin/xattr",
        args: ["-dr", "com.apple.quarantine", "{{appdir}}/KanjiBar.app"],
        writable_paths: ["KanjiBar.app"],
        writable_base: :appdir
  end

  zap trash: [
    "~/Library/Application Support/KanjiBar",
  ]

  caveats <<~EOS
    KanjiBar lives in the menu bar (no Dock icon). Click it for a card.
    Quit it from the right-click menu.
  EOS
end
