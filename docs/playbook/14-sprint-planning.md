# Sprint Planning
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 10 — Sprint Planning
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `13-development-roadmap.md`, `05-prd.md` (v1.1.0), `06-srs.md` (v1.1.0)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Pendekatan: Rolling Wave Planning

Merencanakan **seluruh** 32 minggu ke tingkat sprint sejak sekarang akan melanggar prinsip *Simplicity over Cleverness* — rencana detail untuk pekerjaan 6 bulan ke depan hampir pasti akan berubah begitu pembelajaran nyata terjadi (scope bergeser, estimasi meleset, prioritas berubah). Karena itu, dokumen ini menerapkan **Rolling Wave Planning**:

- **Release 1** (dekat, akan segera dikerjakan): direncanakan **detail penuh** ke tingkat sprint & backlog item.
- **Release 2-5** (jauh): direncanakan **garis besar saja** (jumlah sprint, tema tiap sprint) — akan **didetailkan ulang** (dokumen Sprint Planning baru atau revisi) begitu Release sebelumnya mendekati selesai, dengan informasi ter-update dari pembelajaran nyata.

Ini bukan kemalasan perencanaan — ini adalah praktik nyata yang dipakai tim engineering profesional (detailed planning horizon pendek, roadmap panjang tetap garis besar).

---

## 1. Sprint Cadence

- **Durasi sprint**: 2 minggu.
- **Kapasitas asumsi**: paruh waktu (bukan tim penuh), disesuaikan realita belajar individu — kapasitas per sprint akan dikalibrasi ulang setelah Sprint 1-2 selesai (velocity nyata, bukan asumsi).
- **Sprint Ceremony (adaptasi solo)**:
  - **Sprint Planning** (awal sprint): pilih backlog item dari Release yang sedang berjalan, pastikan tiap item memenuhi Definition of Ready (§4).
  - **Daily Check-in** (opsional, self-tracking): catatan singkat progres harian — tidak formal, cukup untuk menjaga momentum.
  - **Sprint Review**: demo hasil sprint terhadap Acceptance Criteria terkait (PRD/SRS).
  - **Sprint Retrospective**: refleksi singkat — apa yang berjalan baik, apa yang perlu disesuaikan di sprint berikutnya (termasuk kalibrasi ulang estimasi).

---

## 2. Release 1 — Sprint Breakdown Detail (Sprint 1-3)

### Sprint 1 (Minggu 1-2): Project Foundation + Awal Authentication

**Sprint Goal**: Skeleton monorepo berjalan penuh via Docker Compose, CI hijau, dan alur register dasar berfungsi.

| Backlog Item | Terkait | Prioritas |
|---|---|---|
| Setup monorepo (`go.work`, pnpm workspace, struktur folder Playbook §19) | M1 | Must |
| Setup Docker Compose (API, Web, PostgreSQL, Redis, MinIO, Traefik) | M1 | Must |
| Setup CI pipeline dasar (`ci.yml`: format, lint, test, build) | M1 | Must |
| Health check endpoint (`/healthz`, `/readyz`) | M1 | Must |
| Migrasi awal: `users`, `sessions` (Database Design §2.1) | M2 | Must |
| Endpoint `POST /auth/register` + Argon2id hashing | M2 (FR-AUTH-01/02) | Must |
| Unit test: validasi registrasi (email/username unik, password policy) | M2 | Must |

**Definition of Done Sprint**: `docker compose up` berjalan tanpa error, CI hijau untuk PR kosong, `curl POST /auth/register` berhasil membuat user baru dengan password ter-hash.

### Sprint 2 (Minggu 3-4): Authentication Lengkap

**Sprint Goal**: Alur autentikasi penuh (login, refresh rotation, logout-all, session management) berfungsi end-to-end.

| Backlog Item | Terkait | Prioritas |
|---|---|---|
| Endpoint `POST /auth/login` (email/username fleksibel) | FR-AUTH-03 | Must |
| JWT access token generation & middleware validasi | FR-AUTH-04 | Must |
| Refresh token rotation + `POST /auth/refresh` | FR-AUTH-04 | Must |
| HttpOnly Cookie storage + CSRF double-submit (Security Design §2/§6) | Security Design | Must |
| `POST /auth/logout-all` | FR-AUTH-05 | Must |
| `GET/DELETE /auth/sessions` (Device Management) | FR-AUTH-06 | Should |
| Rate limiting login (Redis sliding window, LLD §2.8) | §3.5 SRS | Must |
| Integration test: alur login → refresh → logout-all | — | Must |

**Definition of Done Sprint**: Skenario end-to-end (register → login → akses endpoint terproteksi → refresh → logout-all → token lama ditolak) lolos test otomatis.

### Sprint 3 (Minggu 5-6): Workspace, Permission, Channel

**Sprint Goal**: User dapat membuat workspace, invite anggota, mengatur role/permission, dan membuat channel.

| Backlog Item | Terkait | Prioritas |
|---|---|---|
| Migrasi: `workspaces`, `categories`, `invites`, `members`, `roles`, `member_role_assignments` | Database Design §2.2 | Must |
| `POST /workspaces`, `GET /workspaces` | FR-WS-01 | Must |
| `POST /workspaces/{id}/invites`, `POST /invites/{code}/redeem` (idempotent) | FR-WS-06 | Must |
| `POST /workspaces/{id}/roles`, assignment role ke member | FR-WS-02/04 | Must |
| Permission Resolver (LLD §2.1) + unit test multi-level override | FR-WS-07 | Must |
| Migrasi: `channels`, `channel_permission_overrides` | Database Design §2.3 | Must |
| `POST /workspaces/{id}/channels` (tipe text dulu, voice/video menyusul Release 3) | FR-CH-01 | Must |
| `PATCH /channels/{id}/permission-overrides` | FR-WS-05 | Must |

