# Koneksi

`rc status` mencetak alamat yang tersedia. Jika tidak ada, tampil `tidak terhubung`.

## Jalur

- **LAN** — alamat RFC1918 host.
- **USB** — tethering Android (rentang 192.168.42.0/24 atau nama antarmuka usb/rndis/enx).
- **Tailnet** — alamat 100.64.0.0/10 bila Tailscale aktif; bila terpasang tapi belum login, tampil `terpasang, belum login`.

## Android client

Gunakan aplikasi RustDesk resmi. Masukkan alamat IP dari `rc status` dan password dari `rc status`/installer.

Bila `rustdesk --option` tidak didukung oleh versi terpasang, aktifkan direct IP secara manual di UI RustDesk (Settings -> Network -> Direct IP access).
