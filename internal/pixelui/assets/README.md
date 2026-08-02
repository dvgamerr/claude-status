# Icon sources

- `anthropic.svg` is the bare white Anthropic mark, with no background or
  surrounding frame, based on the
  [Icons8 Claude/Anthropic set](https://icons8.com/icons/set/anthropic-claude-icon).
- `clawd-sleeping.svg` and `clawd-exclamation-mark.svg` reproduce the pixel
  geometry of the matching 100 px Icons8 Color previews as SVG paths for the
  framebuffer renderer. Each has a `-2` sibling (`clawd-sleeping-2.svg`,
  `clawd-exclamation-mark-2.svg`): a second hand-edited pose — drifted Zzz, a
  pulsing alert dot — that `icons.go` rasterizes alongside the original and
  `render.go` alternates between on each state's beat (see
  `mascotPoseForActivity`). This keeps the mascot's second beat an actual
  edited SVG pose, not just a geometric transform of the same raster.
- `clawd-coding.svg`/`clawd-coding-2.svg` are no longer loaded by `icons.go`
  (kept on disk, unreferenced, for history) — the legacy `working` state now
  plays the same traced sequence as `typing` (see below) instead of this
  original static pose.

Clawd artwork is by Icons8: [Coding](https://icons8.com/icon/aStqmH9WIxrR/clawd-coding),
[Sleeping](https://icons8.com/icon/IJM9GlYiqcQn/clawd-sleeping), and
[Exclamation Mark](https://icons8.com/icon/2CuabJDbRdZ8/clawd-exclamation-mark).

- `clawd-thinking-01.svg`..`-09.svg`, `clawd-typing-01.svg`..`-09.svg`,
  `clawd-building-01.svg`..`-09.svg`, `clawd-headphones-groove-01.svg`..`-10.svg`,
  and `clawd-juggling-01.svg`..`-09.svg` are full frame-by-frame traces (same
  `0 0 100 100` viewBox, `crispEdges`, path-only grouped-fill conventions as
  above) of a real reference animation per pose (thought bubble rising, hands
  typing, hammer swinging, head bobbing to music, balls arcing) supplied for
  this project — each pose's source GIF and extracted PNG frames live under
  the repo-root `assets/` folder (outside this package), not committed here.
  Frame numbers are zero-padded specifically so a lexical sort is also
  playback order; `icons.go`'s `loadFrameSequence` glob-embeds and sorts them,
  and `render.go`'s `gifFrameIndex` advances through the sequence at
  `gifFrameDuration` (300ms — 5 source frames' worth at the original ~60ms/
  frame GIF rate) per displayed frame, looping.
