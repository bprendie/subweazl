# Subweazl Quality-of-Life Port Plan

> Implementation status (2026-08-24): complete. Theme integration, Omarchy
> agent support, the private remote bridge, persistent widget session, media-key
> routing, installer/uninstaller, documentation, automated tests, and live
> Omarchy validation have all landed in the working tree.

## Goal

Bring the best Omarchy-specific quality-of-life work from Omasub back into
Subweazl without rebranding Subweazl, weakening its curator safeguards, or
coupling the core application to the widget.

This plan covers three outcomes:

1. Make the Subweazl TUI follow the active Omarchy theme, including live theme
   changes, with a safe fallback outside Omarchy.
2. Add `omarchy` as a third curator provider. It resolves and invokes the
   user's current Omarchy default agent while preserving Ollama and vLLM.
3. Ship an Omarchy widget with the proven Omasub session behavior:
   launch or reattach through tmux, keep the unlocked vault and playback alive
   while the terminal is detached, and offer an explicit quit-and-lock action.

The locked Omasub v1.0.0 widget is the implementation reference. It validates
against the installed Omarchy schema and fixes the widget contract at a schema-
v1 manifest, one `BarWidget.qml`, one pure status parser in `Model.js`, and one
tmux/focus launcher in `launch.sh`.

## Guiding Constraints

- Preserve Subweazl branding, config paths, environment variables, application
  identifiers, and existing user data.
- Preserve every Weazl-facing product element: the Subweazl name and banner,
  DJ-Weazl curator identity, vault and playlist language, help copy, keybindings,
  and the application's established personality. Omasub is a QOL reference,
  never a branding or voice source.
- Existing Ollama and vLLM configurations must continue working without a
  migration or surprise provider change.
- Keep the curator's closed-world contract: agents receive bounded candidate
  data, and every returned track ID is validated against the encrypted cache.
- Never expose vault contents, credentials, curator prompts, candidate crates,
  playlists, or queue contents through widget state.
- Do not make Omarchy a hard runtime dependency. The core TUI must remain usable
  on Linux, macOS, and Windows with deterministic fallback colors and the
  existing HTTP providers.
- Never edit packaged files under `/usr/share/omarchy/`. The widget must install
  as a user-owned plugin.
- Port behavior selectively. Do not merge the Omasub fork wholesale; its
  branding changes, file splitting, installer changes, and unrelated remote
  work are outside the immediate core-app scope.

## Source Map

The Omasub implementation already provides useful reference code:

| Capability | Omasub reference | Subweazl integration area |
| --- | --- | --- |
| Semantic theme palette | `internal/tui/theme.go` | New theme adapter in `internal/tui/` |
| Live theme refresh | `internal/tui/theme_refresh.go` | `Model` state and the existing tick path |
| Theme-aware styles | `internal/tui/styles.go`, `gradient.go` | Replace fixed palette use throughout TUI |
| Omarchy agent resolution | `internal/llm/agent.go` | New provider runner in `internal/llm/` |
| Provider dispatch | `internal/llm/client.go` | `Complete` and `StreamComplete` |
| Provider configuration | `internal/config/config.go` | Readiness/default normalization |
| In-app provider picker | `internal/tui/llm_setup.go`, `setup_view.go` | Third provider and no-endpoint flow |
| Runtime agent resolution | `internal/tui/llm_curator_part2.go` | Resolve immediately before each run |
| Remote bridge | `internal/remote/`, `internal/tui/remote.go` | Widget-enabling core work |
| Plugin manifest | `manifest.json` | New Subweazl schema-v1 bar-widget manifest |
| Bar and popup | `widget/omasub/BarWidget.qml`, `Model.js` | Port under `widget/subweazl/` |
| Persistent launch | `widget/omasub/launch.sh` | Port with Subweazl tmux/app identities |
| Media-key routing | `scripts/omasub-media-key`, `omasub-bindings.lua` | Subweazl helpers and marked bindings |
| Install lifecycle | `scripts/install.sh`, `uninstall.sh` | Optional user-plugin install/removal |

## Phase 0: Baseline and Port Boundaries

Before changing behavior:

- Run the current Subweazl test suite, `go vet ./...`, and a clean build.
- Record the current rendering under a fixed terminal size so theme changes can
  be compared without accepting accidental layout changes.
- Inventory direct uses of the fixed colors in `internal/tui`; every use should
  map to a semantic role rather than receive an arbitrary Omarchy color.
