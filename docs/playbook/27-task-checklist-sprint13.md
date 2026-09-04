# Detailed Task Checklist — Sprint 13
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 13: Extract First Service — Milestone 13, Transisi ke Phase C)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `03-adr.md` (ADR-001 Monorepo, ADR-010), `07-hld.md` (§1.3 Phase C, §5 Service Extraction Plan), `04-learning-roadmap.md` (Milestone 13), `26-task-checklist-sprint12.md`
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen & Makna Sprint Ini

**Ini adalah sprint paling krusial secara pedagogis di seluruh proyek** (Learning Roadmap M13: "praktik nyata bagaimana rasanya memisahkan modul yang sudah dirancang dengan boundary tegas sejak Phase A, membuktikan — atau membantah — asumsi bahwa disiplin domain boundary sejak awal benar-benar memudahkan ekstraksi").

**Prasyarat**: Sprint 12 selesai (Phase B — Event-Driven Modular Monolith aktif). Domain yang diekstraksi harus benar-benar loosely coupled — bila ternyata tidak, sprint ini akan **menemukan itu secara langsung**, dan temuan tersebut sama berharganya dengan ekstraksi yang mulus.

**Sprint Goal**: Domain **Identity** (kandidat urutan pertama, HLD §5 — paling independen, tidak ada dependency masuk dari domain lain kecuali referensi ID) berjalan sebagai service Go independen (`services/identity-svc/`), dengan database sendiri, deployment/CI terpisah, dan monolith memanggilnya lewat REST.

**Prinsip kerja sprint ini** (LLD/HLD §1.3 Best Practice): **definisikan kontrak API/event stabil dulu, baru pindahkan kode** — bukan sebaliknya.

---

## EPIC 18: Service Extraction — Identity

### Feature 18.1: Validasi Ulang Boundary (Sebelum Memindahkan Apapun)

#### Task 18.1.1: Audit Dependency Masuk/Keluar Domain Identity

- **Deskripsi**: Sebelum ekstraksi, **verifikasi ulang** (bukan asumsi) bahwa domain Identity benar-benar tidak punya *hidden coupling* — HLD §1.3 mengingatkan ini adalah momen paling umum ditemukannya Distributed Monolith yang sebenarnya sudah ada sejak dalam monolith.
- **Acceptance Criteria**: Audit mencakup: (a) apakah ada foreign key dari tabel domain lain langsung ke `users`/`sessions` tanpa lewat referensi ID murni (FK memang wajar, tapi cek apakah ada **JOIN langsung lintas domain** di query manapun — pelanggaran §2.3 Playbook yang lolos review sebelumnya), (b) apakah ada shared transaction (satu `BEGIN...COMMIT` yang menulis ke tabel Identity DAN tabel domain lain sekaligus).
- **Definition of Done**: Laporan audit tertulis — daftar seluruh titik kontak Identity dengan domain lain, ditandai "aman untuk diekstraksi" atau "perlu refactor dulu".
- **Dependency**: Sprint 12 selesai
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] `grep`/code search: cari seluruh query yang JOIN tabel `users`/`sessions` dengan tabel domain lain secara langsung
- [ ] Cari seluruh `BEGIN...COMMIT` yang mencakup insert/update ke `users` bersamaan dengan tabel domain lain (kandidat: `WorkspaceService.Create` — Task 3.2.1 Sprint 3, insert `workspace`+`member`+`role`, apakah ada dependency ke `users` di transaksi yang sama selain FK biasa?)
- [ ] Tandai temuan: aman / perlu refactor
- [ ] **Bila ditemukan hidden coupling**: refactor DULU sebelum lanjut ke Feature 18.2 — laporkan ke pengguna sebagai konflik (sesuai pola proyek ini di ADR-007/SRS §6) sebelum melanjutkan

---

### Feature 18.2: Definisikan Kontrak API Identity (Sebelum Memindahkan Kode)

#### Task 18.2.1: Kontrak REST `identity-svc` — Internal API

