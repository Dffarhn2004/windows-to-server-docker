# Prompt untuk coding agent berikutnya (salin seluruh bagian ini)

Saya ingin melanjutkan implementasi homelab berdasarkan repository/ZIP ini, **bukan mendesain ulang dari nol**. Target host saya Windows 11 Home, Docker Desktop dengan WSL2, tanpa VirtualBox. Saya ingin subdomain dinamis via wildcard Cloudflare Tunnel + Traefik labels, dua test apps dahulu, lalu WAHA dan 9Router di private Docker network. 

Baca dan patuhi `AGENTS.md`, `README.md`, `docs/00-HANDOFF.md`, `docs/01-ARCHITECTURE.md`, dan `docs/02-WINDOWS-SETUP.md` s.d. `docs/05-ADD-NEW-PROJECT.md`. Periksa file YAML, scripts, kebutuhan host, dan akses Cloudflare; jangan menebak status mesin atau secrets. **Jangan expose database, WAHA, atau 9Router ke publik.**

Mulailah dengan:
1. Audit environment yang benar-benar tersedia (Docker Desktop, WSL2, file `.env` yang harus tetap rahasia, connectivity Cloudflare).
2. Jalankan validasi Compose, jelaskan perbaikan jika ada error.
3. Deploy infrastruktur + dua aplikasi tes dan verifikasi route internal, lalu browser publik.
4. Catat secara eksplisit apa yang sudah **dieksekusi** vs hanya **direncanakan**.
5. Bila semua lulus, lanjutkan integrasi service secara bertahap tanpa membuka port host dan dengan backup.

Tanyakan hanya data yang tidak mungkin diketahui dari mesin atau file, terutama domain milik saya dan apakah Cloudflare nameserver sudah aktif. Jangan meminta token untuk ditempel ke chat: arahkan saya mengisi `.env` lokal sendiri.

Terakhir, perbarui `docs/00-HANDOFF.md` dengan status aktual dan langkah lanjutan untuk agent selanjutnya.
