# claude-status

Native pixel dashboard สำหรับดู Claude Code และ Codex usage บน Raspberry Pi 4 โดยรับ
ข้อมูลจาก Claude `statusLine` หรือ Codex turn notification + local rollout metadata
แล้วสร้าง sanitized snapshot ไม่ scrape หน้าเว็บและไม่อ่าน/ส่ง credential ของ provider

หน้าหลักวาดลง `/dev/fb0` แบบ RGB565 ที่ 800×480 โดยตรง ไม่ใช้ terminal grid,
Desktop, Chromium หรือ X/Wayland จึงควบคุม typography, spacing, สี และ rounded cards
ได้ทุกพิกเซล

## หน้าตาปัจจุบันของ dashboard

- ซ้าย: rail มี mascot (Clawd) เป็นจุดสนใจหลัก ใต้ pill สถานะ (WORKING/IDLE/NEEDS
  APPROVAL) มีแค่ระยะเวลา ("working for 12s") — ไม่แสดงชื่อหรือ ID ของ session
  เลย ต่อด้วย Pi health (CPU/MEM/GPU)
- ขวาบน: "5 HOUR LIMIT" และ "7 DAY LIMIT" วางเต็มความกว้างซ้อนกันคนละแถว
  (label+เวลารีเซ็ตแถวเดียวกัน ตามด้วย bar แล้วเลขเปอร์เซ็นต์ใหญ่)
- กลาง: หัวข้อ model + reasoning level (เช่น "SONNET 5" / "HIGH EFFORT") สไตล์
  เดียวกับการ์ด Codex ตามด้วย "CONTEXT WINDOW" และ token chip INPUT/OUTPUT
- ขวาล่าง: การ์ด Codex (model, reasoning effort, context) ทั้งการ์ด Codex และ
  chip INPUT/OUTPUT ชิดขอบล่างของจอเป็นแนวเดียวกัน ไม่มีที่ว่างเหลือด้านล่าง
- header มีแค่ Anthropic mark กับคำว่า CLAUDE เท่านั้น ไม่มีนาฬิกา ไม่มี
  model/session identity (ข้อมูลนั้นย้ายไปอยู่กลางจอแทน)

## สิ่งที่โปรแกรมทำ

- `claude-status ingest` อ่าน Claude JSON จาก stdin, sanitize, เขียน state แบบ atomic
  แล้วพิมพ์ status line สั้นกลับให้ Claude Code
- `claude-status activity` อ่าน Claude Code hook event (`UserPromptSubmit`,
  `PreToolUse`, `Stop`, `Notification`) จาก stdin แล้วอัปเดตแค่สถานะ
  working/idle/waiting-approval ของ session นั้น โดยไม่แตะ field อื่นและไม่เก็บ
  ข้อความ hook ดิบไว้เลย ใช้ขับ animation ของ mascot บนหน้าจอ
- `claude-status usage --five-hour PCT --seven-day PCT` เขียนแค่ 5h/7d limit
  ทับ session ล่าสุด (หรือ `--session ID` เจาะจง) โดยไม่แตะ field อื่น — ใช้ตอน
  `statusLine` ไม่ยิงจริง (เท่าที่เจอ ยิงได้แค่จาก CLI terminal ไม่ยิงจาก
  Claude Code แบบ VS Code extension) เอาไว้กรอกเลขจากหน้า Account & Usage มือ
- `claude-status codex-notify` รับ Codex turn-complete notification แล้วอ่านเฉพาะ
  model, context/token usage และ 5-hour/7-day usage จาก rollout ของ thread นั้น
- `claude-status import` รับเฉพาะ sanitized snapshot schema สำหรับเครื่อง Pi
- `claude-status relay` เป็น process แยกที่อ่าน snapshot ล่าสุดจาก local state,
  ส่งไป Pi ผ่าน SSH และ retry อัตโนมัติเมื่อเครือข่ายขาด
- `claude-status service install|remove|start|stop|status` ติดตั้ง relay ให้รัน
  เป็น background service — คำสั่งเดียวกันทำงานเหมือนกันบน Windows (Windows
  Service จริง), Linux (`systemd --user`) และ macOS (launchd `LaunchAgent`)
