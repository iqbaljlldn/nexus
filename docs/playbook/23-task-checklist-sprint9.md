# Detailed Task Checklist — Sprint 9
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 9: Voice)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `03-adr.md` (ADR-005 — LiveKit), `06-srs.md` (Learning Roadmap M9), `07-hld.md` (§2.12), `09-database-design.md` (§2.7), `10-api-specification.md` (§3), `11-security-design.md` (§4), `12-deployment-architecture.md` (§8 — LiveKit resource allocation terpisah)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen & Prasyarat

Melanjutkan penomoran yang sudah diamandemen di `22-task-checklist-sprint8.md` §0 (Voice sekarang Sprint 9, bukan "Sprint 8" seperti garis besar awal).

**Prasyarat**: Channel tipe `voice` sudah bisa dibuat sejak Sprint 3 (Task 3.6.2 hanya mencakup tipe `text`, namun skema `channels` mendukung tipe `voice` sejak awal — Sprint 9 menambahkan endpoint khusus voice channel yang belum ada).

**Sprint Goal**: User dapat join/leave voice channel via LiveKit; daftar partisipan aktif tersinkronisasi realtime ke seluruh member channel.

**Catatan infrastruktur penting**: Sesuai Deployment Architecture §8, LiveKit **direncanakan self-hosted terpisah** dari `apps/api` sejak Tahap 2 deployment karena profil resource sangat berbeda (bandwidth-bound). Untuk Sprint 9 (masih development/Tahap 1), LiveKit dijalankan sebagai container tambahan di Docker Compose — namun **komunikasi ke LiveKit selalu lewat Server SDK/webhook**, tidak pernah asumsi co-location dengan `apps/api`, agar pemisahan fisik nanti tidak butuh perubahan kode.

---

## EPIC 14: Voice

### Feature 14.1: Setup LiveKit di Docker Compose

#### Task 14.1.1: Tambah Service LiveKit ke `docker-compose.yml`

- **Deskripsi**: Sesuai Deployment Architecture §1, tambahkan container LiveKit server.
- **Acceptance Criteria**: LiveKit server berjalan dengan konfigurasi API key/secret sendiri (bukan berbagi kredensial dengan komponen lain).
- **Definition of Done**: `docker compose up` menjalankan LiveKit; `livekit-cli` (atau curl ke endpoint) dapat memverifikasi server merespons.
- **Dependency**: Task 1.2.2 (Sprint 1)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tambah service `livekit` ke `docker-compose.yml` (image resmi LiveKit)
- [ ] Konfigurasi `NEXUS_API_LIVEKIT_API_KEY`, `NEXUS_API_LIVEKIT_API_SECRET`, `NEXUS_API_LIVEKIT_URL` (Playbook §7.3 konvensi)
- [ ] Verifikasi server LiveKit merespons (health endpoint bawaan LiveKit)

---

### Feature 14.2: Migrasi Database

#### Task 14.2.1: Migrasi Tabel `voice_sessions`, `voice_participants`

- **Deskripsi**: DDL sesuai Database Design §2.7 — state ringan (sebagian besar state ada di LiveKit sendiri, HLD §2.12).
- **Acceptance Criteria**: `voice_sessions.livekit_room_name` unik per channel per sesi aktif.
- **Definition of Done**: `migrate up` sukses.
- **Dependency**: Task 3.6.1 (Sprint 3 — tabel `channels`)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `voice_sessions` (up & down)
- [ ] Tulis migrasi `voice_participants` (up & down, composite PK)
- [ ] sqlc query dasar: `CreateVoiceSession`, `EndVoiceSession`, `AddParticipant`, `RemoveParticipant`, `ListActiveParticipants`

---

### Feature 14.3: LiveKit Token Service

#### Task 14.3.1: `LiveKitTokenService` — Generate Room Access Token

- **Deskripsi**: Sesuai LLD/Security Design §4 — token **wajib** di-generate backend, tidak pernah API secret di-embed frontend (Learning Roadmap M9 kesalahan umum yang dihindari).
- **Acceptance Criteria**: Token memiliki claim spesifik (`room`, `identity`, grants terbatas — mis. tidak bisa join room lain selain yang diotorisasi).
- **Definition of Done**: Unit test: token yang di-generate dapat di-decode dan claim-nya sesuai (room & identity benar); token untuk channel yang user tidak punya akses tidak pernah ter-generate (dicegah di layer service sebelum sampai ke LiveKit SDK).
- **Dependency**: Task 14.1.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `internal/voice/application/livekit_token_service.go` (LiveKit Server SDK Go)
- [ ] Validasi permission baca channel voice SEBELUM generate token (reuse `ChannelAuthorizationService`)
- [ ] Unit test: token valid untuk channel yang diizinkan; permintaan untuk channel tanpa akses ditolak sebelum sampai ke LiveKit

---

### Feature 14.4: Join/Leave Voice Channel

#### Task 14.4.1: Handler — `POST /channels/{id}/voice/join`

- **Deskripsi**: Sesuai HLD §6.2 sequence diagram.
- **Acceptance Criteria**: Response berisi token + LiveKit server URL; `voice_sessions` dibuat otomatis bila belum ada sesi aktif untuk channel tersebut (satu channel voice = satu room LiveKit aktif pada satu waktu).
- **Definition of Done**: Test: join pertama kali → `voice_sessions` baru dibuat; join kedua (user lain) ke channel sama → memakai `voice_sessions` yang sudah ada (tidak duplikat room).
- **Dependency**: Task 14.3.1, Task 14.2.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `POST /channels/{id}/voice/join` — cek/buat `voice_sessions`, generate token, insert `voice_participants`
- [ ] Response: `{ token, livekit_url, room_name }`
- [ ] Test: join pertama (buat room baru), join kedua (reuse room)