- **Deskripsi**: Endpoint yang akan dipanggil monolith untuk operasi identity (register, login, verify token, dst.) — **kontrak ini didefinisikan dan disepakati dulu**, sebelum satu baris kode dipindahkan (HLD §1.3 Best Practice).
- **Acceptance Criteria**: Kontrak mencakup seluruh endpoint yang sudah ada di API Specification §1 (`/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/logout-all`, `/auth/sessions`), plus **satu endpoint baru internal-only**: `POST /internal/v1/tokens/verify` (dipanggil Auth Middleware monolith untuk validasi token — lihat Task 18.4.2).
- **Definition of Done**: Dokumen kontrak (`docs/services/identity-svc-contract.md`) ditinjau dan disepakati — **amandemen kecil terhadap `10-api-specification.md`** menandai endpoint mana yang sekarang "dilayani oleh identity-svc" (secara eksternal tidak berubah dari sisi client, hanya catatan internal).
- **Dependency**: Task 18.1.1 (audit bersih)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis `docs/services/identity-svc-contract.md` — daftar endpoint, request/response (reuse dari API Spec §1, tidak didesain ulang)
- [ ] Definisikan endpoint internal `POST /internal/v1/tokens/verify` — request `{ token }`, response `{ valid, user_id, expires_at }`
- [ ] Update `10-api-specification.md` dengan catatan "dilayani oleh identity-svc sejak Sprint 13"
- [ ] Tinjau kontrak — pastikan tidak ada breaking change ke client eksternal (frontend)

---

### Feature 18.3: Struktur `services/identity-svc/`

#### Task 18.3.1: Inisialisasi Module & Struktur Folder

- **Deskripsi**: Sesuai Playbook §1.4/§19 — `services/identity-svc/` dengan `go.mod` sendiri, ditambahkan ke `go.work`.
- **Acceptance Criteria**: Struktur folder identik dengan `apps/api/internal/identity/` (Clean Architecture layer yang sama), namun sebagai module independen.
- **Definition of Done**: `go build ./...` dari root tetap sukses dengan module baru ini terdaftar di `go.work`.
- **Dependency**: Task 18.2.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Buat `services/identity-svc/` dengan struktur `cmd/server/`, `internal/domain/`, `internal/application/`, `internal/infrastructure/`, `internal/interface/http/`
- [ ] `go mod init github.com/<org>/nexus/services/identity-svc`
- [ ] `go work use ./services/identity-svc`
- [ ] Verifikasi `go build ./...` dari root tetap sukses

#### Task 18.3.2: **Pindahkan** (Bukan Salin) Kode Domain Identity

- **Deskripsi**: Git `mv` kode dari `apps/api/internal/identity/` ke `services/identity-svc/internal/` — perpindahan folder ini sendiri adalah **artifact pembelajaran** yang terlihat di git history (Playbook §1.4 rationale, dikutip ulang di sini karena inilah momennya benar-benar terjadi).
- **Acceptance Criteria**: Kode dipindahkan (bukan diduplikasi) — `apps/api/internal/identity/` **dihapus** setelah pemindahan, tidak ada dua salinan kode identity yang hidup berdampingan.
- **Definition of Done**: `git log --follow` pada file yang dipindahkan menunjukkan history sebelum-sesudah pindah tetap terlacak (memakai `git mv`, bukan hapus+buat baru).
- **Dependency**: Task 18.3.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] `git mv apps/api/internal/identity/* services/identity-svc/internal/`
- [ ] Sesuaikan import path (module path berubah dari `apps/api` ke `services/identity-svc`)
- [ ] `sqlc.yaml` baru khusus `identity-svc` (query yang sama, tapi generate di lokasi baru)
- [ ] Verifikasi `git log --follow` masih terhubung ke history lama
- [ ] Hapus folder lama `apps/api/internal/identity/` sepenuhnya (bukan dibiarkan sebagai duplikat "untuk jaga-jaga")

---

### Feature 18.4: Database-per-Service untuk Identity

#### Task 18.4.1: Migrasi Database Terpisah `nexus_identity_dev`/`nexus_identity_prod`

