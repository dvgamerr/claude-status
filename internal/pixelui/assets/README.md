# Icon sources

- `anthropic.svg` is the bare white Anthropic mark, with no background or
  surrounding frame, based on the
  [Icons8 Claude/Anthropic set](https://icons8.com/icons/set/anthropic-claude-icon).
- `clawd-exclamation-mark.svg` reproduces the pixel geometry of the matching
  100 px Icons8 Color preview as SVG paths for the framebuffer renderer, with
  a `-2` sibling (`clawd-exclamation-mark-2.svg`): a second hand-edited
  pose — a pulsing alert dot — that `icons.go` rasterizes alongside the
  original and `render.go` alternates between on the state's beat (see
  `mascotPoseForActivity`). `waiting_approval` is the only state left on this
  2-frame pose+alternation system; everything else below plays back a full
  traced GIF sequence instead.
- `clawd-coding.svg`/`clawd-coding-2.svg` and `clawd-sleeping.svg`/
  `clawd-sleeping-2.svg` are no longer loaded by `icons.go` (kept on disk,
  unreferenced, for history) — the legacy `working` state now plays the same
  traced sequence as `typing`, and `idle` now plays its own traced sequence
  (see below), instead of these original static poses.

Clawd artwork is by Icons8: [Coding](https://icons8.com/icon/aStqmH9WIxrR/clawd-coding),
[Sleeping](https://icons8.com/icon/IJM9GlYiqcQn/clawd-sleeping), and
[Exclamation Mark](https://icons8.com/icon/2CuabJDbRdZ8/clawd-exclamation-mark).

- `clawd-idle.svg`, `clawd-thinking.svg`, `clawd-typing.svg`,
  `clawd-building.svg`, `clawd-headphones-groove.svg`, and
  `clawd-juggling.svg` are each a single rigged SVG (same `0 0 100 100`
  viewBox, `crispEdges` as above): a static body plus a handful of named
  `<g>` groups (`eyes`, `leftArm`/`rightArm`, `armHammer`, `upperBody`,
  `thoughtBubble`, `ball1`/`ball2`/`ball3`, ...) animated with real SMIL
  (`<animate>`/`<animateTransform>`, `values`/`keyTimes`/`dur`/
  `repeatCount="indefinite"`) — genuinely valid, browser-viewable animated
  SVGs, not a Go-managed array of pre-rasterized frames. These replaced an
  earlier generation of hand-traced per-frame sequences (10/9/9/9/10/9
  frames respectively) once it turned out those traced frames were flat,
  ungrouped pixel-rect traces with no rig to tween between — see the
  "Replace traced-GIF mascot frames" maintenance entry in the project
  `CLAUDE.md` for that investigation. `internal/svganim.Evaluate` bakes each
  rig's animation for a given instant (since `oksvg` itself has no SMIL
  support), and `icons.go`/`render.go` rasterize the result fresh every
  render tick instead of caching frames at startup.
