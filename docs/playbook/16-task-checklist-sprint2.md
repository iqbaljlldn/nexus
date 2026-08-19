# Detailed Task Checklist — Sprint 2

## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 2: Authentication Lengkap)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `14-sprint-planning.md` (Sprint 2), `06-srs.md` (§2.1 FR-AUTH), `10-api-specification.md` (§1), `11-security-design.md` (§2, §6, §9), `15-task-checklist-sprint1.md` (kelanjutan Epic 2)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen & Prasyarat

Sesuai **Rolling Wave Planning** (`14-sprint-planning.md` §0), dokumen ini mendetailkan **Sprint 2 saja**. Melanjutkan **Epic 2: Authentication** dari Sprint 1 (Feature 2.1-2.3 sudah selesai: migrasi identity, register, testing dasar).

**Prasyarat**: Seluruh task Sprint 1 (`15-task-checklist-sprint1.md`) berstatus Done — khususnya Task 2.2.6 (Wiring Register) dan Task 2.3.2 (Test Suite Sprint 1 hijau). Bila belum, task Sprint 2 di bawah (khususnya Feature 2.5 Middleware) akan gagal karena bergantung pada struktur `AuthService` yang sudah ada.

**Sprint Goal** (dari `14-sprint-planning.md` Sprint 2): Alur autentikasi penuh (login, refresh rotation, logout-all, session management) berfungsi end-to-end, dengan refresh token disimpan sebagai HttpOnly Cookie + CSRF protection (keputusan final Security Design §2/§6).

---

## EPIC 2 (lanjutan): Authentication — Login, Session, & Token Lifecycle

### Feature 2.4: Login

#### Task 2.4.1: Repository — FindUserByIdentifier (Email atau Username)

- **Deskripsi**: Perluas `UserRepository` (dibuat Sprint 1) dengan method pencarian fleksibel email/username, sesuai FR-AUTH-03.
- **Acceptance Criteria**: Fungsi mendeteksi tipe identifier otomatis (mengandung `@` → treat sebagai email, query kolom `email` CITEXT; selain itu → query kolom `username` CITEXT).
- **Definition of Done**: Unit test lolos untuk kasus login via email dan via username terhadap user yang sama.
- **Dependency**: Task 2.2.3 (Sprint 1 — UserRepository dasar)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Tambah query sqlc `FindUserByEmailOrUsername` (bila belum ada dari Sprint 1, lengkapi; bila sudah, verifikasi ulang)
- [x] Implementasi logic deteksi tipe identifier di repository/service layer
- [x] Unit test: login via email, login via username, identifier tidak ditemukan

#### Task 2.4.2: Service Layer — AuthService.Login

- **Deskripsi**: Orkestrasi verifikasi password (Argon2id, dibuat Sprint 1 Task 2.2.2) dan pembuatan sesi baru.
- **Acceptance Criteria**: Login sukses menghasilkan access token + refresh token; login gagal (identifier tidak ada ATAU password salah) mengembalikan error **generik yang sama** (mitigasi user enumeration, SRS FR-AUTH-03).
- **Definition of Done**: Unit test memverifikasi pesan error identik untuk kedua skenario gagal (perbandingan string/error code, bukan hanya "sama-sama gagal").
- **Dependency**: Task 2.4.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Implementasi `AuthService.Login(ctx, identifier, password, deviceInfo) (*TokenPair, error)`
- [x] Pastikan path "user tidak ditemukan" dan "password salah" mengembalikan sentinel error yang SAMA (`ErrInvalidCredentials`)
- [x] Unit test: login sukses, kedua skenario gagal menghasilkan error identik

#### Task 2.4.3: HTTP Handler — `POST /api/v1/auth/login`

- **Deskripsi**: Sesuai kontrak `10-api-specification.md` §1.
- **Acceptance Criteria**: Response 200 dengan access_token di body; refresh_token dikirim sebagai **HttpOnly Secure SameSite=Strict Cookie** (bukan di body — keputusan final Security Design §2), plus cookie kedua non-HttpOnly `csrf_token` (Security Design §6).
- **Definition of Done**: `curl -v` menunjukkan `Set-Cookie` dengan flag `HttpOnly; Secure; SameSite=Strict` untuk refresh token; response body TIDAK mengandung refresh token.
- **Dependency**: Task 2.4.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Request DTO `{ identifier, password }` + validasi
- [x] Set cookie refresh token (HttpOnly, Secure, SameSite=Strict) via `http.SetCookie`
- [x] Set cookie `csrf_token` (non-HttpOnly, readable JS)
- [x] Response body hanya berisi `access_token`, `expires_in`
- [x] Test HTTP: verifikasi header `Set-Cookie` sesuai flag yang benar, body tidak bocorkan refresh token

