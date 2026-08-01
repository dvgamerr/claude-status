# Raspberry Pi display target

- Target host: `pilab`
- Board: Raspberry Pi 4 Model B Rev 1.4
- Display: original Raspberry Pi Touch Display 7-inch, 800x480
- Native display path: `/dev/fb0`, `vc4drmfb`, 800x480, RGB565 (16 bpp),
  stride 1600 bytes
- Linux console font remains TerminusBold 12x24 only for text-mode fallback
- Text-mode fallback grid: 66 columns x 20 rows
- Installed/supported TerminusBold sizes: 10x20, 12x24, 14x28, and 16x32
- Previous console configuration backup:
  `/etc/default/console-setup.codex-backup-20260731`
- Touch controller: `10-0038 generic ft5x06` at `/dev/input/event0`
  (`ID_INPUT_TOUCHSCREEN=1`, `INPUT_PROP_DIRECT`). Confirmed by raw capture
  that `ABS_MT_POSITION_X/Y` and the legacy `ABS_X/Y` mirror already report
  in panel pixels (0-799, 0-479) — no scaling needed. `input_event` on this
  64-bit kernel is 24 bytes (8+8+2+2+4), also confirmed by capture rather
  than assumed from headers.

The primary UI is the native `gfx` dashboard, rendered at exactly 800x480
pixels into `/dev/fb0`; the old 66x20 TUI is fallback only. After every edit
to `internal/pixelui` (layout, colors, animation), run
`go run ./cmd/claude-status preview` and look at the resulting
`pixel-dashboard-preview.png` before calling the change done — don't just
read the diff and assume the pixels landed where the numbers say they
should. Final visual QA before shipping to `pilab` should still be against
real RGB565 output, not only this RGB PNG preview, because subtle gradients
band on the physical 16-bit framebuffer in ways the preview won't show.
Keep the warm Claude-first theme with a
flat dark background, high-contrast type, equal Claude 5-hour/7-day limit
rows (each with its reset countdown), and a Codex card that deliberately
never shows a session id/name — only its model, reasoning effort, and
context %, since Codex is "the other tool" here, not a session to track.
Provider selection is independent: a newer Codex event must never displace
Claude from the primary UI. The display follows the newest snapshot for each
provider without local input, except for the touch ripple below — touch
never changes what data is shown, only a purely visual acknowledgment that
the screen was tapped.

The panel is a touchscreen, so `internal/touch` (Linux-only; a `!linux` stub
elsewhere) reads `/dev/input/event0` directly and `internal/pixelui/render.go`
draws a fading ring at each tap (`renderTouchRipples`/`blendRing`,
`touchRippleLifetime`). This is feedback only, not a control surface — the
dashboard has no buttons to press. `claude-status gfx --touch-device` can
point at a different evdev node or be set to `""` to disable it; a failure
to open the device is logged as a warning and the dashboard keeps running
without touch feedback, the same "secondary feature must never break the
primary display" pattern as `pixelui.resolveActivity`'s fallbacks. The
`claude-status-tty1.service` unit's `SupplementaryGroups=` must include
`input` (added alongside `video`) or the device open fails with a
permission error.

