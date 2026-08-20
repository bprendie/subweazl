# Phase 8: DJ-Weazl Curator

## Product modes

DJ-Weazl produces closed-world playlists using only tracks present in the synced, encrypted Subsonic cache.

### AI Mix

- Press `G` to build a general LLM mix. `G` never changes meaning based on playback state.
- Produce exactly 40 tracks.
- Create or overwrite the case-insensitive Navidrome server playlist named `AI Mix` after the first track validates.
- Grow `AI Mix` in validated batches, navigate to it in area 4, and refresh an open playlist in place just like Mood.
- Preserve current playback and use the existing upcoming queue as a safety buffer while inference runs.
- Prioritize tracks from Navidrome's newest albums.
- Use back-catalog and deep cuts to complete a cohesive sequence.
- Exclude recently played tracks and avoid repeated tracks.

### Mood Radio

- Press `M` to build or overwrite the server-side playlist named `Mood`.
- Use the currently playing track as the seed. If nothing is playing, fall back to the queue's current or selected track.
- Produce exactly 20 tracks, including the seed.
- Preserve the seed's mood, energy, texture, and momentum.
- Include compatible tracks from Navidrome's newest albums and use the back nine for mood-adjacent deep cuts.
- Build a one-shot, 20-track playlist rather than continuously refilling it.
- Create the server-side `Mood` playlist immediately with the seed as track one, then grow it as validated selections arrive.
- A later generation replaces the existing case-insensitive server `Mood` playlist instead of creating duplicates.
- Navigate to server Playlists (area 4) and select `Mood`; never save Mood into the vault automatically.
- Applying Mood never restarts, stops, pauses, seeks, or otherwise interrupts playback.
- Keep the pre-generation upcoming queue as a safety buffer until enough generated tracks exist to replace it safely.
- Insert validated generated tracks immediately behind whichever track is currently playing. If playback advances during inference, do not rewind to the original seed.
- When generation completes, remove obsolete safety-buffer entries and leave only the unplayed portion of the completed Mood sequence behind the current track.
- If the active seed is not present in the synced cache, Mood fails without changing playback or the existing queue.
- On failure, preserve playback and any still-needed safety-buffer entries. Keep the last fully valid server `Mood` revision rather than publishing ghosts, duplicates, or malformed IDs.

### Server and vault playlist boundary

- Area 4 contains Navidrome server playlists; area 5 contains encrypted vault-only playlists.
- With a playlist selected, `v` copies it across the boundary.
- Area 4 to area 5 replaces a case-insensitive same-named vault playlist.
- Area 5 to area 4 replaces a case-insensitive same-named server playlist.
- Copying never deletes or mutates the source playlist.

## Meaning of NEW

`NEW` is grounded exclusively in Navidrome's newest-album ordering. It does not mean a recent release year and is not inferred by the LLM.

- Cache sync fetches approximately the top 20 albums from Navidrome's `newest` classification.
- Tracks belonging to those albums receive `new=true` and a `new_rank` derived from Navidrome's ordering.
- Each completed sync replaces the previous newest classification.

## Closed-world candidate contract

- Candidate tracks come exclusively from non-missing records in the synced cache.
- The LLM receives real track IDs and compact cached metadata.
- Every returned ID is checked against the exact candidate snapshot used for that run.
- Invented, altered, stale, duplicate, and malformed IDs are rejected before queue construction.
- A playlist never contains a track absent from that snapshot.
- At least 20 eligible, distinct candidates must exist before generation starts.
- Exclude spacer artifacts whose normalized title is `silence` and duration is ten seconds or less. Do not exclude full-length songs merely because their title is "Silence."

## Validation and repair loop

Generation accumulates valid selections until exactly 20 unique tracks have been accepted.

