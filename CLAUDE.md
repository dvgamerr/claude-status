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

The primary UI is the native `gfx` dashboard, rendered at exactly 800x480
pixels into `/dev/fb0`; the old 66x20 TUI is fallback only. Design and visual
QA against RGB565 output, not only an RGB PNG preview, because subtle gradients
band on the physical 16-bit framebuffer. Keep the warm Claude-first theme with a
flat dark background, high-contrast type, equal Claude 5-hour/7-day quota cards,
and a compact Codex card containing only its latest session/model and context.
Provider selection is independent: a newer Codex event must never displace
Claude from the primary UI. The display follows the newest snapshot for each
provider without local input.

The left-hand status rail holds the animated clay starburst mascot and is the
dashboard's focal point (`internal/pixelui/render.go`'s `renderRail`): its
motion communicates the source session's activity state at a glance, per
Nielsen's visibility-of-system-status heuristic, without requiring anyone to
read text. Three states, each with a distinct speed/color/behavior profile in
`visualForActivity`: `working` (fast, bright orange, orbiting dot), `idle`
(slow, dim breathing), and `waiting_approval` (yellow, plus a pulsing "?"
badge overlaid on the mascot and a yellow glow around the rail card). Activity
state is independent of the statusLine refresh cycle — see below.

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
  `/home/pi/.local/bin/claude-status gfx --refresh 250ms --framebuffer /dev/fb0 --tty /dev/tty1`
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