---

### Feature 2.5: JWT Access Token & Auth Middleware

#### Task 2.5.1: JWT Signing & Verification Utility

- **Deskripsi**: Implementasi generic JWT helper di `pkg/jwt` (bukan domain-specific — sesuai Playbook §3.1).
- **Acceptance Criteria**: `Sign(claims) (string, error)` dan `Verify(token) (Claims, error)`; token berumur 15 menit (SRS FR-AUTH-04).
- **Definition of Done**: Unit test: token valid ter-verify, token expired ditolak, token dengan signature dipalsukan ditolak.
- **Dependency**: Task 1.1.4 (Sprint 1 — tooling dasar)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Implementasi `pkg/jwt/jwt.go` (`Sign`, `Verify`) memakai `NEXUS_API_JWT_SECRET` dari env (RULES.md §3 — jangan hardcode)
- [x] Claims minimal: `user_id`, `exp`, `iat`
- [x] Unit test: sign-verify round-trip, token expired, signature invalid

#### Task 2.5.2: Gin Auth Middleware

- **Deskripsi**: Middleware yang memvalidasi `Authorization: Bearer <token>`, menaruh `user_id` ke Gin context.
- **Acceptance Criteria**: Request tanpa token/token invalid → `401 UNAUTHORIZED` (envelope error standar); request valid → handler menerima `user_id` dari context.
- **Definition of Done**: Integration test: endpoint terproteksi menolak request tanpa token, menerima request dengan token valid.
- **Dependency**: Task 2.5.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Implementasi `internal/platform/middleware/auth.go`
- [x] Parse header `Authorization`, verify via `pkg/jwt`
- [x] Set `user_id` ke `gin.Context` (`c.Set("user_id", ...)`)
- [x] Response 401 dengan error code `UNAUTHORIZED` sesuai Error Code Catalog (API Spec §0)
- [x] Test: request tanpa header, header malformed, token expired, token valid

---

### Feature 2.6: Refresh Token Rotation

#### Task 2.6.1: Repository — SessionRepository Lengkap

- **Deskripsi**: Perluas `sessions` table access (dibuat skema Sprint 1) dengan method rotation.
- **Acceptance Criteria**: `RotateRefreshToken(ctx, oldTokenHash, newTokenHash) error` — menandai sesi lama `revoked` DAN membuat sesi baru dalam **satu transaksi database**.
- **Definition of Done**: Integration test: setelah rotate, token lama tidak dapat dipakai lagi (status `revoked` terverifikasi di DB).
- **Dependency**: Task 2.1.1 (Sprint 1 — migrasi `sessions`)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Query sqlc: `RevokeSession`, `CreateSession`, `FindSessionByTokenHash`
- [x] Implementasi `RotateRefreshToken` dalam satu DB transaction (`pgx.Tx`)
- [x] Integration test: rotate berhasil, token lama gagal dipakai setelah rotate

#### Task 2.6.2: Service & Handler — `POST /api/v1/auth/refresh` + CSRF

- **Deskripsi**: Endpoint refresh membaca cookie, validasi CSRF header, rotate token.
- **Acceptance Criteria**: Sesuai Security Design §6 — request tanpa header `X-CSRF-Token` yang cocok dengan cookie `csrf_token` ditolak `403 FORBIDDEN`; request valid menghasilkan access+refresh token baru (cookie baru di-set, cookie lama otomatis diganti).
- **Definition of Done**: Test HTTP: refresh tanpa CSRF header gagal, refresh dengan CSRF header cocok berhasil dan cookie baru berbeda dari sebelumnya.
- **Dependency**: Task 2.6.1
- **Estimasi Kesulitan**: Sedang-Tinggi
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Middleware/handler validasi double-submit cookie (`X-CSRF-Token` header vs cookie `csrf_token`)
- [x] Baca refresh token dari cookie HttpOnly (bukan dari body)
- [x] Panggil `AuthService.RefreshToken` → `RotateRefreshToken`
- [x] Set cookie refresh token BARU + csrf_token BARU
- [x] Test: tanpa CSRF header → 403; CSRF cocok → 200 dengan cookie baru; refresh token expired/revoked → 401

