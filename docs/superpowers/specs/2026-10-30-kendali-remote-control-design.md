# kendali — Desain Aplikasi Kendali Device Jarak Jauh

- Tanggal: 2026-10-30
- Status: disetujui, siap masuk tahap rencana implementasi
- Nama kerja: `kendali` (bisa diganti tanpa memengaruhi arsitektur)

## 1. Ringkasan

Aplikasi untuk mengendalikan komputer Linux dan Windows dari jarak jauh, dengan tiga
mode di atas satu sesi: **Layar** (lihat desktop + kontrol mouse/keyboard), **Terminal**
(sesi shell), dan **File** (jelajah dan transfer). Pengendali berjalan di Android, Linux,
dan Windows. Tiga jalur koneksi: **LAN**, **USB** (tethering), dan **Internet** (tailnet).

Sasaran sekarang: dipakai sendiri. Arah berikutnya: membantu/mengawasi device orang lain,
tanpa membongkar arsitektur.

## 2. Mengapa

Alternatif yang ada menuntut salah satu dari: data lewat server pihak lain, lisensi
tertutup, atau ketergantungan pada satu jalur koneksi saja. Yang diinginkan adalah satu
bundel yang bisa dipasang di mesin sendiri, jalur koneksinya jelas, dan **mudah
dirawat bertahun-tahun** — bukan proyek sekali jadi.

"Ma mudah dirawat" diterjemahkan menjadi keputusan konkret:

- tidak mem-fork komponen besar milik pihak lain;
- kode sendiri sekecil mungkin dan terkumpul di satu tempat;
- naik/turun versi satu perintah, dengan rollback otomatis bila gagal;
- setiap perilaku punya tes, termasuk perilaku saat gagal;
- dokumentasi operasional (runbook) ditulis bersamaan dengan kode.

## 3. Goals

1. Host (device yang dikendalikan) berjalan di Linux dan Windows, terpasang dengan satu
   perintah, idempotent, dan bisa di-uninstall bersih.
2. Client di Android, Linux, Windows dapat melihat layar, mengendalikan input, dan
   mentransfer file.
3. Terminal dapat dibuka dari ketiga client tanpa memasang client tambahan.
4. Koneksi berhasil lewat LAN, USB (tethering), dan Internet (tailnet) dengan alamat
   yang dicetak oleh satu perintah.
5. Upgrade dan rollback aman, terverifikasi, dan tidak mengharuskan pengguna mengubah
   konfigurasi.
6. Seluruh perilaku penting tercakup tes otomatis; rilis tidak bisa keluar bila ada yang
   merah.

## 4. Non-goals (sekarang)

- Host di Android/iOS/macOS. Android hanya client.
- Audio dua arah, remote printing, dan multi-pengguna bersamaan.
- Aplikasi client bermerek sendiri (memakai aplikasi RustDesk resmi dulu).
- Relay/signaling sendiri (ditunda; `infra/` disiapkan untuk itu).
- Mode "unattended massal" dengan akun, grup, dan audit.

## 5. Anti-goals (dianggap gagal walau secara teknis jalan)

- Mem-fork RustDesk lalu menggabungkan perubahan upstream secara manual setiap rilis.
- Menulis sendiri pipeline video/audio, codec, hole punching, atau injeksi input.
- Membuka port terminal ke jaringan luas tanpa lapisan tambahan.
- Upgrade yang meninggalkan mesin dalam keadaan setengah terpasang.
- Tes yang bergantung pada `sleep` atau jaringan internet nyata.

## 6. Batasan nyata

- **Lisensi.** RustDesk AGPLv3. Kode tambahan kita ikut AGPL; distribusi wajib menyertakan
  sumber. Keputusan sadar, dicatat di `docs/SECURITY-LICENSE.md`.
- **USB.** Kabel USB-A↔USB-A antar dua PC tidak berfungsi (tidak ada mode host-ke-host).
  Jalur USB berarti telepon menjadi adapter jaringan (tethering) atau adapter
  USB-Ethernet/USB link cable. Bagi aplikasi, ini hanya "LAN lewat kabel".
