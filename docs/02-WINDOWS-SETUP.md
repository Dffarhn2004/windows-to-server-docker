# 02 — Persiapan Windows 11 Home

1. Buka PowerShell as Administrator, periksa virtualisasi tersedia di BIOS/UEFI.
2. Install WSL2 sesuai panduan Microsoft. Contoh:

```powershell
wsl --install --no-distribution
wsl --update
wsl --status
```

Restart Windows jika diminta. `--no-distribution` berarti tidak memasang Ubuntu WSL terpisah; Docker Desktop akan memakai WSL backend. Virtualisasi tetap dipakai **secara internal** oleh WSL2.

3. Install Docker Desktop Windows dan pilih **Linux containers / WSL2 backend**.
4. Jalankan Docker Desktop, verifikasi:

```powershell
docker version
docker compose version
docker info
```

5. Buka Docker Desktop settings: aktifkan startup **when you sign in** (bukan layanan headless before login). Matikan Windows sleep ketika server dipakai. Jangan tutup laptop jika closing lid membuat sleep.
6. Simpan proyek di path pendek misalnya `C:\homelab`. Untuk heavy I/O/volume/DB sebaiknya gunakan Docker named volumes, bukan mount bind directory Windows untuk data intensif.

## Resource planning
Untuk laptop 16 GB RAM mulailah dengan dua demo saja; WAHA + 9Router + DB ditambahkan bertahap sambil memantau `docker stats`, task manager, storage. Koneksi internet, laptop sleep, dan Windows Update tetap single point of failure.

## Error umum
- `docker info` gagal: Docker Desktop belum aktif, engine Linux belum dipilih.
- WSL error: update WSL, restart, cek virtualization BIOS.
- Error bind mount: folder belum di-share/akses path tidak tepat.
- Docker Desktop tidak aktif setelah reboot: user Windows harus login; jangan klaim uptime VPS.

Rujukan: https://docs.docker.com/desktop/features/wsl/ dan https://docs.docker.com/desktop/setup/install/windows-install/
