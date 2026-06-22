# Subweazl

```text
 .________    ___.   __      __          _______________.__   
 |   ____/__ _\_ |__/  \    /  \ ____   /  |  \____    /|  |  
 |____  \|  |  \ __ \   \/\/   // __ \ /   |  |_/     / |  |  
 /       \  |  / \_\ \        /\  ___//    ^   /     /_ |  |__
/______  /____/|___  /\__/\  /  \___  >____   /_______ \|____/
       \/          \/      \/       \/     |__|       \/
SIGNAL // SELF-HOSTED // BARE METAL
```

Subweazl is a terminal-native Subsonic/Navidrome client with a private local
vault. It is built for the daily path: connect to your server, unlock your
vault, jump back into music, search fast, queue tracks, save private playlists,
and generate recommendations from music the app actually knows exists.

There is no local-folder library mode. Subweazl talks to the Subsonic API for
music and uses the local vault only for private app state: play history, queue
snapshots, private playlists, synced server metadata, deterministic recipes, and
optional LLM curator runs.

`mpv` handles playback. `ffmpeg` feeds the Harmonica VU meters. Cover art
renders directly in the TUI.

## Install

Requirements:

- Go 1.25+
- `mpv` in PATH for playback
- `ffmpeg` in PATH for the visualizer

Linux / macOS:

```sh
SUBWEAZL_SKIP_LAUNCH=1 SUBWEAZL_SKIP_LLM_SETUP=1 ./scripts/install.sh
```

Windows:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1 -SkipLaunch
```

Run from source:

```sh
go run ./cmd/subweazl
```

The Linux installer can offer optional LLM setup during an interactive install.
Set `SUBWEAZL_SKIP_LLM_SETUP=1` for automated builds.

## First Run

Start the app:

```sh
subweazl
```

First run is staged:

1. Enter your Subsonic/Navidrome server URL, username, and password.
2. Test and save the connection.
3. Create or unlock the single Subweazl vault.
4. Land on Home, with jump-back-in entries and discovery shortcuts.

Subsonic credentials are stored in `~/.config/subweazl/config.json`. The app
state file is `~/.config/subweazl/state.json`. The encrypted vault lives under
Subweazl's data directory as `vault.sqlite3`.

For scripted runs, use environment variables:

```sh
export SUBWEAZL_SERVER="<server-url>"
export SUBWEAZL_USER="<username>"
export SUBWEAZL_PASSWORD="<password>"
```

## Optional LLM Curator

AI is disabled until you explicitly configure it. Subweazl does not ship with a
provider, model, endpoint, or server default.

Interactive setup:

```sh
subweazl --configure-llm
```

The setup asks for a provider label, base URL, chat completion path, optional
model-list path, model name, context window, and optional API key. Blank provider
label disables AI.

The curator only receives vaulted cache candidates and summary context. It must
return cached track IDs, and Subweazl validates every returned ID before building
a queue. Invented or unknown IDs are rejected. Run metadata is stored encrypted
in the vault.

## Daily Controls

Network and discovery:

- `h`: home / jump back in
- `1`: newest albums
- `2`: server playlists
- `3`: random albums
- `4`: queue
- `5`: private vaulted playlists
- `y`: sync vaulted Subsonic metadata cache
- `g`: generate deterministic queue from vaulted cache
- `G`: curate a queue with the configured optional LLM
- `/`: search cached tracks first, then fall back to server search

Navigation:

- `enter`: open an album or playlist, play a track, jump to a queue row, or load a private playlist
- `left`: go back
- `esc`: cancel search or go back
- `q` / `ctrl+c`: quit

Playback and queue:

- `space`: pause/resume
- `n` / `p`: next/previous queue track
- `a`: enqueue the selected or playing track
- `w`: save the current queue as a private playlist
- `x`: remove the selected queue row
- `delete` / `backspace`: delete the selected private playlist
- `c`: clear the queue
- `u` / `d`: move the selected queue row up/down
- `s`: stop playback
- `r`: create a saved server station playlist from the active track
- `ctrl+r`: rename the selected server playlist, current station, or private playlist

Setup:

- `tab`: cycle input fields
- `enter`: test and save the connection
- `ctrl+s`: force save the connection payload

## Vaulted Features

- Home restores useful private state after unlock.
- Play history is private and vaulted.
- Queue snapshots survive restart.
- Private playlists are local-only and do not mutate the Subsonic server.
- Cache sync is manual, encrypted, and used for fast search and recommendations.
- Deterministic recommendations never require AI.
- Optional LLM recommendations are validated against cached Subsonic track IDs.

Weaz the juice.
