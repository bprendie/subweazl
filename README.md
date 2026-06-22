# Subweazl

```text
 .________    ___.   __      __           _______________.__   
 |    ____/__ _\_ |__/  \    /  \ ____   /  |  \____    /|  |  
 |____  \|  |  \ __ \   \/\/   // __ \ /   |  |_/     / |  |  
 /       \  |  / \_\ \        /\  ___//    ^   /     /_ |  |__
/______  /____/|___  /\__/\  /  \___  >____   /_______ \|____/
       \/          \/      \/       \/     |__|       \/
SIGNAL // ENCRYPTED VAULTS // BARE METAL
```

Cloud sync is just telemetry disguised as convenience. Letting a corporation host your playlists means letting them log your vibe, track your hours, and memory-hole your curated tracks whenever licensing rights shift.

Subweazl is the exploit. It’s a sovereign, terminal-native Subsonic client built for the daily path. Connect to your server, unlock your vault, and jump straight into the music. It jacks into the Subsonic API for the raw FLACs, but your curation stays strictly on the bare metal.

There is no local-folder mode. We pull the audio from the server and keep the state in the vault. Play history, queue snapshots, private playlists, cached metadata, and deterministic recipes are locked under paranoid local encryption.

Cover art renders right in the grid. `mpv` does the heavy lifting. `ffmpeg` feeds the live Harmonica VU meters.

No cloud sync. No telemetry. Just your music, locked down tight.

## Forge The Binary

You need Go 1.25+, `mpv`, `ffmpeg`, and a C compiler for the SQLite vault. If you don't have them, shave the yak.

**Linux / macOS:**

```sh
SUBWEAZL_SKIP_LAUNCH=1 SUBWEAZL_SKIP_LLM_SETUP=1 ./scripts/install.sh
```

**Windows (MSYS2 required for C compiler):**

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1 -SkipLaunch
```

No wizards. No corporate installers. The script compiles the Go binary, drops it into your path, and gets out of the way.

Want to run it raw?

```sh
go run ./cmd/subweazl
```

## Boot Sequence & The Vault

```sh
subweazl
```

First boot is staged. You punch in your Navidrome/Subsonic server coordinates, test the connection, and then Subweazl locks the door. You set a bcrypt vault password. The SQLite database is choked down to `0600` permissions.

Your Subsonic credentials drop into `~/.config/subweazl/config.json`. The encrypted vault lives under your data directory as `vault.sqlite3`. Forget the password, and you lose your mixtapes. Back up your metal.

Want to script the connection? Environment variables override the config file entirely:

```sh
export SUBWEAZL_SERVER="<server-url>"
export SUBWEAZL_USER="<username>"
export SUBWEAZL_PASSWORD="<password>"
```

## Tactical AI (The Curator)

AI is a weapon, not a default. Subweazl does not ship with a provider, model, or endpoint. The feature is completely dead until you explicitly arm it.

Jack in your local provider:

```sh
subweazl --configure-llm
```

The setup demands your provider label, base URL, and model details. Blanking the provider disables the AI entirely.

**The Sandbox:** The curator only receives vaulted cache candidates and summary context. It must return cached track IDs. Subweazl validates every returned ID before building the queue. If the model hallucinates an invented or unknown ID, we reject it. Run metadata is stored encrypted in the vault. Zero algorithmic sludge.

## Hardware Interrupts

Mouse clicks are dead here. The BBS relies on hotkeys.

**The Network & Discovery**

* `h`: Home / jump back in
* `1`: Newest albums
* `2`: Server playlists
* `3`: Random albums
* `4`: The Queue
* `5`: Private vaulted playlists
* `y`: Sync vaulted Subsonic metadata cache
* `g`: Forge a deterministic queue from the vaulted cache
* `G`: Command the local LLM to curate a queue
* `/`: Search cached tracks first, fallback to the server

**Navigation & Execution**

* `enter`: Crack open an album, fire a track, jump to a queue row, or load a private playlist
* `left`: Eject to the previous section
* `esc`: Kill the search prompt
* `q` / `ctrl+c`: Kill the app entirely

**The Amp & Queue Desk**

* `space`: Pause/resume the audio
* `s`: Kill the playback process
* `n` / `p`: Next/previous track in the queue
* `a`: Enqueue the selected or active track
* `w`: Forge the current queue into a private vaulted playlist
* `x`: Nuke the selected queue row
* `delete` / `backspace`: Burn the selected private playlist
* `c`: Clear the queue entirely
* `u` / `d`: Move the selected queue row up/down
* `r`: Forge a saved server station playlist from the active track
* `ctrl+r`: Rename the selected server playlist, current station, or private playlist

**The Setup Deck**

* `tab`: Cycle input fields
* `enter`: Test and save the connection
* `ctrl+s`: Force save the connection payload

## The Vaulted State

The local vault is not just a database; it is the entire memory of your session.

* Home restores useful private state the second you unlock.
* Play history stays private and never hits the server.
* Queue snapshots survive a hard restart.
* Private playlists stay local—they do not mutate your Subsonic server.
* Cache sync is manual and encrypted, explicitly used for high-speed local searches and local curation.

Weaz the juice.