- **Deskripsi**: Sesuai ADR-010/HLD §1.4 — Database-per-Service dimulai dari domain pertama yang diekstraksi.
- **Acceptance Criteria**: Tabel `users`, `sessions` **dipindahkan** ke database baru `nexus_identity_*`; database monolith utama (`nexus_dev`) **tidak lagi** memiliki tabel ini.
- **Definition of Done**: Migrasi data dijalankan (dump dari `nexus_dev.users`/`sessions` → restore ke `nexus_identity_dev`), diverifikasi tidak ada data hilang (`SELECT COUNT(*)` sebelum-sesudah cocok).
- **Dependency**: Task 18.3.2
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 4 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Buat database `nexus_identity_dev` baru di PostgreSQL (bisa instance sama untuk development, dipisah fisik nanti di produksi bila diperlukan — cukup logical separation dulu)
- [ ] Jalankan migrasi `users`/`sessions` di database baru (migration files dipindahkan bersama kode ke `services/identity-svc/migrations/`)
- [ ] Dump data dari `nexus_dev` (`pg_dump --table=users --table=sessions`), restore ke `nexus_identity_dev`
- [ ] Verifikasi `COUNT(*)` cocok sebelum-sesudah
- [ ] **Hapus** tabel `users`/`sessions` dari `nexus_dev` **setelah** verifikasi identity-svc berfungsi penuh (Task 18.7.1) — jangan dihapus prematur sebelum service baru terbukti bekerja

#### Task 18.4.2: Adaptasi Foreign Key Lintas Database (Tantangan Nyata Database-per-Service)

- **Deskripsi**: **Ini adalah titik di mana kompleksitas Database-per-Service benar-benar terasa** — tabel lain di monolith (`workspaces.owner_id`, `members.user_id`, `messages.author_id`, dst.) memiliki `REFERENCES users(id)`, namun `users` sekarang di database **berbeda**. PostgreSQL **tidak mendukung foreign key lintas database**.
- **Acceptance Criteria**: Constraint `REFERENCES users(id)` **dihapus** dari seluruh tabel monolith yang mereferensikan `users` — `user_id`/`owner_id`/`author_id` dst. tetap ada sebagai kolom UUID biasa, **tanpa** FK constraint, integritas referensial menjadi tanggung jawab aplikasi (bukan lagi database).
- **Definition of Done**: Migrasi yang menghapus FK constraint tersebut ditulis dan diverifikasi; test yang sebelumnya bergantung pada FK constraint (mis. "insert dengan `owner_id` tidak valid ditolak database" — Task 3.1.1 Sprint 3) **direvisi** menjadi validasi di service layer (panggil `identity-svc` untuk verifikasi user exists, atau terima eventual consistency untuk kasus non-kritikal).
- **Dependency**: Task 18.4.1
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 4 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Identifikasi seluruh FK constraint ke `users(id)` di skema monolith (`workspaces.owner_id`, `members.user_id`, `messages.author_id`, `sessions` sudah pindah, dll. — grep `REFERENCES users`)
- [ ] Tulis migrasi `DROP CONSTRAINT` untuk masing-masing (expand-contract tidak diperlukan di sini karena ini pelonggaran constraint, bukan pengetatan)
- [ ] **Keputusan desain eksplisit**: operasi yang butuh verifikasi user exists sebelum insert (mis. `WorkspaceService.Create` dengan `owner_id`) memanggil `identity-svc` internal endpoint (`GET /internal/v1/users/{id}/exists`) — **synchronous call**, bukan event, karena dibutuhkan sebelum commit (masih dalam Phase C, REST antar service tetap dipakai untuk kasus butuh jawaban langsung, sesuai HLD §4)
- [ ] Revisi test Sprint 3 yang bergantung pada FK constraint — ganti dengan test yang memverifikasi service-level check
- [ ] Test: insert dengan `owner_id` tidak valid → tetap ditolak, tapi sekarang lewat pengecekan aplikasi (call ke identity-svc), bukan database constraint

---

### Feature 18.5: HTTP Client Internal (Monolith → identity-svc)

#### Task 18.5.1: `IdentityServiceClient` — Adapter di Monolith

- **Deskripsi**: Monolith butuh mengganti seluruh pemanggilan langsung `AuthService`/`UserRepository` (in-process, karena kode sudah pindah) dengan HTTP client ke `identity-svc`.
- **Acceptance Criteria**: Interface `IdentityServiceClient` didefinisikan di `apps/api/internal/platform/` (bukan di domain manapun — ini adalah adapter infrastruktur lintas service), diimplementasikan sebagai HTTP client dengan **timeout eksplisit dan circuit breaker** (HLD §1.3 Best Practice — "network is reliable" adalah asumsi yang tidak lagi valid).
- **Definition of Done**: Unit test dengan mock HTTP server: timeout dihormati, circuit breaker terbuka setelah N kegagalan berturut (mencegah cascading failure ke monolith bila identity-svc down).
- **Dependency**: Task 18.2.1
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 3.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `apps/api/internal/platform/identityclient/client.go`
- [ ] Timeout eksplisit per request (mis. 3 detik) — **JANGAN** biarkan default `http.Client{}` tanpa timeout (kesalahan umum yang menyebabkan cascading failure)
- [ ] Circuit breaker sederhana (library `sony/gobreaker` atau implementasi minimal sendiri) — terbuka setelah 5 kegagalan berturut, half-open setelah cooldown
- [ ] Unit test: timeout dihormati, circuit breaker terbuka & pulih

