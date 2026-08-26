#!/usr/bin/env bash

set -euo pipefail

APP_NAME="subweazl"
PLUGIN_ID="io.github.bprendie.subweazl"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_ROOT="${SUBWEAZL_HOME:-"$HOME/.subweazl"}"
CONFIG_ROOT="${XDG_CONFIG_HOME:-"$HOME/.config"}"
CACHE_ROOT="${XDG_CACHE_HOME:-"$REPO_ROOT/.gocache"}"
MOD_CACHE="${GOMODCACHE:-"$REPO_ROOT/.gomodcache"}"
BIN_DIR="$INSTALL_ROOT/bin"
BIN_PATH="$BIN_DIR/$APP_NAME"
PLUGIN_DIR="$CONFIG_ROOT/omarchy/plugins/$PLUGIN_ID"
PLUGIN_SOURCE_DIR="$REPO_ROOT/widget/subweazl"
SKIP_LAUNCH="${SUBWEAZL_SKIP_LAUNCH:-0}"
SKIP_LLM="${SUBWEAZL_SKIP_LLM_SETUP:-0}"
SKIP_WIDGET="${SUBWEAZL_SKIP_WIDGET:-0}"
SKIP_ACTIVATE="${SUBWEAZL_SKIP_ACTIVATE:-0}"

usage() {
  cat <<'EOF'
Usage: ./scripts/install.sh [options]

Options:
  --no-launch       Install without launching Subweazl
  --no-llm-setup    Skip the optional curator setup prompt
  --no-widget       Skip Omarchy widget and media-key integration
  -h, --help        Show this help
EOF
}

while (( $# > 0 )); do
  case "$1" in
    --no-launch) SKIP_LAUNCH=1 ;;
    --no-llm-setup) SKIP_LLM=1 ;;
    --no-widget) SKIP_WIDGET=1 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done

go_version_number() {
  go version | awk '{print $3}' | sed 's/^go//' | cut -d. -f1,2
}

version_at_least() {
  local current_major current_minor required_major required_minor
  current_major="${1%%.*}"
  current_minor="${1#*.}"
  required_major="${2%%.*}"
  required_minor="${2#*.}"
  [[ $current_major =~ ^[0-9]+$ && $current_minor =~ ^[0-9]+$ ]] || return 1
  [[ $required_major =~ ^[0-9]+$ && $required_minor =~ ^[0-9]+$ ]] || return 1
  (( current_major > required_major || (current_major == required_major && current_minor >= required_minor) ))
}

install_dependencies() {
  local -a packages=()
  command -v go >/dev/null 2>&1 || packages+=(go)
  command -v mpv >/dev/null 2>&1 || packages+=(mpv)
  command -v cc >/dev/null 2>&1 || packages+=(gcc)
  if [[ $SKIP_WIDGET != 1 ]] && command -v omarchy >/dev/null 2>&1; then
    command -v tmux >/dev/null 2>&1 || packages+=(tmux)
    command -v jq >/dev/null 2>&1 || packages+=(jq)
  fi
  (( ${#packages[@]} == 0 )) && return
  if command -v omarchy >/dev/null 2>&1 && [[ ${SUBWEAZL_INSTALL_DEPS:-1} == 1 ]]; then
    echo "Installing dependencies: ${packages[*]}"
    omarchy pkg add "${packages[@]}"
    return
  fi
  echo "Missing dependencies: ${packages[*]}" >&2
  echo "Install them and rerun this installer." >&2
  exit 1
}

check_go_version() {
  local required current
  required="$(awk '/^go / {print $2; exit}' "$REPO_ROOT/go.mod" | cut -d. -f1,2)"
  current="$(go_version_number)"
  if ! version_at_least "$current" "$required"; then
    echo "Go $required or newer is required. Found Go $current." >&2
    exit 1
  fi
}

profile_path() {
  case "$(basename "${SHELL:-bash}")" in
    zsh) echo "$HOME/.zshrc" ;;
    fish) echo "$CONFIG_ROOT/fish/config.fish" ;;
    bash) [[ -f $HOME/.bashrc ]] && echo "$HOME/.bashrc" || echo "$HOME/.profile" ;;
    *) echo "$HOME/.profile" ;;
  esac
}