- `claude-status pi install` เขียนและเปิดใช้งาน systemd unit ของ dashboard บน
  Raspberry Pi เอง ชี้ไปที่ path ของ binary ปัจจุบันโดยตรง (รองรับ `go install`
  โดยไม่ต้อง clone repo)
- `claude-status gfx` เปิด native framebuffer dashboard 800×480, animate ที่ 66ms (~15fps)
  และเลือก snapshot ล่าสุดของ Claude/Codex แยกกันเพื่อให้ Claude เป็นหน้าหลักเสมอ
  จอเป็น touchscreen จริง แตะแล้วจะเห็น ripple จาง ๆ ตรงจุดที่แตะ (`--touch-device`
  ปรับ evdev device ได้ ใส่ค่าว่างเพื่อปิด) เป็น feedback อย่างเดียว ไม่ใช่ปุ่มกด
- `claude-status preview` render frame เดียวกันเป็น PNG สำหรับ visual QA
- quota ที่ provider ไม่ส่งจะแสดง unavailable; Codex account ที่ระบุ unlimited
  จะแสดง `UNMETERED` แทนการสร้างเปอร์เซ็นต์ขึ้นเอง
- `claude-status tui` ยังเก็บไว้เป็น fallback สำหรับเครื่องที่ไม่มี framebuffer
- แสดง `LIVE`/`STALE` ชัดเจน ป้องกันการเข้าใจ snapshot เก่าว่าเป็นข้อมูลสด
- รองรับ field ที่หาย, เป็น `null` และ field ใหม่ที่โปรแกรมยังไม่รู้จัก

## สถาปัตยกรรม: ข้อมูลเดินทางยังไงตั้งแต่ต้นจนขึ้นจอ

ระบบแยกเป็น 3 ชั้นที่ไม่พึ่งพากันโดยตรง ชั้นไหนพังก็ไม่ทำให้ชั้นอื่นค้าง:

1. **Source adapter** (เครื่องที่รัน Claude Code/Codex จริง) — `ingest`,
   `activity`, `usage`, `codex-notify` แต่ละตัวเป็น process สั้น ๆ ที่ถูกเรียก
   ต่อ event เดียว (statusLine refresh, hook, หรือ turn-complete notify) อ่าน
   input, sanitize ผ่าน allowlist แล้ว **เขียนแค่ local state แบบ atomic**
   (temp file + fsync + rename) จบแล้วก็ออก — ไม่เปิด network เอง ไม่รอ SSH
   ไม่ block hook หรือ statusLine แม้แต่ nanoseconds เดียว
2. **Relay** (`claude-status relay`) — process ระยะยาวตัวเดียวที่ watch local
   state directory เดียวกันนั้น เจอ snapshot ที่เปลี่ยน (เทียบด้วย hash/mtime)
   ก็ส่งไป Pi ผ่าน SSH โดย retry เองเมื่อเครือข่ายขาดหรือ Pi ปิดอยู่ชั่วคราว
   เป็นเจ้าของ SSH transport เพียงจุดเดียวในทั้งระบบ ปลายทางเสมอคือ
   `claude-status import` บน Pi ซึ่งรับเฉพาะ `model.Snapshot` schema ที่รู้จัก
   และ reject field แปลกปลอมทันที
3. **Renderer** (`claude-status gfx` บน Pi) — loop เดียวที่อ่าน state
   directory ของตัวเอง (ที่ `import` เขียนไว้), เลือก snapshot ล่าสุดของ
   Claude กับ Codex แยกกัน (Claude เป็นหลักเสมอ ต่อให้ Codex event ใหม่กว่า),
   แล้ว composite เฟรมด้วย `internal/pixelui` วาดตรงลง `/dev/fb0` ทุก
   `--refresh` (ดีฟอลต์ 66ms/~15fps) — ไม่รอ SSH, ไม่รอ relay, ไม่ block

