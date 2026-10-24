# LibreDental — LAN Server Setup Guide

> **Audience:** IT staff & system administrators.  
> **Supported OS:** Windows 10/11 & Linux (Ubuntu 22.04+).

## 1. Overview & Architecture

LibreDental offers two separate binaries for two distinct deployment models:

- **Desktop App (`libredental`):** Standalone single-machine desktop application for solo practices operating on a single PC with zero networking.
- **LAN Server (`libredental-server`):** Headless background service running on one central server PC that serves the web application to browser clients over HTTP, or over HTTPS when you turn it on (see [Secure connections](#secure-connections-https)).

> **Note:** Client workstations connect to the server using standard web browsers (Chrome, Edge, Firefox). Remote workstation desktop binaries cannot connect over the network to a server instance. However, running the desktop app locally on the server PC itself is fully supported alongside the server process (safely sharing `libredental.db` via SQLite WAL mode).

## 2. Server Requirements & OS Setup

### Requirements

- **CPU / RAM:** Dual-core (Quad-core recommended), 4 GB RAM.
- **OS:** Windows 10/11 or Ubuntu 22.04+ LTS.
- **Network:** Wired Ethernet with a **Static IP** (e.g. `192.168.1.100`, assigned via router DHCP reservation or OS network settings).

### Credential Storage (Integrations)

API keys for claim clearinghouses and notification providers are stored in the operating
system's credential store, not in the database. Windows uses Credential Manager and needs
no setup. On Linux the server process needs access to an unlocked Secret Service
(e.g. GNOME Keyring) on its user's D-Bus session. Without one, saving integration
credentials fails with an error mentioning `dbus-launch`.

The system-level systemd unit in §3 runs outside any login session, so it has no keyring
even on Ubuntu Desktop. If you use integrations on Linux, run the server inside the
desktop user's session instead, for example as a user unit
(`~/.config/systemd/user/libredental.service` with the same `[Service]` section minus
`User=`, `WantedBy=default.target`, then `systemctl --user enable --now libredental`),
and enable automatic login for that user so the keyring is unlocked after a reboot.

### Automatic Appointment Reminders

To set up reminders for a practice, see [Setting Up Appointment Reminders](appointment-reminders-setup.md).
If the practice turns on reminders (**My Clinic > Reminders**), they are sent by whichever
LibreDental process is running: the server, and also the desktop app if it runs on the
server PC. Both can run at once; each reminder is claimed in the shared database before it's
sent, so it goes out only once. Reminders need the notification providers' credentials, so
on Linux the server must run where it can reach the keyring (see above). Problems such as an
unreadable keyring are shown under **Reminder activity** on the Reminders screen.

The practice timezone (**My Clinic > Practice Profile & Standards**) must match the timezone of the
workstations used to book appointments; the server PC's own timezone isn't used.

### Prevent System Sleep

The Server PC must **never enter sleep/suspend mode** while the practice is open (the display monitor may turn off).

- **Windows:** Settings → System → Power & Sleep → set **Sleep** to **Never**.
- **Linux:**

  ```bash
  sudo systemctl mask sleep.target suspend.target hibernate.target hybrid-sleep.target
  ```

## 3. Server Installation & Firewall

Download `libredental-server.exe` (Windows) or `libredental-server` (Linux) from the LibreDental releases page.

### Linux (Ubuntu)

1. **Install Binary:**

   ```bash
   sudo mkdir -p /opt/libredental
   sudo cp libredental-server /opt/libredental/
   sudo chmod +x /opt/libredental/libredental-server
   ```

2. **Firewall (UFW):** Allow port 4242 from local LAN only:

   ```bash
   sudo ufw allow from 192.168.1.0/24 to any port 4242
   ```

3. **Autostart (systemd):** Create `/etc/systemd/system/libredental.service`:

   ```ini
   [Unit]
   Description=LibreDental LAN Server
   After=network.target

   [Service]
   Type=simple
   User=YOUR_USERNAME
   ExecStart=/opt/libredental/libredental-server
   Restart=on-failure

   [Install]
   WantedBy=multi-user.target
   ```

   Enable and start:

   ```bash
   sudo systemctl daemon-reload && sudo systemctl enable --now libredental
   ```

### Windows

1. **Install Binary:** Copy `libredental-server.exe` to `C:\LibreDental\libredental-server.exe`.
2. **Firewall (PowerShell as Admin):**

   ```powershell
   New-NetFirewallRule -DisplayName "LibreDental LAN" -Direction Inbound -Protocol TCP -LocalPort 4242 -RemoteAddress 192.168.1.0/24 -Action Allow
   ```

3. **Autostart (Task Scheduler):**
   Create a task named `LibreDental Server` triggered **At startup**, running with highest privileges, launching `C:\LibreDental\libredental-server.exe`.

## 4. Workstation Connection & Data Backups

### Client Workstation Connection

On any computer on the local network, open a web browser and navigate to:

<http://192.168.1.100:4242>

*(Replace `192.168.1.100` with your server's static IP. Bookmark for easy access).*

### Secure connections (HTTPS)

By default the server speaks plain HTTP, which anyone on the same network can read, PINs
included. Turn on HTTPS before anyone uses it over Wi-Fi, and always before real patient data
goes in. It adds no software to install. Pick one:

**Self-signed certificate (easiest).** Set `LIBREDENTAL_TLS=auto` and restart the server. The
server makes a certificate the first time and keeps it in the `tls` folder next to the
database (`libredental.db`), covering `localhost`, the computer's name and its current network
addresses. A new network address, or a certificate close to expiry, gets a new one on the next
start. Then open `https://192.168.1.100:4242` (note the `s`).

A browser doesn't know a certificate it made itself, so each device shows a warning the first
time: the connection is encrypted, but nobody has vouched for who is on the other end. On a
workstation choose Advanced, then continue to the site. On a phone, do the same. For a
warning-free setup, use a certificate your devices already trust, below.

**Your own certificate.** Set `LIBREDENTAL_TLS_CERT` and `LIBREDENTAL_TLS_KEY` to the full paths
of a certificate and its private key (both PEM files). A tool such as
[mkcert](https://github.com/FiloSottile/mkcert) makes one that your computers trust once its
root is installed on them, and a practice with its own domain name can use a real certificate.
These two settings win over `LIBREDENTAL_TLS=auto`.

If HTTPS is turned on and the certificate can't be used, the server refuses to start rather
than quietly falling back to HTTP. Plain HTTP is only for a server kept on one computer
(`LIBREDENTAL_HOST=127.0.0.1`) or for testing with demo data. At startup the server logs a
warning when it is serving plain HTTP to the whole network.

On Windows, set the variable for one run in PowerShell with
`$env:LIBREDENTAL_TLS = "auto"` before starting the server, or permanently under System
Properties, Environment Variables. On Linux add `Environment=LIBREDENTAL_TLS=auto` to the
`[Service]` section of the systemd unit.

### Data Location & Daily Backups

All patient data resides in a single SQLite database on the server PC:

- **Linux:** `~/.config/LibreDental/libredental.db`
- **Windows:** `C:\Users\<Username>\AppData\Roaming\LibreDental\libredental.db`

**Linux Hot Backup Script (`/opt/libredental/backup.sh`):**

```bash
#!/bin/bash
BACKUP_DIR="/mnt/backup/libredental"
mkdir -p "$BACKUP_DIR"
sqlite3 "$HOME/.config/LibreDental/libredental.db" ".backup '$BACKUP_DIR/libredental-$(date +%Y-%m-%d).db'"
find "$BACKUP_DIR" -name "libredental-*.db" -mtime +30 -delete
```

Schedule via `crontab -e`: `0 23 * * * /opt/libredental/backup.sh`

## 5. Quick Reference & Troubleshooting

- **Server Self-Access:** If the LAN network switch fails, the Server PC can still access the system at `http://localhost:4242`.
- **Concurrent Server & Desktop Execution:** Running the Desktop App (`libredental`) alongside the LAN Server (`libredental-server`) on the server machine under the same OS user account is supported and safe. Because LibreDental uses SQLite WAL mode (`journal_mode=WAL` with `busy_timeout=5000`), both binaries safely share access to `libredental.db` without data corruption.
- **Host, Port & HTTPS Customization:** The server binds to `0.0.0.0` (all interfaces) on port `4242` by default. Set `LIBREDENTAL_PORT=8080` or `LIBREDENTAL_HOST=127.0.0.1` (or a specific IP like `192.168.1.100`) to customize. `LIBREDENTAL_TLS=auto`, or `LIBREDENTAL_TLS_CERT` with `LIBREDENTAL_TLS_KEY`, turns on HTTPS (see above). With HTTPS on, the self-access address is `https://localhost:4242`.
- **Service Management (Linux):** `sudo systemctl [status|restart|stop] libredental`