#### Task 18.5.2: Refactor Auth Middleware — Verify Token via `identity-svc`

- **Deskripsi**: Auth Middleware (Sprint 2, Task 2.5.2) sebelumnya verifikasi JWT **lokal** (karena secret ada di proses yang sama) — sekarang harus memanggil `identity-svc` internal endpoint (Task 18.2.1).
- **Acceptance Criteria**: **Pertimbangan trade-off eksplisit**: memanggil `identity-svc` di **setiap** request terproteksi menambah latensi nyata (network round-trip tambahan untuk hampir semua endpoint). Solusi: JWT **tetap diverifikasi lokal** di monolith (signature check tidak butuh network call, hanya butuh public key/shared secret) — `identity-svc` **hanya** dipanggil untuk operasi yang butuh state terkini (login, refresh, revoke), bukan untuk setiap request biasa.
- **Definition of Done**: Test: endpoint terproteksi biasa (mis. kirim pesan) **tidak** memanggil `identity-svc` sama sekali (hanya verifikasi signature JWT lokal); endpoint yang butuh cek revocation real-time (jarang, bila ada) baru memanggil `identity-svc`.
- **Dependency**: Task 18.5.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] **Keputusan didokumentasikan eksplisit**: JWT verification tetap lokal (stateless, tidak butuh network call) — ini **bukan** penyimpangan dari filosofi service extraction, karena JWT secara desain memang dibuat untuk stateless verification (ADR/Security Design sudah memilih JWT justru karena alasan ini)
- [ ] Auth Middleware tetap seperti Sprint 2 (verify signature lokal), **tidak diubah**
- [ ] Hanya endpoint yang eksplisit butuh data sesi terkini (mis. daftar sesi aktif) yang memanggil `IdentityServiceClient`
- [ ] Test: regression Sprint 2 (auth middleware) tetap lolos tanpa modifikasi signifikan

---

### Feature 18.6: CI/CD Terpisah untuk `identity-svc`

#### Task 18.6.1: Path-Based Trigger CI/CD

- **Deskripsi**: Sesuai Playbook §6.1 (Phase C: "path-based trigger, hanya build/deploy service yang berubah").
- **Acceptance Criteria**: Perubahan di `services/identity-svc/**` men-trigger pipeline khusus (`ci-identity-svc.yml`), terpisah dari `ci.yml` monolith; perubahan di `apps/api/**` **tidak** men-trigger build identity-svc.
- **Definition of Done**: Test: PR yang hanya mengubah `apps/web/` tidak men-trigger build/test Go apapun (baik monolith maupun identity-svc) — verifikasi path filter bekerja presisi.
- **Dependency**: Task 18.3.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Buat `.github/workflows/ci-identity-svc.yml` dengan `paths: ['services/identity-svc/**']`
- [ ] Buat `.github/workflows/build-and-push-identity-svc.yml`, `deploy-identity-svc.yml` (independen dari pipeline monolith)
- [ ] Update `ci.yml` monolith dengan `paths-ignore` untuk `services/**` (agar tidak tumpang tindih)
- [ ] Test: PR mengubah masing-masing path (`apps/web/`, `apps/api/`, `services/identity-svc/`) men-trigger pipeline yang tepat saja

---

### Feature 18.7: Deployment & Verifikasi End-to-End

#### Task 18.7.1: Deploy `identity-svc` ke Docker Compose (Service Terpisah)

- **Deskripsi**: Update `docker-compose.yml` — `identity-svc` sebagai container terpisah dengan database sendiri.
- **Acceptance Criteria**: Traefik routing `/api/v1/auth/*` diarahkan ke `identity-svc` (bukan `apps/api`) — **amandemen konfigurasi Traefik**, transparan bagi client (URL path tidak berubah, hanya tujuan routing internal).
- **Definition of Done**: `docker compose up` menjalankan `identity-svc` + `nexus_identity_dev` sebagai container terpisah; request ke `/api/v1/auth/login` benar-benar diproses oleh `identity-svc` (verifikasi via log/trace, bukan asumsi).
- **Dependency**: Task 18.4.1, Task 18.6.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Update `docker-compose.yml` — service `identity-svc` + `postgres-identity` terpisah
- [ ] Update label Traefik routing `/api/v1/auth/*` → `identity-svc`
- [ ] Verifikasi via log: request login benar-benar diproses `identity-svc`, bukan `apps/api`

