#!/usr/bin/env bash
# Builds a distributable zip of KanjiBar.app and stamps Casks/kanjibar.rb with
# the version + sha256, so `brew install --cask kanjibar` installs this exact
# build.
#
#   ./scripts/package-release.sh            # uses VERSION
#   ./scripts/package-release.sh 1.2.0      # bumps VERSION and packages 1.2.0
#
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ $# -ge 1 ]]; then
  printf '%s\n' "$1" > VERSION
fi
VERSION="$(cat VERSION)"

echo "==> building KanjiBar ${VERSION}"
./build.sh

ZIP="dist/KanjiBar-${VERSION}.zip"
rm -f "$ZIP"
# ditto keeps the bundle intact (resource forks, signatures).
ditto -c -k --keepParent "dist/KanjiBar.app" "$ZIP"
SHA="$(shasum -a 256 "$ZIP" | awk '{print $1}')"

echo "==> updating Casks/kanjibar.rb (version ${VERSION}, sha256 ${SHA})"
python3 - "$VERSION" "$SHA" <<'PY'
import re, sys
version, sha = sys.argv[1], sys.argv[2]
path = "Casks/kanjibar.rb"
text = open(path).read()
text = re.sub(r'^\s*version\s+"[^"]*"', f'  version "{version}"', text, count=1, flags=re.M)
text = re.sub(r'^\s*sha256\s+"[^"]*"', f'  sha256 "{sha}"', text, count=1, flags=re.M)
open(path, "w").write(text)
PY

echo
echo "==> packaged: $ZIP"
echo "    sha256:   $SHA"
echo
echo "Next: publish the release, then sync the cask into your tap repo."
echo
echo "  1) gh release create v${VERSION} \"$ZIP\" \\"
echo "       --title \"KanjiBar ${VERSION}\" --generate-notes --repo baguskna/mac_kanji_toggle"
echo "  2) publish the tap (once): create github.com/baguskna/homebrew-tap,"
echo "     copy Casks/kanjibar.rb there, then:"
echo "       brew tap baguskna/tap"
echo "       brew install --cask kanjibar"
echo
echo "  (Upload + hash must match; re-run this script whenever you change the app.)"
