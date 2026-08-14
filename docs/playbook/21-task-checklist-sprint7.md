# Detailed Task Checklist — Sprint 7
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 7: Presence & Realtime Signal)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `06-srs.md` (§2.5 FR-PRES), `07-hld.md` (§2.9), `08-lld.md` (§2.6 cache invalidation pattern, referensi SCAN), `09-database-design.md` (§2.4 `read_receipts`), `10-api-specification.md` (§10 WebSocket Protocol), `13-development-roadmap.md` (Release 2)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen & Prasyarat

Sprint 7 adalah **sprint terakhir Release 2** (Sprint Planning §2: "M7 Presence & Realtime Signal, 1 minggu"). Setelah sprint ini, **Release 2 selesai sepenuhnya** dan gerbang kelulusan Development Roadmap ("Demo — chat penuh termasuk DM terasa seperti Discord dari sisi UX, attachment berfungsi") dapat diverifikasi total.

**Prasyarat**: Sprint 4 (WebSocket Infrastructure, khususnya Task 6.2.1 — event router `typing.start`/`presence.set_status`/`heartbeat` sudah di-stub) dan Sprint 6 (Upload) selesai.

**Sprint Goal**: Status online/idle/dnd/invisible/offline berfungsi dengan deteksi disconnect otomatis (TTL-based); typing indicator dan read receipt berfungsi realtime.

---

## EPIC 11: Presence

### Feature 11.1: Presence Service (Redis TTL-Based)

#### Task 11.1.1: Implementasi `PresenceService` — Redis TTL Store

- **Deskripsi**: Sesuai HLD §2.9 — presence **tidak** persisten di PostgreSQL, murni Redis-backed dengan TTL 30 detik (FR-PRES-02).
- **Acceptance Criteria**: `SetStatus(ctx, userID, status)` menulis key Redis `presence:{user_id}` dengan value status dan TTL 30 detik; `GetStatus(ctx, userID)` mengembalikan `offline` bila key tidak ada (expired atau belum pernah set).
- **Definition of Done**: Unit test (Redis test container/miniredis): set status → get status benar; tunggu TTL habis (atau majukan clock simulasi) → get status kembali `offline`.
- **Dependency**: Task 1.2.2 (Sprint 1 — Redis)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `internal/presence/domain/presence.go` (`PresenceStatus` enum: online/idle/dnd/invisible/offline)
- [ ] Implementasi `internal/presence/infrastructure/redis_presence_store.go` — `SetStatus`, `GetStatus`, `RefreshTTL`
- [ ] Unit test: set-get round-trip, TTL expiry → default offline

#### Task 11.1.2: Handling Status `invisible`

- **Deskripsi**: FR-PRES-01 — `invisible` disimpan sebagai status nyata secara internal, namun ditampilkan sebagai `offline` ke user lain.
- **Acceptance Criteria**: `GetStatus` dipanggil oleh **pemilik status sendiri** → mengembalikan `invisible` (asli); dipanggil untuk **user lain** → mengembalikan `offline` (disamarkan). Perbedaan ini di level service, bukan disimpan sebagai dua key terpisah.
- **Definition of Done**: Unit test: `GetStatusForViewer(ctx, targetUserID, viewerUserID)` — viewer=target sendiri → `invisible`; viewer≠target → `offline`.
- **Dependency**: Task 11.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `GetStatusForViewer(ctx, targetUserID, viewerUserID)` dengan logic masking `invisible`→`offline`
- [ ] Unit test: kedua skenario viewer

---

### Feature 11.2: Wiring WebSocket Presence Events

#### Task 11.2.1: Tuntaskan Handler `presence.set_status` dan `heartbeat` (dari Stub Sprint 4)

