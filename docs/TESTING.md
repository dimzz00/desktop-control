# Pengujian

- Unit Go: `go test -race ./...`
- Coverage gate: `make coverage` (min 80%, kecuali `internal/platform/exec.go`).
- Lint: `make lint` (golangci-lint, staticcheck).
- Lisensi: `make license` (SPDX header).
- Uji container: `bash tests/idempotency.sh <archive>` di ubuntu:22.04.
- Windows: Pester di `host/windows/tests`.

Tidak ada `sleep` di tes; gunakan helper tunggu-dengan-batas.
