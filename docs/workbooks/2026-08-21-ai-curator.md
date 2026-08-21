# Phase Plan — Interactive AI Curator — 2026-08-21

> **Historical workbook — superseded.** This captures the private, queue-independent `G` design before live testing established immediate launch playback and authoritative-seed grounding. The canonical current contract is [`../../ai_contract.md`](../../ai_contract.md). Do not implement this workbook's “no playback or queue mutation from `G`” requirements.

## Objective

Turn `G` into an intentional, private-playlist creation workflow while preserving the proven immediacy of Mood. Reject generic SaaS recommendation behavior: the user asks Weazl for a record crate, Weazl judges the library, and nothing is published to the server without an explicit command.

## Fixed product boundary

### Mood remains unchanged

- `M` immediately extends the current listening flow from the playing or queued seed.
- It creates or overwrites the Navidrome server playlist `Mood`.
- It contains exactly 20 unique, validated tracks including the seed.
- Playback continues uninterrupted while the playlist streams and repairs itself.
- No chooser or text prompt is added to Mood.

### `G` becomes interactive

Pressing `G` opens a modal chooser:

1. `zero_tax_grindage`
2. `Tell Weazl what you want`
3. `esc` cancels without changing playback, queue, playlists, or navigation.

The chooser uses the same focused row treatment as the LLM provider wizard. Existing global hotkeys do not fire while it is open.

## Mode 1: `zero_tax_grindage`

- Generate exactly 40 unique, validated tracks without a seed.
- Save into the encrypted vault as the private playlist `zero_tax_grindage`.
- Replace an existing case-insensitive same-named private playlist rather than creating duplicates.
- Navigate to Private Playlists (area 5), select it, and show validated tracks arriving in place.
- Do not modify, replace, start, stop, pause, or otherwise touch the active queue or playback.
- Do not create a Navidrome playlist automatically.
- The user may press `v` to copy it explicitly to the server.

## Mode 2: Tell Weazl what you want

1. Select `Tell Weazl what you want`.
2. Focus a single-line prompt input.
3. Enter a request such as `synthwave tracks for focus`.
4. `enter` starts curation; `esc` cancels without side effects.
5. Generate exactly 40 unique, validated tracks matching the request.
6. Save and stream the result into a private vault playlist.
7. Navigate to area 5, select the playlist, and update its open track list in place.
8. Leave playback and the queue untouched.

### Prompt-derived names

- Derive the display name locally; do not spend a second inference request on naming.
- Remove filler words, normalize whitespace, title-case meaningful keywords, and cap the result to a compact TUI-safe length.
- Prefix the result with `AI Mix: `.
- Example: `synthwave tracks for focus` becomes `AI Mix: Synthwave Focus`.
- If no meaningful keywords remain, use `AI Mix: Weazl Cut`.
- A repeated normalized name replaces the same private playlist.

## Curation balance

### Dedicated AI Mix prompt contract

- Mood retains its seed-continuation system prompt; interactive AI mixes use `internal/curator/prompts/ai_mix.md`.
- The user request is the governing eligibility filter for prompted mixes.
- NEW is only a tie-breaker among candidates that already fit the request. It never makes an irrelevant track eligible.
- Runtime user text, mode instructions, repair state, and candidate metadata remain in the user message rather than the system prompt.
- `zero_tax_grindage` receives an explicit discovery-mode instruction; repair rounds receive an explicit shortfall instruction.
- The model judges relevance and sequencing. The application remains responsible for ID validity, exact counts, quotas, diversity, and adjacency.

NEW remains grounded exclusively in Navidrome's top approximately 20 newly uploaded albums.

- NEW is a preference among relevant tracks, not permission to dump whole new albums into the playlist.
- Target at most 60% NEW: no more than 24 tracks in a 40-track playlist.
- Require at least 40% back nine: at least 16 tracks from outside NEW.
- Prompt relevance may reduce the NEW share below 60%; it may never reduce the back-nine floor below 16.
- Enforce the quota in candidate selection and validation. Prompt wording alone is insufficient.
- Repair requests carry the remaining NEW and back-nine requirements explicitly.
- Deterministic completion must honor the same quota.

## Diversity and sequencing contract

Enforce these constraints before accepting a track:

- Maximum 3 tracks per artist.
- Maximum 2 tracks per album.
- At least 8 distinct artists in the completed 40-track playlist.
- Never accept adjacent tracks by the same artist.
- Preserve the LLM's accepted order when it satisfies the constraints.
- Reject over-cap selections and request replacements for the exact shortfall.
- Deterministic completion uses the same artist, album, adjacency, NEW, and back-nine constraints.
- Never publish full-album blocks or candidate-order dumps like the Teal Album smoke test.

The existing short-spacer blacklist remains active before prompt construction: normalized title `silence` with duration ten seconds or less is ineligible. Legitimate full-length songs named "Silence" remain eligible.

## Streaming and repair contract

