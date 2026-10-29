# Pemecahan masalah

- `rc doctor` melaporkan drift versi RustDesk: kendali tidak upgrade RustDesk; perbarui manual bila perlu.
- Checksum tidak cocok: unduhan rusak; ulangi instalasi.
- Tailscale mati: jalankan `tailscale up`.
- Service mati: `systemctl start rustdesk` (Linux) atau Services.msc (Windows).
- Konfigurasi skema masa depan: `config_version` lebih baru dari yang didukung; downgrade aplikasi atau pulihkan backup.