- Confirm the current Omarchy color command and default-agent command on the
  supported Omarchy release. Use the public `omarchy ...` CLI route where it
  provides equivalent machine-readable output; otherwise isolate the packaged
  helper behind a tiny adapter so command changes have one repair point.
- Keep the widget directory and installer untouched in this phase.

Exit criteria:

- The baseline is green and recorded.
- Theme, provider, remote bridge, and widget changes have separate ownership
  boundaries and can land in separate commits.

## Phase 1: Omarchy Theme Adapter

### Palette contract

Add a small theme adapter based on Omasub's `theme.go`, but retain Subweazl's
existing palette as the fallback rather than adopting Omasub's generic ANSI
fallback. Define semantic roles for:

- accent/selection
- secondary accent/active border
- success/status
- warning/error
- foreground text
- muted/help text
- panel surface
- panel border
- application canvas
- three gradient stops

Read the active Omarchy color contract and map its named colors to those roles.
Missing keys, malformed output, a missing command, or a non-Omarchy host must
all yield the complete Subweazl fallback palette; partial Omarchy output may
override only valid values.

### Apply the palette consistently

- Convert the fixed color constants in `styles.go` into palette-backed semantic
  values.
- Make the banner, decorative slashes, active track gradient, panels, footer,
  sidebar, help popup, inputs, list delegate, spinner, cover-art frame, and
  visualizer use those semantic roles.
- Keep layout, copy, spacing, borders, and keybindings unchanged.
- Verify readability in both dark and light themes. In particular, avoid
  assuming that `lighter_background` is dark or that foreground is white.

### Live refresh

Use the Omasub pattern of keeping a theme signature in the model and rebuilding
all cached Lip Gloss/Bubbles styles when it changes. Check no more than once per
second from the existing animation/tick path; do not add a second high-frequency
timer. Re-style every stateful component that copied styles at construction:

- main list and its delegate
- text inputs, including vault and setup inputs
- spinner
- cached application styles and gradients

If querying the theme command proves visible in profiles, replace polling with
a cheap signature derived from the active Omarchy theme files while preserving
the same adapter API.

### Theme tests

- Parse a complete Omarchy color response.
- Accept whitespace and ignore unknown keys.
- Fall back for missing, malformed, or invalid colors.
- Verify partial overrides do not zero unrelated roles.
- Verify applying a new palette rebuilds list/input/spinner styles.
- Add representative dark- and light-palette render tests at stable dimensions.
- Confirm theme refresh does not change navigation state, queue state, playback,
  focus, vault state, or curator activity.

Exit criteria:

- Subweazl visually tracks the active Omarchy theme without restart.
- The original Subweazl palette remains the exact non-Omarchy fallback.
- Existing TUI rendering and behavior tests remain green after intentional
  golden-output updates.

## Phase 2: Omarchy Default Agent as a Third Curator Provider

### Configuration behavior

Add `omarchy` beside `ollama` and `vllm` in configuration and the in-app wizard.
For this provider:

- no base URL, model picker, chat path, API key, or context screen is required;
- selecting it saves only `Provider: "omarchy"` plus existing provider-neutral
  limits if any are needed;
- `LLMReady` means the provider is structurally configured, while runtime
  resolution reports whether the current system default is actually usable;
- environment overrides retain the `SUBWEAZL_*` namespace.

Compatibility policy:

- Never rewrite a saved Ollama or vLLM selection.
- Treat Omarchy as a third choice rather than changing existing installations.
- Recommended fresh-install behavior: choose `omarchy` automatically only when
  running on Omarchy and a supported default agent resolves; otherwise leave
  the curator unconfigured and present the normal setup flow. This avoids
  claiming readiness on non-Omarchy hosts. Confirm this product choice before
  implementation.

The picker should display all three choices clearly. Omarchy should be the
first choice on Omarchy hosts; elsewhere it may remain visible with an
actionable availability description, or appear after the HTTP providers.

### Runtime resolution

Resolve the selected Omarchy default agent immediately before every curator
run—not only during setup—so changing the desktop default affects the next
request. Return clear errors for:

- Omarchy command unavailable
- no default agent selected
- selected agent executable unavailable
- selected agent not yet supported by the runner
- authentication or noninteractive execution failure

Record both `omarchy` and the resolved executable/agent name in encrypted run
metadata so failures and outputs remain diagnosable.

### Agent runner

Port Omasub's isolated runner behind the existing LLM client interface:

- create a fresh temporary working directory per request;
- explicitly prohibit tool use, filesystem inspection, and system mutation in
  the prompt;
- use noninteractive, read-only/ephemeral flags appropriate to each supported
  agent;
