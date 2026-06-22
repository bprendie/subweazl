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

Spotify is a rent trap. Apple Music is telemetry wrapped in a curated
algorithm.

Subweazl is the exploit. It is a sovereign, terminal-native client built for
the Subsonic API. Need a backend? Spin up Navidrome. It is a pure Go binary that
sips RAM, serves your FLAC hoard without the legacy Java bloat, and Subweazl
jacks straight into it. We bypass the streaming cartel completely.

Cover art gets rendered right in the grid. `mpv` does the heavy lifting for
playback. `ffmpeg` feeds the Harmonica VU meters so you can watch the actual
decoded audio signal bounce on the metal.

No Electron bloat. No subscription fees. No bullshit. Just your music.

## Forge The Binary

You need Go 1.25+, `mpv`, and `ffmpeg`. If you do not have them, shave the yak.

Linux / macOS:

```sh
SUBWEAZL_SKIP_LAUNCH=1 ./scripts/install.sh
```

Windows:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1 -SkipLaunch
```

No wizards. No corporate telemetry. The script compiles the Go binary, drops it
into your path, and gets out of the way.

Want to run it raw?

```sh
go run ./cmd/subweazl
```

## The Connection And The Vault

```sh
subweazl
```

On first boot, Subweazl opens a connection setup screen. Enter your
Navidrome/Subsonic server coordinates and connect. After the server is saved,
Subweazl requires a single private vault for personal play history, queues,
private playlists, synced server cache, and recommendation context. After vault
unlock, Subweazl opens to a home screen for jumping back in or discovering
server music quickly.

We drop the config payload into `~/.config/subweazl/config.json`. We also
cache your last played track in `~/.config/subweazl/state.json` so you boot
right back into the vibe on the next launch.

Want to script it? Inject your credentials via environment variables to override
the file entirely:

```sh
export SUBWEAZL_SERVER="https://navidrome.yourmetal.io"
export SUBWEAZL_USER="your-user"
export SUBWEAZL_PASSWORD="your-password"
```

## Requirements And The Stack

- Go 1.25+
- `mpv` in PATH for playback
- `ffmpeg` in PATH for the visualizer

Subweazl is the interface; `mpv` does the heavy lifting. Cover art gets
rendered directly in the TUI grid.

`ffmpeg` is the meter. Subweazl uses it to quietly decode a mono copy of the
active stream and feed frequency-band energy straight into the Harmonica
terminal bar visualizer, identical to the WeazlTunes treatment.

Pure Go. No CGO yak-shaving, no databases, and no native extension puzzle boxes.
If Go can build Bubble Tea apps on your metal, it can build this.

## Hardware Interrupts

Mouse clicks are dead here. The BBS relies on hotkeys.

The Network:

- `h`: home / jump back in
- `1`: newest albums (Subsonic/Navidrome)
- `2`: playlists (Subsonic/Navidrome)
- `3`: random albums (Subsonic/Navidrome)
- `/`: search the server for tracks

Navigation And Execution:

- `enter`: crack open an album/playlist, or fire the selected track
- `left`: eject to the previous section
- `esc`: kill the search prompt

The Amp:

- `space`: pause/resume the audio
- `s`: kill the playback process
- `r`: forge a saved station playlist from the active track
- `ctrl+r`: rename the selected playlist or current station

The Setup Deck:

- `tab`: cycle input fields
- `enter`: test and save the connection
- `ctrl+s`: force save the payload
- `q` / `ctrl+c`: kill the app entirely

Weaz the juice.