Activity state (working/idle/waiting-approval) เดินคนละ path จาก
snapshot ทั่วไป: hook เขียนแค่ field `Activity{State, UpdatedAt}` merge เข้า
ไปในของเดิม ไม่ทำให้ statusLine event ที่มาก่อน/หลังกันมาทับข้อมูลกัน และ
mascot จะ fallback กลับ idle เองถ้า state ค้างเกิน 10 นาที (เผื่อ `Stop`
hook หลุดไป)

ถ้าเครื่องรัน Claude Code/Codex เป็นเครื่องเดียวกับที่มี `/dev/fb0` (Pi ตัวเดียว
ทำหมด) ก็ข้าม relay ไปเลยได้ — `ingest`/`activity`/`gfx` อ่านเขียน state
directory เดียวกันตรง ๆ

## ติดตั้งบน Raspberry Pi

ต้องใช้ Raspberry Pi OS 64-bit ตัว dashboard ใช้งานบน Pi 4 RAM 2 GB ได้ แต่ถ้าจะรัน
Claude Code บน Pi เครื่องเดียวกัน ควรใช้ RAM 4 GB หรือ 8 GB ตามข้อกำหนดของ Claude Code

วิธีที่ง่ายที่สุด: ติดตั้งผ่าน `go install` (ต้องมี Go 1.25+ บน Pi) แล้วสั่ง
`pi install` ตัวเดียวจบ — ไม่ต้อง clone repo, ไม่ต้องเขียน systemd unit เอง:

```bash
uname -m                         # ควรได้ aarch64
go install github.com/dvgamerr/claude-status/cmd/claude-status@latest
sudo $(go env GOPATH)/bin/claude-status pi install
```

`pi install` เขียน systemd unit `claude-status-tty1.service` ชี้ไปที่ path ของ
binary ปัจจุบันโดยตรง (ไม่ hardcode `/home/pi/.local/bin` เหมือนเดิมอีกต่อไป)
ตั้ง `User`/`Group`/`WorkingDirectory` ตามผู้ใช้ที่เรียก (`$SUDO_USER`, override ได้
ด้วย `--user NAME`) แล้ว `systemctl daemon-reload` + `enable --now` ให้ในคำสั่งเดียว
— ต้องรันด้วยสิทธิ์ root (จะ `sudo` ให้เองอัตโนมัติถ้ายังไม่ได้รันแบบ root) ตัวเลือกอื่น:
`--refresh 66ms`, `--framebuffer /dev/fb0`, `--tty /dev/tty1`, `--touch-device
/dev/input/event0` (ใส่ค่าว่างเพื่อปิด touch feedback)

ถ้าไม่อยากพึ่ง Go module proxy ก็ยังใช้วิธี clone + build เองได้เหมือนเดิม:

```bash
git clone https://github.com/dvgamerr/claude-status
cd claude-status
bash scripts/install.sh          # หรือ: bash scripts/install.sh ./claude-status (มี binary cross-build แล้ว)
sudo ~/.local/bin/claude-status pi install
```

จากนั้นเพิ่มใน `~/.claude/settings.json` (หรือ project settings) ถ้า Claude Code
รันบน Pi เครื่องเดียวกัน:

```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.local/bin/claude-status ingest",
    "padding": 1,
    "refreshInterval": 5
  },
  "hooks": {
    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "~/.local/bin/claude-status activity" }] }],
    "PreToolUse": [{ "matcher": "*", "hooks": [{ "type": "command", "command": "~/.local/bin/claude-status activity" }] }],
    "Stop": [{ "hooks": [{ "type": "command", "command": "~/.local/bin/claude-status activity" }] }],
    "Notification": [{ "hooks": [{ "type": "command", "command": "~/.local/bin/claude-status activity" }] }]
  }
}
```

hook ทั้งสี่ทำให้ mascot บนจอรู้สถานะจริงของ session: `UserPromptSubmit`/
`PreToolUse` → กำลังทำงาน (Clawd Coding พิมพ์อยู่ ตากะพริบ), `Stop` → idle
(Clawd Sleeping หายใจช้า ๆ พร้อม Zzz ลอย), `Notification` ที่มีคำว่า permission →
รอ approval (Clawd Exclamation Mark สั่นเตือน จุดตกใจกะพริบ) แต่ละสถานะมี SVG
2 ท่าสลับกันตามจังหวะของตัวเอง (ไม่ใช่แค่ขยับ/ย่อขยาย raster เดิม) ส่วน rail,
card และ halo ด้านหลังอยู่นิ่งทั้งหมด ถ้าไม่ตั้ง hook พวกนี้ dashboard จะยัง
ทำงานได้ปกติ แต่จะเดาสถานะจาก statusLine freshness แทน

