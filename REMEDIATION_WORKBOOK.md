# Subweazl Remediation Workbook

This workbook exists to get Subweazl back onto a coherent daily-driver path.
The current app has strong pieces, but the product surface is split between a
Subsonic client, a local encrypted library, and future recommendation/history
ideas. The remediation goal is to make the main experience as dependable and
direct as WeazlTunes while preserving only the complexity that serves that goal.

## Baseline Direction

Subweazl should be a TUI Subsonic/Navidrome music client. Local file playback
and local folder indexing are out of scope for the remediation cycle.

The "Weazl" ethos is total paranoia: the app should treat personal listening
history, local server-library cache, generated queues, and private playlists as
vaulted data. The vault is not an optional side feature; it is the private
workspace for a Subsonic client.

The daily path should be:

1. Open the app.
2. See useful server music immediately.
3. Search or browse without remembering hidden state.
4. Play, queue, pause, skip, and stop reliably.
5. Trust that private local state is protected in the vault.
6. Trust that no local-file-library concept gets in the way of Subsonic use.

WeazlTunes is the usability reference:

- Fast boot into useful playable content.
- Small set of memorable keys.
- Clear status and error feedback.
- No setup or storage concept unless it is needed for the current task.
- Playback remains the center of gravity.

WeazlWrite is the vault reference:

- Vaults are first-class when they are part of the product.
- Create/unlock/confirm flows are explicit modes.
- Encrypted storage failures stay scoped to encrypted-storage features.

## Current UX Problems

### Setup Asks Too Much

The first-run setup combines Subsonic credentials with local music folders. A
new user should not have to think about local indexing before hearing server
music.

### Local Validation Blocks Remote Setup

Local folder validation currently happens during setup save/test. A bad local
folder can block an otherwise valid Subsonic connection.

### Remote Playback Touches Local Storage

Subsonic playback currently attempts to record play history through the local
encrypted store. Local database or vault problems should not surface during
normal server playback.

### Navigation Has More Concepts Than the Core Use Case Needs

The sidebar/source/local/vault model is heavier than WeazlTunes. Subweazl needs
album/library navigation, but the top-level model should still feel immediate.

### Vault Flow Is Neither Central Nor Invisible

The local vault is hidden behind local-library actions, but local-library
concepts leak into setup and playback. This should be either removed from the
active product or made an explicit advanced mode.

## North Star Engineering Rule

No module/source file may exceed 400 LOC. Split behavior by responsibility
before files become too large.

## Top-Level Decisions Needed

Baseline answers are recorded below. The next step is to break each answer into
sub-decisions and implementation phases.

### 1. Product Scope

Decision: Subsonic/Navidrome only.

Implications:

- Remove local folder setup from first-run.
- Remove local folder browsing/indexing from active UI.
- Remove local file playback from the product surface.
- Keep the vault only for private Subsonic-related state: play history,
  playlists, queue snapshots, cached server metadata, recommendation context,
  and generated playlist state.

### 2. First-Run Experience

Decision: first run asks for server URL, username, password, and initial vault
setup.

Implications:

- Subsonic connection setup and vault setup are both part of onboarding.
- The vault should be single-purpose and single-vault, unlike WeazlWrite's
  contextual multi-vault model.
- The setup flow should make clear that the vault protects private local app
  state, not Subsonic credentials unless explicitly implemented later.
- A failed vault action should be clearly recoverable.

### 3. Default Home Screen

Decision: balance immediate satisfaction with structured discovery.

The app should have a "jump back in" home surface powered by the vault, plus a
sidebar/navigation model for search and discovery.

Implications:

- Home should prioritize resuming the previous session.
- Recently played, last queue, and private playlists should come from the vault.
- Discovery can keep a sidebar concept for search, newest albums, playlists,
  random albums, and server-library browsing.
- The first post-login screen should be useful even before a full server-library
  sync completes.

### 4. Playback Model

Decision: a real queue system is required.

Implications:

- Add queue, next, previous, clear, remove, reorder, and show queue.
- Playing an album or playlist should enqueue enough context for continuous
  playback.
- A queue should be easy to save as a private vaulted playlist.
- Later, a saved private playlist can be synced/exported to Subsonic when the
  server can represent it cleanly.

### 5. Local Vault And History

Decision: use a WeazlWrite-style vault flow, but with exactly one app vault.

