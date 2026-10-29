# Keamanan

- RustDesk: password permanen per device, dibuat saat instalasi, ditampilkan `rc status`.
- Terminal: token 32 byte, constant-time, rate limit, idle timeout 30m, bind localhost default.
- Siklus token: dibuat saat instalasi, disimpan di rc.yaml mode 0600, ditampilkan `rc token`, diganti `rc token --rotate`.
- Pemulihan token memerlukan akses ke layar host.
- Rilis: checksum + HTTPS; tanda tangan kunci sebelum 1.0.0.
- Log tidak memuat token, password, atau isi sesi.
- `rc doctor --bundle` (0.3.0) mengumpulkan diagnostik dengan redaksi.