**Definition of Done Sprint**: Skenario end-to-end (buat workspace → invite user kedua → assign role custom → buat channel dengan override permission → verifikasi user kedua tidak bisa akses channel privat) lolos test otomatis, sesuai Milestone Rilis Release 1 di Development Roadmap.

---

## 3. Release 2-5 — Sprint Overview (Garis Besar, Akan Didetailkan Ulang)

### Release 2 — Core Realtime (~3 sprint)

| Sprint | Tema Utama |
|---|---|
| Sprint 4-5 | WebSocket infrastructure + Messaging inti (kirim/edit/delete/reply/thread/mention/reaction) + DM |
| Sprint 6 | Upload (MinIO integration, Asynq worker, thumbnail) + Presence & Realtime Signal |

### Release 3 — Engagement (~3 sprint)

| Sprint | Tema Utama |
|---|---|
| Sprint 7 | Notification (realtime + Brevo email, preferensi, rate limiting) |
| Sprint 8 | Voice (LiveKit integration) |
| Sprint 9 | Video (perluasan LiveKit) |

### Release 4 — Hardening & EDA (~2 sprint)

| Sprint | Tema Utama |
|---|---|
| Sprint 10 | Profiling (pprof), benchmark, perbaikan bottleneck teridentifikasi |
| Sprint 11 | Outbox Pattern, Redis Streams consumer, idempotency untuk 3 domain event kunci |

### Release 5 — Distributed System (~6 sprint, dievaluasi ulang penuh saat tiba)

| Sprint | Tema Utama |
|---|---|
| Sprint 12 | Ekstraksi service pertama (Identity atau Notification) |
| Sprint 13-14 | Hybrid Architecture, trace propagation, API Gateway routing |
| Sprint 14-15 (paralel) | Observability penuh (OpenTelemetry, Grafana dashboard) |
| Sprint 15 | Blue-Green Deployment teruji |
| Sprint 16-17 | Microservices Migration lanjutan sesuai Service Extraction Plan |
| Sprint 18 | Production Hardening (security review, DR drill, load test penuh) |

> **Catatan eksplisit**: breakdown Release 3-5 di atas **akan direvisi** menjadi dokumen Sprint Planning baru (atau amandemen dokumen ini) begitu Release sebelumnya mendekati selesai — sesuai prinsip Rolling Wave Planning §0.

---

## 4. Definition of Ready (Sebelum Item Masuk Sprint)

Sebuah backlog item hanya dapat masuk sprint bila:

- [ ] Requirement terkait sudah ada di PRD/SRS (dapat ditelusuri ID FR-nya).
- [ ] Desain teknis terkait sudah ada di HLD/LLD/Database Design/API Spec (tidak ada keputusan arsitektur besar yang masih terbuka untuk item ini).
- [ ] Dependency terhadap item lain sudah selesai (atau termasuk dalam sprint yang sama dengan urutan jelas).
- [ ] Acceptance Criteria dapat diverifikasi secara konkret (bukan "kelihatannya sudah jalan").

---

## Ringkasan Keputusan

1. **Rolling Wave Planning** diterapkan: Release 1 detail penuh (3 sprint), Release 2-5 garis besar yang akan didetailkan ulang mendekati waktunya.
2. Sprint cadence 2 minggu dengan ceremony yang diadaptasi untuk konteks solo learning (tanpa menghilangkan disiplin Planning/Review/Retrospective).
3. Setiap sprint di Release 1 memiliki Definition of Done yang dapat diverifikasi lewat test otomatis, bukan penilaian subjektif.

## Trade-off yang Diterima

- Sprint Planning untuk Release 2-5 sengaja tidak detail — risiko: bila dibaca sekilas, terlihat "kurang lengkap" dibanding Release 1. Ini diterima secara sadar karena detail prematur untuk pekerjaan berbulan-bulan ke depan cenderung salah dan harus dibuang, sesuai prinsip *YAGNI* diterapkan pada proses perencanaan itu sendiri.

## Risiko Arsitektur

- Velocity asumsi (2 minggu per sprint) belum divalidasi data nyata — Sprint 1-2 akan menjadi kalibrasi awal; bila meleset signifikan, estimasi Release 2-5 di Development Roadmap perlu direvisi lebih awal dari perkiraan.

## Technical Debt yang Sengaja Diterima

- Belum ada task-level breakdown (per-task checklist individual) — itu adalah scope **Phase 11: Detailed Task Checklist**, dokumen berikutnya.

## Hal yang Perlu Dikonfirmasi Sebelum Lanjut ke Fase Berikutnya

1. Apakah breakdown Sprint 1-3 (Release 1) sudah cukup detail dan actionable untuk langsung mulai dikerjakan?
2. Apakah pendekatan **Rolling Wave Planning** (detail dekat, garis besar jauh) dapat diterima, dibanding mendetailkan seluruh 18 sprint sejak sekarang?
3. Lanjut ke **Phase 11 — Detailed Task Checklist** (untuk Sprint 1, sesuai prinsip Rolling Wave — bukan seluruh 18 sprint)?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen pertama Phase 10: Release 1 detail penuh (Sprint 1-3), Release 2-5 garis besar dengan prinsip Rolling Wave Planning |