Implications:

- Remove local music folders from the vault concept.
- Keep private play history in the vault.
- Keep private playlists in the vault.
- Add a local synced Subsonic metadata cache inspired by Nautiline-style local
  DB sync for fast search and recommendations.
- Allow the LLM playlist/recommendation concept to use the vaulted Subsonic
  cache and vaulted listening history.
- The LLM must never invent tracks; it should select from known cached track
  IDs and the app must validate every result.

## Decision Log

Record final answers here before breaking them into sub-decisions.

1. Product scope: Subsonic/Navidrome only.
2. First-run experience: Subsonic credentials plus initial single-vault setup.
3. Default home screen: jump-back-in home with sidebar discovery.
4. Playback model: real queue required; queues can become private playlists.
5. Local vault and history: WeazlWrite-style single vault for private state,
   synced Subsonic cache, history, private playlists, and LLM recommendation
   context.

## Remediation Backlog Placeholder

Replace this placeholder with the phase plan below as phases are completed.

## Working Method

Each phase should be worked to completion before starting the next one.

Completion loop:

1. Implement the phase.
2. Run automated verification.
3. Build the installed binary.
4. Smoke test the installed binary.
5. Mark the phase complete in this workbook.

Standard verification commands:

```sh
GOCACHE=/home/bobp/Code/subweazl/.gocache GOMODCACHE=/home/bobp/Code/subweazl/.gomodcache go test ./...
GOCACHE=/home/bobp/Code/subweazl/.gocache GOMODCACHE=/home/bobp/Code/subweazl/.gomodcache go build -buildvcs=false -o /tmp/subweazl ./cmd/subweazl
SUBWEAZL_SKIP_LAUNCH=1 ./scripts/install.sh
```

Smoke tests should use the installed `subweazl` binary, not only `go run`.

## Phase Plan

### Phase 0. Baseline And Safety

Status: complete.

Goal: establish the current working baseline before behavioral remediation.

Scope:

- Confirm the renamed `subweazl` binary builds and installs.
- Confirm current config/state paths are under `subweazl`.
- Record known dirty-tree state before remediation begins.
- Do not change product behavior in this phase unless a rename fallout bug is
  discovered.

Exit criteria:

- `go test ./...` passes.
- `/tmp/subweazl` build passes.
- Installed binary builds with `SUBWEAZL_SKIP_LAUNCH=1 ./scripts/install.sh`.
- Smoke test confirms app launches far enough to show setup or main UI.

Completion notes:

- Completed Phase 0 baseline verification.
- Dirty-tree state recorded before remediation work: rename to `subweazl`,
  existing local-library work, and remediation workbook are present.
- `go test ./...` passed with project-local caches.
- `go build -buildvcs=false -o /tmp/subweazl ./cmd/subweazl` passed.
- `SUBWEAZL_SKIP_LAUNCH=1 ./scripts/install.sh` installed
  `/home/bobp/.subweazl/bin/subweazl`.
- Installed binary smoke launch reached the TUI and was terminated by the
  expected 3-second timeout.
- Known warning: `go-sqlite3` emits C compiler const-discard warnings during
  build/test.

### Phase 1. Scope Cleanup: Subsonic Only

Status: complete.

Goal: remove local-file-library behavior from the active product surface while
keeping the vault concept for private Subsonic state.

Scope:

- Remove local music folder fields from first-run setup.
- Remove local folder browsing/indexing from active TUI navigation.
- Remove local file playback from active controls.
- Rename or isolate local-library wording that conflicts with the new product
  direction.
- Ensure normal Subsonic playback does not surface local folder/indexing
  concepts.

Out of scope:

- Full vault redesign.
- Queue implementation.
- Subsonic metadata cache sync.

Exit criteria:

- First-run setup is Subsonic plus vault-oriented, not local-folder-oriented.
- No active UI path presents "local folders" as a primary source.
- README and help text no longer promise local folder playback/indexing.
- Verification/build/install/smoke loop passes.

Completion notes:

- Removed local folder setup from config and first-run UI.
- Removed active local folder browsing, indexing, source switching, and local file
  playback from the TUI.
