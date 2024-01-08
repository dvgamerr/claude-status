1. Install `k0sctl` tools
2. set nopassword with sudo `echo -n "orangepi ALL=(ALL) NOPASSWD: ALL\n" > /etc/sudoers.d/010_orangepi-nopasswd`
3. เช็ค machine-id ด้วย ถ้าซ้ำกันก็ให้ `dbus-uuidgen > /etc/machine-id`