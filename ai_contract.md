# AI Curator Contract

## Objective

Build grounded, private AI playlists from a user's musical intent without treating NEW uploads as recommendations by themselves. Retrieval establishes eligibility; DJ-Weazl judges and sequences eligible music; application code enforces truth and limits.

## Product boundaries

### Mood remains unchanged

- `M` starts from the playing or queued track.
- It creates or overwrites the Navidrome server playlist `Mood`.
- It streams 20 unique, validated tracks without interrupting playback.
- Its current seed-based prompt and similarity behavior remain independent from AI Mix.

### Interactive AI Mix

- `G` offers `zero_tax_grindage` or `Tell Weazl what you want`.
- Both produce 40-track private vault playlists.
- `G` explicitly takes over playback with the new mix, even when another track is playing.
- Publish three grounded launch tracks immediately, replace the active queue with them, and start the authoritative seed.
- Append the remaining 37 validated tracks behind the launch tracks as inference completes.
- Never restart or stop the playing anchor while backfilling the playlist.
- The user explicitly presses `v` to copy the completed private playlist to Navidrome.

## Governing principle

The user request is the eligibility filter. NEW is only a tie-breaker among relevant music.

Never ask an 8B model to infer acoustic suitability from an arbitrary NEW-first track dump. Sparse title, artist, album, genre, and year metadata cannot reliably describe how a recording sounds, and recognizable cover titles can mislead the model about the actual recording.

## Authoritative-seed retrieval pipeline

### 1. Resolve a primary seed without the LLM

The critical path first attempts deterministic local grounding:

- search the encrypted cache with the normalized full phrase, meaningful phrases, and n-grams;
- rank exact artist and track matches above album and genre matches;
- prefer canonical studio recordings;
- reject covers, remixes, live, acoustic, novelty, karaoke, and tribute versions unless requested;
- choose the strongest real library track as the primary seed.

For `new wave like New Order`, a real New Order artist match must outrank tracks with merely recognizable 1980s titles.

### 2. Expand to three launch tracks deterministically

- Ask Navidrome for the primary seed's similarity neighborhood.
- Select the authoritative primary seed plus two strong launch neighbors from different artists and albums.
- Validate all three against the synced cache.
- Prefer Navidrome similarity order while enforcing recording-variant exclusions.
- Do not call the LLM when this fast path succeeds.

### 3. Use inference only as a grounding fallback

If deterministic local grounding cannot find a credible primary seed:

- run one compact intent inference to produce library search concepts;
- search the cache with those concepts and retry deterministic expansion;
- use closed-world LLM anchor selection only as the final fallback over a compact real pool;
- keep every fallback stage visible through the animated status.

Never send the entire library to the model.

### 4. Publish a playable draft immediately

As soon as three launch tracks are validated:

- create or replace the private playlist with those launch tracks;
- navigate to Private Playlists and select it;
- replace the active queue with the launch tracks;
- begin playback at the authoritative seed;
- show `3/40` while background curation starts;
- preserve the playlist and queue identity while later tracks append.

Long-form inference must never delay the first playable playlist.

### 5. Launch-track constraints

Launch-track constraints:

- three distinct tracks;
- prefer three different artists;
- maximum one track per album;
- every launch track must independently fit the request;
- reject covers, remixes, live recordings, acoustic versions, novelty recordings, and tribute versions unless the request calls for them;
- validate every ID against the supplied anchor pool.

If fewer than three credible launch tracks exist after the fallback budget, fail clearly. Before publication, failure leaves the existing playlist, queue, and playback untouched.

### 6. Build the grounded candidate crate

- Treat track one as the sole authoritative semantic anchor.
- Tracks two and three are provisional launch tracks for responsiveness, not independent retrieval anchors.
- Request the candidate neighborhood from the authoritative seed only, matching Mood's proven retrieval shape.
- Never allow a convenient secondary neighbor to redefine or broaden the requested genre.
- A secondary neighborhood may be considered later only when it substantially overlaps the primary neighborhood; it is not part of the initial contract.
- Exclude recent plays, blacklisted tracks, missing tracks, and the wrong recording variants.
- Add NEW tracks only when they are compatible with the interpreted request or appear in the authoritative seed's grounded similarity neighborhood.
- Retain authoritative-seed provenance in the encrypted run record.
- Interleave candidate classes and artists before inference so list order cannot become an album dump.

### 7. Run prompt-aware Mood curation

Run DJ-Weazl over the grounded crate with:

- the original user request;
- the interpreted intent;
- the authoritative seed plus the two provisional launch tracks, clearly labeled by role;
- candidate metadata and provenance;
- the remaining class and diversity requirements;
- previously accepted and rejected IDs during repair rounds.

The original request remains active throughout every round so similarity expansion cannot drift into generic adjacent music.

The launch tracks occupy the first three positions so playback can begin immediately. The authoritative seed owns the mix's identity. DJ-Weazl curates the remaining 37 tracks and protects the transition from launch track three into track four.

## AI Mix system-prompt contract