- Removed the obsolete local-index package and local-library TUI files.
- Removed the obsolete local-library workbook.
- Kept `internal/localstore` for the Phase 2 private vault rebuild.
- Subsonic playback no longer opens or reports errors from local/private storage.
- README and help text now describe Subsonic-only controls.
- `go test ./...` passed with project-local caches.
- `go build -buildvcs=false -o /tmp/subweazl ./cmd/subweazl` passed.
- `SUBWEAZL_SKIP_LAUNCH=1 ./scripts/install.sh` installed
  `/home/bobp/.subweazl/bin/subweazl`.
- Installed binary smoke launch reached the Subsonic setup TUI and was terminated
  by the expected 3-second timeout.
- 400 LOC rule checked; largest Go file is `internal/subsonic/client.go` at 374
  lines.

### Phase 2. Single Vault Onboarding

Status: complete.

Goal: make a single private app vault integral to onboarding, using the
WeazlWrite flow as the reference but without multi-vault selection.

Scope:

- Add or reshape setup into a staged flow:
  - Subsonic connection details.
  - Connection test/save.
  - Create or unlock the single Subweazl vault.
- Store vault state under Subweazl data paths.
- Make vault purpose explicit in UI copy: private play history, queues,
  playlists, server cache, recommendations.
- Keep Subsonic credentials in config for now unless explicitly moved in a
  later phase.
- Ensure vault failure is recoverable without corrupting config.

Out of scope:

- Queue persistence beyond any minimal placeholder needed by the flow.
- LLM provider setup.
- Full server metadata sync.

Exit criteria:

- Fresh install guides user through Subsonic setup and vault create/unlock.
- Existing config with no vault prompts for vault setup.
- Bad vault password gives clear error and retry path.
- App reaches a usable Subsonic home after vault unlock.
- Verification/build/install/smoke loop passes.

Completion notes:

- Added a dedicated `modeVault` onboarding gate after Subsonic config is ready.
- Added single private vault create, confirm, unlock, retry, and quit handling.
- Added dedicated vault view explaining private play history, queues, private
  playlists, synced server cache, and recommendation context.
- Renamed the default private store file to `vault.sqlite3`.
- Updated the vault key prefix to `subweazl/private-vault/`.
- Existing ready config with no vault now opens the vault setup screen before
  loading server content.
- Existing vault opens in unlock mode; wrong password reports a recoverable
  error.
- Added focused TUI tests for vault create/unlock, mismatch retry, existing
  vault unlock, quit cleanup, and vault view rendering.
- `go test ./...` passed with project-local caches.
- `go build -buildvcs=false -o /tmp/subweazl ./cmd/subweazl` passed.
- `SUBWEAZL_SKIP_LAUNCH=1 ./scripts/install.sh` installed
  `/home/bobp/.subweazl/bin/subweazl`.
- Installed binary smoke test used temporary config/data paths, created a vault,
  unlocked with the provided smoke-test password, and loaded 40 newest albums
  from the provided Subsonic server.

### Phase 3. Jump-Back-In Home And Discovery Shell

Status: complete.

Goal: create the balanced daily-driver shell: immediate "jump back in" from
vaulted state plus sidebar discovery for Subsonic browsing.

Scope:

- Add a home mode/surface.
- Show last played, recent play history, last queue, and private playlists when
  available.
- Show useful fallbacks before history exists.
- Keep sidebar/discovery entries for search, newest albums, playlists, random
  albums, and future library cache views.
- Clarify key help so the daily path is visible.

Out of scope:

- Full queue mechanics if Phase 4 has not landed yet.
- LLM recommendations.
- Full metadata sync.

Exit criteria:

- App lands on home after setup/unlock.
- A user can resume something from prior private state when available.
- A new user can still discover and play server music immediately.
- Verification/build/install/smoke loop passes.

Completion notes:

- Added `modeHome` as the post-vault landing surface.
- Added home entries for last played, recent vaulted play history, newest
  albums, playlists, random albums, search, last queue placeholder, and private
  playlist placeholder.
- Added `h` as the home key and sidebar entry.
- Updated back behavior to return to home at top level.
- Added home action handling for resume, discovery shortcuts, search, and honest
  Phase 4/5 placeholders.
- Reintroduced vaulted Subsonic play-history recording now that the private vault
  is part of onboarding.
- Added tests for home fallbacks, last played, recent vaulted history, search
  action, and play-history recording.
