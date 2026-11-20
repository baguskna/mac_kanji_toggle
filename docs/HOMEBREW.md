# Homebrew distribution

KanjiBar ships as a **Homebrew cask** in a personal tap:

- Tap repo: <https://github.com/baguskna/homebrew-tap>
- Release asset: <https://github.com/baguskna/mac_kanji_toggle/releases>

The official `Homebrew/homebrew-cask` repository is **not possible yet** — see
[Not in homebrew-cask yet](#not-in-homebrew-cask-yet) below.

## For users

```sh
brew tap baguskna/tap
brew trust baguskna/tap           # Homebrew 5+ requires trusting third-party taps
brew install --cask kanjibar
```

Then launch **KanjiBar** from `/Applications` (or Spotlight). It appears in the
menu bar; there is no Dock icon.

```sh
brew upgrade --cask kanjibar      # update
brew uninstall --cask kanjibar    # remove the app
brew untap baguskna/tap           # remove the tap
```

The cask removes the download quarantine flag after install because releases
are currently ad-hoc signed (not notarized). This is done with a
`postflight_steps` stanza in `Casks/kanjibar.rb`; it can be dropped once
releases are Developer-ID signed and notarized.

## For maintainers: cutting a release

1. Bump `VERSION` (e.g. `1.1.0`) and commit.
2. Package and stamp the cask:

   ```sh
   ./scripts/package-release.sh
   ```

   This builds `dist/KanjiBar.app`, creates `dist/KanjiBar-<VERSION>.zip`,
   computes its sha256 and writes both into `Casks/kanjibar.rb`.

3. Publish the GitHub release:

   ```sh
   gh release create v$(cat VERSION) "dist/KanjiBar-$(cat VERSION).zip" \
     --repo baguskna/mac_kanji_toggle --title "KanjiBar $(cat VERSION)" --generate-notes
   ```

4. Sync the cask into the tap repository and push:

   ```sh
   cp Casks/kanjibar.rb ../homebrew-tap/Casks/kanjibar.rb
   cd ../homebrew-tap && git commit -am "kanjibar $(cat ../mac_kanji_toggle/VERSION)" && git push
   ```

5. Sanity check as a user:

   ```sh
   brew update && brew reinstall --cask kanjibar
   /Applications/KanjiBar.app/Contents/MacOS/KanjiBar --version
   ```

The zip that gets uploaded **must** be the exact file the cask's `sha256` was
computed from; re-run the packaging script if anything is rebuilt.

## Why a tap?

- It works today: no review, no notarization gatekeeping.
- Users pay one extra command (`brew trust baguskna/tap`) because Homebrew 5+
  does not load third-party taps by default.

## Not in homebrew-cask yet

A PR to [`Homebrew/homebrew-cask`](https://github.com/Homebrew/homebrew-cask)
currently has two hard blockers:

1. **Notability.** New casks are rejected when the project is too obscure — the
   current bar (per homebrew-cask's authoring guide) is roughly **30 forks or
   watchers, or 75 stars**. This is a brand-new repo.
2. **Signing/notarization.** Homebrew 5+ deprecates unsigned/non-notarized
   casks and its audits check codesigning and notarization. Shipping an
   un-notarized cask with a quarantine-removal step would be rejected.

To qualify later:

- Enroll in the Apple Developer Program (paid).
- Sign `KanjiBar.app` with a **Developer ID Application** certificate and the
  hardened runtime.
- Notarize with `notarytool submit --wait` and `stapler staple`.
- Remove the quarantine step from the cask.
- Run `brew create --cask <release-zip-url>`, `brew audit --cask --new kanjibar`,
  then open a PR. Casks are reviewed by the Homebrew maintainers.

Until then, the tap above is the "everyone can `brew install` it" path.

## Useful references

- Acceptable casks / package acceptance policy:
  <https://docs.brew.sh/Package-Acceptance-Policy>
- Tap trust (Homebrew 5+): <https://docs.brew.sh/Tap-Trust>
- Cask cookbook: <https://docs.brew.sh/Cask-Cookbook>