---

### Feature 2.7: Logout & Device Management

#### Task 2.7.1: `POST /api/v1/auth/logout-all`

- **Deskripsi**: Merevoke seluruh sesi milik user (FR-AUTH-05).
- **Acceptance Criteria**: Response 204; seluruh baris `sessions` milik user berstatus `revoked` setelahnya.
- **Definition of Done**: Integration test: buat 3 sesi (login 3x), logout-all, verifikasi ketiganya `revoked` dan refresh dengan token manapun gagal.
- **Dependency**: Task 2.6.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [ ] Query sqlc `RevokeAllSessionsByUserID`
- [ ] Handler + service method `LogoutAll`
- [ ] Hapus cookie refresh_token & csrf_token di response (`Max-Age=-1`)
- [ ] Test: 3 sesi aktif → logout-all → seluruh refresh gagal setelahnya

#### Task 2.7.2: `GET /api/v1/auth/sessions` — Daftar Sesi Aktif

- **Deskripsi**: FR-AUTH-06, Device Management.
- **Acceptance Criteria**: Mengembalikan daftar sesi (device/user agent, IP, created_at) milik user yang login, TIDAK menampilkan token hash.
- **Definition of Done**: Test HTTP: response tidak mengandung field token/hash apapun.
- **Dependency**: Task 2.6.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Should

**Subtask & Checklist**:

- [ ] Query sqlc `ListActiveSessionsByUserID`
- [ ] Handler, response DTO (exclude `refresh_token_hash`)
- [ ] Test: pastikan hash token tidak pernah muncul di response JSON

#### Task 2.7.3: `DELETE /api/v1/auth/sessions/{sessionId}` — Revoke Sesi Individual

- **Deskripsi**: User dapat mencabut satu sesi spesifik (mis. device yang hilang).
- **Acceptance Criteria**: Hanya pemilik sesi yang dapat menghapus sesinya sendiri; mencoba hapus sesi user lain → `403 FORBIDDEN`.
- **Definition of Done**: Test: user A tidak dapat menghapus sesi user B.
- **Dependency**: Task 2.7.2
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Should

**Subtask & Checklist**:

- [ ] Handler cek kepemilikan sesi sebelum revoke
- [ ] Query sqlc `RevokeSessionByID`
- [ ] Test: revoke sesi sendiri berhasil, revoke sesi orang lain ditolak

---

### Feature 2.8: Rate Limiting Login

#### Task 2.8.1: Redis Sliding Window Rate Limiter (Generic)

- **Deskripsi**: Implementasi Lua script sliding window sesuai LLD §2.8, sebagai `pkg/ratelimit` generic (dapat dipakai domain lain di sprint berikutnya).
- **Acceptance Criteria**: `Allow(ctx, key, window, limit) (bool, error)` atomik (satu `EVAL` call).
- **Definition of Done**: Unit test (memakai Redis test container/miniredis): request ke-6 dalam window ditolak saat limit=5.
- **Dependency**: Task 1.2.2 (Sprint 1 — Redis berjalan)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [ ] Tulis `rate_limit.lua` sesuai LLD §2.8
- [ ] Implementasi `pkg/ratelimit/limiter.go` — load & eksekusi script via `EVAL`
- [ ] Unit test: request ke-(limit+1) ditolak, request setelah window lewat diterima lagi

#### Task 2.8.2: Terapkan Rate Limit ke Login (Lockout Progresif)

- **Deskripsi**: Sesuai SRS §3.5 — 5 percobaan/15 menit, lockout progresif (5 menit → 15 menit → 1 jam).
- **Acceptance Criteria**: Percobaan login gagal ke-6 dalam window ditolak `429 RATE_LIMIT_EXCEEDED` dengan header `Retry-After`.
- **Definition of Done**: Integration test: 5x login gagal berturut-turut → percobaan ke-6 ditolak rate limit (bukan `401` biasa).
- **Dependency**: Task 2.8.1, Task 2.4.3
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [ ] Middleware/interceptor rate limit khusus endpoint login, key berbasis identifier (bukan IP saja — SRS §3.5)
- [ ] Implementasi lockout progresif (tracking jumlah lockout berturut sebelumnya, di Redis dengan key terpisah)
- [ ] Header `Retry-After` pada response 429
- [ ] Test: 5 gagal → percobaan ke-6 → 429; setelah window lewat → percobaan berikutnya normal lagi

