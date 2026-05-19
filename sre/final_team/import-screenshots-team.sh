#!/usr/bin/env bash
#
# Take the 15 most recent screenshots from ~/Desktop and copy them to
# sre/final_team/screenshots/new/ with the names REPORT.md expects.
#
# Order is oldest → newest, matching SCREENSHOTS_GUIDE.md sections A→E.

set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST="$HERE/screenshots/new"
mkdir -p "$DEST"

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

NEED=15

if [[ ${#SHOTS[@]} -lt $NEED ]]; then
    echo "Found only ${#SHOTS[@]} screenshots on ~/Desktop (need $NEED)."
    echo "Take all 15 first (see SCREENSHOTS_GUIDE.md), then re-run."
    printf '  %s\n' "${SHOTS[@]:-<none>}"
    exit 1
fi

# Take the $NEED most recent, then reverse so OLDEST = #1.
RECENT=("${SHOTS[@]:0:$NEED}")
ORDERED=()
for ((i=NEED-1; i>=0; i--)); do ORDERED+=("${RECENT[$i]}"); done

TARGETS=(
  01-repo-tree.png
  02-sre-folder-tree.png
  03-k8s-folder-listing.png
  04-terraform-folder.png
  05-ansible-folder.png
  06-cicd-yml-vscode.png
  07-github-actions-runs.png
  08-github-actions-detail.png
  09-website-live.png
  10-server-ssh-deploy.png
  11-server-docker-ps.png
  12-prometheus-targets-live.png
  13-grafana-live.png
  14-alert-firing-live.png
  15-team-photo.png
)

echo "Will copy and rename:"
for i in $(seq 0 $((NEED-1))); do
    printf '  %s  ->  %s\n' "${ORDERED[$i]}" "${TARGETS[$i]}"
done
echo
read -r -p "Proceed? [y/N] " ans
[[ "$ans" =~ ^[Yy] ]] || { echo "Aborted."; exit 1; }

for i in $(seq 0 $((NEED-1))); do
    cp "$HOME/Desktop/${ORDERED[$i]}" "$DEST/${TARGETS[$i]}"
done

echo
echo "Imported into $DEST :"
ls -la "$DEST"
echo
echo "Next: cd $HERE && ./build-pdf-team.sh"
