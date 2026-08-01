# claude-status

Native pixel dashboard สำหรับดู Claude Code และ Codex usage บน Raspberry Pi 4 โดยรับ
ข้อมูลจาก Claude `statusLine` หรือ Codex turn notification + local rollout metadata
แล้วสร้าง sanitized snapshot ไม่ scrape หน้าเว็บและไม่อ่าน/ส่ง credential ของ provider

หน้าหลักวาดลง `/dev/fb0` แบบ RGB565 ที่ 800×480 โดยตรง ไม่ใช้ terminal grid,
Desktop, Chromium หรือ X/Wayland จึงควบคุม typography, spacing, สี และ rounded cards
ได้ทุกพิกเซล พร้อม Claude-first hero context meter, token chips, 5h/7day cards,
animated Claude mark และ Pi health ส่วน Codex ย่อเหลือ current session/model + context

## สิ่งที่โปรแกรมทำ

- `claude-status ingest` อ่าน Claude JSON จาก stdin, sanitize, เขียน state แบบ atomic
  แล้วพิมพ์ status line สั้นกลับให้ Claude Code
- `claude-status activity` อ่าน Claude Code hook event (`UserPromptSubmit`,
  `PreToolUse`, `Stop`, `Notification`) จาก stdin แล้วอัปเดตแค่สถานะ
  working/idle/waiting-approval ของ session นั้น โดยไม่แตะ field อื่นและไม่เก็บ
  ข้อความ hook ดิบไว้เลย ใช้ขับ animation ของ mascot บนหน้าจอ
- `claude-status codex-notify` รับ Codex turn-complete notification แล้วอ่านเฉพาะ
  model, context/token usage และ 5-hour/7-day usage จาก rollout ของ thread นั้น
- `claude-status import` รับเฉพาะ sanitized snapshot schema สำหรับเครื่อง Pi
- `claude-status gfx` เปิด native framebuffer dashboard 800×480, animate ที่ 250ms
  และเลือก snapshot ล่าสุดของ Claude/Codex แยกกันเพื่อให้ Claude เป็นหน้าหลักเสมอ
  จอเป็น touchscreen จริง แตะแล้วจะเห็น ripple จาง ๆ ตรงจุดที่แตะ (`--touch-device`
  ปรับ evdev device ได้ ใส่ค่าว่างเพื่อปิด) เป็น feedback อย่างเดียว ไม่ใช่ปุ่มกด
- `claude-status preview` render frame เดียวกันเป็น PNG สำหรับ visual QA
- quota ที่ provider ไม่ส่งจะแสดง unavailable; Codex account ที่ระบุ unlimited
  จะแสดง `UNMETERED` แทนการสร้างเปอร์เซ็นต์ขึ้นเอง
- `claude-status tui` ยังเก็บไว้เป็น fallback สำหรับเครื่องที่ไม่มี framebuffer
- แสดง `LIVE`/`STALE` ชัดเจน ป้องกันการเข้าใจ snapshot เก่าว่าเป็นข้อมูลสด
- รองรับ field ที่หาย, เป็น `null` และ field ใหม่ที่โปรแกรมยังไม่รู้จัก

## ติดตั้งบน Raspberry Pi

ต้องใช้ Raspberry Pi OS 64-bit และ Go 1.25 ขึ้นไป ตัว dashboard ใช้งานบน Pi 4
RAM 2 GB ได้ แต่ถ้าจะรัน Claude Code บน Pi เครื่องเดียวกัน ควรใช้ RAM 4 GB หรือ
8 GB ตามข้อกำหนดของ Claude Code

```bash
uname -m                         # ควรได้ aarch64
git clone <repository-url> claude-status
cd claude-status
bash scripts/install.sh
~/.local/bin/claude-status version
sudo install -m 0644 configs/claude-status-tty1.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now claude-status-tty1.service
```

หากมี binary ที่ cross-build มาแล้ว:

```bash
bash scripts/install.sh ./claude-status
```

จากนั้นเพิ่มใน `~/.claude/settings.json` (หรือ project settings):

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
`PreToolUse` → กำลังทำงาน (animation เร็ว สีส้มสด), `Stop` → idle (animation
ช้า สีจาง), `Notification` ที่มีคำว่า permission → รอ approval (สีเหลือง
พร้อม badge "?" กระพริบบน mascot) ถ้าไม่ตั้ง hook พวกนี้ dashboard จะยัง
ทำงานได้ปกติ แต่จะเดาสถานะจาก statusLine freshness แทน

ถ้า Claude/Codex รันบน Windows และ Pi ใช้ SSH alias `pilab` ให้ build Windows กับ
ARM64 binary ก่อน จากนั้นติดตั้ง integration ฝั่ง Windows:

```powershell
pwsh -File scripts/verify.ps1
pwsh -File scripts/install-windows.ps1 -MirrorHost pilab
```

installer จะสำรองและแก้สองไฟล์โดยรักษาค่าอื่นไว้:

- `%USERPROFILE%\.claude\settings.json`: ตั้ง `statusLine` ให้เรียก `ingest`
  และเพิ่ม hook สี่ตัวให้เรียก `activity` (merge เข้าไปแบบไม่ลบ hook อื่นที่มีอยู่)
- `%USERPROFILE%\.codex\config.toml`: ห่อ `notify` เดิมด้วย `codex-notify`
  และ forward event กลับไป notifier เดิมด้วย จึงไม่ทำ Computer Use เดิมหาย

ทั้งสองทางส่งไป Pi ด้วย SSH เฉพาะ snapshot ที่ผ่าน allowlist แล้ว ปลายทางเรียก
`/home/pi/.local/bin/claude-status import`; ไม่ส่ง statusLine JSON ดิบ, prompt,
response, transcript, OAuth token หรือ session auth ไป Pi

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
--refresh 250ms       รอบ animation, อ่านข้อมูล และ Pi metrics
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

ให้ `ingest`/`codex-notify` ทำงานที่เครื่องต้นทาง แล้วส่งเฉพาะ snapshot ที่ sanitize
แล้วไปยัง Pi ผ่าน SSH ห้าม copy credential หรือส่ง JSON ดิบจาก provider ไปยัง Pi

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