- **Deskripsi**: Perluas router event WS (Task 6.2.1 Sprint 4) — implementasi penuh, bukan lagi stub.
- **Acceptance Criteria**: `presence.set_status` memanggil `PresenceService.SetStatus`; `heartbeat` memanggil `PresenceService.RefreshTTL` (refresh TTL tanpa mengubah status — FR-PRES-02, interval client 15 detik, margin 2x sebelum TTL 30 detik habis).
- **Definition of Done**: Test: kirim event `presence.set_status` via WS test client → status di Redis berubah; kirim `heartbeat` berkala → TTL tidak pernah habis selama koneksi aktif.
- **Dependency**: Task 6.2.1 (Sprint 4), Task 11.1.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi handler `presence.set_status` — panggil `SetStatus`, lalu broadcast (Task 11.3.1)
- [ ] Implementasi handler `heartbeat` — panggil `RefreshTTL`
- [ ] Set status `online` otomatis saat koneksi WS pertama kali terbuka (bila belum ada status eksplisit)
- [ ] Set status `offline` (hapus key Redis) saat koneksi ditutup secara graceful (bukan menunggu TTL bila disconnect terdeteksi eksplisit — optimasi, TTL tetap jadi fallback untuk disconnect tidak normal)
- [ ] Test: set status manual, heartbeat menjaga TTL, disconnect graceful langsung set offline

---

### Feature 11.3: Presence Broadcast (Scoped)

#### Task 11.3.1: Broadcast `presence.updated` — Scoped ke Workspace Bersama

- **Deskripsi**: FR-PRES §Best Practice — broadcast **hanya** ke member yang berbagi workspace dengan user tersebut, bukan broadcast global (kritikal untuk NFR 100.000 member/server).
- **Acceptance Criteria**: Query "siapa saja yang berbagi workspace dengan user X" dipakai untuk menentukan target broadcast, **BUKAN** broadcast ke seluruh koneksi aktif di server.
- **Definition of Done**: Test: User A dan User B sama-sama di Workspace 1; User C tidak di Workspace 1. User A ubah status → User B menerima event, User C **tidak** menerima.
- **Dependency**: Task 11.2.1, Task 3.1.3 (Sprint 3 — query member)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Query sqlc `ListWorkspacesSharedByUsers` atau `ListMemberUserIDsSharingWorkspaceWith(userID)` (efisien, memanfaatkan index `idx_members_user_id`/`idx_members_workspace_id` yang sudah ada Sprint 3)
- [ ] Broadcast hanya ke `ConnectionRegistry` untuk `user_id` dalam daftar hasil query tersebut (bukan per-channel seperti broadcast pesan — perlu variasi method `BroadcastToUsers(userIDs, msg)` di `ConnectionRegistry`, perluasan dari Task 6.1.2 Sprint 4)
- [ ] Test: isolasi antar workspace terverifikasi

---

## EPIC 12: Realtime Signal

### Feature 12.1: Typing Indicator

#### Task 12.1.1: Tuntaskan Handler `typing.start` (dari Stub Sprint 4)

- **Deskripsi**: FR-PRES-03 — server **tidak** menyimpan state typing secara persisten (bahkan tidak di Redis), cukup broadcast langsung; timeout 5 detik murni logic **client-side**.
- **Acceptance Criteria**: Event `typing.start` diterima server → langsung di-broadcast sebagai `typing.updated` ke channel terkait (via `ConnectionRegistry.Broadcast`, reuse Task 6.1.2 Sprint 4) — **tanpa** menyentuh database maupun Redis sama sekali (server sepenuhnya stateless untuk fitur ini, sesuai HLD §3 "kapan event tidak perlu Outbox").
- **Definition of Done**: Test: Client A kirim `typing.start` → Client B (di channel sama) menerima `typing.updated` dalam waktu singkat; verifikasi tidak ada write ke Redis/PostgreSQL untuk event ini (code review checklist eksplisit, bukan hanya functional test).
- **Dependency**: Task 6.2.1 (Sprint 4), Task 6.1.2 (Sprint 4)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi handler `typing.start` — validasi permission baca channel (reuse `ChannelAuthorizationService`), lalu broadcast langsung
- [ ] **JANGAN** tambahkan penyimpanan state apapun untuk fitur ini (RULES.md prinsip presence/typing fire-and-forget)
- [ ] Test: broadcast diterima, verifikasi tidak ada write ke storage manapun

