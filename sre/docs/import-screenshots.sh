#!/usr/bin/env bash
#
# Pick the 6 most recent screenshots from ~/Desktop and copy them to
# sre/docs/screenshots/ under the names REPORT.md expects.
#
# Order is by mtime ascending (oldest first), matching the order in
# REPORT_GUIDE.md:
#   1. docker compose ps
#   2. curl payment / user-profile
#   3. prometheus targets
#   4. prometheus alerts (inactive)
#   5. grafana dashboard
#   6. prometheus alerts (pending / firing)

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST="$HERE/screenshots"
mkdir -p "$DEST"

# Match common macOS screenshot names in EN and RU.
shopt -s nullglob
cd ~/Desktop

mapfile -t SHOTS < <(
  ls -t \
    "Screenshot "*.png \
    "Screen Shot "*.png \
    "Снимок экрана "*.png \
    "CleanShot "*.png \
    2>/dev/null
)

if [[ ${#SHOTS[@]} -lt 6 ]]; then
    echo "Found only ${#SHOTS[@]} screenshots on ~/Desktop."
    echo "Take all 6 screenshots first (see REPORT_GUIDE.md), then re-run."
    printf '  %s\n' "${SHOTS[@]:-<none>}"
    exit 1
fi

# Take the 6 most recent, then reverse so the OLDEST becomes #1.
RECENT=("${SHOTS[@]:0:6}")
ORDERED=()
for ((i=5; i>=0; i--)); do ORDERED+=("${RECENT[$i]}"); done

TARGETS=(
  01-docker-ps.png
  02-microservices-running.png
  03-prometheus-targets.png
  04-prometheus-alerts.png
  05-grafana-dashboard.png
  06-incident-simulation.png
)

echo "Will copy and rename:"
for i in 0 1 2 3 4 5; do
    printf '  %s  ->  %s\n' "${ORDERED[$i]}" "${TARGETS[$i]}"
done
echo
read -r -p "Proceed? [y/N] " ans
[[ "$ans" =~ ^[Yy] ]] || { echo "Aborted."; exit 1; }

for i in 0 1 2 3 4 5; do
    cp "$HOME/Desktop/${ORDERED[$i]}" "$DEST/${TARGETS[$i]}"
done

echo
echo "Imported into $DEST :"
ls -la "$DEST"
echo
echo "Now rebuild the PDF:"
echo "  cd $HERE && ./build-pdf.sh"