- AI Mix uses `internal/curator/prompts/ai_mix.md`.
- Mood retains its separate seed-continuation prompt.
- User intent is law.
- NEW means recently uploaded to this Navidrome library, not recently released.
- NEW never makes an irrelevant candidate eligible.
- Do not copy candidate order or select contiguous album blocks.
- Runtime user text, mode instructions, repair state, and candidates belong in the user message rather than the system prompt.
- The LLM judges relevance, flow, and sequencing; application code enforces all mechanical constraints.

## Application-enforced playlist constraints

- exactly 40 unique, real track IDs;
- no more than 24 NEW tracks;
- at least 16 BACK-NINE tracks;
- no more than 3 tracks per artist;
- no more than 2 tracks per album;
- at least 8 distinct artists;
- no adjacent tracks by the same artist;
- no invented, stale, duplicate, blacklisted, or ineligible IDs;
- normalized title `silence` with duration ten seconds or less remains blacklisted;
- deterministic fallback and repair rounds obey the same constraints.

Prompt relevance may reduce the NEW share below 60%. It may never admit an irrelevant NEW track merely to fill a quota.

## Streaming and failure contract

- Do not create an empty private playlist.
- Create or replace it immediately after all three launch tracks are validated.
- Replace the queue with those launch tracks and start the authoritative seed immediately.
- Persist, append to the queue, and display each of the remaining 37 validated tracks as it arrives.
- Keep the Weazl spinner animated through interpretation, anchor resolution, similarity retrieval, curation, repair, and persistence.
- Preserve accepted valid tracks across repair rounds.
- Stop inference once 40 acceptable tracks are secured.
- Failure before publication restores the pre-existing same-named playlist and leaves queue and playback untouched.
- Failure, cancellation, or supersession after publication preserves the playable partial playlist and queue. It reports the incomplete count without rolling back or stopping playback.

## `zero_tax_grindage`

`zero_tax_grindage` uses the same immediate three-launch-track publication and playback pipeline without a user query. Its deterministic fast path chooses an authoritative discovery seed, then uses that seed's Navidrome neighborhood for two provisional launch tracks and the final candidate crate. It does not wait for intent inference and must not collapse into a NEW-album sampler.

Its 60/40 class balance and all diversity limits are enforced by the application.

## Naming

Playlist naming needs a separate refinement after grounded retrieval works. Names should describe the interpreted musical intent rather than mechanically title-casing the complete user sentence.

Example target: `new wave like New Order` may become `AI Mix: New Order New Wave`.

Naming must remain local and deterministic unless a later contract explicitly authorizes another inference call.

## Explicitly separate follow-up work

Do not mix these changes into anchor retrieval unless required by a shared primitive:

- playing-track follow is complete: an already-open active playlist or queue selects and reveals the active queue index on playback transitions;
- playlist deletion is complete: `Delete` removes a selected server or vault playlist only after `Y/N` confirmation;
- further playlist-name refinement.

## Implementation phases

### Phase A: Deterministic fast grounding

- Add phrase and n-gram cache search with deterministic scoring.
- Resolve a primary seed without inference when the request contains searchable library evidence.
- Expand through Navidrome and choose two distinct-artist, distinct-album launch tracks.
- Test exact named-artist priority and recording-variant rejection.

### Phase B: Immediate playlist and playback handoff

- Create the private playlist at exactly three validated launch tracks.
- Navigate to it, replace the queue, and start the authoritative seed.
- Preserve playback while later tracks append to both playlist and queue.
- Distinguish failures before and after publication.

### Phase C: Fallback grounding and similarity crate

- Retain compact intent inference only when deterministic grounding fails.
- Retain closed-world LLM anchor selection only as the last fallback.
- Retrieve the similarity neighborhood from the authoritative seed only.
- Keep provisional launch tracks out of candidate expansion so they cannot cause semantic drift.
- Interleave the candidate payload to remove NEW, artist, and album ordering bias.
- Fail cleanly when three credible launch tracks cannot be established.

### Phase D: Constrained prompt-aware curation

- Feed original intent, authoritative seed, and provisional launch-track roles into the dedicated AI Mix prompt.
- Enforce NEW/BACK-NINE, artist, album, distinct-artist, and adjacency rules during acceptance.
- Make repair requests constraint-aware.
- Make deterministic completion obey the same contract.

### Phase E: Streaming integration

- Connect the grounded pipeline to the existing private-playlist stream.
- Preserve spinner animation and generation-session isolation across all stages.
- Preserve live vault refresh, queue appends, active playback, and generation-session isolation.

### Phase F: Verification

- Run full tests and `go vet`.
- Rebuild `./subweazl` and `~/.subweazl/bin/subweazl`.
- Smoke-test at least:
  - `new wave like New Order`;
  - `synthwave like Information Society`;
  - one mood/energy request without a named artist;
  - `zero_tax_grindage`.
- Inspect musical relevance, recording identity, artist/album diversity, class balance, sequence flow, and live streaming behavior before commit/push.
- Measure time from pressing Enter to the three-track playlist appearing and anchor one starting. The deterministic fast path should take seconds, not minutes.

## Completion gate

The feature is not complete merely because 40 valid IDs were produced. It passes only when three grounded launch tracks become playable promptly, later recordings credibly satisfy the request, every mechanical constraint is respected, and slow inference never stops the music.
