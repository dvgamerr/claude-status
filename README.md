# claude-status

Terminal dashboard สำหรับดู Claude Code usage บน Raspberry Pi 4 โดยรับข้อมูลจาก
`statusLine` JSON โดยตรง ไม่ scrape หน้าเว็บและไม่อ่าน credential ของ Claude

```text
 Claude Status  21:42:03 ICT  LIVE

 Model        Opus
 5-hour       █████░░░░░  51%  reset 23:18 (in 1h36m)
 Weekly       ███░░░░░░░  34%  reset Fri 14:00 (in 2d16h)

 Context      ███████░░░  72%  144k / 200k
 Session      01:42:18
 Est. cost    $1.28
 Code         +186  -42

 Pi           CPU 18%  RAM 1.1/4.0 GB  Temp 52°C  Load 0.42  Up 1d02h

 Last update  4s ago
 [q] Quit  [r] Refresh  [s] Sessions
```

## สิ่งที่โปรแกรมทำ

- `claude-status ingest` อ่าน JSON จาก stdin, sanitize, เขียน state แบบ atomic
  แล้วพิมพ์ status line สั้นกลับให้ Claude Code
- `claude-status tui` แสดง 5-hour/weekly quota, context, session, estimated cost,
  code activity และ CPU/RAM/load/temperature/uptime ของ Pi
- แยก state ตาม `session_id` และเลือก session ด้วย `s`, ลูกศร และ Enter
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
  }
}
```

Claude Code ต้องได้รับ trust สำหรับ project ก่อนจึงจะเรียก command status line ได้
ค่า `rate_limits` จะปรากฏเฉพาะบัญชี Claude.ai Pro/Max และหลัง API response แรกของ
session เท่านั้น

เปิด dashboard:

```bash
claude-status tui
```

ตัวเลือกสำคัญ:

```text
--state-dir DIR       เปลี่ยนที่เก็บ snapshot
--session ID          เปิด session ที่ระบุตั้งแต่เริ่ม
--refresh 1s          รอบอ่านข้อมูลและ Pi metrics
--stale-after 15s     อายุข้อมูลก่อนแสดง STALE
--inline              ไม่ใช้ alternate screen
```

ตั้ง path ด้วย environment variable ได้เช่นกัน:

```bash
export CLAUDE_STATUS_STATE_DIR=/var/lib/claude-status
```

## ทดลองด้วย mock data

```bash
claude-status ingest < examples/statusline-input.json
claude-status tui
```

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

ชื่อไฟล์ session เป็น hash เพื่อไม่ให้ `session_id` กลายเป็น path traversal หรือรั่วใน
directory listing ข้อมูล cost เป็นค่าประมาณจาก token ไม่ใช่ยอดเรียกเก็บจริง

## Claude อยู่บน PC/Mac แต่ใช้ Pi เป็นจอ

ให้ `ingest` ทำงานที่เครื่อง Claude ก่อน แล้ว sync เฉพาะ directory snapshot ที่ sanitize
แล้วไปยัง Pi ผ่าน SSH/rsync จากนั้นเปิด TUI ด้วย `--state-dir` ที่ปลายทาง ห้าม copy
credential หรือส่ง JSON ดิบจาก statusLine ไปยัง Pi

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
- [Claude Code: System requirements](https://code.claude.com/docs/en/setup)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Raspberry Pi OS documentation](https://www.raspberrypi.com/documentation/computers/os.html)