The header uses a bare white Anthropic logo with no background or frame, and
deliberately omits Claude model and session identity. The left-hand status rail
holds context-specific Clawd SVG artwork and is the dashboard's focal point
(`internal/pixelui/render.go`'s `renderRail`): `working` uses Clawd Coding with
a fast typing bob, `idle` uses Clawd Sleeping with a slow inhale/exhale scale,
and `waiting_approval` uses Clawd Exclamation Mark with a tight urgent shake.
The rail, card, and halo stay completely still in every state; only the mascot
artwork moves. The embedded SVGs are rasterized once at renderer startup, not
on every frame. Activity state is independent of the statusLine refresh cycle
— see below. The rail caption's session name is drawn with the bundled UI
font (Go's core "gofont" set, Latin-only) — a name in Thai or another
non-Latin script has no glyphs in that font, so `pixelui.sessionName` checks
glyph coverage first and falls back to the truncated session ID rather than
drawing a row of unmapped glyph boxes.

## Provider ingestion and Windows source

- The dashboard accepts both Claude Code and Codex snapshots. A snapshot has a
  `provider` field, and the UI must label it `CLAUDE STATUS` or `CODEX STATUS`.
- Windows installs the source binary at
  `%LOCALAPPDATA%\Programs\claude-status\claude-status.exe`.
- Claude Code calls `ingest` through `%USERPROFILE%\.claude\settings.json`.
- Codex calls `codex-notify` through the external `notify` array in
  `%USERPROFILE%\.codex\config.toml`. Preserve and forward any notifier that
  was configured before claude-status is installed.
- Both source adapters may mirror to `pilab`, but only by serializing the
  allowlisted `model.Snapshot` and piping it over SSH to
  `/home/pi/.local/bin/claude-status import`.
- Never mirror raw Claude statusLine JSON, a raw Codex notify payload, rollout
  lines, prompts, responses, transcripts, credentials, OAuth tokens, or
  session auth. The Pi's `import` command must reject unknown JSON fields.
- Codex context usage uses the latest `last_token_usage.total_tokens` divided
  by `model_context_window`. The 5-hour and 7-day bars use the 300-minute and
  10,080-minute rate-limit windows when the account exposes them; unavailable
  enterprise limits show `UNMETERED` only when Codex explicitly reports
  `credits.unlimited=true`; otherwise unavailable values remain `--`.

## Activity state (mascot animation, approval badge)

- The statusLine refresh alone cannot tell whether Claude is working, idle, or
  blocked on a permission prompt, so `%USERPROFILE%\.claude\settings.json`
  also registers four hooks — `UserPromptSubmit`, `PreToolUse` (matcher `*`),
  `Stop`, and `Notification` — that all call `claude-status activity`
  (`scripts/install-windows.ps1`'s `Set-ClaudeStatusHook`). That merge is
  additive: it only ever replaces the `claude-status ... activity` hook group
  it previously installed on each event and leaves every other tool's hook
  group on that event untouched.
- `claude-status activity` reads one hook payload from stdin, classifies it
  with `claude.ActivityForHook`, and merges just the `Activity{State,
  UpdatedAt}` field into that session's already-stored snapshot — it never
  rebuilds the snapshot from scratch, so a hook firing before or after the
  next statusLine event can't clobber the other's fields. For `Notification`
  it inspects the hook's `message` text locally to detect a permission prompt
  ("needs your permission to use ...") vs. an idle-nudge; either way the
  message text itself is discarded, never persisted or mirrored, matching the
  "never mirror raw payloads" rule above.
- This command must always exit 0. `PreToolUse` hooks can block the tool call
  if their hook exits non-zero, and this activity side channel must never be
  able to do that.
- `pixelui.resolveActivity` degrades gracefully when hooks aren't installed
  (falls back to a statusLine-freshness proxy) and when a state gets stuck
  (falls back to idle after `activityStaleAfter`, 10 minutes, in case a `Stop`
  hook was missed).

## statusLine does not work from the VS Code extension (confirmed, not a bug)

- Windows gotcha noticed while debugging this: `~/.claude.json`'s per-project
  `hasTrustDialogAccepted` is keyed by the project path as a literal string,
  case-sensitive drive letter included. `E:/.dvgamerr/aide-lab` and
  `e:/.dvgamerr/aide-lab` are two separate entries; trusting one does not
  trust the other, and Claude Code silently skips `statusLine`/hooks for an
  untrusted entry with no visible error. If hooks ever stop firing after
  reopening a project, check both letter-case variants before assuming a
  config regression.
- `statusLine` is CLI-terminal-only. Confirmed via Claude Code's own docs
  (the VS Code extension feature-comparison table omits `statusLine`
  entirely) and GitHub issue #55643 ("Support custom statusLine in VS Code
  extension"), closed as **not planned**. It renders as a bar at the bottom
  of a real terminal; the VS Code extension's chat panel has no such
  surface, so `ingest` (the command `statusLine` invokes) never runs for
  that session — not a settings.json mistake, not a trust problem. Hooks
  are unaffected by this: they fire from any interface.
- Rate limits only ever appear in the statusLine JSON for Pro/Max accounts,
  and only after the session's first API response — a second, independent
  reason the numbers can be briefly absent even when statusLine does fire.
- Hooks never carry usage/rate-limit numbers, by design, in any interface —
  `claude.HookInput` only has `session_id`, `hook_event_name`, and
  `message` (Notification only) because that's all Claude Code sends. Don't
  go looking for a hook-based way to get real percentages; there isn't one.
- Non-obvious discovery: statusLine *does* fire from the short-lived CLI
  processes Claude Code spawns internally for subagents (the Task/Agent
  tool) and some slash commands (`/usage`) — those are real CLI
  invocations with a terminal-like context, unlike the main chat session.
  Their `ingest` writes a real snapshot under
  `%LOCALAPPDATA%\claude-status\sessions\*.json` with genuine
  `rate_limits` (account-wide, so any fresh one is valid regardless of
  which session produced it) — but that `ingest` call's `--mirror-ssh`
  step does not reliably reach `pilab` from inside that spawned process
  (cause unconfirmed; possibly no SSH/network context there). So the real
  numbers exist locally moments after any subagent or `/usage` run, but
  don't show up on the Pi on their own.
- Two ways to get real numbers onto `pilab` from a VS Code extension
  session: (a) `claude-status usage --five-hour PCT --seven-day PCT`
  (merges just the two percentages into the latest session, mirrors it —
  see "Activity state" above for the shape it preserves), fed from numbers
  read off `/usage`'s own output or the Account & Usage panel; or (b) find
  the freshest real snapshot under `sessions\*.json` (matching `/usage`'s
  numbers) and pipe it straight into
  `ssh pilab "/home/pi/.local/bin/claude-status import"` to mirror that
  exact snapshot.

References:

- https://www.raspberrypi.com/documentation/accessories/display.html
- https://manpages.debian.org/trixie/console-setup/console-setup.5.en.html

## SSH-only control

- `pilab` has no keyboard attached. Perform all input, control, recovery, and
  process management remotely through `ssh pilab`; never ask the user to press
  a key such as `q` on the Raspberry Pi.
- The physical pixels are `/dev/fb0`; `/dev/tty1` is switched to `KD_GRAPHICS`
  while the service runs. A normal SSH PTY and a PowerShell window
  on Windows are different terminals, so clearing or drawing in either one does
  not change the Raspberry Pi display.
- The dashboard on `/dev/tty1` is owned by the enabled system service
  `claude-status-tty1.service`. It runs
  `/home/pi/.local/bin/claude-status gfx --refresh 66ms --framebuffer /dev/fb0 --tty /dev/tty1`
  (~15fps; the flag floor is 20ms, so this is intentionally not maxed out)
  with `Restart=always`.
- Inspect it with
  `ssh pilab "systemctl --no-pager --full status claude-status-tty1.service"`.
  Start, stop, or restart it through SSH with `sudo systemctl`; do not try to
  control the service by sending keys to the physical console.
- `/home/pi/.local/bin` is not guaranteed to be in Fish's non-interactive
  `PATH`, so use the absolute path when invoking `claude-status` outside the
  service.

## Lessons learned from the failed control attempt

- Running `clear` in an ordinary SSH session cleared only that SSH terminal,
  not `/dev/tty1`.
- Running bare `claude-status` only prints command help; the primary dashboard
  command is `claude-status gfx`.
- Opening a visible PowerShell/SSH TUI on Windows did not update the Raspberry
  Pi's physical display and left an interactive session the user could not
  control from the Pi.
- Sending `TERM` or `KILL` directly to the dashboard process was the wrong
  control method: systemd immediately respawned it because the service uses
  `Restart=always`. Control `claude-status-tty1.service` instead.
- This file is the canonical project instruction file. Do not write these notes
  to `C:\Users\dvgamerr\Desktop\CLAUDE.md`.