- capture stdout as the model response and sanitize concise stderr for errors;
- respect caller context cancellation and existing curator deadlines;
- delete the temporary directory after completion;
- never set its working directory to the Subweazl checkout, config directory,
  data directory, or vault location.

Start with Codex because it is the current Omarchy default. Add other dispatcher
agents only after verifying their current CLI flags on the installed versions;
do not blindly freeze the Omasub command table, because agent CLIs change.

### Streaming semantics

Omasub currently implements Omarchy `StreamComplete` by waiting for the whole
agent response and emitting one delta. That is acceptable for an initial port
only if the existing curator progress UI remains honest and cancellation works.
Document it as buffered agent output, not true streaming. A later enhancement
may add per-agent streaming adapters without changing the curator contract.

### Security and validation

Do not fork a second curation path. Omarchy output must enter the same parser,
repair, candidate-ID validation, deduplication, quota, and publication pipeline
as HTTP-provider output. Agent prose, Markdown fences, malformed JSON, invented
IDs, and oversized output must fail or repair exactly as they do today.

### Provider tests

- Resolve the current default on every request.
- Preserve configured Ollama/vLLM selections.
- Save and load an Omarchy-only config.
- Skip endpoint/model screens for Omarchy.
- Run Codex in an isolated temporary directory with the expected safe flags.
- Cover missing command, empty default, missing executable, unsupported agent,
  nonzero exit, empty output, timeout, and cancellation.
- Verify roles and structured-output constraints survive prompt conversion.
- Verify malicious or noisy output cannot bypass the existing track-ID validator.
- Verify buffered `StreamComplete` sends exactly one delta and propagates
  callback errors.

Exit criteria:

- A user can choose Omarchy, Ollama, or vLLM from both configuration paths.
- The next curation uses the then-current Omarchy default agent.
- Codex-backed AI Mix and Mood workflows pass the same validation guarantees as
  the HTTP providers.
- Non-Omarchy platforms build and retain their current workflows.

## Phase 3: Core Remote Bridge for the Future Widget

This phase may begin after the theme/provider work, because it belongs to the
core app and its contract is already clear. It must land independently of any
QML implementation.

Port and adapt the Omasub remote bridge under Subweazl names:

- The running TUI/player remains the single owner of playback, queue, and the
  unlocked vault.
- Publish an atomic `0600` status snapshot under the XDG state directory.
- Listen on a `0600` Unix socket under the XDG runtime directory, with a safe
  Windows no-op/unsupported transport that preserves builds.
- Expose only: running/playing/paused/idle, title, artist, album, duration,
  playback mode, and update timestamp. Add progress only if the player exposes
  it cheaply and accurately.
- Accept only: status, play/pause, previous, next, stop, cycle mode, and explicit
  quit-and-lock.
- Reject a second active Subweazl instance rather than create two owners.
- Detect stale snapshots and stale sockets deterministically.
- Route commands into Bubble Tea messages so state mutation stays on the TUI
  update loop.

Vault/session semantics are non-negotiable:

- Detaching or closing the terminal must leave the tmux-hosted Subweazl process,
  playback, queue, and unlocked vault alive.
- Reopening from the widget must attach/focus that same session, returning the
  user directly to the already-open vault state.
- `stop` stops playback only; it does not lock the vault or end the session.
- `quit-and-lock` stops playback, closes the vault store/clears its in-memory
  key, exits Bubble Tea, closes the socket, writes a stopped snapshot, and ends
  the persistent session.
- Normal in-app `q` behavior should remain explicit and consistent with the
  security model; test whether it ends the listening session and locks the
  vault, while terminal-window close merely detaches.

Remote bridge tests should cover permissions, atomic writes, stale state,
single-instance enforcement, command validation, concurrent reads, cleanup,
playback-mode publication, and vault closure on remote quit.

Exit criteria:

- `subweazl remote status` reports the live process safely.
- Remote commands control the existing process.
- Detach/reattach preserves the open vault; explicit quit closes it.
- The bridge can be tested completely without installing a widget.

## Phase 4: Omarchy Widget Integration

Port the locked Omasub v1.0.0 plugin as a narrow Subweazl-specific adaptation.
Preserve its component boundaries and interactions; change product identifiers,
command names, labels, paths, and application identity without redesigning it.

### Repository layout and manifest

Add:

```text
manifest.json
widget/subweazl/BarWidget.qml
widget/subweazl/Model.js
widget/subweazl/launch.sh
scripts/subweazl-media-key
scripts/subweazl-bindings.lua
scripts/uninstall.sh
```