#### Task 14.4.2: LiveKit Webhook — Sinkronisasi Partisipan

- **Deskripsi**: Sesuai HLD §6.2 — LiveKit mengirim webhook `participant_joined`/`participant_left`, backend menyinkronkan ke database & broadcast WS.
- **Acceptance Criteria**: Endpoint webhook memverifikasi signature LiveKit (mitigasi request palsu — Security Design prinsip attack surface).
- **Definition of Done**: Test: simulasikan webhook `participant_joined` → `voice_participants` terupdate, broadcast `voice.participant_joined` terkirim ke member channel.
- **Dependency**: Task 14.4.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `POST /webhooks/livekit` — verifikasi signature (LiveKit SDK helper)
- [ ] Handle event `participant_joined` → update `voice_participants`, broadcast WS
- [ ] Handle event `participant_left` → update `left_at`, broadcast WS; bila partisipan terakhir keluar → tandai `voice_sessions.ended_at`
- [ ] Test: kedua event, termasuk auto-end session saat partisipan terakhir keluar

#### Task 14.4.3: Handler — `GET /channels/{id}/voice/participants`

- **Deskripsi**: Sesuai API Specification §3.
- **Acceptance Criteria**: Mengembalikan daftar partisipan aktif (`left_at IS NULL`) dari `voice_sessions` yang sedang berjalan.
- **Definition of Done**: Test: list mencerminkan state terkini setelah join/leave.
- **Dependency**: Task 14.4.2
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `GET /channels/{id}/voice/participants`
- [ ] Test: list sesuai state join/leave

---

### Feature 14.5: Integration Test End-to-End Sprint 9

#### Task 14.5.1: Skenario Penuh — Voice Join/Leave

- **Deskripsi**: Verifikasi Sprint Goal.
- **Acceptance Criteria**: Alur: User A join voice channel → room dibuat, token valid diterima; User B join channel sama → room di-reuse, kedua user muncul di `GET participants`; simulasikan webhook `participant_left` untuk User A → User A hilang dari daftar, broadcast WS diterima User B; User B (partisipan terakhir) leave → `voice_sessions.ended_at` terisi.
- **Definition of Done**: Test hijau konsisten 3x run berturut.
- **Dependency**: Seluruh task Epic 14
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis skenario penuh (mock webhook LiveKit untuk simulasi join/leave, karena test tidak memerlukan koneksi WebRTC sungguhan — cukup verifikasi orkestrasi backend)
- [ ] Jalankan 3x berturut, pastikan tidak flaky
- [ ] Update `docs/AGENTS.md` §7 — Sprint 9 selesai

---

## Ringkasan Keputusan

1. Sprint 9 mencakup **1 Epic, 5 Feature, 8 task**, menuntaskan integrasi Voice via LiveKit.
2. Token generation **selalu** di backend, tidak pernah API secret di-embed frontend — prinsip keras dari ADR-005/Learning Roadmap M9 dipatuhi presisi di Task 14.3.1.
3. Webhook LiveKit (Task 14.4.2) **wajib** verifikasi signature — konsisten dengan Security Design (attack surface eksternal).
4. Satu channel voice = satu room LiveKit aktif pada satu waktu (Task 14.4.1) — desain sederhana yang cukup untuk skala proyek ini, dibanding multi-room per channel yang tidak dibutuhkan.

## Trade-off yang Diterima

- Test end-to-end (Task 14.5.1) memakai **mock webhook**, bukan koneksi WebRTC sungguhan — cukup untuk memverifikasi orkestrasi backend (yang menjadi tanggung jawab kode kita), sementara kualitas audio/koneksi WebRTC sungguhan adalah tanggung jawab LiveKit sendiri (sudah diverifikasi vendor, di luar scope test kita — konsisten dengan rationale ADR-005 memilih LiveKit).

## Risiko Arsitektur

- LiveKit dijalankan co-located dengan `apps/api` di Docker Compose development — perlu diingat ini **sementara**; Deployment Architecture §8 sudah mencatat rencana pemisahan resource sejak Tahap 2 produksi. Pastikan tidak ada asumsi kode yang menganggap LiveKit selalu di host yang sama (mis. hardcode `localhost` — harus selalu dari env var `NEXUS_API_LIVEKIT_URL`).

## Technical Debt yang Sengaja Diterima

- Belum ada mekanisme pembersihan `voice_sessions` yang "menggantung" (mis. server crash saat ada sesi aktif, `ended_at` tidak pernah terisi) — akan ditambahkan sebagai scheduled cleanup (pola sama seperti Task 10.7.1 Sprint 6) bila terbukti jadi masalah nyata di Milestone 11.

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah desain "satu channel voice = satu room LiveKit aktif" sudah sesuai ekspektasi, atau ada kebutuhan multi-room per channel (mis. breakout room) yang perlu diantisipasi lebih awal?
2. Lanjut ke **Sprint 10** (Video — perluasan LiveKit untuk kamera + screen share), atau berhenti dulu?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 9: 1 Epic, 5 Feature, 8 task, menuntaskan integrasi Voice via LiveKit dengan token generation backend-only dan webhook signature verification |
