#!/bin/bash
# Remove the Study Pomodoro plugin, its keybindings, CLI link, and weekly timer,
# and put OmaCal's own bar widget back where it was.
# Keeps OmaCal, the planner config, and events already in your calendar.

set -euo pipefail

PLUGIN_ID="phuniverse.study-pomodoro"
OMACAL_ID="omacal.upcoming"
BINDINGS="$HOME/.config/hypr/bindings.lua"

section="$(/usr/bin/python3 - "$HOME/.config/omarchy/shell.json" "$PLUGIN_ID" <<'EOF'
import json, sys
try:
    layout = json.load(open(sys.argv[1]))["bar"]["layout"]
except Exception:
    sys.exit(0)
for name, entries in layout.items():
    if any(isinstance(e, dict) and e.get("id") == sys.argv[2] for e in entries):
        print(name)
EOF
)"

omarchy plugin disable "$PLUGIN_ID" >/dev/null 2>&1 || true
rm -rf "$HOME/.config/omarchy/plugins/$PLUGIN_ID"
omarchy-shell -q shell rescanPlugins
if [[ -d $HOME/.config/omarchy/plugins/$OMACAL_ID ]]; then
  omarchy plugin enable "$OMACAL_ID" --section "${section:-right}" >/dev/null 2>&1 || true
fi
rm -f "$HOME/.local/bin/study-pomodoro"

systemctl --user disable --now study-pomodoro-plan.timer >/dev/null 2>&1 || true
rm -f "$HOME/.config/systemd/user/study-pomodoro-plan."{service,timer}
systemctl --user daemon-reload

if grep -q '>>> study-pomodoro' "$BINDINGS" 2>/dev/null; then
  cp "$BINDINGS" "$BINDINGS.bak.$(date +%s)"
  sed -i '/-- >>> study-pomodoro/,/-- <<< study-pomodoro/d' "$BINDINGS"
  hyprctl reload >/dev/null || true
fi

rm -rf "$HOME/.config/study-pomodoro" "$HOME/.local/state/study-pomodoro"
echo "Study Pomodoro removed; OmaCal's own widget is back in the bar."