Use a schema-v1 `bar-widget` manifest modeled directly on Omasub:

- stable plugin id: `io.github.bprendie.subweazl`
- entry point: `widget/subweazl/BarWidget.qml`
- display name: `Subweazl`
- category: `Media`
- `allowMultiple: false`
- `defaultSection: right`
- MIT metadata and a description limited to playback status/remote controls

Set the QML `moduleName` to the same plugin id. Keep the QML, JavaScript, and
launcher files under 300 lines and validate the repository/plugin directory
with `omarchy plugin validate` before installation.

### Status model

Port Omasub's pure `Model.js` shape exactly under Subweazl names:

- `emptyStatus()` returns a stopped, inactive, off-mode record, matching
  Subweazl's established playback terminology;
- `parseStatus()` accepts only a JSON object and normalizes every field;
- malformed, empty, or non-object output falls back without throwing into the
  shell;
- `stateLabel()` distinguishes not running, paused, playing, and idle;
- `trackLabel()` formats title and artist for the bar tooltip.

The model must consume only the Phase 3 public snapshot fields. Do not add
vault state, server details, queue contents, playlist data, or credentials.

### Bar behavior

Port the locked `BarWidget.qml` interaction model:

- poll `subweazl remote status` every second through a non-overlapping
  Quickshell `Process`;
- reset to `emptyStatus()` when the status command exits unsuccessfully;
- show a dim inactive music glyph when Subweazl is absent;
- animate glyph opacity only while playing;
- show the current title on a horizontal bar, capped and elided; collapse it on
  a vertical bar;
- left-click inactive state launches Subweazl;
- left-click active state toggles the popup;
- middle-click active state advances to the next track;
- mouse wheel sends previous/next while active;
- use the normal Omarchy tooltip service with the normalized track label.

Use only Omarchy shell theme and layout primitives (`barForeground`,
`foreground`, `Color.accent`, `Style`, `Border`, `BorderSurface`, `PopupCard`,
and `Button`). Retain the Nerd Font music glyph convention used by the locked
widget. Do not hardcode app-theme colors in QML.

### Popup behavior

Keep the locked popup intentionally small rather than turning it into a second
music client:

- hero glyph plus title, artist/status, and album;
- previous, play/pause, next, and stop buttons;
- Open Subweazl / Launch Subweazl action;
- one label-changing mode button using the snapshot's playback-mode text;
- separate Quit and lock vault action;
- disabled/dim controls when no live Subweazl process exists;
- close the popup before launching/focusing or quitting.

Remote actions must be serialized through one action process so repeated clicks
cannot create a command storm. Refresh status immediately after each action
exits rather than waiting for the next timer tick.

### Tmux launcher and focus behavior

Port the locked launcher with these Subweazl identities:

- tmux session: `subweazl`
- executable: `subweazl`
- terminal app id: `org.omarchy.subweazl`
- installed launcher path derived from `XDG_CONFIG_HOME`, falling back to
  `~/.config/omarchy/plugins/io.github.bprendie.subweazl/`

Unset `NO_COLOR` before starting the session so the themed TUI is not flattened
by the shell process environment.

The launch algorithm is fixed:

1. If the tmux session has an attached client and remote status is live, walk
   its client PIDs, resolve each terminal parent PID, find the matching Hyprland
   client address with `hyprctl clients -j`/`jq`, and focus it.
2. If a session exists but is detached, run `tmux attach-session` inside
   `omarchy-launch-tui`.
3. If no session exists, run `tmux new-session -s subweazl env -u NO_COLOR
   subweazl` inside `omarchy-launch-tui`.
4. Poll briefly for the new tmux client and focus its terminal; fail clearly if
   no client appears within the bounded retry window.

Never start a second Subweazl process when the named listening session exists.
Closing the terminal window must detach the tmux client, leaving playback,
queue, and the unlocked vault inside the original process. Open Subweazl must
reattach to that exact process.

### Media keys

Port the locked media router and bindings under Subweazl names:

- `subweazl-media-key` accepts only `toggle`, `next`, or `previous`;
- if `subweazl remote status` succeeds, send the action to Subweazl;
- otherwise fall back to `omarchy-shell media playPause|next|previous` so the
  user's selected Omarchy media source still works;
- install bindings for Play, Pause, Next, Previous, Alt+Play as Next, and
  Alt+Shift+Play as Previous;
- allow the bindings while locked, matching the locked reference widget;
- wrap all binding edits in unique begin/end markers for idempotent removal.

