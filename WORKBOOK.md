# Subweazl Workbook

This is the working checklist for the next rounds of Subweazl development.
Keep credentials out of this file and out of commits.

## Development Rules

- Keep every source file under 400 LOC.
- Preserve the WeazlTunes visual treatment: Lip Gloss panels, gradient ASCII
  logo, Harmonica VU meters, and the installer style.
- Prefer small focused packages over a large TUI file.
- After each successful feature step, rebuild the installed binary for testing:

```sh
SUBWEAZL_SKIP_LAUNCH=1 ./scripts/install.sh
```

- Also run the explicit verification build when useful:

```sh
GOCACHE=/home/bobp/Code/subweazl/.gocache GOMODCACHE=/home/bobp/Code/subweazl/.gomodcache go test ./...
GOCACHE=/home/bobp/Code/subweazl/.gocache GOMODCACHE=/home/bobp/Code/subweazl/.gomodcache go build -buildvcs=false -o /tmp/subweazl ./cmd/subweazl
```

## Live Test Server

Use the user-provided Subsonic/Navidrome test server only through environment
variables or the local config file. Do not commit the username or password.

```sh
export SUBWEAZL_SERVER="https://weazltunes.prendie.io"
export SUBWEAZL_USER="<provided-user>"
export SUBWEAZL_PASSWORD="<provided-password>"
```

Station creation creates a real playlist on the server, so ask before running
that smoke test.

## Product Direction

Keep Subweazl separate from WeazlTunes.

- WeazlTunes remains a terminal radio tuner for SomaFM, Icecast/Xiph, pasted
  stream URLs, and eight preset slots.
- Subweazl remains a terminal music-library client for Subsonic/Navidrome,
  generated station playlists, album art, and future local folder playback.
- Share visual language and, later, possibly small helper packages. Do not merge
  the apps unless there is a stronger product reason.

## Next Feature Sequence

### 1. Resume Last Played Item

Status: done.

Goal: Subweazl should open focused on the last played item.

- Persist last playback state locally.
- Store only non-secret playback metadata: track ID, title, artist, album,
  parent album or playlist context when available, and last source view.
- On launch, restore enough context to make the previous track obvious.
- If the track cannot be resolved, fall back to newest albums with a harmless
  status message.

### 2. Connection Setup UI

Status: done.

Goal: first run should make connection choices obvious.

- Add a setup view before the main browser when no usable source is configured.
- Provide a Subsonic/Navidrome form: server URL, username, password, and test
  connection action.
- Provide a local music folder section for one or more folders, even if the
  first implementation only stores them for later indexing work.
- Clearly show the active source in the main UI after setup.
- Do not print or persist secrets outside the existing config path.

### 3. Local Folder Foundation

Status: foundation done; indexing/playback remains future work.

Goal: prepare local music support without turning Subweazl into a large
indexing rewrite.

- Add config support for local music folders.
- Validate folders exist and are readable.
- Keep the first pass focused on configuration and UI plumbing.
- Add indexing/playback in a later feature step.

### 4. Split Large TUI Update File

Status: done; command helpers moved out of `internal/tui/update.go`.

Goal: keep `internal/tui/update.go` below the 400 LOC rule while new setup and
resume work is added.

- Move setup-view update logic into its own file.
- Move station or playlist mutation update helpers if needed.
- Keep behavior unchanged during the split.

### 5. Station Quality

Goal: improve generated station playlists after the setup/resume foundations.

Status: foundation done; live playlist-creation smoke test still requires
explicit approval.

- Prefer metadata-aware candidates if Navidrome exposes genre/year/artist data.
- Keep generated playlists saved immediately.
- Ask before running a live smoke test that creates a real playlist.

### 6. Media Key Support

Goal: let desktop media keys control Subweazl when it is running.

Status: planned.

- Support play/pause, stop, next, and previous where the terminal/desktop stack
  exposes those events.
- Prefer Bubble Tea key handling when media keys arrive as terminal input.
- If terminal media keys are not reliable under Hyprland/Omarchy, add a small
  documented command surface that can be bound externally, for example
  `subweazl --control play-pause`.
- Keep control commands separate from credentials and config mutation.
- Verify behavior with the installed binary after implementation.

### 7. SubTUI-Style Navigation Polish

Goal: make section movement and the playback surface feel more like a focused
music client.

Status: done.

- Left arrow backs out to the previous section.
- The main view shows source badges for streaming Subsonic and local folders.
- The now-playing surface is a bottom play bar with honest cover availability
  status; actual TCT rendering still belongs to mpv, not the Bubble Tea bar.
- Subsonic and Local now have separate top-level source sections instead of
  sharing one search/navigation surface.

### 8. Native Album Art

Goal: render Subsonic cover art inside the Bubble Tea UI instead of relying on
mpv `--vo=tct` output.

Status: done; first pass uses a built-in ANSI half-block renderer.

- Fetch cover art through the authenticated Subsonic client.
- Decode cover images in Go and cache them by cover ID.
- Render the current playing track's artwork in the TUI footer or media panel.
- Keep mpv focused on playback; do not let mpv write TCT output into the Bubble
  Tea screen.
- Preserve graceful fallback for tracks without cover art or failed image fetch.

## Done Criteria For Each Step

- Code stays under the 400 LOC file limit.
- `go test ./...` passes with project-local caches.
- `/tmp/subweazl` build passes when run.
- Installed binary is rebuilt with `SUBWEAZL_SKIP_LAUNCH=1 ./scripts/install.sh`.
- README is updated when user-facing behavior or keys change.