- **Linux.** X11 adalah jalur utama (mesin pengembang: Pop!\_OS 22.04, GNOME, X11).
  Wayland didukung sebatas kemampuan portal; keterbatasan didokumentasikan.
- **Windows.** Aplikasi berjalan elevated (UAC) tidak dapat dikendalikan dari sesi
  non-elevated. Didokumentasikan, bukan diperbaiki.
- **Tailscale.** Butuh akun dan aplikasi Tailscale di tiap device. Dependensi pihak ketiga
  yang diterima untuk Tahap 1.
- **Infrastruktur.** Tidak ada VPS/server milik pengguna saat ini.

## 7. Pendekatan yang dipertimbangkan

| # | Pendekatan | Kelebihan | Kekurangan | Putusan |
|---|------------|-----------|------------|---------|
| 1 | Berdiri di atas RustDesk, tanpa fork | Cepat bisa dipakai; client Android/Linux/Windows sudah ada; NAT traversal & codec matang; perawatan upstream gratis | AGPL; bergantung arah upstream; tidak bisa menutup sumber | **Dipilih** |
| 2 | Tulis sendiri seluruh stack | Bebas lisensi; arsitektur persis sesuai; kendali penuh | Berbulan-bulan; kualitas awal lebih rendah; menanggung seluruh bug platform | Ditolak untuk sekarang (tetap jalur keluar bila kelak butuh lepas AGPL) |
| 3 | Hibrida (agent sendiri + WebRTC) | Tanpa batasan lisensi; tetap memakai komponen media yang matang | Kerja besar; dua ekosistem untuk dirawat | Ditolak |

Alasan memilih #1: tujuan utamanya aplikasi yang benar-benar terpakai dengan beban
perawatan kecil. Nilai tambah kita berada di lapisan pengalaman pakai, terminal, dan
operasional — bukan di pipeline video.

**Prinsip pelaksanaan #1: nol patch pada upstream.** RustDesk dipakai sebagai binary rilis
resmi yang dikunci versi + checksum. Semua kode kita berada di luar binari itu.

## 8. Arsitektur

### 8.1 Komponen

```
                 ┌──────────── client (Android / Linux / Windows) ────────────┐
                 │  aplikasi RustDesk resmi  +  browser (UI terminal)         │
                 └───────────────┬───────────────────────┬────────────────────┘
                     layar/input │                       │ tunnel TCP
                                 ▼                       ▼
┌──────────────────────────── host: Linux / Windows ───────────────────────────┐
│  RustDesk (binary resmi, tanpa patch)                                        │
│    • screen capture, codec, input injection, clipboard, file transfer       │
│    • TCP tunneling  ──────────────►  kendali-term (127.0.0.1:47654)          │
│  kendali-term (Go)                                                           │
│    • PTY (Unix) / ConPTY (Windows), UI web xterm.js, token auth              │
│  rc (Go, CLI)                                                                │
│    • instalasi, status, doctor, upgrade, rollback, log, alamat koneksi       │
│  Tailscale (opsional, jalur internet)                                        │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 8.2 Alur koneksi — satu bentuk untuk tiga jalur

`rc status` mencetak tiga alamat siap pakai:

```
LAN       192.168.1.42
USB       192.168.42.129      (tethering)
Tailnet   100.101.102.103
Terminal  tunnel → 127.0.0.1:47654   (token: lihat `rc token`)
```

Client cukup menempelkan alamat ke aplikasi RustDesk. Tidak ada perbedaan konfigurasi
antar-jalur, sehingga tidak ada tiga jalur kode yang harus dirawat.

### 8.3 Terminal

- Binary Go, mendengarkan **hanya** `127.0.0.1:47654` (port dapat diubah di `rc.yaml`).
- Diakses melalui fitur TCP tunneling RustDesk: tidak ada port yang terbuka ke jaringan.
- Opsi `bind: tailnet` tersedia untuk akses langsung via IP tailnet, **default mati**.
- Antarmuka web (xterm.js) berjalan di browser Android, Linux, dan Windows — tidak perlu
  client ketiga.
- Auth: token 32 byte acak, dibandingkan constant-time, batas percobaan, timeout idle,
  dan pembersihan sesi saat koneksi putus.

### 8.4 Struktur repo

```
VERSION                    # versi aplikasi kita (satu-satunya sumber)
deps.lock                  # pin upstream: nama, versi, URL, sha256
rc.yaml.example            # konfigurasi contoh + config_version
Makefile                   # build, test, lint, release
go/
  cmd/rc/                  # CLI
  cmd/kendali-term/        # sidecar terminal
  internal/config/         # parse, validasi, migrasi
  internal/release/        # unduh, verifikasi, tukar versi, rollback
  internal/term/           # PTY + sesi + auth + HTTP
  internal/tailnet/        # deteksi alamat tailnet (interface tipis, mudah di-fake)
