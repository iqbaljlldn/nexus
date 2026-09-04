# Detailed Task Checklist — Sprint 9
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 9: Voice)
**Versi:** 1.2.0
**Status:** Accepted (v1.1 revisi skema, v1.2 amandemen frontend — lihat Changelog)
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

#### Task 14.2.1: Migrasi Tabel `voice_participants` *(disederhanakan — lihat catatan revisi di bawah)*

> **Revisi**: Draft awal task ini mencakup dua tabel (`voice_sessions` + `voice_participants`). Setelah dibandingkan dengan desain Discord — **voice channel bersifat persisten dan ITU SENDIRI adalah room** (1:1 dengan `channel_id`), bukan entity yang dibuat-hapus dinamis mengikuti join-leave — `voice_sessions` **dihapus sepenuhnya** dari desain. `livekit_room_name` tidak disimpan; dihitung langsung dari `channel_id.String()` di kode aplikasi.

- **Deskripsi**: DDL sesuai Database Design §2.7 (revisi) — hanya satu tabel `voice_participants`, room = `channel_id` langsung.
- **Acceptance Criteria**: `PRIMARY KEY (channel_id, user_id, joined_at)`; index parsial `idx_voice_participants_active` untuk query cepat "siapa aktif di channel ini" (`WHERE left_at IS NULL`).
- **Definition of Done**: `migrate up` sukses.
- **Dependency**: Task 3.6.1 (Sprint 3 — tabel `channels`)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 0.5 jam *(turun dari 1 jam — satu tabel, bukan dua)*
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `voice_participants` (up & down) — skema Database Design §2.7 revisi
- [ ] sqlc query dasar: `AddParticipant`, `MarkParticipantLeft`, `ListActiveParticipants(channelID)`, `IsRoomActive(channelID) bool`

---

### Feature 14.3: LiveKit Token Service

#### Task 14.3.1: `LiveKitTokenService` — Generate Room Access Token

- **Deskripsi**: Sesuai LLD/Security Design §4 — token **wajib** di-generate backend, tidak pernah API secret di-embed frontend (Learning Roadmap M9 kesalahan umum yang dihindari). Room name = `channel_id.String()` langsung (tidak ada lookup/pembuatan room terpisah).
- **Acceptance Criteria**: Token memiliki claim spesifik (`room = channel_id.String()`, `identity`, grants terbatas — mis. tidak bisa join room lain selain yang diotorisasi).
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

- **Deskripsi**: Sesuai HLD §6.2 sequence diagram (disederhanakan mengikuti revisi §0).
- **Acceptance Criteria**: Response berisi token + LiveKit server URL. **Tidak ada** logic "cek/buat sesi" — backend cukup generate token untuk `room = channel_id.String()` (room LiveKit dibuat otomatis oleh LiveKit sendiri saat peserta pertama connect, `auto_create` — bukan tanggung jawab `apps/api`) dan insert baris `voice_participants` optimistically (dikonfirmasi ulang oleh webhook `participant_joined`, Task 14.4.2).
- **Definition of Done**: Test: join oleh user manapun (pertama atau bukan) menghasilkan alur identik — tidak ada percabangan "room baru vs room existing" di kode (itulah keuntungan penyederhanaan ini).
- **Dependency**: Task 14.3.1, Task 14.2.1
- **Estimasi Kesulitan**: Mudah *(turun dari Sedang — tidak ada lagi logic cek/buat sesi)*
- **Estimasi Waktu**: 1.5 jam *(turun dari 2.5 jam)*
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `POST /channels/{id}/voice/join` — validasi permission, generate token (`room = channel_id`), insert `voice_participants` (optimistic)
- [ ] Response: `{ token, livekit_url, room_name: channel_id }`
- [ ] Test: join oleh user pertama dan user berikutnya menghasilkan alur kode yang sama persis

#### Task 14.4.2: LiveKit Webhook — Sinkronisasi Partisipan

- **Deskripsi**: Sesuai HLD §6.2 — LiveKit mengirim webhook `participant_joined`/`participant_left`, backend menyinkronkan ke database & broadcast WS. **Tidak ada** lagi logic "auto-end session saat partisipan terakhir keluar" — channel voice tetap "ada" (persisten) baik kosong maupun berisi orang, konsisten dengan revisi §0.
- **Acceptance Criteria**: Endpoint webhook memverifikasi signature LiveKit (mitigasi request palsu — Security Design prinsip attack surface).
- **Definition of Done**: Test: simulasikan webhook `participant_joined` → `voice_participants` terupdate (`left_at = NULL`), broadcast `voice.participant_joined` terkirim ke member channel; webhook `participant_left` → `left_at` terisi, tidak ada efek lain (tidak ada tabel "sesi" yang perlu ditutup).
- **Dependency**: Task 14.4.1
- **Estimasi Kesulitan**: Mudah *(turun dari Sedang)*
- **Estimasi Waktu**: 1.5 jam *(turun dari 2.5 jam)*
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `POST /webhooks/livekit` — verifikasi signature (LiveKit SDK helper)
- [ ] Handle event `participant_joined` → upsert `voice_participants` (`left_at = NULL`), broadcast WS
- [ ] Handle event `participant_left` → update `left_at`, broadcast WS
- [ ] Test: kedua event — **tidak** perlu test "auto-end session" (sudah tidak ada konsepnya)