- Updated Phase 2 vault tests to expect home after unlock.
- `go test ./...` passed with project-local caches.
- `go build -buildvcs=false -o /tmp/subweazl ./cmd/subweazl` passed.
- `SUBWEAZL_SKIP_LAUNCH=1 ./scripts/install.sh` installed
  `/home/bobp/.subweazl/bin/subweazl`.
- Installed binary smoke test used temporary config/data paths, created and
  unlocked a vault, landed on home, and loaded newest albums from the home
  discovery shortcut.

### Phase 4. Queue Core

Status: complete.

Goal: replace single-track-only behavior with a real queue model suitable for
album, playlist, and search-result playback.

Scope:

- Add in-memory queue model.
- Add queue view.
- Add play now, enqueue, next, previous, remove, clear, and reorder behavior.
- Playing an album or playlist should enqueue context for continuous playback.
- Persist enough queue snapshot data in the vault to support "jump back in".
- Keep files below 400 LOC by splitting queue model, queue commands, queue view,
  and persistence as needed.

Out of scope:

- Queue-to-playlist save.
- LLM queue generation.
- Server playlist sync/export.

Exit criteria:

- Queue survives basic playback workflows.
- Next/previous are reliable.
- Queue state can be restored after app restart when vault is unlocked.
- Verification/build/install/smoke loop passes.

Completion notes:

- Added a focused `internal/playqueue` model for queue current index, next,
  previous, append, remove, clear, reorder, and snapshot restore.
- Added encrypted single-row queue snapshot persistence in the private vault.
- Added queue view on `4` plus queue controls for enqueue, next, previous,
  remove, clear, and move up/down.
- Selecting a song from album, playlist, station, or search results now builds a
  queue context around that list for continuous playback.
- The home `Last queue` action now opens the restored queue view.
- Queue state restores after vault unlock.
- Added queue model, vault persistence, and TUI queue tests.
- `go test ./...` passed with project-local caches.
- `go build -buildvcs=false -o /tmp/subweazl ./cmd/subweazl` passed.
- `SUBWEAZL_SKIP_LAUNCH=1 ./scripts/install.sh` installed
  `/home/bobp/.subweazl/bin/subweazl`.
- Installed-binary smoke test used temporary config/data paths, created a vault,
  loaded newest albums, opened an album, enqueued a track, opened queue view,
  and verified a non-plaintext queue snapshot row in the temp vault.

### Phase 5. Private Playlists And Queue-To-Playlist

Status: complete.

Goal: make queues useful as private vaulted playlists.

Scope:

- Add private playlist storage in the vault.
- Save current queue as private playlist.
- Load private playlist into queue.
- Rename/delete private playlists.
- Clearly distinguish private vaulted playlists from Subsonic server playlists.

Out of scope:

- Server playlist sync/export unless trivial and safe.
- LLM generated playlists.

Exit criteria:

- User can create a queue, save it, restart, unlock vault, and load it again.
- Private playlist operations do not mutate the Subsonic server.
- Verification/build/install/smoke loop passes.

Completion notes:

- Added encrypted private playlist storage using the vault `local_playlists` table.
- Added save-current-queue as private playlist with `w`.
- Added private playlist view on `5` and Home `Private playlists` routing.
- Loading a private playlist replaces the queue and persists the queue snapshot.
- Added private playlist rename with `ctrl+r` and delete with `delete`/`backspace`.
- Private playlist operations are vault-only and do not call Subsonic playlist mutation APIs.
- Added localstore and TUI workflow tests for save, list, load, rename, delete,
  and encrypted payload storage.
- `go test ./...` passed with project-local caches.
- `go build -buildvcs=false -o /tmp/subweazl ./cmd/subweazl` passed.
- `SUBWEAZL_SKIP_LAUNCH=1 ./scripts/install.sh` installed
  `/home/bobp/.subweazl/bin/subweazl`.
- Installed-binary smoke test used temporary config/data paths, created and
  unlocked a vault, and verified Phase 5 private playlist controls were visible.
  The full private playlist save/load/rename/delete workflow is covered by
  automated localstore and TUI tests.

### Phase 6. Subsonic Metadata Cache Sync

Status: pending.

Goal: add a vaulted local Subsonic metadata cache for fast search and future
recommendations, inspired by Nautiline-style local DB sync.

Scope:

- Define cached entities and schema: tracks, albums, artists, playlists, genres,
  cover IDs, starred/favorite flags when available.
