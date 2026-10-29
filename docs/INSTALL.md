# Instalasi kendali 0.1.0

## Linux

```sh
sudo ./host/linux/install.sh --archive kendali-0.1.0-linux-amd64.tar.gz
```

Tanpa jaringan, gunakan `--archive`. Untuk melihat langkah tanpa menulis: `--dry-run`.

## Windows (PowerShell sebagai Administrator)

```powershell
.\host\windows\install.ps1 -Archive kendali-0.1.0-windows-amd64.zip
```

## Verifikasi

```sh
rc status
rc doctor
```