- Reuse the proven vLLM/Ollama streaming parser and generation-session IDs.
- Keep the Weazl spinner animated throughout candidate preparation, inference, repair, validation, and vault persistence.
- Validate every completed streamed ID against the exact cached candidate snapshot.
- Invalid, invented, stale, duplicate, blacklisted, over-quota, and over-diversity-cap IDs never consume a slot.
- Preserve every accepted valid ID across repair rounds.
- Ask each repair round for exactly the remaining shortfall and remaining class requirements.
- Stop inference immediately when 40 acceptable tracks are secured, even if the model continues emitting candidates.
- After the repair budget, deterministic completion may fill remaining slots only if every contract remains satisfiable.
- A terminal failure leaves the pre-existing private playlist, queue, playback, and navigation intact.

## Live private-playlist behavior

- Do not create an empty private playlist.
- Create or replace the destination after the first validated track arrives.
- Persist validated tracks in small batches so the vault has recoverable progress.
- Update the private-playlist count live in area 5.
- If the playlist is open, append validated tracks without exiting or reopening it.
- Preserve cursor and viewport position during refresh.
- Final reconciliation leaves the user inside the open playlist when applicable.
- Curator progress remains visible even when ordinary list-loading messages complete.

## Server/vault boundary

- AI curator output belongs to the private vault by default.
- `v` copies the selected private playlist to Navidrome, replacing a case-insensitive same-named server playlist.
- `v` on a server playlist copies it into the vault using the same replacement rule.
- Copying never mutates or deletes the source.
- No background or automatic server publication is allowed for `G`.

## Playing-track follow

When the playlist containing the active queue is already open:

- Select the currently playing row using the existing gradient selection treatment.
- Keep the selected row visible by moving the list cursor and viewport on playback transitions.
- `Now` continues showing the playing track in the player/status area.
- Pause retains the row highlight.
- Next and previous move selection to the corresponding queue row.
- Prefer the active queue index over track ID so duplicate tracks resolve correctly.
- Manual browsing may move selection temporarily; the next playback transition resumes following.
- Do not automatically navigate into a playlist merely because one of its tracks starts playing.

## Out of scope

- Do not change shuffle. Existing playback modes already own shuffle behavior.
- Do not modify Mood's product contract.
- Do not add cloud/SaaS recommendation language, telemetry, publishing, sharing, engagement features, or automatic server synchronization.
- Do not expose raw prompts, JSON, IDs, repair diagnostics, or chain-of-thought in the TUI.

## Implementation order

1. Add `G` chooser and prompt-entry modes with cancellation tests.
2. Add deterministic local prompt-to-playlist naming.
3. Separate curator destination behavior: Mood to server; interactive mixes to vault.
4. Add quota-aware candidate windows and acceptance accounting.
5. Add artist, album, distinct-artist, and adjacency validation.
6. Extend repair and deterministic completion to satisfy remaining constraints.
7. Add incremental private-playlist persistence and live area-5 refresh.
8. Ensure `G` never mutates the active queue or playback.
9. Add playing-track follow for open playlists and queue views.
10. Verify symmetric `v` copying after private curator completion.
11. Run full tests and vet; rebuild workspace and installed binaries.
12. Perform live vLLM smoke tests for both `zero_tax_grindage` and a typed request.

## Required tests

- `G` opens the chooser and `esc` is side-effect free.
- Autogeneration always targets private `zero_tax_grindage`.
- Typed prompts derive stable, bounded private-playlist names.
- Repeated normalized names overwrite instead of duplicate.
- A completed mix contains exactly 40 real unique tracks.
- No more than 24 tracks are NEW and at least 16 are back nine.
- No artist exceeds 3 tracks and no album exceeds 2.
- The completed mix contains at least 8 artists.
- Adjacent tracks never share an artist.
- Repair rounds retain valid tracks and request the exact class-aware shortfall.
- Deterministic completion cannot violate quota or diversity rules.
- `[silence]` spacer artifacts remain excluded.
- Private playlist count and an open track list refresh during inference.
- `G` leaves the active queue, playing track, pause state, and player process unchanged.
- `v` copies the finished mix to Navidrome and back without mutating the source.
- Playback transitions select and reveal the matching row in an already-open playlist.
- Superseded generation events cannot mutate the newer playlist.

## Completion gate

The phase is complete when live vLLM runs produce:

- An overwriteable private `zero_tax_grindage` with exactly 40 balanced, diverse tracks.
- A prompt-derived private mix whose content credibly matches the request.
- Live in-place private-playlist growth with an animated spinner.
- No playback or queue mutation from `G`.
- Successful explicit server export through `v`.
- Correct playing-row follow behavior in an open playlist.

## Verification commands

```sh
env GOCACHE=/tmp/subweazl-go-cache GOTMPDIR=/tmp go test ./...
env GOCACHE=/tmp/subweazl-go-cache GOTMPDIR=/tmp go vet ./...
env GOCACHE=/tmp/subweazl-go-cache GOTMPDIR=/tmp go build -buildvcs=false -o ./subweazl ./cmd/subweazl
SUBWEAZL_SKIP_LAUNCH=1 SUBWEAZL_SKIP_LLM_SETUP=1 ./scripts/install.sh
```

## Repository hygiene

- Preserve `screenshot-2026-08-20_17-15-37.png` as an untracked local diagnostic unless explicitly requested for documentation.
- Do not commit credentials, vault contents, generated binaries, or live playlist payloads.
