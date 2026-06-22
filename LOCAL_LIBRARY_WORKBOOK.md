# Subweazl Local Library Workbook

This is the plan for turning Subweazl into a vault-backed local music client
without weakening the Subsonic-first flow.

## Decisions

- Database path: `~/.local/share/subweazl/library.sqlite3`.
- Vault encryption is required for local library use.
- Encrypt everything in the local library database: paths, metadata, playlists,
  ratings, play history, AI summaries, and recommendation runs.
- Use `ffprobe`/`ffmpeg` as the indexing authority. If ffmpeg can read it,
  Subweazl should be able to index it.
- Local and Subsonic tracks can appear in the same hybrid playlist.
- Hybrid rows must be source-coded with Lip Gloss so local and Subsonic tracks
  are visually distinct.
- Local playlists should eventually sync/export to Navidrome.
- AI playlist providers should target vLLM/OpenAI-compatible endpoints first
  and Ollama second.
- The LLM can use encrypted summarized listening history after vault unlock.
- Add a first-class Recently Played view/playlist from play history.

## WeazlChat Vault Pattern

Use `../weazlchat/internal/storage` as the house pattern:

- SQLite store opened from an app-owned path.
- Code-driven migrations.
- `vault` table with one row and a bcrypt password hash.
- AES-GCM encrypted payloads.
- Key derived from the vault password with SHA3-256 plus an app-specific prefix.
- Store starts locked; encrypted reads/writes fail until unlock.
- Honest security language: this protects local privacy and casual prying eyes,
  but it is only as strong as the vault password.

Subweazl should use its own key prefix, for example:

```text
subweazl/local-library/
```

## Proposed Storage Shape

All user/library payload fields should be encrypted. Tables can still keep
opaque IDs, timestamps, and stable hashes in plaintext when needed for joins and
incremental indexing.

Suggested tables:

- `vault`: password hash and vault creation metadata.
- `folders`: encrypted folder path payloads plus scan state.
- `tracks`: encrypted ffprobe metadata payload, stable track ID, file hash,
  modified time, and size.
- `albums`: encrypted album summary payloads.
- `artists`: encrypted artist summary payloads.
- `local_playlists`: encrypted playlist names, descriptions, recipes, and flags.
- `local_playlist_tracks`: playlist ID, track ID, position, source.
- `play_history`: encrypted play event payloads for local and Subsonic tracks.
- `ratings`: encrypted ratings/favorites for local and Subsonic tracks.
- `station_recipes`: encrypted deterministic station generation recipes.
- `recommendation_runs`: encrypted AI/deterministic recommendation inputs,
  summaries, selected candidates, and final playlist outputs.

## Indexing Plan

1. Read configured local folders after vault unlock.
2. Walk files incrementally.
3. Use `ffprobe` to extract metadata for any readable media format.
4. Generate a stable track identity from normalized path, file size, modified
   time, and optionally content hash.
5. Store encrypted metadata payloads.
6. Detect deleted/moved files without destroying history immediately.
7. Keep indexing cancelable and resumable.

The first implementation should favor correctness over fancy watching. File
watchers can come later.

## Local Playback

- Reuse the existing `mpv` player path.
- Reuse the existing `ffmpeg` visualizer path.
- Native cover art should support embedded art later; first pass can use
  metadata/external sidecar art when discovered.
- Play events should write to encrypted `play_history`.

## Recently Played

Add a first-class Recently Played view after play history exists.

Expected behavior:

- Shows recent local and Subsonic tracks together.
- Source-code rows with Lip Gloss.
- `enter` resumes a selected track.
- Useful as a "jump back into the session" path.
- Can seed deterministic and AI recommendations.

## Playlist Model

Local playlists should support:

- Pure local tracks.
- Pure Subsonic tracks.
- Hybrid local/Subsonic tracks.
- Manual ordering.
- Generated station recipes.
- AI-generated metadata explaining why tracks were selected.

Navidrome sync/export should be a later phase:

- Export only tracks that can be mapped to Subsonic IDs.
- Keep unmapped local-only tracks in the local playlist.
- Show sync status and conflicts clearly.

## Recommendation Architecture

Build this as layers:

```text
vaulted library + history -> deterministic candidate generator -> optional LLM curator -> validated playlist
```

The deterministic candidate generator should work without AI:

- same artist / album artist
- same genre
- same year range
- random unseen tracks
- favorites/ratings boost
- skip recently played unless requested
- blend local and Subsonic sources with weights

The LLM curator should never invent tracks. It receives known candidate IDs and
returns selected/reordered IDs plus optional rationale. The app validates every
ID, deduplicates, and enforces playlist length.

## AI Setup Pattern From WeazlChat

WeazlChat’s installer builds the app, then runs a separate setup command:

- Linux/macOS installer:
  - builds `weazlchat`
  - adds `~/.weazlchat/bin` to `PATH`
  - runs `go run -buildvcs=false ./cmd/weazlchat-setup`
- Windows installer:
  - builds `weazlchat.exe`
  - builds `weazlchat-setup.exe`
  - runs the setup executable

The setup command:

- prompts for provider: `vllm` or `ollama`
- uses default base URLs:
  - vLLM: `http://localhost:8000`
  - Ollama: `http://localhost:11434`
- normalizes URLs by trimming `/v1` for vLLM and `/api` for Ollama
- discovers models:
  - vLLM: `GET <base>/v1/models`
  - Ollama: `GET <base>/api/tags`
- falls back to manual model entry if discovery fails
- asks context window:
  - small: 8192
  - medium: 16384
  - large: 32768
  - xl: 128000
- writes provider config

Subweazl should follow this pattern with a future `cmd/subweazl-setup`
or an installer-triggered setup mode.

Subweazl-specific setup fields:

- Recommendation provider: `vllm`, `ollama`, or disabled.
- Base URL.
- Model.
- Context window.
- Whether AI recommendations may use summarized play history after vault
  unlock.
- Default playlist size.
- Local/Subsonic source blend defaults.

## AI Privacy Rules

- AI is disabled until explicitly configured.
- AI runs require an unlocked vault.
- Raw encrypted history should not be sent wholesale.
- The app should generate a compact recommendation summary from decrypted
  history after unlock.
- Store AI run inputs/outputs encrypted.
- Never send credentials.
- Never send raw file paths unless the user explicitly enables that.

## Implementation Phases

1. [done] Add `internal/localstore` skeleton based on WeazlChat storage
   patterns.
2. [done] Add vault create/unlock flow for local library.
3. [done] Add migrations for folders, tracks, playlists, history,
   recommendations.
4. [done] Add local folder indexing with `ffprobe`.
5. [done] Add local library TUI views.
6. [done] Add local TUI indexing and playback through existing player/meter.
7. [done] Add encrypted play history.
8. Add Recently Played view.
9. Add local playlists.
10. Add hybrid playlist rendering and source color coding.
11. Add Navidrome sync/export planning and mapping.
12. Add deterministic recommendation engine.
13. Add `subweazl-setup` AI provider setup patterned after WeazlChat.
14. Add vLLM/OpenAI-compatible recommendation curator.
15. Add Ollama recommendation curator.

## Done Criteria

- Local library cannot be used before vault unlock.
- Every local-library payload is encrypted at rest.
- Existing Subsonic flow still works without touching the vault.
- All source files remain under 400 LOC.
- `go test ./...` passes with project-local caches.
- `/tmp/subweazl` build passes.
- Installed binary is rebuilt before smoke testing large behavior changes.