#### Task 18.7.2: Regression Test Total — Sekaligus Pembuktian Kualitas Boundary Sprint 1-12

- **Deskripsi**: **Ini adalah gerbang paling penting di seluruh proyek sejauh ini** — membuktikan (atau membantah) hipotesis inti Learning Roadmap M13.
- **Acceptance Criteria**: **Seluruh** regression suite Sprint 1-12 dijalankan ulang terhadap topologi baru (monolith + identity-svc terpisah). Bila lolos tanpa modifikasi test yang signifikan (di luar yang sudah diantisipasi di Task 18.4.2) → **hipotesis boundary domain yang disiplin sejak Phase A terbukti benar**. Bila banyak test gagal/butuh modifikasi besar → **temuan itu sendiri harus didokumentasikan sebagai pembelajaran**, bukan disembunyikan.
- **Definition of Done**: Laporan eksplisit (`docs/reports/sprint13-extraction-findings.md`) — apa yang berjalan mulus, apa yang ternyata butuh perbaikan tak terduga, dan pelajaran untuk ekstraksi domain berikutnya (Notification, urutan ke-2 di HLD §5).
- **Dependency**: Seluruh task Epic 18
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 4 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Jalankan seluruh test suite Sprint 1-12 terhadap topologi baru
- [ ] Catat setiap kegagalan/modifikasi yang dibutuhkan — **jangan diam-diam diperbaiki tanpa dicatat**, ini adalah data pembelajaran paling berharga di sprint ini
- [ ] Tulis laporan temuan jujur (termasuk bila ternyata ada hidden coupling yang lolos dari Task 18.1.1 audit awal)
- [ ] Update `docs/AGENTS.md` §7 — **Sprint 13 selesai. Fase Arsitektur berubah menjadi "Phase C — Hybrid Architecture"**, `services/identity-svc/` resmi berisi service pertama

---

## Catatan Frontend *(amandemen retroaktif)*

Sama seperti Sprint 12, ekstraksi `identity-svc` seharusnya **transparan** bagi frontend — URL path (`/api/v1/auth/*`) tidak berubah (Traefik yang mengarahkan ke tujuan berbeda secara internal, Task 18.7.1), dan frontend tidak tahu maupun perlu tahu bahwa permintaannya sekarang diproses service terpisah.

**Namun**, ini adalah sprint dengan risiko regresi tertinggi sejauh ini (dicatat eksplisit di §Risiko backend) — sehingga frontend **wajib** menjalankan regression penuh, bukan asumsi "pasti transparan":

- [ ] Jalankan **seluruh** regression Playwright Sprint 1-12 terhadap topologi baru (monolith + `identity-svc`)
- [ ] Perhatian khusus: alur auth (register, login, refresh, logout, logout-all, device management — Task 3.2.1, 3.3.1, 3.4.1, 3.4.2 frontend Sprint 1-2) karena inilah domain yang benar-benar berpindah lokasi eksekusi
- [ ] **Bila Task 18.7.2 backend menemukan kegagalan** (dicatat di laporan `sprint13-extraction-findings.md`) yang berdampak pada kontrak API — laporkan sebagai temuan bersama, bukan tambal sendiri di frontend tanpa sinkronisasi dengan perubahan backend

---

## Ringkasan Keputusan

