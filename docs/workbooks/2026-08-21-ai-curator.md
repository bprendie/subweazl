# Subweazl Workbook — 2026-08-21

## Objective

Finish and validate the general AI curator (`G`) now that the streaming Mood (`M`) workflow is proven in production.

## Starting point

- `M` creates or overwrites the Navidrome server playlist `Mood` with 20 unique tracks.
- `G` creates or overwrites the Navidrome server playlist `AI Mix` with 40 unique tracks.
- Both modes stream opaque IDs from vLLM, validate them against the synced closed-world library, repair shortfalls, and cancel excess output after reaching their target.
- Current playback continues throughout inference. Existing upcoming tracks remain as a safety buffer until curated tracks are ready.
- Server playlists grow in validated batches and an open curated playlist refreshes in place without losing its cursor.
- The curator spinner remains animated even when an ordinary playlist load completes during inference.
- `v` copies a selected playlist between Navidrome (area 4) and the encrypted vault (area 5), replacing a case-insensitive same-named destination.
- Short spacer artifacts with normalized title `silence` and duration of ten seconds or less are excluded before prompt construction. Full-length songs named "Silence" remain eligible.

## Proven live behavior

- A Helena Mood run created the real server playlist immediately and grew it to 20 tracks.
- Re-running Mood overwrote the same Navidrome playlist ID.
- Final verification showed 20 unique real IDs with Helena first.
- Long-running inference no longer depends on a 400-token completion cap; streamed parsing stops after enough validated IDs arrive.

## Tomorrow's live AI Mix test

1. Restart Subweazl so the installed binary includes the latest changes.
2. Keep a track playing and note the current queue/current track.
3. Press `G` once.
4. Confirm the spinner remains animated and reports progress toward `40/40`.
5. Confirm `AI Mix` appears in Navidrome area 4 after its first validated track.
6. Open `AI Mix` during inference and confirm tracks appear without leaving/re-entering.
7. Confirm playback never pauses, restarts, seeks, or runs out while inference continues.
8. Watch Navidrome Web UI confirm the playlist grows in batches.
9. At completion, verify exactly 40 tracks and 40 unique IDs.
10. Check that NEW-album material is represented and the back nine supplies cohesive deep cuts.
11. Press `G` again and confirm the existing `AI Mix` playlist is overwritten in place rather than duplicated.
12. Confirm no short `[silence]` spacer appears.

## Follow-up checks

- Open `AI Mix`, select it, and press `v`; verify the vault copy appears in area 5.
- From area 5, press `v` on the vaulted copy; verify the same-named server playlist is replaced without mutating the vault source.
- Trigger `M`, then quickly trigger another `M`; verify late events from the cancelled generation cannot reset or mutate the newer playlist.
- Navigate between playlist index, open playlist, and queue during inference; spinner and progress must remain visible.
- Review the final 40-track arc for excessive artist clustering, abrupt transitions, or weak deterministic completion choices.

## Completion gate

AI Mix is complete when a live run produces one overwriteable, server-side 40-track playlist; all IDs are real and unique; the list refreshes live; playback remains uninterrupted; NEW/back-nine priorities are credible; and server/vault copying works in both directions.

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
