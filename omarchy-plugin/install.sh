#!/bin/bash
# Install the Study Pomodoro Omarchy plugin (mounted on OmaCal).
#
#   ./install.sh                  Full setup (plugin, keybindings, planner, daily planning timer)
#   ./install.sh --plugin-only    Just (re)install the plugin files; use this to update
#   ./install.sh --no-keybindings Skip adding Hyprland keybindings
#
# Requires OmaCal (https://omacal.app): its app provides the calendar data and
# its omacal.upcoming widget provides the shared QML this plugin builds on.
# Re-running is safe: every step checks what's already in place.

set -euo pipefail

PLUGIN_ID="phuniverse.study-pomodoro"
OMACAL_ID="omacal.upcoming"
OMACAL_BASE_VERSION="1.11" # the omacal.upcoming version Panel.qml was forked from
SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGINS="$HOME/.config/omarchy/plugins"
DEST="$PLUGINS/$PLUGIN_ID"
OMACAL_DIR="$PLUGINS/$OMACAL_ID"
SHELL_JSON="$HOME/.config/omarchy/shell.json"
BIN_LINK="$HOME/.local/bin/study-pomodoro"
CONFIG_DIR="$HOME/.config/study-pomodoro"
PLANNER_CONFIG="$HOME/.config/study-pomodoro/planner.json"
BINDINGS="$HOME/.config/hypr/bindings.lua"
UNIT_DIR="$HOME/.config/systemd/user"
export PATH="$HOME/.local/bin:$PATH"

plugin_only=0
keybindings=1
for arg in "$@"; do
  case $arg in
    --plugin-only) plugin_only=1 ;;
    --no-keybindings) keybindings=0 ;;
    -h|--help) sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Unknown option: $arg" >&2; exit 1 ;;
  esac
done

step() { printf '\n\033[1m==> %s\033[0m\n' "$*"; }
ok() { printf '  ✓ %s\n' "$*"; }
warn() { printf '  ! %s\n' "$*"; }
ask() { local reply; read -r -p "  $1 " reply; echo "$reply"; }
confirm() { local reply; read -r -p "  $1 [Y/n] " reply; [[ -z $reply || $reply =~ ^[Yy] ]]; }

# Section of the bar ("left"/"center"/"right") holding a widget id, or empty.
bar_section() {
  /usr/bin/python3 - "$SHELL_JSON" "$1" <<'EOF'
import json, sys
try:
    layout = json.load(open(sys.argv[1]))["bar"]["layout"]
except Exception:
    sys.exit(0)
for section, entries in layout.items():
    if any(isinstance(e, dict) and e.get("id") == sys.argv[2] for e in entries):
        print(section)
        break
EOF
}

# ---------------------------------------------------------------- checks
step "Checking requirements"
command -v omarchy-shell >/dev/null || { echo "  omarchy-shell not found — this plugin needs Omarchy." >&2; exit 1; }
if ! command -v omacal >/dev/null; then
  cat >&2 <<EOF
  OmaCal is required. Install it, open it once, and sign in:
    curl -fsSL https://omacal.app/install.sh | sh
    omacal
  Then run ./install.sh again.
EOF
  exit 1
fi
if [[ ! -f $OMACAL_DIR/manifest.json ]]; then
  cat >&2 <<EOF
  OmaCal's bar widget ($OMACAL_DIR) is missing. OmaCal installs it on its
  first start; if you removed it, OmaCal won't put it back. Restore it with:
    git clone --depth 1 https://github.com/x3me/omacal /tmp/omacal
    cp -r /tmp/omacal/packaging/omarchy-plugin $OMACAL_DIR
EOF
  exit 1
fi
omacal_version="$(/usr/bin/python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["version"])' "$OMACAL_DIR/manifest.json")"
if [[ $omacal_version != "$OMACAL_BASE_VERSION".* ]]; then
  warn "omacal.upcoming is $omacal_version; this plugin was forked from $OMACAL_BASE_VERSION.x."
  warn "If the popup misbehaves, re-apply the \"Study Pomodoro:\" changes onto the new Panel.qml."
fi
ok "OmaCal $(omacal doctor 2>/dev/null | awk '/version/ {print $3; exit}') and its widget ($omacal_version) found"

# ---------------------------------------------------------------- plugin
step "Installing plugin to $DEST"
mkdir -p "$DEST" "$HOME/.local/bin" "$CONFIG_DIR" "$HOME/.local/state/study-pomodoro"
# Copy (not symlink): the shell's hot-reload watcher doesn't follow symlinks.
rsync -a --delete --exclude '.git' --exclude 'tests' --exclude '__pycache__' --exclude 'install.sh' --exclude 'uninstall.sh' "$SRC/" "$DEST/"
chmod +x "$DEST/bin/study-pomodoro"
ln -sfn "$DEST/bin/study-pomodoro" "$BIN_LINK"
rm -rf "$HOME/.cache/study-pomodoro" # left over from the pre-OmaCal version
ok "Files copied, CLI linked at $BIN_LINK"