#### Task 14.4.3: Handler — `GET /channels/{id}/voice/participants`

- **Deskripsi**: Sesuai API Specification §3.
- **Acceptance Criteria**: Mengembalikan daftar partisipan aktif (`left_at IS NULL`) untuk `channel_id` yang diminta — query langsung ke `voice_participants`, tanpa perantara tabel sesi.
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
- **Acceptance Criteria**: Alur: User A join voice channel → token valid diterima; User B join channel sama → alur kode identik (tidak ada percabangan room baru/existing), kedua user muncul di `GET participants`; simulasikan webhook `participant_left` untuk User A → User A hilang dari daftar, broadcast WS diterima User B; User B (partisipan terakhir) leave → daftar partisipan kosong, namun channel voice tetap "ada" (tidak ada entity sesi yang perlu ditutup — verifikasi eksplisit: query channel masih ada, hanya `voice_participants` aktif = 0).
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

## EPIC 15: Frontend — Voice UI (LiveKit Client)

### Feature 15.1: Join/Leave Voice Channel

#### Task 15.1.1: `useVoiceRoom` Composable

- **Deskripsi**: Implementasi persis Frontend Architecture §8 — wrapper LiveKit Client SDK, `shallowRef` untuk objek `Room` (bukan `reactive`, alasan performa dijelaskan di §8).
- **Acceptance Criteria**: `join()` memanggil `POST /channels/{id}/voice/join` (Task 14.4.1 backend), connect ke LiveKit dengan token yang diterima.
- **Definition of Done**: E2E test (mock LiveKit SDK — koneksi WebRTC sungguhan di luar scope test frontend, konsisten dengan pendekatan backend Task 14.5.1 yang juga mock webhook): join channel voice → `room.connect` terpanggil dengan parameter benar.
- **Dependency**: Task 8.1.1 (Sprint 4 — pola composable serupa)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Install LiveKit Client SDK (`livekit-client`)
- [ ] Implementasi `composables/useVoiceRoom.ts` persis §8
- [ ] Test dengan mock SDK: join memanggil connect dengan token benar

#### Task 15.1.2: `VoiceParticipantGrid.vue` — Sinkronisasi Partisipan

- **Deskripsi**: Perluas event router (Task 8.1.2 Sprint 4) dengan case `voice.participant_joined`/`participant_left` (§10 API Spec) — **selain** event LiveKit SDK native (`RoomEvent.ParticipantConnected`), backend juga broadcast lewat WS aplikasi (Task 14.4.2 backend) untuk sinkronisasi UI di luar konteks LiveKit murni (mis. menampilkan partisipan di sidebar channel meski user belum join).
- **Acceptance Criteria**: Daftar partisipan aktif tampil di `ChannelSidebar` (bukan hanya di dalam UI voice call itu sendiri) — memakai data dari WS broadcast aplikasi, bukan LiveKit SDK event (yang hanya tersedia bagi partisipan yang sudah connect).
- **Definition of Done**: E2E test: User A join voice → User B (belum join, hanya melihat sidebar) melihat User A muncul di daftar partisipan channel tersebut.
- **Dependency**: Task 15.1.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Perluas event router — case `voice.participant_joined`/`participant_left`
- [ ] Komponen `VoiceParticipantGrid.vue` di sidebar channel (daftar avatar partisipan)
- [ ] E2E test: partisipan terlihat oleh user yang belum join

#### Task 15.1.3: `VoiceControls.vue` — Mute/Unmute, Leave

- **Deskripsi**: Kontrol dasar dalam sesi voice aktif.
- **Acceptance Criteria**: Tombol mute/unmike (local track enable/disable, tidak perlu request ke backend — murni client-side LiveKit SDK), tombol leave memanggil disconnect + (opsional) beri tahu backend.
- **Definition of Done**: E2E test (mock SDK): toggle mute memanggil method SDK yang benar; leave memutus koneksi.
- **Dependency**: Task 15.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Komponen `VoiceControls.vue` (mute/unmute, leave)
- [ ] Test dengan mock SDK

---

### Feature 15.2: Integration Test End-to-End Frontend Sprint 9

#### Task 15.2.1: Skenario Penuh — Mencerminkan Gerbang Backend (Task 14.5.1)

- **Deskripsi**: Versi frontend, dengan mock LiveKit SDK (kualitas audio/WebRTC sungguhan tetap di luar scope, sama seperti pendekatan backend).
- **Acceptance Criteria**: Join → partisipan muncul di sidebar (user lain) → leave (mock webhook) → partisipan hilang.
- **Definition of Done**: Playwright test hijau konsisten 3x run berturut.
- **Dependency**: Seluruh task Epic 15
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Skenario Playwright dengan mock LiveKit SDK
- [ ] Jalankan 3x berturut, pastikan tidak flaky
- [ ] Update `docs/AGENTS.md` §7 — Sprint 9 frontend selesai bersamaan backend