---

### Feature 12.2: Read Receipt

#### Task 12.2.1: Update Read Receipt saat Client Membaca Channel

- **Deskripsi**: FR-PRES-04 — per-user per-channel (`last_read_message_id`), bukan per-pesan (tabel `read_receipts` sudah dimigrasikan Sprint 4, Task 7.1.2).
- **Acceptance Criteria**: Endpoint `PUT /channels/{id}/read-receipt` (tidak ada di API Spec awal secara eksplisit — **ditambahkan sebagai amandemen kecil API Specification**, konsisten dengan FR-PRES-04 yang sudah ada di SRS) menerima `last_read_message_id`, upsert ke `read_receipts`.
- **Definition of Done**: Test: update read receipt → query berikutnya mengembalikan `last_read_message_id` terbaru; update dengan `message_id` lebih lama dari yang sudah tersimpan **ditolak** (read receipt tidak boleh mundur — validasi berdasarkan `created_at` pesan, bukan hanya menerima nilai apa adanya).
- **Dependency**: Task 7.1.2 (Sprint 4 — migrasi `read_receipts`)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Should

**Subtask & Checklist**:
- [ ] Query sqlc `UpsertReadReceipt` (`INSERT ... ON CONFLICT (user_id, channel_id) DO UPDATE ... WHERE excluded lebih baru`)
- [ ] Handler `PUT /channels/{id}/read-receipt` — **catat amandemen ini di `10-api-specification.md` Changelog**
- [ ] Validasi: tolak update mundur (bandingkan `created_at` pesan baru vs pesan tersimpan)
- [ ] Test: update maju berhasil, update mundur ditolak/diabaikan

#### Task 12.2.2: Broadcast `read_receipt.updated` (Opsional untuk Visual "Sudah Dibaca")

- **Deskripsi**: Notifikasi realtime ke pengirim pesan bahwa pesannya sudah dibaca.
- **Acceptance Criteria**: Broadcast hanya ke `author_id` pesan yang bersangkutan (bukan seluruh channel — cukup pengirim yang perlu tahu status baca pesannya).
- **Definition of Done**: Test: User A kirim pesan, User B baca (update read receipt) → User A menerima event realtime.
- **Dependency**: Task 12.2.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Should

**Subtask & Checklist**:
- [ ] Broadcast event ke `author_id` pesan terkait via `ConnectionRegistry.BroadcastToUsers` (reuse Task 11.3.1)
- [ ] Test: broadcast diterima hanya oleh pengirim relevan

---

### Feature 12.3: Integration Test End-to-End Sprint 7 — Sekaligus Penutup Release 2

#### Task 12.3.1: Skenario Penuh — Presence & Realtime Signal

- **Deskripsi**: Verifikasi Sprint Goal + **gerbang kelulusan Release 2 secara keseluruhan** (menggabungkan hasil Sprint 4-7).
- **Acceptance Criteria**: Alur: (a) User A online → User B (workspace sama) melihat status online, User C (workspace beda) tidak; (b) User A set `invisible` → User B melihat `offline`; (c) User A mulai mengetik di channel → User B melihat indikator typing; (d) User A kirim pesan, User B baca → User A melihat read receipt update; (e) User A disconnect tanpa graceful close (simulasi) → setelah TTL habis, status otomatis jadi `offline` bagi User B.
- **Definition of Done**: Test hijau konsisten 3x run berturut.
- **Dependency**: Seluruh task Epic 11, 12
- **Estimasi Kesulitan**: Sedang-Tinggi
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis skenario (a)-(e) sebagai test suite terstruktur (reuse WS test harness dari Sprint 4/6)
- [ ] Skenario (e) — simulasikan disconnect tidak normal (tutup koneksi TCP paksa, bukan close frame), verifikasi TTL fallback bekerja (test dengan TTL dipersingkat khusus untuk test environment agar tidak menunggu 30 detik asli)
- [ ] Jalankan 3x berturut, pastikan tidak flaky
- [ ] **Verifikasi gerbang Release 2**: jalankan ulang seluruh integration test Sprint 4-7 sekaligus (regression check) — pastikan tidak ada fitur sebelumnya yang rusak akibat perubahan sprint ini
- [ ] Update `docs/AGENTS.md` §7 — **Release 2 (Core Realtime) selesai penuh**, siap lanjut Release 3 (Sprint 8: Notification)