# One OmaCal item in the bar: ours takes omacal.upcoming's place. Disabling
# (not removing) keeps its files, which this plugin imports, and lets OmaCal
# keep updating them.
omacal_section="$(bar_section "$OMACAL_ID")"
our_section="$(bar_section "$PLUGIN_ID")"
omarchy-shell -q shell rescanPlugins
for _ in $(seq 1 20); do
  omarchy plugin list --json 2>/dev/null | grep -q "\"$PLUGIN_ID\"" && break
  sleep 0.5
done
if [[ -n $omacal_section ]]; then
  omarchy plugin disable "$OMACAL_ID" >/dev/null && ok "Disabled OmaCal's own widget in the bar (files kept)"
fi
if [[ -z $our_section ]]; then
  section="${omacal_section:-right}"
  if omarchy plugin enable "$PLUGIN_ID" --section "$section" >/dev/null 2>&1; then
    ok "Enabled in the bar ($section)"
  else
    warn "Could not enable automatically. Run: omarchy plugin enable $PLUGIN_ID --section $section"
  fi
elif [[ -n $omacal_section && $omacal_section != "$our_section" ]]; then
  # Taking over from omacal.upcoming just now: sit where it sat.
  omarchy bar move "$PLUGIN_ID" --section "$omacal_section" >/dev/null 2>&1 \
    && ok "Moved to OmaCal's spot in the bar ($omacal_section)" \
    || warn "Could not move it. Run: omarchy bar move $PLUGIN_ID --section $omacal_section"
else
  ok "Already in the bar ($our_section)"
fi

# Hot reload doesn't reliably pick this widget up (Qt caches the plugin
# folder's listing and the imported OmaCal files), so start the shell fresh.
omarchy restart shell >/dev/null 2>&1 && ok "Restarted the Omarchy shell to load it"

(( plugin_only )) && { echo; echo "Plugin updated."; exit 0; }

# ---------------------------------------------------------------- keybindings
if (( keybindings )) && ! grep -q '>>> study-pomodoro' "$BINDINGS" 2>/dev/null; then
  step "Adding keybindings to $BINDINGS"
  cp "$BINDINGS" "$BINDINGS.bak.$(date +%s)"
  cat >>"$BINDINGS" <<EOF

-- >>> study-pomodoro (remove this block to drop the bindings)
o.bind("SUPER + ALT + P", "Pomodoro start/pause", "$BIN_LINK toggle")
o.bind("SUPER + SHIFT + ALT + P", "Pomodoro skip phase", "$BIN_LINK skip")
o.bind("SUPER + CTRL + ALT + P", "Calendar and study popup", "$BIN_LINK popup")
-- <<< study-pomodoro
EOF
  hyprctl reload >/dev/null
  errors="$(hyprctl configerrors 2>/dev/null || true)"
  if [[ -n ${errors//[[:space:]]/} && $errors != *"no errors"* ]]; then
    warn "Hyprland reported config errors:"; echo "$errors" | sed 's/^/    /'
  else
    ok "Super+Alt+P start/pause · Super+Shift+Alt+P skip · Super+Ctrl+Alt+P popup"
  fi
fi

# ---------------------------------------------------------------- omacal account
step "OmaCal account"
has_calendars() { omacal calendars --json 2>/dev/null | /usr/bin/python3 -c 'import json,sys; sys.exit(0 if json.load(sys.stdin).get("data") else 1)'; }
until has_calendars; do
  pgrep -x omacal >/dev/null || (setsid -f omacal >/dev/null 2>&1)
  echo "  OmaCal has no calendars yet. Sign in to Google in the OmaCal window."
  reply="$(ask "Press Enter once your calendars show up (or type 'skip'):")"
  [[ $reply == skip ]] && { echo "  Skipped. Re-run ./install.sh after signing in to finish setup."; exit 0; }
done
ok "Calendars available"

# ---------------------------------------------------------------- planner config
if [[ ! -f $PLANNER_CONFIG ]]; then
  step "Configuring the study planner"
  echo "  Pick your calendars and study hours, then rate each course's difficulty"
  echo "  (1 easy – 5 hard) and give its exam code if you know it (e.g. the \"GT\" in"
  echo "  \"GT written\"). Change any of it later with: study-pomodoro setup"
  "$BIN_LINK" setup
fi
ok "Planner config at $PLANNER_CONFIG"

step "Preview of the next 7 days"
"$BIN_LINK" plan --dry-run || true
if confirm "Create these study sessions now (through OmaCal)?"; then
  "$BIN_LINK" plan || warn "Planning failed; see the message above. Retry with: study-pomodoro plan"
fi

# ---------------------------------------------------------------- daily timer
step "Daily planning timer"
mkdir -p "$UNIT_DIR"
cp "$SRC/systemd/study-pomodoro-plan.service" "$SRC/systemd/study-pomodoro-plan.timer" "$UNIT_DIR/"
systemctl --user daemon-reload
systemctl --user enable --now study-pomodoro-plan.timer >/dev/null
ok "Plans the next 7 days at 06:30 and 14:00 (OmaCal must be running; retries every 10 min)"

omarchy-shell -q omacal.upcoming refresh

cat <<EOF

Done. Click OmaCal's mark or the 🍅 in the bar for today's events and study sessions.
  study-pomodoro --help            all commands
  ./install.sh --plugin-only       update after editing the plugin
  ./uninstall.sh                   remove it and bring OmaCal's own widget back
EOF
