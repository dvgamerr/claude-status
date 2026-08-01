# Icon sources

- `anthropic.svg` is the bare white Anthropic mark, with no background or
  surrounding frame, based on the
  [Icons8 Claude/Anthropic set](https://icons8.com/icons/set/anthropic-claude-icon).
- `clawd-coding.svg`, `clawd-sleeping.svg`, and
  `clawd-exclamation-mark.svg` reproduce the pixel geometry of the matching
  100 px Icons8 Color previews as SVG paths for the framebuffer renderer.
- Each of those three has a `-2` sibling (`clawd-coding-2.svg`,
  `clawd-sleeping-2.svg`, `clawd-exclamation-mark-2.svg`): a second hand-edited
  pose — blinking "<..>" eyes, drifted Zzz, a pulsing alert dot — that
  `icons.go` rasterizes alongside the original and `render.go` alternates
  between on each state's beat. This keeps the mascot's second beat an actual
  edited SVG pose, not just a geometric transform of the same raster.

Clawd artwork is by Icons8: [Coding](https://icons8.com/icon/aStqmH9WIxrR/clawd-coding),
[Sleeping](https://icons8.com/icon/IJM9GlYiqcQn/clawd-sleeping), and
[Exclamation Mark](https://icons8.com/icon/2CuabJDbRdZ8/clawd-exclamation-mark).
