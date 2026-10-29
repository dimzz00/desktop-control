# Terminal (0.2.0)

kendali-term menyajikan shell pengguna lewat browser (xterm.js).

## Menjalankan

Service `kendali-term` berjalan otomatis sebagai pengguna login. Manual:

```
kendali-term serve --config /etc/kendali/rc.yaml
```

## Mengakses

- Desktop (tunnel): buka URL dari `rc term url` (localhost:47654).
- Android/tailnet: `rc term url` menampilkan alamat tailnet.

## Auth

Token 32 byte di `rc.yaml` (`terminal.token`). Kirim di URL query pada koneksi pertama;
setelah itu cookie sesi HttpOnly dipakai. `rc token --rotate` mengganti token.

## Bind

- `localhost` (default): hanya tunnel desktop.
- `tailnet`: akses langsung via IP tailnet (termasuk Android). Memerlukan konfirmasi.