---

## Ringkasan Keputusan

1. Sprint 9 mencakup **1 Epic, 5 Feature, 8 task** backend, menuntaskan integrasi Voice via LiveKit. *(Direvisi: ditambah Epic 15 — 2 Feature, 4 Task frontend, amandemen retroaktif — `useVoiceRoom` dengan `shallowRef`, sinkronisasi partisipan via WS aplikasi terpisah dari event LiveKit SDK native.)*
2. **Revisi arsitektur (mengikuti pola Discord nyata)**: entity `voice_sessions` **dihapus sepenuhnya**. Voice channel bersifat persisten dan ITU SENDIRI adalah room LiveKit (`room = channel_id.String()`, dihitung langsung, tidak disimpan sebagai kolom). Hanya `voice_participants` yang eksis sebagai entity — melacak siapa terkoneksi ke channel mana, kapan. Ini menghilangkan kelas bug "room ganda" dan "sesi menggantung" yang sebelumnya dicatat sebagai risiko terbuka.
3. Token generation **selalu** di backend, tidak pernah API secret di-embed frontend — prinsip keras dari ADR-005/Learning Roadmap M9 dipatuhi presisi di Task 14.3.1.
4. Webhook LiveKit (Task 14.4.2) **wajib** verifikasi signature — konsisten dengan Security Design (attack surface eksternal).

## Trade-off yang Diterima

- Test end-to-end (Task 14.5.1) memakai **mock webhook**, bukan koneksi WebRTC sungguhan — cukup untuk memverifikasi orkestrasi backend (yang menjadi tanggung jawab kode kita), sementara kualitas audio/koneksi WebRTC sungguhan adalah tanggung jawab LiveKit sendiri (sudah diverifikasi vendor, di luar scope test kita — konsisten dengan rationale ADR-005 memilih LiveKit).
- Pembuatan room LiveKit itu sendiri (bukan hanya token) diserahkan ke `auto_create` bawaan LiveKit saat peserta pertama connect — `apps/api` tidak pernah secara eksplisit memanggil "create room" API, cukup generate token. Ini mengurangi satu titik kegagalan (tidak ada risiko token valid tapi room gagal dibuat oleh langkah terpisah di `apps/api`).

## Risiko Arsitektur

- LiveKit dijalankan co-located dengan `apps/api` di Docker Compose development — perlu diingat ini **sementara**; Deployment Architecture §8 sudah mencatat rencana pemisahan resource sejak Tahap 2 produksi. Pastikan tidak ada asumsi kode yang menganggap LiveKit selalu di host yang sama (mis. hardcode `localhost` — harus selalu dari env var `NEXUS_API_LIVEKIT_URL`).
- Insert `voice_participants` di Task 14.4.1 bersifat **optimistic** (sebelum webhook konfirmasi) — bila user gagal benar-benar connect ke LiveKit setelah dapat token (mis. WebRTC gagal di sisi client), baris `voice_participants` bisa "menggantung" tanpa `left_at` terisi. Mitigasi: TTL/cleanup berbasis heartbeat dapat dipertimbangkan di Milestone 11 bila terbukti jadi masalah nyata (lihat Technical Debt).

## Technical Debt yang Sengaja Diterima

- Baris `voice_participants` yang "menggantung" akibat insert optimistic tanpa konfirmasi webhook (kegagalan koneksi WebRTC di sisi client) belum punya mekanisme cleanup — akan dievaluasi di Milestone 11 berdasarkan data nyata, bukan diasumsikan jadi masalah sekarang (YAGNI).

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah penyederhanaan "channel = room permanen" (menghilangkan `voice_sessions`) sudah sesuai yang Anda maksud, dan skema `voice_participants` revisi (Database Design §2.7) dapat diterima?
2. Lanjut ke **Sprint 10** (Video — perluasan LiveKit untuk kamera + screen share), atau berhenti dulu?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 9: 1 Epic, 5 Feature, 8 task, menuntaskan integrasi Voice via LiveKit dengan token generation backend-only dan webhook signature verification |
| 1.1.0 | Revisi | Dihapus entity `voice_sessions` mengikuti desain nyata Discord (channel = room persisten, bukan sesi dinamis). Task 14.2.1, 14.4.1, 14.4.2, 14.4.3, 14.5.1 disederhanakan; estimasi waktu turun total ~2.5 jam akibat hilangnya logic cek/buat/tutup sesi. |
| 1.2.0 | Amandemen | Ditambahkan Epic 15: Frontend (`useVoiceRoom` dengan `shallowRef`, sinkronisasi partisipan via WS aplikasi, kontrol mute/leave) — amandemen retroaktif |