---

## Ringkasan Keputusan

1. Sprint 7 mencakup **2 Epic, 5 Feature, 7 task**, menuntaskan seluruh scope Presence & Realtime Signal sekaligus **menutup Release 2 secara keseluruhan**.
2. Typing indicator (Task 12.1.1) diimplementasikan sepenuhnya **stateless** di server — tidak ada penulisan ke Redis maupun PostgreSQL, murni broadcast pass-through, konsisten dengan keputusan HLD §3.
3. Status `invisible` (Task 11.1.2) ditangani lewat **logic masking di service layer** (satu key Redis, ditampilkan berbeda tergantung viewer), bukan dua state terpisah — desain lebih sederhana dan konsisten.
4. Endpoint read receipt (Task 12.2.1) adalah **amandemen kecil** terhadap API Specification — dicatat eksplisit agar dokumen tetap konsisten sebagai Source of Truth.
5. Task 12.3.1 sengaja mencakup **regression check** terhadap seluruh Sprint 4-7 — bukan hanya fitur baru sprint ini — karena ini adalah gerbang kelulusan Release 2 secara utuh.

## Trade-off yang Diterima

- Task 12.2.1/12.2.2 (Read Receipt) berprioritas *Should* — dapat digeser bila kapasitas sprint tidak cukup, tanpa mengorbankan Sprint Goal inti (Presence + Typing Indicator adalah *Must*, Read Receipt adalah pelengkap sesuai prioritas asli di PRD §5).
- Deteksi disconnect murni mengandalkan TTL 30 detik sebagai fallback (bukan deteksi instan) untuk kasus disconnect tidak normal — trade-off yang sudah disetujui sejak SRS/Learning Roadmap Milestone 7, diterima kembali di sini.

## Risiko Arsitektur

- Task 11.3.1 (Broadcast Scoped) menambah query "siapa berbagi workspace dengan user X" pada **setiap** perubahan status — berpotensi menjadi titik performa yang perlu dipantau di Milestone 11 bila user dengan banyak workspace (each dengan banyak member) sering berganti status secara bersamaan (mis. saat restart massal setelah downtime). Dicatat sebagai kandidat optimasi (caching daftar "shared workspace members") bila diperlukan.
- Perluasan `ConnectionRegistry.BroadcastToUsers` (dipakai Task 11.3.1 dan 12.2.2) menambah code path baru di komponen yang sudah ditandai "Estimasi Kesulitan Tinggi" sejak Sprint 4 (Task 6.1.2) — pastikan test race detector tetap dijalankan untuk method baru ini, bukan hanya method `Broadcast` asli.

## Technical Debt yang Sengaja Diterima

- Caching daftar "shared workspace members" untuk presence broadcast belum diimplementasikan Sprint 7 — dievaluasi di Milestone 11 berdasarkan data beban nyata, bukan diasumsikan perlu sekarang (YAGNI).

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah Task 12.2.1/12.2.2 (Read Receipt, *Should*) dikerjakan penuh di Sprint 7, atau digeser?
2. Dengan selesainya Sprint 7, **Release 2 (Core Realtime) sudah terencana detail penuh** dari Sprint 4 hingga 7. Apakah Anda ingin saya lanjut menyiapkan **Sprint 8** (awal Release 3 — Notification), atau berhenti dulu di titik ini menunggu Release 2 dieksekusi (sesuai checkpoint "Minimum Viable Learning" di Development Roadmap §3, yang sebenarnya baru tercapai setelah Release 3)?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 7: 2 Epic, 5 Feature, 7 task, menuntaskan Presence & Realtime Signal sekaligus menutup Release 2 dengan regression check menyeluruh |