host/linux/                # install.sh, uninstall.sh, kendali-term.service
host/windows/             # install.ps1, uninstall.ps1
infra/                     # compose relay (belum dipakai; disiapkan untuk nanti)
scripts/                   # release.sh, dev-update.sh, hooks/
docs/                      # INSTALL, CONNECT, UPGRADE, TROUBLESHOOT, SECURITY, qa/
tests/                     # integrasi + fixture
.codegraph/                # indeks lokal (tidak masuk git)
```

### 8.5 Lokasi terpasang

| | Linux | Windows |
|---|---|---|
| Program | `/opt/kendali/releases/<ver>/` + `current` (symlink) | `%ProgramData%\kendali\releases\<ver>\` + penunjuk `current` |
| Konfigurasi | `/etc/kendali/rc.yaml` | `%ProgramData%\kendali\rc.yaml` |
| Log | `/var/log/kendali/` | `%ProgramData%\kendali\logs\` |
| Autostart | systemd `kendali-term.service` + service RustDesk | Windows service RustDesk + `kendali-term` |

## 9. Keputusan yang sudah diambil

| Keputusan | Alasan |
|---|---|
| Tanpa fork; pakai binary rilis resmi | Beban perawatan paling kecil; upgrade = ganti versi |
| Tailscale untuk jalur internet (Tahap 1) | Tidak perlu server; enkripsi berlapis; tiga jalur jadi "sambung ke IP" |
| `infra/` relay disiapkan sejak awal, belum dipakai | Pindah ke relay sendiri nanti hanya mengganti satu alamat |
| Go untuk `rc` dan `kendali-term` | Binary statis, lintas-compile mudah, dependensi sedikit, mudah dirawat |
| Terminal hanya lewat tunnel (default) | Permukaan serang minimal |
| UI terminal berbasis web | Satu implementasi melayani Android + Linux + Windows |
| Nama kerja `kendali` | Netral, mudah diganti |

## 10. Versioning

Empat jenis versi, dipisahkan tegas.

### 10.1 Versi aplikasi — SemVer

- Sumber tunggal: `VERSION`. Disuntik ke binary via `-ldflags "-X main.version=$(cat VERSION)"`
  dan ke installer/paket. Tidak ada nomor versi manual di kode.
- `0.x` selama pembangunan: `0.1.0` Tahap 1, `0.2.0` terminal, `0.3.0` CI+docs.
  `1.0.0` setelah dipakai harian tanpa masalah.
- Pra-rilis `X.Y.Z-rc.N`; build lokal `X.Y.Z-dev.<sha>` (tidak dipublikasi).
- Aturan: MINOR untuk fitur/flag kompatibel, PATCH untuk perbaikan, MAJOR bila pengguna
  wajib mengubah konfigurasi.
- `rc --version` mencetak versi kita + versi RustDesk + skema konfigurasi.

### 10.2 Dependensi — `deps.lock`

- Format per baris: `nama versi url sha256`.
- Mengunci RustDesk, versi Go untuk build, dan Tailscale minimum.
- Diverifikasi saat instalasi dan saat upgrade; tidak cocok = batal tanpa menyentuh
  instalasi berjalan.
- Job CI mingguan mengunduh ulang semua URL dan membandingkan checksum untuk mendeteksi
  rilis ulang diam-diam oleh upstream.

### 10.3 Skema konfigurasi — `config_version`

- `rc.yaml` memiliki nomor skema sendiri, terpisah dari versi aplikasi.
- Migrasi otomatis setelah membuat cadangan `rc.yaml.bak-<versi>`.
- Skema lebih baru dari yang dikenal → ditolak dengan pesan jelas, tidak ditebak.

### 10.4 Rilis dan pemasangan

- Conventional Commits; tag `vX.Y.Z` memicu build rilis.
- Artefak `kendali-<ver>-linux-amd64.tar.gz` + `.sha256` + daftar isi; `CHANGELOG.md`
  format Keep a Changelog.
- Pemasangan versioned: `releases/<ver>/` + penunjuk `current`. Linux memakai symlink;
  Windows mengarahkan ulang `binPath` service lewat `sc config`.
- Dua versi terakhir disimpan.
- **Upgrade atomik:** unduh ke direktori sementara → verifikasi checksum → tukar penunjuk
  → restart → pemeriksaan kesehatan 30 detik (service hidup, port mendengarkan, versi
  menjawab). Gagal → rollback otomatis + laporan.
- `rc upgrade` memakai lock file; dua proses bersamaan ditolak.

## 11. Pengujian dan gerbang kualitas

Prinsip: setiap perilaku punya tes; setiap jalur gagal diuji; rilis diblokir bila merah.

### 11.1 Unit (Go, `-race`)

- Parsing/validasi/migrasi konfigurasi: table-driven + fuzz (`FuzzConfigParse`).
- Pemilihan versi untuk rollback; perbandingan versi SemVer.
- Verifikasi checksum: cocok, diubah, terpotong, hilang.
- Token, perbandingan constant-time, rate limiter, timeout idle (jam palsu).
- Sanitasi alias; parsing alamat/IP; deteksi alamat tailnet (di balik interface).
- Logika PTY di balik interface agar dapat dites dengan PTY palsu.

### 11.2 Integrasi (container Ubuntu 22.04; runner Windows terpisah)

- Instalasi dua kali → snapshot filesystem+service harus identik (idempotensi).
- install → uninstall → install bersih.
- Siklus upgrade → rollback memakai server rilis palsu (`httptest`).
- Artefak korup / checksum salah → batal, state tidak berubah.
- Proses service dimatikan paksa → hidup lagi.
- Konkurensi: dua `rc upgrade` bersamaan → satu ditolak lock.
- Migrasi konfigurasi dari versi lama.

### 11.3 Terminal end-to-end (container)

- Sesi PTY nyata: `echo`, Ctrl-C, resize.
- Token salah → 401; percobaan berlebih → 429; idle → sesi ditutup.
- Putus tanpa close → sesi dibersihkan (tanpa kebocoran proses).
- Paste sangat besar dan byte non-UTF8 tidak menjatuhkan server.
- `ss -ltn` membuktikan hanya `127.0.0.1` yang mendengarkan.

### 11.4 Windows

- Pester: install, uninstall, idempotensi, keberadaan service, penanganan non-admin.
- Smoke test ConPTY.

### 11.5 E2E nyata (manual terskrip, per rilis)

Layar sungguhan tidak dapat diandalkan di CI. Checklist penerimaan di `docs/qa/`:
Android→Linux dan Android→Windows lewat LAN, USB (tethering), dan Tailscale; clipboard;
transfer file; terminal lewat tunnel. Hasil dicatat sebelum tag.

### 11.6 Uji kegagalan wajib

Tidak ada jaringan saat install · checksum tidak cocok · unduhan terputus · disk penuh ·
dijalankan bukan root · versi lama terpasang · `config_version` dari masa depan ·
Tailscale tidak ada/belum login · dua upgrade bersamaan · konfigurasi rusak/terpotong.

### 11.7 Gerbang statis dan kualitas

`go vet`, `staticcheck`, `golangci-lint`, `govulncheck`, `shellcheck`, `shfmt -d`,
PSScriptAnalyzer, `docker compose config`, `gitleaks`, header lisensi AGPL (REUSE), cek
tautan dokumen, `systemd-analyze verify` pada unit file.

Coverage minimal **80%** untuk paket logika; lapisan OS-shim dikecualikan tetapi daftarnya
tertulis di `docs/TESTING.md`. Tidak ada `sleep` di tes; gunakan helper tunggu-dengan-batas.

### 11.8 Aturan bug

Setiap bug didahului tes yang gagal, lalu diperbaiki. Tes itu tinggal sebagai penjaga
permanen dan disebut di pesan commit.

### 11.9 Gerbang rilis

Tag hanya jika: seluruh CI hijau, checklist manual ditandatangani di PR, siklus
upgrade→rollback diuji, dan `CHANGELOG.md` diperbarui.

## 12. CodeGraph

- Setelah setiap perubahan kode: `codegraph sync -q` (inkremental; dirancang untuk hook).
- Setelah menambah/memindah banyak berkas atau refactor besar: `codegraph index`.
- Hook `pre-commit` memanggil `codegraph sync -q`, dipasang via `core.hooksPath`.
- Definisi selesai tiap tugas: tes hijau **dan** `codegraph status` bersih.
- CI membangun indeks dari nol untuk memastikan seluruh kode dapat diparse.
- `.codegraph/` bersifat lokal (sudah di-gitignore oleh CodeGraph) — tidak di-commit.

## 13. Keamanan

- RustDesk: password permanen per device, dibuat installer dan ditampilkan `rc status`;
  akses lewat tailnet dibatasi ACL Tailscale.
- Terminal: token 32 byte, constant-time, rate limit, idle timeout, bind localhost default.
- Relay (nanti): kunci `hbbs`/`hbbr` disimpan di `.env` yang tidak di-commit.
- Log tidak memuat token, password, atau isi sesi terminal.
- `rc doctor --bundle` mengumpulkan diagnostik dengan redaksi untuk laporan bug.

## 14. Tahapan

1. **0.1.0** — kerangka repo, `VERSION`/`deps.lock`, `rc` (up/status/doctor/logs), installer
   Linux+Windows, RustDesk terkunci, panduan koneksi LAN/USB/Tailscale. Keluaran: layar dan
   input berfungsi dari Android ke host Linux dan Windows.
2. **0.2.0** — `kendali-term` + integrasi tunnel + auth, UI web terminal.
3. **0.3.0** — CI lengkap, uji upgrade/rollback, dokumentasi dan runbook, `rc doctor --bundle`.
4. **Nanti** — relay sendiri (gratis tier VPS), launcher bermerek, akun/izin untuk
   membantu device orang lain.

## 15. Kriteria penerimaan Tahap 1

1. Instalasi satu perintah berhasil di Linux (Pop!\_OS 22.04) dan Windows 10/11.
2. Menjalankan installer dua kali tidak mengubah keadaan.
3. `rc status` mencetak alamat LAN, USB, tailnet, dan token terminal.
4. Android dapat melihat layar dan mengendalikan input lewat LAN, USB (tethering), dan
   Tailscale.
5. `rc upgrade` lalu `rc rollback` berhasil tanpa merusak konfigurasi; kegagalan kesehatan
   memicu rollback otomatis.
6. `rc doctor` mendiagnosis dengan benar: Tailscale mati, service mati, checksum salah.
7. Seluruh tes CI hijau, coverage ≥80% pada paket logika.
8. `codegraph status` bersih setelah pekerjaan selesai.

## 16. Yang diserahkan ke builder

- Detail tata letak UI `rc status` dan format pesan `rc doctor`.
- Pemilihan pustaka PTY/ConPTY dan pustaka web terminal.
- Nama port default dan struktur internal paket Go selama tetap sesuai §8.4.
- Cara memasang hook `pre-commit` di mesin pengembang.
- Bahasa dokumentasi (disarankan Indonesia, istilah teknis tetap Inggris).
