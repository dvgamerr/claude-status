
# K3S Config
1. Install `k0sctl` tools
2. set nopassword with sudo `echo -n "orangepi ALL=(ALL) NOPASSWD: ALL\n" > /etc/sudoers.d/010_orangepi-nopasswd`
3. เช็ค machine-id ด้วย ถ้าซ้ำกันก็ให้ `dbus-uuidgen > /etc/machine-id`
4. เปิด `openebs-ndm-config` ใน `cm` แก้ตรง `key: path-filter` ใน `exclude` เป็น `""` แล้วใส่ `include` path ที่จะใช้เก็บข้อมูล `/mnt/WDS250G2B0C`
---

# Config RasPI4
```bash
vi /boot/firmware/config.txt
```

```bash
# Enable DRM VC4 V3D driver
# dtoverlay=vc4-kms-v3d # comment this
max_framebuffers=2
display_lcd_rotate=1 # Rotate 90 
```