Because bindings modify user Hyprland configuration, the installer must back
up changed files, reload with `hyprctl reload`, and require an empty
`hyprctl configerrors` result. Installation must abort/report if validation
fails rather than leave a silently broken input configuration.

### Installer and uninstaller

Extend the existing installer without discarding its current
`SUBWEAZL_HOME=~/.subweazl` convention:

- add `--no-widget` and matching `SUBWEAZL_SKIP_WIDGET` support;
- on Omarchy, ensure `tmux` and `jq` exist (using the normal Omarchy package
  command when dependency installation is enabled);
- install the app and media helper into `$SUBWEAZL_HOME/bin`;
- install manifest/QML/JS/launcher files under
  `${XDG_CONFIG_HOME:-~/.config}/omarchy/plugins/io.github.bprendie.subweazl`;
- use modes `0644` for manifest/QML/JS and `0755` for shell helpers;
- run `omarchy plugin validate` before activation;
- rescan plugins, then enable the widget before `omarchy.audio`;
- skip all shell/plugin/media-key work cleanly when Omarchy is absent;
- keep installation idempotent and avoid touching `/usr/share/omarchy/`.

Add a safe uninstaller modeled on Omasub that:

- removes/disables the plugin through `omarchy plugin remove ... --yes` when
  available;
- removes only marked Subweazl PATH and media-binding blocks;
- removes only installed Subweazl binaries/helpers;
- reloads Hyprland and checks config errors;
- preserves configuration, encrypted vault, cache, queue, private playlists,
  and listening history;
- preserves the plugin directory with a warning if Omarchy is unavailable
  rather than deleting unregistered state blindly.

### Widget tests and live smoke test

- Validate the source tree and installed plugin with `omarchy plugin validate`.
- Unit-test the JavaScript parser if the plugin test harness supports it;
  otherwise exercise valid, malformed, empty, and stale CLI responses through
  a disposable installed plugin.
- Stub `subweazl`, `tmux`, `hyprctl`, `jq`, and `omarchy-launch-tui` to test all
  launcher branches without opening windows.
- Verify inactive launch, attached focus, detached reattach, and missing-command
  failures.
- Verify every popup and mouse action maps to exactly one remote command.
- Verify a remote command failure does not crash or wedge Omarchy shell.
- Verify dark/light theme changes update the widget through native shell colors.
- Verify one-second polling does not overlap or produce persistent shell logs.
- Install into a disposable XDG config home, validate permissions and paths,
  then uninstall and prove user data remains.
- Finish with a live test: launch, unlock vault, play, close terminal, control
  from widget/media keys, reopen into the unlocked vault, stop without locking,
  then Quit and lock vault and confirm the tmux session/socket are gone.

Exit criteria:

- The widget validates, installs, enables, hot-reloads, and uninstalls cleanly.
- It reflects and controls the one live Subweazl process promptly.
- The locked Omasub interaction model is preserved under Subweazl branding.
- Detach/reattach preserves the open vault; explicit quit ends the session and
  clears the vault key.
- Media keys prefer live Subweazl and otherwise preserve Omarchy media control.

## Recommended Delivery Order

1. Baseline and theme adapter.
2. Complete semantic recoloring and live refresh.
3. Omarchy provider configuration and runtime resolution.
4. Codex runner and curator integration.
5. Core remote bridge and tmux/vault behavioral tests.
6. Port the locked manifest, status model, bar/popup, and tmux launcher.
7. Add idempotent widget/media-key install and safe uninstall flows.
8. Validate and run the full live widget/session/vault smoke test.

Each numbered delivery should be independently reviewable, buildable, and
reversible. Do not combine theme, provider, remote bridge, and widget changes in
one large merge.

## Final Acceptance Checklist

- `go test ./...`
- `go vet ./...`
- clean builds on supported platforms
- Subweazl fallback visuals unchanged outside Omarchy
- live dark/light Omarchy theme switching without restart
- existing Ollama and vLLM configs and curator flows unchanged
- Omarchy/Codex curator flow passes malformed-output and cancellation tests
- no credentials or vault material in prompts, temp working directories, state
  snapshots, logs, or remote responses
- remote detach/reattach keeps playback, queue, and the unlocked vault alive
- explicit quit closes the vault and listening session
- source and installed widgets pass `omarchy plugin validate`
- inactive launch, active focus, detached reattach, and quit-and-lock pass live
- hardware media keys prefer live Subweazl and fall back to Omarchy media
- uninstall removes integration while preserving all Subweazl user data
