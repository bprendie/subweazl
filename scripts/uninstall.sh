#!/usr/bin/env bash

set -euo pipefail

PLUGIN_ID="io.github.bprendie.subweazl"
INSTALL_ROOT="${SUBWEAZL_HOME:-"$HOME/.subweazl"}"
CONFIG_ROOT="${XDG_CONFIG_HOME:-"$HOME/.config"}"
BIN_DIR="$INSTALL_ROOT/bin"

profile_path() {
  case "$(basename "${SHELL:-bash}")" in
    zsh) echo "$HOME/.zshrc" ;;
    fish) echo "$CONFIG_ROOT/fish/config.fish" ;;
    bash) [[ -f $HOME/.bashrc ]] && echo "$HOME/.bashrc" || echo "$HOME/.profile" ;;
    *) echo "$HOME/.profile" ;;
  esac
}

remove_marked_block() {
  local target="$1" begin="$2" end="$3" temp
  [[ -f $target ]] || return
  temp="$(mktemp "${target}.subweazl.XXXXXX")"
  awk -v begin="$begin" -v end="$end" '
    $0 == begin { skipping=1; next }
    $0 == end { skipping=0; next }
    !skipping { print }
  ' "$target" >"$temp"
  if cmp -s "$target" "$temp"; then
    rm -f "$temp"
    return
  fi
  cp -p "$target" "${target}.bak.$(date +%s)"
  mv "$temp" "$target"
}

remove_plugin() {
  local plugin_dir="$CONFIG_ROOT/omarchy/plugins/$PLUGIN_ID"
  [[ -d $plugin_dir ]] || return
  if command -v omarchy >/dev/null 2>&1; then
    omarchy plugin remove "$PLUGIN_ID" --yes
    [[ ! -d $plugin_dir ]] || rm -rf -- "$plugin_dir"
  else
    echo "Omarchy is unavailable; preserving plugin at $plugin_dir" >&2
  fi
}

remove_plugin
remove_marked_block "$(profile_path)" "# >>> subweazl path >>>" "# <<< subweazl path <<<"
remove_marked_block "$CONFIG_ROOT/hypr/bindings.lua" \
  "-- >>> subweazl media keys >>>" "-- <<< subweazl media keys <<<"
rm -f "$BIN_DIR/subweazl" "$BIN_DIR/subweazl-media-key"

if command -v hyprctl >/dev/null 2>&1 && hyprctl status >/dev/null 2>&1; then
  hyprctl reload >/dev/null
  errors="$(hyprctl configerrors)"
  [[ -z $errors ]] || { echo "$errors" >&2; exit 1; }
fi

echo "Removed the Subweazl application, widget, PATH entry, and media bindings."
echo "Your configuration, encrypted Weazl vault, cache, queue, playlists, and history were preserved."