- Add manual sync command.
- Consider background startup sync only if it does not slow first interaction.
- Provide cache status in UI.
- Use cache for rapid search where possible, falling back to server search when
  cache is missing or stale.

Out of scope:

- LLM recommendations.
- Offline playback.
- Local file library.

Exit criteria:

- User can trigger a sync and see progress/status.
- Cached search is meaningfully faster after sync.
- Stale or missing cache does not block server browsing/playback.
- Verification/build/install/smoke loop passes.

### Phase 7. Recommendation Foundation

Status: pending.

Goal: build deterministic playlist/queue generation from vaulted cache and
history before involving an LLM.

Scope:

- Generate candidates from known cached track IDs only.
- Support deterministic recipes such as same artist, same genre, nearby year,
  favorites/starred, random unseen, and avoid recently played.
- Produce a queue from a seed track or recipe.
- Store recipe metadata in the vault.

Out of scope:

- LLM provider setup.
- Natural-language playlist prompts.

Exit criteria:

- User can generate a useful queue without LLM.
- Generated queue references only valid cached/server track IDs.
- Verification/build/install/smoke loop passes.

### Phase 8. LLM Curator

Status: pending.

Goal: add optional LLM curation over known candidates without letting the model
invent music.

Scope:

- Add provider setup patterned after WeazlWrite where applicable.
- Make the installer/setup flow ask for the LLM provider once LLM support lands.
- Support OpenAI-compatible/vLLM first, with `https://granite.prendie.io` as
  the known vLLM smoke/default endpoint; keep Ollama second if still desired.
- Summarize vaulted listening history without sending raw secrets or
  credentials.
- Send candidate IDs and metadata to the model.
- Validate returned IDs, dedupe, enforce queue/playlist length, and reject
  invented tracks.
- Store recommendation run inputs/outputs encrypted in the vault.

Out of scope:

- Cloud account management.
- Sending credentials or raw config secrets.

Exit criteria:

- LLM can create a validated queue from known candidates.
- Bad/model-invented IDs are rejected cleanly.
- AI remains disabled until explicitly configured.
- Installer/setup can configure the provider, URL, model, and context window
  using the WeazlWrite-style flow.
- Verification/build/install/smoke loop passes.

### Phase 9. Polish, Docs, And Install

Status: pending.

Goal: make the remediated app coherent as a daily-driver release.

Scope:

- Update README to match actual product behavior.
- Update workbook status.
- Audit help menus and error messages.
- Confirm install scripts use current names and paths.
- Run complete verification and installed-binary smoke test.

Exit criteria:

- README matches app behavior.
- No stale local-file-library promises remain.
- All phases marked accurately.
- Verification/build/install/smoke loop passes.

## Sub-Decision Queue

Break these down next.

### Product Scope Sub-Decisions

- Which local-library packages are deleted versus retained as references?
- Should the SQLite store be renamed from local-library language to vault/cache
  language?
- What migration, if any, should exist from the current experimental local DB?

### Onboarding Sub-Decisions

- Should setup be one screen or a staged flow?
- Should vault password creation happen before or after Subsonic ping succeeds?
- Should config credentials remain in `config.json`, or should credentials move
  into the vault later?

### Home Surface Sub-Decisions

- What exact blocks appear on "jump back in"?
- What is shown before enough vaulted history exists?
- Which keys jump directly to search, queue, playlists, and discovery?

### Queue Sub-Decisions

- What data belongs in the in-memory queue versus the vaulted queue snapshot?
- Should next/previous track history be linear or support branching after manual
  selection?
- What keys should control queue operations?

### Vault And Cache Sub-Decisions

- Which Subsonic entities are cached first: artists, albums, tracks, playlists,
  genres, starred, play counts?
- How does cache sync run: manual key, startup background sync, scheduled, or
  all of the above?
- What remains usable while sync is running or stale?

### LLM Sub-Decisions

- Provider setup baseline: borrow the WeazlWrite flow. Prompt for provider
  (`vllm` first, `ollama` second), base URL, discovered/manual model, and
  context window. Normalize vLLM URLs without `/v1`.
- Default/smoke vLLM endpoint: `https://granite.prendie.io`.
- What minimum deterministic playlist generator exists before LLM curation?
- What private data is allowed into prompts, and how is it summarized?
