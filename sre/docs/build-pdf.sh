#!/usr/bin/env bash
#
# Build the End Term Project final PDF from REPORT.md.
#
# Prerequisites (one-time):
#   brew install pandoc
#   brew install --cask basictex   # provides xelatex
#   eval "$(/usr/libexec/path_helper)"
#   sudo tlmgr update --self
#   sudo tlmgr install collection-fontsrecommended unicode-math
#
# Usage:
#   cd sre/docs
#   ./build-pdf.sh
#
# Output: REPORT.pdf in the current directory.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$HERE"

INPUT=REPORT.md
OUTPUT=REPORT.pdf

# Prefer xelatex (supports unicode + ASCII boxes nicely).  Fallback to
# weasyprint (HTML pipeline) if no LaTeX is installed, then to chrome
# headless print-to-pdf.

build_with_xelatex() {
    pandoc "$INPUT" \
        --pdf-engine=xelatex \
        -V geometry:margin=1.5cm \
        -V mainfont="Helvetica Neue" \
        -V monofont="Menlo" \
        -V colorlinks=true \
        -V linkcolor=blue \
        -V urlcolor=blue \
        --toc \
        --toc-depth=2 \
        -o "$OUTPUT"
}

build_with_weasyprint() {
    pandoc "$INPUT" -t html5 -s --metadata title="EduLMS SRE — End Term" \
        -c <(printf '%s' "
            body{font-family:-apple-system,Helvetica,sans-serif;max-width:780px;margin:40px auto;padding:0 24px;color:#222}
            h1{font-size:28px;margin-top:40px}h2{font-size:20px;margin-top:32px}h3{font-size:16px;margin-top:24px}
            pre,code{font-family:Menlo,Consolas,monospace;font-size:12px}
            pre{background:#f4f4f4;padding:12px;border-radius:6px;overflow:auto}
            table{border-collapse:collapse;width:100%;margin:12px 0}
            th,td{border:1px solid #ccc;padding:6px 10px;text-align:left;font-size:13px}
            img{max-width:100%;border:1px solid #ddd;margin:8px 0}
            a{color:#0366d6;text-decoration:none}
        ") -o REPORT.html
    weasyprint REPORT.html "$OUTPUT"
    rm REPORT.html
}

build_with_chrome() {
    # Render through Markdown -> HTML, then Chrome's headless print-to-PDF.
    pandoc "$INPUT" -t html5 -s --metadata title="EduLMS SRE — End Term" \
        -o REPORT.html
    local CHROME
    for c in \
        "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
        "/Applications/Chromium.app/Contents/MacOS/Chromium" \
        "google-chrome" "chromium" "chrome"; do
        if [[ -x "$c" || -n "$(command -v "$c" 2>/dev/null)" ]]; then
            CHROME="$c"; break
        fi
    done
    [[ -z "${CHROME:-}" ]] && return 1
    "$CHROME" --headless --disable-gpu --no-sandbox \
        --print-to-pdf="$PWD/$OUTPUT" "file://$PWD/REPORT.html"
    rm REPORT.html
}

echo ">>> Building $OUTPUT from $INPUT ..."

if command -v xelatex >/dev/null 2>&1; then
    build_with_xelatex
elif command -v weasyprint >/dev/null 2>&1; then
    build_with_weasyprint
elif build_with_chrome; then
    :
else
    echo "ERROR: install one of:" >&2
    echo "  brew install --cask basictex   # then add /Library/TeX/texbin to PATH" >&2
    echo "  brew install weasyprint" >&2
    echo "  Google Chrome (for headless PDF)" >&2
    exit 1
fi

echo "OK  -> $HERE/$OUTPUT"
ls -lh "$OUTPUT"