ถ้า Claude Code/Codex รันอยู่คนละเครื่องกับจอ Pi ดูหัวข้อ "Claude/Codex อยู่บน
PC/Mac แต่ใช้ Pi เป็นจอ" ด้านล่าง สำหรับวิธีตั้งค่าฝั่งเครื่องต้นทาง

Claude Code ต้องได้รับ trust สำหรับ project ก่อนจึงจะเรียก command status line ได้
ค่า `rate_limits` จะปรากฏเฉพาะบัญชี Claude.ai Pro/Max และหลัง API response แรกของ
session เท่านั้น

ตั้งแต่ Claude Code v2.1.132 ค่า `total_input_tokens` และ `total_output_tokens`
หมายถึง token ใน context ปัจจุบันจาก API response ล่าสุด; รุ่นก่อนหน้านั้นเป็นยอดสะสม
ของ session

เปิด dashboard:

```bash
claude-status gfx --framebuffer /dev/fb0 --tty /dev/tty1
```

ตัวเลือกสำคัญ:

```text
--state-dir DIR       เปลี่ยนที่เก็บ snapshot
--refresh 66ms        รอบ animation, อ่านข้อมูล และ Pi metrics (ค่าเริ่มต้น ~15fps, ต่ำสุด 20ms)
--stale-after 15s     อายุข้อมูลก่อนแสดง STALE
--framebuffer PATH    framebuffer device; ค่าเริ่มต้น /dev/fb0
--tty PATH            console ที่สลับเข้า graphics mode; ค่าเริ่มต้น /dev/tty1
```

ตั้ง path ด้วย environment variable ได้เช่นกัน:

```bash
export CLAUDE_STATUS_STATE_DIR=/var/lib/claude-status
```

## ทดลองด้วย mock data

```bash
claude-status ingest < examples/statusline-input.json
claude-status preview --output dashboard.png
```

ใน Nushell ใช้ `open --raw` และ external-command marker `^`:

```nu
open --raw examples/statusline-input.json | ^go run ./cmd/claude-status ingest
```

การรัน `claude-status ingest` เปล่า ๆ ไม่มี stdin จะ error ตามตั้งใจ เพราะตอนใช้งานจริง
Claude Code เป็นผู้ pipe JSON เข้ามาให้อัตโนมัติ

state จะอยู่ที่ `${XDG_CACHE_HOME:-~/.cache}/claude-status/` บน Linux:

```text
claude-status/
├── sessions/
│   └── <sha256-prefix>.json
└── latest.json
```

directory ใช้ permission `0700`; snapshot ใช้ `0600`; การเขียนใช้ temporary file,
`fsync` และ atomic rename

## Privacy และ security

โปรแกรมสร้าง struct ใหม่จาก allowlist เท่านั้น ไม่ serialize input ดิบ จึงไม่เก็บ:

- `transcript_path`
- prompt หรือข้อความสนทนา
- `~/.claude/.credentials.json`
- OAuth token หรือ API key
- path ของ workspace

ฝั่ง Codex จะเปิด local rollout ของ `thread-id` ที่ notify ส่งมา แต่ decode เฉพาะ
`session_meta`, `turn_context` และ `token_count`; บรรทัดอื่นรวมถึงข้อความสนทนาจะถูกข้าม
ทั้งหมด จากนั้นจึง serialize เฉพาะ `Snapshot` schema เดียวกับ Claude

ชื่อไฟล์ session เป็น hash เพื่อไม่ให้ `session_id` กลายเป็น path traversal หรือรั่วใน
directory listing ข้อมูล cost เป็นค่าประมาณจาก token ไม่ใช่ยอดเรียกเก็บจริง

