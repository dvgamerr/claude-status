# Raspberry Pi display target

- Target host: `pilab`
- Board: Raspberry Pi 4 Model B Rev 1.4
- Display: original Raspberry Pi Touch Display 7-inch, 800x480
- Linux console font: TerminusBold 12x24, configured persistently and loaded
  without requiring a reboot
- Effective console grid: 66 columns x 20 rows (previously about 100x30 with
  an 8x16 font)
- Installed/supported TerminusBold sizes: 10x20, 12x24, 14x28, and 16x32
- Previous console configuration backup:
  `/etc/default/console-setup.codex-backup-20260731`

Keep the primary TUI at or below 66x20 cells, including borders. Prefer short,
high-contrast labels and progress bars that remain legible on the physical
touch display. The session picker must paginate rather than grow past 20 rows.

References:

- https://www.raspberrypi.com/documentation/accessories/display.html
- https://manpages.debian.org/trixie/console-setup/console-setup.5.en.html