1. For Mood, accept the seed immediately and ask the LLM for the remaining 19 IDs. For AI Mix, ask for 20.
2. Stream the response and recognize each complete JSON string as it arrives rather than waiting for the complete response body.
3. Validate each streamed ID against the candidate snapshot before emitting a TUI progress event.
4. Preserve valid IDs in their returned order.
5. Reject IDs that are unavailable, malformed, duplicated, or already accepted.
6. Stop and cancel inference as soon as the requested number of valid IDs has arrived. Extra output is irrelevant; Mistral is allowed to ignore the requested array length without delaying completion.
7. If a response ends before the playlist is full, ask for exactly the remaining number of replacements.
8. A repair request contains only candidates that have never been accepted or rejected.
9. Repeat until 20 valid, unique tracks have been accepted or the overall deadline expires.

The repair loop remains active during streaming:

- Invalid, invented, stale, duplicated, malformed, and incomplete IDs never consume a playlist position.
- Valid IDs from every attempt are committed in order and are never discarded merely because another ID in the same response failed validation.
- When a streamed response stops, calculate `remaining = 20 - accepted` and immediately begin another streamed request for exactly that shortfall.
- Include the accepted IDs in the repair context and remove accepted and rejected candidates from the next candidate payload.
- Keep the same generation ID, animated spinner, queue safety buffer, and server `Mood` playlist throughout all repair rounds.
- Incremental Navidrome updates contain only the seed and validated IDs; an invalid ID can never briefly appear in the server playlist or queue.
- Continue repair rounds while useful progress is being made and the ten-minute overall deadline has not expired.
- If a round produces no valid IDs, retry with a smaller, clearer candidate window and stronger exact-count language rather than restarting the playlist.
- Cancel immediately once the twentieth valid unique track is accepted.

Do not use a 400-token response cap for opaque Navidrome IDs. Streaming cancellation, not a tiny token limit, bounds excess model output. Configure enough headroom for the model to close a valid response while retaining the ten-minute overall deadline.

After the repair budget is exhausted, fill any remaining slots deterministically from the same ranked, closed-world candidate snapshot. Record these IDs separately as library-filled selections. This guarantees eventual completion without admitting a ghost or duplicate and without discarding valid LLM choices.

The full operation has a ten-minute deadline so a cold or busy local inference server has enough room for repair rounds. A timeout produces a concise DJ-Weazl error, preserves uninterrupted playback, retains already validated generated tracks, and keeps enough of the original safety buffer to prevent the queue from running dry.

## Prompt and response contract

- The system prompt lives in `internal/curator/prompts/dj_weazl_draft.md` and will be embedded in the binary.
- Prompts are kept compact for an 8B-class local model.
- The model returns only a JSON array of track IDs.
- Requests specify the exact count, mode, seed when applicable, newest/back-nine quotas, accepted IDs, and remaining candidates.
- Temperature remains low for consistent structured output.

## Run records

Encrypted recommendation-run metadata retains:

- Candidate snapshot identity and mode.
- Seed and requested quotas.
- Every LLM request and raw response.
- Accepted and rejected IDs for each attempt.
- Rejection reasons.
- Final ordered track IDs or terminal failure.

## Remediation implementation order

1. Add a streaming completion API for vLLM and Ollama with cancellation after the required number of validated IDs.
2. Introduce a generation-session ID and typed progress messages for accepted tracks, phases, completion, and failure.
3. Batch the spinner tick and inference commands so rendering remains animated for the entire request.
4. Implement the non-destructive queue safety buffer and incremental splice behavior.
5. Implement immediate server `Mood` creation, debounced incremental updates, and exact final replacement.
6. Keep `G` and `M` as separate explicit commands and align sidebar/help copy.
7. Complete symmetric `v` server/vault copying.
8. Add slow-inference, overlong-output, cancellation, queue-continuity, server-upsert, and stale-message tests.
9. Run the full suite and vet, then rebuild both the workspace and installed binaries for interactive testing.