1. Sprint 13 mencakup **1 Epic, 7 Feature, 12 task** (murni backend), menandai transisi Phase B → **Phase C — Hybrid Architecture**. *(Amandemen retroaktif: tidak ada task frontend baru — ekstraksi service seharusnya transparan bagi client, namun regression penuh Sprint 1-12 wajib dijalankan mengingat ini sprint berisiko regresi tertinggi.)*
2. Urutan kerja **kontrak dulu, baru pindah kode** (Feature 18.2 sebelum 18.3) dipatuhi persis sesuai Best Practice HLD §1.3.
3. **Temuan arsitektural signifikan**: PostgreSQL tidak mendukung FK lintas database — Database-per-Service berarti **melepas** integritas referensial level-database untuk relasi lintas service (Task 18.4.2), digantikan verifikasi level-aplikasi. Ini bukan detail kecil — ini adalah salah satu trade-off paling fundamental dari Database-per-Service pattern, dan proyek ini sekarang benar-benar merasakannya, bukan hanya membacanya di teori.
4. JWT verification **tetap lokal** di monolith (Task 18.5.2) — keputusan sadar bahwa tidak semua hal harus memanggil service yang diekstraksi; JWT secara desain memang stateless untuk alasan ini.
5. Task 18.7.2 dirancang eksplisit untuk **mencatat kejujuran hasil**, bukan hanya "pastikan hijau" — kegagalan yang ditemukan sama berharganya dengan keberhasilan untuk Learning Objective sprint ini.

## Trade-off yang Diterima

- Menghapus FK constraint lintas database (Task 18.4.2) berarti kemungkinan **data inconsistency** yang sebelumnya dicegah database kini bergantung pada disiplin aplikasi — diterima sebagai konsekuensi inheren Database-per-Service, dimitigasi dengan verifikasi service-level di titik-titik kritikal (create workspace, dst.), namun tidak akan sekuat FK constraint asli.
- Circuit breaker (Task 18.5.1) menambah kompleksitas kode yang sebelumnya tidak ada (in-process call tidak pernah "gagal" secara network) — diterima sebagai kebutuhan riil begitu domain benar-benar jadi service terpisah.

## Risiko Arsitektur

- Task 18.4.2 adalah **risiko tertinggi** di sprint ini — kemungkinan besar akan menemukan lebih banyak titik yang bergantung pada FK constraint `users` daripada yang terlihat dari audit awal (Task 18.1.1), karena constraint database seringkali menjadi "pengaman tak terlihat" yang baru terasa hilang saat benar-benar dilepas.
- Bila Task 18.7.2 menemukan kegagalan signifikan (bukan hanya penyesuaian kecil), ini adalah sinyal bahwa Sprint 13 mungkin perlu diperpanjang di luar estimasi — konsisten dengan peringatan yang sama yang dicatat di Sprint 11 §Risiko.

## Technical Debt yang Sengaja Diterima

- Database `nexus_identity_dev` masih di instance PostgreSQL yang sama dengan `nexus_dev` (pemisahan logical, bukan fisik) — pemisahan fisik penuh (instance/server terpisah) ditunda ke Deployment Architecture Tahap 4/5 bila benar-benar dibutuhkan skalanya, bukan dipaksakan di development.
- Circuit breaker (Task 18.5.1) memakai implementasi sederhana — belum ada dashboard observability untuk memonitor state circuit breaker (open/half-open/closed), akan datang bersamaan Milestone 15.

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah keputusan **JWT verification tetap lokal** (bukan selalu memanggil `identity-svc`) sudah sesuai pemahaman Anda tentang kapan sebuah operasi *benar-benar* butuh memanggil service yang diekstraksi vs kapan tidak?
2. Apakah Anda memahami dan menerima konsekuensi **hilangnya FK constraint lintas database** (Task 18.4.2) sebagai trade-off yang inheren dari Database-per-Service — bukan sesuatu yang bisa dihindari selama tetap memakai PostgreSQL untuk masing-masing service?
3. Sprint 13 adalah sprint dengan ketidakpastian tertinggi sejauh ini (hasil sungguhan baru diketahui setelah dieksekusi, terutama Task 18.7.2). Apakah Anda ingin saya lanjut menyiapkan **Sprint 14** (Hybrid Architecture — trace propagation, API Gateway routing penuh), atau berhenti dulu menunggu Sprint 13 benar-benar dieksekusi dan temuannya diketahui (mengingat temuan itu bisa memengaruhi bagaimana Sprint 14 seharusnya didesain)?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 13: 1 Epic, 7 Feature, 12 task. Mengekstraksi domain Identity sebagai service pertama, menandai transisi Phase B → Phase C, dengan penekanan eksplisit pada kejujuran pelaporan hasil ekstraksi |
| 1.1.0 | Amandemen | Ditambahkan Catatan Frontend — dikonfirmasi tidak ada task baru diperlukan (ekstraksi seharusnya transparan), namun regression penuh Sprint 1-12 wajib mengingat risiko regresi tertinggi di sprint ini |
