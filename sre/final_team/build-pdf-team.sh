#!/usr/bin/env bash
#
# Build both deliverables for the team final:
#   - REPORT.pdf
#   - PRESENTATION.pdf
#
# Usage:
#   cd sre/final_team
#   ./build-pdf-team.sh

set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$HERE"

# ── 1. REPORT.pdf via pandoc + xelatex ───────────────────────────────
echo ">>> Building REPORT.pdf ..."
if ! command -v pandoc >/dev/null; then
    echo "  pandoc not found. Install: brew install pandoc && brew install --cask basictex"
    exit 1
fi

pandoc REPORT.md \
    --pdf-engine=xelatex \
    -V geometry:margin=1.8cm \
    -V mainfont="Helvetica Neue" \
    -V monofont="Menlo" \
    -V colorlinks=true -V linkcolor=blue -V urlcolor=blue \
    --toc --toc-depth=2 \
    -o REPORT.pdf 2>/tmp/pandoc.log \
  || { echo "  pandoc failed; falling back to DejaVu Sans"; \
       pandoc REPORT.md --pdf-engine=xelatex \
         -V geometry:margin=1.8cm \
         -V mainfont="DejaVu Sans" -V monofont="DejaVu Sans Mono" \
         -V colorlinks=true -V linkcolor=blue -V urlcolor=blue \
         --toc --toc-depth=2 -o REPORT.pdf; }

echo "  ✓ $HERE/REPORT.pdf  ($(du -h REPORT.pdf | cut -f1))"

# ── 2. PRESENTATION.pdf via Marp CLI ─────────────────────────────────
echo
echo ">>> Building PRESENTATION.pdf ..."
cd "$HERE/presentation"

if command -v marp >/dev/null 2>&1; then
    marp PRESENTATION.md --pdf --allow-local-files -o PRESENTATION.pdf
elif command -v npx >/dev/null 2>&1; then
    npx --yes @marp-team/marp-cli PRESENTATION.md --pdf --allow-local-files -o PRESENTATION.pdf
else
    echo "  Neither 'marp' nor 'npx' is installed."
    echo "  Easiest fix:  npm install -g @marp-team/marp-cli   (requires Node.js)"
    echo "  Alternative:  install VS Code + the 'Marp for VS Code' extension and export from there."
    exit 1
fi

echo "  ✓ $HERE/presentation/PRESENTATION.pdf  ($(du -h PRESENTATION.pdf | cut -f1))"

echo
echo "════════════════════════════════════════════"
echo "  All done.  Files to submit:"
echo "    $HERE/REPORT.pdf"
echo "    $HERE/presentation/PRESENTATION.pdf"
echo "════════════════════════════════════════════"