replace_marked_block() {
  local target="$1" begin="$2" end="$3" content="$4" temp
  mkdir -p "$(dirname "$target")"
  touch "$target"
  temp="$(mktemp "${target}.subweazl.XXXXXX")"
  awk -v begin="$begin" -v end="$end" '
    $0 == begin { skipping=1; next }
    $0 == end { skipping=0; next }
    !skipping { print }
  ' "$target" >"$temp"
  printf '%s\n%s\n%s\n' "$begin" "$content" "$end" >>"$temp"
  if cmp -s "$target" "$temp"; then
    rm -f "$temp"
    return
  fi
  [[ -s $target ]] && cp -p "$target" "${target}.bak.$(date +%s)"
  mv "$temp" "$target"
}

add_to_path() {
  local profile shell_name line
  profile="$(profile_path)"
  shell_name="$(basename "${SHELL:-bash}")"
  if [[ $shell_name == fish ]]; then
    line="fish_add_path \"$BIN_DIR\""
  else
    line="export PATH=\"$BIN_DIR:\$PATH\""
  fi
  replace_marked_block "$profile" "# >>> subweazl path >>>" "# <<< subweazl path <<<" "$line"
  echo "Added $BIN_DIR to PATH in $profile"
}

build_application() {
  local temp_binary
  mkdir -p "$BIN_DIR" "$CACHE_ROOT" "$MOD_CACHE"
  temp_binary="$(mktemp "$BIN_DIR/.subweazl.XXXXXX")"
  trap 'rm -f "${temp_binary:-}"' EXIT
  echo "Building Subweazl..."
  (
    cd "$REPO_ROOT"
    GOCACHE="$CACHE_ROOT" GOMODCACHE="$MOD_CACHE" \
      go build -buildvcs=false -trimpath -o "$temp_binary" ./cmd/subweazl
  )
  chmod 0755 "$temp_binary"
  mv "$temp_binary" "$BIN_PATH"
  trap - EXIT
  install -m755 "$REPO_ROOT/scripts/subweazl-media-key" "$BIN_DIR/subweazl-media-key"
}

install_media_bindings() {
  local bindings="$CONFIG_ROOT/hypr/bindings.lua" content
  content="$(<"$REPO_ROOT/scripts/subweazl-bindings.lua")"
  content="${content//\$HOME\/.subweazl\/bin\/subweazl-media-key/$BIN_DIR\/subweazl-media-key}"
  replace_marked_block "$bindings" "-- >>> subweazl media keys >>>" "-- <<< subweazl media keys <<<" "$content"
  if [[ $SKIP_ACTIVATE != 1 ]] && hyprctl status >/dev/null 2>&1; then
    hyprctl reload >/dev/null
    local errors
    errors="$(hyprctl configerrors)"
    if [[ -n $errors ]]; then
      echo "$errors" >&2
      return 1
    fi
  fi
}

install_widget() {
  [[ $SKIP_WIDGET == 1 ]] && return
  if ! command -v omarchy >/dev/null 2>&1; then
    echo "Omarchy not found; skipping widget installation."
    return
  fi
  mkdir -p "$PLUGIN_DIR/widget/subweazl"
  install -m644 "$REPO_ROOT/manifest.json" "$PLUGIN_DIR/manifest.json"
  rm -f -- "$PLUGIN_DIR/widget/subweazl/Model.js"
  install -m644 "$PLUGIN_SOURCE_DIR/BarWidget.qml" "$PLUGIN_DIR/widget/subweazl/BarWidget.qml"
  install -m755 "$PLUGIN_SOURCE_DIR/launch.sh" "$PLUGIN_DIR/widget/subweazl/launch.sh"
  omarchy plugin validate "$PLUGIN_DIR"
  install_media_bindings
  if [[ $SKIP_ACTIVATE != 1 ]]; then
    omarchy-shell shell rescanPlugins
    omarchy plugin enable "$PLUGIN_ID" --before omarchy.audio
    omarchy bar set "$PLUGIN_ID" binaryPath "$BIN_PATH"
  fi
  echo "Installed Omarchy widget to $PLUGIN_DIR"
}

configure_llm() {
  [[ $SKIP_LLM == 1 || ! -t 0 ]] && return
  echo
  read -r -p "Configure the optional DJ-Weazl provider now? [y/N] " answer
  [[ $answer =~ ^([yY]|yes|YES)$ ]] && "$BIN_PATH" --configure-llm
}

install_dependencies
check_go_version
build_application
add_to_path
install_widget
configure_llm

echo "Installed Subweazl to $BIN_PATH"
if [[ $SKIP_LAUNCH == 1 ]]; then
  echo "Installation complete. Run: $BIN_PATH"
else
  echo "Launching Subweazl..."
  exec "$BIN_PATH"
fi