## Claude/Codex อยู่บน PC/Mac แต่ใช้ Pi เป็นจอ

`ingest`/`activity`/`usage`/`codex-notify` เขียนแค่ local state บนเครื่องต้นทาง
และไม่เปิด network เอง — ตัว **relay** เป็น process ระยะยาวตัวเดียวที่เป็นเจ้าของ
SSH transport ทั้งหมด คอย watch local state แล้วส่งเฉพาะ snapshot ที่ sanitize
แล้วไปเรียก `/home/pi/.local/bin/claude-status import` บน Pi ห้าม copy credential
หรือส่ง JSON ดิบจาก provider ไปยัง Pi เด็ดขาด

ติดตั้ง relay ให้รันเป็น background service — คำสั่งเดียวกัน ทำงานเหมือนกันทั้ง
Windows, Linux, และ macOS:

```bash
go install github.com/dvgamerr/claude-status/cmd/claude-status@latest
claude-status service install --mirror-ssh pilab
```

`service install` ใช้ service manager ของแต่ละ OS โดยตรง (ไม่ผ่านตัวกลางอื่น):

| OS      | กลไกเบื้องหลัง                        | สิทธิ์ที่ต้องใช้                  |
|---------|----------------------------------------|-----------------------------------|
| Windows | Windows Service จริง (SCM)             | รันจาก Administrator shell        |
| Linux   | `systemd --user` unit                  | ไม่ต้อง root                      |
| macOS   | launchd `LaunchAgent`                  | ไม่ต้อง root                      |

คำสั่งอื่นในกลุ่มเดียวกัน: `claude-status service status|start|stop|remove`
(ใช้ชื่อบริการ `claude-status-relay` เดียวกันทุก OS) รัน `install` ซ้ำได้เสมอ —
ถ้ามี instance เดิมรันอยู่จะ stop แล้ว restart ให้เองด้วย binary/flag ใหม่

บน Windows ยังมีสคริปต์ `install-windows.ps1` ที่ทำครบในคำสั่งเดียว (build/ติดตั้ง
binary + เพิ่ม hook ใน `~/.claude/settings.json` + wrap `notify` ใน
`~/.codex/config.toml` + เรียก `service install` ให้ตอนท้าย):

```powershell
pwsh -File scripts/verify.ps1
pwsh -File scripts/install-windows.ps1 -MirrorHost pilab
```

installer จะสำรองไฟล์เดิมไว้ก่อนแก้เสมอ (`.claude-status-backup-<timestamp>`)
บน Linux/macOS ยังต้องแก้ `~/.claude/settings.json` และ `~/.codex/config.toml`
ตามตัวอย่าง JSON ด้านบนเอง ก่อนรัน `service install`

โปรเจกต์ยังไม่เปิด HTTP collector โดยตั้งใจ เพราะการเปิด network endpoint เพิ่มภาระเรื่อง
authentication/TLS โดยไม่จำเป็นสำหรับ MVP; SSH transport ปลอดภัยและดูแลง่ายกว่า

## พัฒนาและทดสอบ

```bash
go test ./...
go vet ./...
go build -o bin/claude-status ./cmd/claude-status
```

Cross-build สำหรับ Pi 4:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -trimpath -o bin/claude-status-linux-arm64 ./cmd/claude-status
```

สร้าง tarballs สำหรับ Linux ARM64/AMD64 พร้อม checksum:

```bash
bash scripts/package.sh v0.1.0
```

ถอนการติดตั้ง binary (ไม่ลบ state):

```bash
bash scripts/uninstall.sh
```

ใช้ `bash scripts/uninstall.sh --purge` เมื่อต้องการลบ state ด้วย

## แหล่งข้อมูลหลัก

- [Claude Code: Customize your status line](https://code.claude.com/docs/en/statusline)
- [Codex configuration reference](https://developers.openai.com/codex/config-reference)
- [Codex CLI source: external notify configuration](https://github.com/openai/codex/blob/main/codex-rs/core/src/config/mod.rs)
- [Claude Code: System requirements](https://code.claude.com/docs/en/setup)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Raspberry Pi OS documentation](https://www.raspberrypi.com/documentation/computers/os.html)