---

### Feature 2.9: Integration Testing End-to-End Sprint 2

#### Task 2.9.1: Test Skenario Penuh — Register → Login → Refresh → Logout-All

- **Deskripsi**: Sesuai Definition of Done Sprint 2 (Sprint Planning): skenario end-to-end lolos test otomatis.
- **Acceptance Criteria**: Satu test suite menjalankan: register → login (dapat cookie) → akses endpoint terproteksi (berhasil) → refresh (dengan CSRF header, dapat cookie baru) → verifikasi refresh token lama gagal dipakai → logout-all → verifikasi refresh token baru pun gagal dipakai setelah logout-all.
- **Definition of Done**: Test ini berjalan di CI (`ci.yml`), hijau konsisten (tidak flaky, dijalankan 3x berturut lolos semua).
- **Dependency**: Seluruh task Feature 2.4-2.8
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [ ] Tulis integration test skenario penuh (`internal/identity/interface/http/auth_flow_test.go`)
- [ ] Jalankan 3x berturut-turut lokal untuk memastikan tidak flaky
- [ ] Verifikasi CI hijau di branch PR
- [ ] Update `docs/AGENTS.md` §7 (Status Proyek) — tandai Sprint 2 selesai setelah merge

---

## Ringkasan Keputusan

1. Sprint 2 melanjutkan Epic 2 dengan **6 Feature baru, 12 Task**, seluruhnya menuntaskan Sprint Goal "Authentication Lengkap" dari `14-sprint-planning.md`.
2. Keputusan final Security Design (HttpOnly Cookie + CSRF double-submit) diimplementasikan presisi di Task 2.4.3 dan 2.6.2 — bukan disederhanakan menjadi localStorage demi kemudahan implementasi.
3. Rate limiter (Task 2.8.1) sengaja dibuat sebagai `pkg/ratelimit` generic, bukan logic khusus login saja — mengantisipasi pemakaian ulang di rate limiting kirim pesan/upload pada sprint mendatang (Playbook §3.3 kriteria shared package terpenuhi: generic, dipakai berulang, tidak prematur).

## Trade-off yang Diterima

- Task 2.7.2/2.7.3 (Device Management UI-facing) berprioritas "Should" — dapat digeser ke Sprint 3 tanpa mengorbankan Sprint Goal inti bila kapasitas 2 minggu tidak cukup (dikalibrasi dari velocity Sprint 1).

## Risiko Arsitektur

- Task 2.6.2 (refresh + CSRF) berestimasi kesulitan Sedang-Tinggi — merupakan task paling berisiko meleset waktu di sprint ini karena melibatkan 3 konsep sekaligus (cookie, rotation, CSRF). Prioritaskan task ini lebih awal dalam sprint agar ada waktu buffer bila meleset.

## Technical Debt yang Sengaja Diterima

- Lockout progresif (Task 2.8.2) diimplementasikan dengan tracking sederhana di Redis — belum ada endpoint admin untuk melihat/reset lockout user tertentu secara manual (akan ditambahkan saat Admin Panel dikerjakan, Release 3).

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah breakdown Sprint 2 sudah selaras dengan hasil aktual Sprint 1 (asumsi: Sprint 1 sudah selesai sesuai Definition of Done-nya)?
2. Task 2.7.2/2.7.3 (Should priority) — tetap dikerjakan di Sprint 2, atau digeser ke Sprint 3 agar Sprint 2 fokus murni ke Must-priority?
3. Lanjut menyiapkan **Sprint 3** (Workspace, Permission, Channel) sekarang, atau tunggu Sprint 2 selesai dieksekusi dulu (sesuai prinsip Rolling Wave)?

---

## Changelog

| Versi | Tanggal    | Perubahan                                                                                                                      |
| ----- | ---------- | ------------------------------------------------------------------------------------------------------------------------------ |
| 1.0.0 | Draft awal | Dokumen Sprint 2: 6 Feature, 12 Task lengkap dengan AC/DoD/Dependency/Estimasi, menuntaskan Sprint Goal Authentication Lengkap |