Do not consider the remediation complete until a live Helena test shows an animated spinner, uninterrupted playback, a visibly growing server `Mood` playlist, an exact 20-track final playlist, and a matching active queue.

## TUI contract

The LLM remains headless. Its prompt, prose, JSON, validation details, rejected IDs, and repair-round responses are never rendered in the TUI.

- `ctrl+l` opens the same staged LLM wizard used by WeazlChat. Uppercase `L` remains a compatibility alias.
- Select `vllm` or `ollama`, enter the provider base URL, wait while models are discovered automatically, select a returned model, and save immediately.
- vLLM discovery uses `/v1/models`; Ollama discovery uses `/api/tags`.
- If discovery fails or returns no models, allow exact manual model-name entry as the WeazlChat fallback.
- `esc` cancels from any wizard stage without modifying the saved provider.

- Start Bubble's spinner tick command in the same batch as the inference command; never discard the initial tick.
- Keep spinner ticks independent of the blocking network request so animation continues for multi-minute inference.
- While generation is active, show Bubble's `spinner.Jump`, styled with the Weazl accent color, followed by an underscore-separated gerund phrase and compact progress such as `7/20`.
- Curator phrases include `sniffing_the_new_bin`, `protecting_the_vibe`, `checking_the_back_nine`, `weeding_ghost_tracks`, `rejecting_taxable_transitions`, `sequencing_the_deep_cuts`, and `wheezing_the_playlist`.
- Select the initial phrase from the generation start time. Keep it stable for short runs and change it at most twice, around 20 and 40 seconds, matching the established WeazlChat behavior.
- Use user-facing phases: preparing candidates, curating, validating, updating Mood, and complete. Do not expose raw model diagnostics.
- Each accepted-ID message is tagged with a generation ID. Ignore late messages from cancelled or superseded generations.
- Debounce Navidrome writes into small batches rather than issuing one API request per token; the server playlist must still visibly grow during inference.
- Success finalizes the exact server playlist, active queue, encrypted run record, and `20/20` status.
- Failure preserves uninterrupted playback and shows one concise, user-actionable error.
- Full prompts, raw responses, rejection reasons, and attempt history remain available only in the encrypted recommendation-run record.
- Glamour is not part of the progress display; it remains a Markdown renderer in WeazlChat. Subweazl's curator spinner uses Bubble and Lip Gloss only.

## Required tests

- Newest status comes from Navidrome album ordering, not release year or sync time.
- A later cache sync replaces the previous newest classification.
- Only non-missing cached tracks become candidates.
- Hallucinated and stale IDs never enter queues or saved playlists.
- Duplicate IDs within and across attempts are rejected.
- Valid selections survive repair rounds in order.
- A 13-valid/7-invalid first response requests exactly seven replacements.
- Multiple streamed repair rounds retain earlier valid IDs and request only the current shortfall.
- A zero-progress repair round narrows the candidate window without clearing accepted tracks or interrupting playback.
- Invalid IDs never appear transiently in the active queue or incremental server playlist.
- A model that ignores the requested count and emits 120 IDs is cancelled after the first required valid IDs.
- Long opaque Navidrome IDs cannot be truncated by a fixed 400-token ceiling.
- Streamed partial JSON never admits an incomplete ID.
- The spinner advances while inference is blocked for several minutes.
- Mood appears in server Playlists immediately with the seed and grows in validated batches.
- Existing playback never restarts, pauses, seeks, or stops when `M` is pressed.
- If the seed finishes before inference, the safety-buffer queue continues playback.
- Completion removes obsolete safety-buffer entries without replaying tracks already heard.
- Repeated `M` runs replace one case-insensitive server `Mood` playlist.
- Late messages from a superseded generation cannot mutate the queue or playlist.
- Generation completes with exactly 20 valid, unique IDs, using ranked closed-world completion only after repair attempts are exhausted.
- `v` copies server playlists to the vault and vault playlists to the server without mutating the source.
