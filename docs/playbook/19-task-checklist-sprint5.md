# Detailed Task Checklist — Sprint 5
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 5: Reply, Thread, Mention, Reaction, Direct Message)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `14-sprint-planning.md` (Release 2 overview), `06-srs.md` (§2.4 FR-MSG-03/04/05/06, §2.9 FR-DM), `07-hld.md` (§2.6, §2.14), `08-lld.md` (§1.2, §2.5), `09-database-design.md` (§2.3-2.4), `10-api-specification.md` (§4-5)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen & Prasyarat

Melanjutkan pemecahan Release 2 dari `18-task-checklist-sprint4.md` §0. Sprint 5 menuntaskan sisa Sprint Goal Release 2 yang belum tersentuh Sprint 4: **Reply, Thread, Mention, Reaction** (perluasan Messaging Core) dan **Direct Message** (fitur baru, amandemen PRD v1.1/SRS v1.1).

**Prasyarat**: Sprint 4 selesai — khususnya `MessageService.Send`/`Edit`/`Delete`, `ConnectionRegistry.Broadcast`, dan skema `mentions`/`reactions` (sudah dimigrasikan Task 7.1.2, namun belum dipakai aktif).

**Sprint Goal**: Pesan mendukung reply/thread/mention/reaction penuh; user dapat memulai DM 1-on-1 dan grup (maks 10 partisipan), dengan block enforcement berfungsi dan diverifikasi end-to-end — menuntaskan **seluruh** scope Release 2 (Sprint Planning §2).

---

## EPIC 8: Messaging Advanced

### Feature 8.1: Reply

#### Task 8.1.1: Dukungan `reply_to_id` pada Send Message

- **Deskripsi**: Perluas `MessageService.Send` (Sprint 4, Task 7.2.1) untuk menerima `reply_to_id` opsional, sesuai FR-MSG-03.
- **Acceptance Criteria**: Bila `reply_to_id` diisi, sistem memverifikasi pesan target ada (boleh sudah soft-deleted — FR-MSG-03 menyatakan reply ke pesan terhapus tetap valid, ditampilkan sebagai "membalas pesan yang telah dihapus" di sisi client, bukan ditolak backend).
- **Definition of Done**: Test: reply ke pesan aktif sukses; reply ke pesan soft-deleted tetap sukses tersimpan (validasi ada di frontend rendering, bukan backend reject); reply ke `message_id` yang sama sekali tidak ada → `404`.
- **Dependency**: Task 7.2.1 (Sprint 4)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tambah parameter `reply_to_id *uuid.UUID` di `SendMessageCommand`
- [ ] Validasi: bila diisi, pesan target harus ada di `channel_id` yang sama (termasuk yang sudah soft-deleted)
- [ ] Test: 3 skenario di atas

---

### Feature 8.2: Thread

#### Task 8.2.1: Handler — `POST /messages/{id}/threads`

- **Deskripsi**: FR-MSG-04 — thread sebagai pesan dengan `thread_root_id` mengarah ke pesan induk.
- **Acceptance Criteria**: Sesuai API Specification §4 — permission `SEND_MESSAGES` di channel induk (bukan permission terpisah untuk thread).
- **Definition of Done**: Test: buat thread dari pesan → `thread_root_id` tersimpan benar; thread dari pesan yang tidak ada → `404`.
- **Dependency**: Task 8.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `POST /messages/{id}/threads` — set `thread_root_id = messageId` pada pesan baru
- [ ] Reuse `MessageService.Send` (bukan service terpisah — DRY, thread adalah variasi Message)
- [ ] Test: create thread sukses, dari pesan tidak ada → 404

#### Task 8.2.2: Query — List Pesan dalam Thread

- **Deskripsi**: Perluas `ListMessagesByChannel` (Sprint 4) atau buat query terpisah `ListMessagesByThreadRoot`.
- **Acceptance Criteria**: Memakai index `idx_messages_thread_root` (sudah dibuat Task 7.1.1) — cursor pagination sama seperti list channel biasa.
- **Definition of Done**: Test: list thread mengembalikan hanya pesan dengan `thread_root_id` yang sesuai, terurut kronologis.
- **Dependency**: Task 8.2.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Query sqlc `ListMessagesByThreadRoot` (cursor-based, pola sama Task 7.1.3)
- [ ] Handler `GET /messages/{id}/threads`
- [ ] Test: list thread terisolasi dari pesan channel utama

---

### Feature 8.3: Mention

#### Task 8.3.1: Parsing & Penyimpanan Mention

- **Deskripsi**: FR-MSG-05 — mention user (`@username`), role (`@role_name`), `@everyone`/`@here`.
- **Acceptance Criteria**: Request `mentions: []uuid` (user IDs, sudah di-parse frontend dari markdown — backend tidak melakukan parsing teks bebas, hanya menerima daftar eksplisit, lebih aman & sederhana daripada regex parsing di server). Mention `@everyone`/`@here` butuh permission `MENTION_EVERYONE` (flag baru, tambahkan ke `PermissionFlag` — Task 3.4.2 Sprint 3).
- **Definition of Done**: Test: mention user biasa tersimpan di tabel `mentions`; mention `@everyone` tanpa permission → `403`; dengan permission → sukses.
- **Dependency**: Task 7.2.1 (Sprint 4), Task 3.4.2 (Sprint 3 — tambah flag `MENTION_EVERYONE`)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tambah flag `MENTION_EVERYONE` ke `internal/workspace/domain/permission.go` (Task 3.4.2)
- [ ] Perluas `SendMessageCommand` dengan `Mentions []uuid.UUID`, `MentionEveryone bool`, `MentionRoles []uuid.UUID`
- [ ] Validasi `MentionEveryone`/mention role besar via Permission Resolver
- [ ] Insert ke tabel `mentions` dalam transaksi yang sama dengan insert message
- [ ] Test: mention user, mention everyone (dengan & tanpa permission)

---

### Feature 8.4: Reaction

#### Task 8.4.1: Handler — `PUT/DELETE /messages/{id}/reactions/{emoji}`

- **Deskripsi**: FR-MSG-06 — idempotent add, unique per `(message_id, user_id, emoji)`.
- **Acceptance Criteria**: `PUT` dua kali dengan emoji sama → tidak error, no-op (idempotent by design, memanfaatkan `ON CONFLICT DO NOTHING` di query sqlc, bukan try-catch di application layer). `DELETE` hanya oleh pemilik reaksi.
- **Definition of Done**: Test: add reaction 2x → hanya 1 baris tersimpan; delete reaction orang lain → `403`/no-op (pilih salah satu perilaku dan dokumentasikan — direkomendasikan `204` no-op tanpa error untuk konsistensi idempotent, karena delete yang tidak ada efeknya bukan pelanggaran keamanan).
- **Dependency**: Task 7.1.2 (Sprint 4 — migrasi `reactions`)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Query sqlc `AddReaction` (`INSERT ... ON CONFLICT (message_id, user_id, emoji) DO NOTHING`)
- [ ] Query sqlc `RemoveReaction` (`DELETE WHERE message_id=$1 AND user_id=$2 AND emoji=$3`)
- [ ] Handler `PUT /messages/{id}/reactions/{emoji}`, `DELETE /messages/{id}/reactions/{emoji}`
- [ ] Broadcast `message.reaction_added`/`message.reaction_removed` via WS (perluas envelope event, API Spec §10)
- [ ] Test: add duplikat (idempotent), remove oleh bukan pemilik

---

## EPIC 9: Direct Message (DM)

### Feature 9.1: Migrasi Database — Channel Members & User Blocks

#### Task 9.1.1: Migrasi Tabel `channel_members`, `user_blocks`

- **Deskripsi**: DDL sesuai Database Design §2.3 — belum dibuat di Sprint 3 (hanya `channels`/`channel_permission_overrides` yang dimigrasikan saat itu).
- **Acceptance Criteria**: `channel_members` dengan composite PK `(channel_id, user_id)`; `user_blocks` dengan composite PK `(blocker_id, blocked_id)`, directional (bukan simetris).
- **Definition of Done**: `migrate up` sukses; test insert block A→B tidak otomatis membuat block B→A (verifikasi directional).
- **Dependency**: Task 3.6.1 (Sprint 3 — tabel `channels` sudah ada)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `channel_members` (up & down)
- [ ] Tulis migrasi `user_blocks` (up & down)
- [ ] Test: insert directional block, verifikasi arah sebaliknya tidak otomatis ada

---

### Feature 9.2: DM Authorization Logic

#### Task 9.2.1: Implementasi `BuildDMChannelKey` (Uniqueness 1-on-1)

- **Deskripsi**: Implementasi persis LLD §2.5 — sort partisipan deterministik, hash untuk `participant_key`.
- **Acceptance Criteria**: `BuildDMChannelKey([userA, userB])` dan `BuildDMChannelKey([userB, userA])` menghasilkan key **identik** (urutan input tidak berpengaruh).
- **Definition of Done**: Unit test: berbagai urutan input menghasilkan key sama; pasangan user berbeda menghasilkan key berbeda.
- **Dependency**: Task 9.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `BuildDMChannelKey` (sort UUID byte-wise, SHA-256 hash) — LLD §2.5 persis
- [ ] Unit test: urutan input berbeda → key sama; pasangan berbeda → key berbeda

#### Task 9.2.2: Implementasi `dmAuthzService.CanWrite` (Block Enforcement)

- **Deskripsi**: Implementasi persis LLD §2.5 — cek membership + cek block dari SEMUA partisipan lain di channel.
- **Acceptance Criteria**: **RULES.md §8 wajib dipatuhi**: cek block SEBELUM mengizinkan create/send DM.
- **Definition of Done**: Unit test: user yang diblokir salah satu partisipan grup DM tidak dapat mengirim pesan ke channel tersebut; user yang tidak diblokir siapapun dapat mengirim normal.
- **Dependency**: Task 9.1.1, Task 1.2 LLD (`ChannelAuthorizationService`, Sprint 4)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `dmAuthzService.CanWrite` persis pseudocode LLD §2.5
- [ ] Wire ke `ChannelAuthorizationService.CanWrite` (percabangan `IF ch.Type == ChannelTypeDM`, sudah di-stub Sprint 4 Task 7.2.1)
- [ ] Unit test: blocked user tidak bisa kirim, non-blocked bisa

---

### Feature 9.3: Create DM Channel

#### Task 9.3.1: Service — DMService.CreateOrFind

- **Deskripsi**: FR-DM-01/02/03 — 1-on-1 unik (find-or-create), grup DM maks 10 partisipan.
- **Acceptance Criteria**: Bila `participant_ids` = 2 (1-on-1) dan channel dengan `participant_key` sama sudah ada → kembalikan channel existing (bukan buat baru). Bila `participant_ids` 3-10 → selalu buat channel grup baru (grup DM tidak unik-constraint seperti 1-on-1). Bila > 10 → `422 BUSINESS_RULE_VIOLATION`.
- **Definition of Done**: Test: create 1-on-1 dua kali dengan pasangan sama → channel ID identik di kedua panggilan; create grup 11 partisipan → ditolak; create dengan partisipan yang saling blokir → ditolak.
- **Dependency**: Task 9.2.1, 9.2.2
- **Estimasi Kesulitan**: Sedang-Tinggi
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `DMService.CreateOrFind(ctx, requesterID, participantIDs)`
- [ ] Cek block SEBELUM cek uniqueness (RULES.md §8 — urutan penting untuk mencegah kebocoran informasi keberadaan channel, sesuai catatan risiko API Spec §Risiko)
- [ ] Untuk 1-on-1: hitung `participant_key`, cek existing via partial unique index (`FindByParticipantKey`), buat baru bila tidak ada
- [ ] Untuk grup (3-10): buat channel baru + insert seluruh partisipan ke `channel_members`
- [ ] Validasi jumlah partisipan (2-10), tolak di luar rentang
- [ ] Test: seluruh skenario di atas

#### Task 9.3.2: Handler — `POST /api/v1/dm`

- **Deskripsi**: Sesuai API Specification §5, dengan `Idempotency-Key` wajib.
- **Acceptance Criteria**: Response `200` bila channel sudah ada (existing), `201` bila baru dibuat.
- **Definition of Done**: Test HTTP: create 1-on-1 pertama kali → 201; create lagi dengan pasangan sama → 200 dengan channel ID sama.
- **Dependency**: Task 9.3.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `POST /dm`, validasi `Idempotency-Key` header ada
- [ ] Response status dinamis (200 vs 201) sesuai apakah channel baru dibuat
- [ ] Test: 201 pertama kali, 200 saat sudah ada

---

### Feature 9.4: Block Management

#### Task 9.4.1: Handler — `POST/DELETE /users/{userId}/block`

- **Deskripsi**: FR-DM-04.
- **Acceptance Criteria**: Sesuai API Specification §5.
- **Definition of Done**: Test: block sukses (204), unblock sukses (204), block diri sendiri ditolak (`422`).
- **Dependency**: Task 9.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Query sqlc `CreateBlock` (`ON CONFLICT DO NOTHING` — idempotent), `RemoveBlock`
- [ ] Handler `POST /users/{userId}/block`, `DELETE /users/{userId}/block`
- [ ] Validasi: `userId == self` → `422 BUSINESS_RULE_VIOLATION`
- [ ] Test: block, unblock, block diri sendiri ditolak

---

### Feature 9.5: Integration Test End-to-End Sprint 5

#### Task 9.5.1: Skenario Penuh — Messaging Advanced + DM

- **Deskripsi**: Verifikasi Sprint Goal, sekaligus **gerbang kelulusan Release 2 penuh** (Development Roadmap §2 — Release 2 "Core Realtime" mencakup Sprint 4-6, namun Sprint 4-5 adalah bagian Messaging+DM yang menjadi fokus verifikasi di sini; Upload (Sprint 6) diverifikasi terpisah).
- **Acceptance Criteria**: Alur: (a) User A kirim pesan, User B reply, verifikasi `reply_to_id` benar; (b) User A buat thread dari pesan, kirim 2 balasan thread, list thread mengembalikan 2 pesan terisolasi dari channel utama; (c) User A mention User B, verifikasi tersimpan di `mentions`; (d) User A react ke pesan, react lagi (idempotent, tetap 1 baris); (e) User A buat DM ke User B, kirim pesan, User B menerima realtime via WS (reuse infrastruktur Sprint 4); (f) User B blokir User A, User A mencoba kirim pesan lagi ke DM tersebut → `403`.
- **Definition of Done**: Test hijau konsisten 3x run berturut.
- **Dependency**: Seluruh task Epic 8, 9
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 4 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis skenario (a)-(f) sebagai satu test suite terstruktur
- [ ] Verifikasi realtime DM (reuse WS test harness dari Task 7.6.1 Sprint 4)
- [ ] Jalankan 3x berturut, pastikan tidak flaky
- [ ] Update `docs/AGENTS.md` §7 — Release 2 (Messaging+DM) selesai, siap lanjut Sprint 6 (Upload)

---

## Ringkasan Keputusan

1. Sprint 5 menuntaskan **2 Epic (8, 9), 9 Feature, 13 task**, melengkapi seluruh scope Messaging (reply/thread/mention/reaction) dan Direct Message yang menjadi amandemen resmi PRD v1.1.
2. Parsing mention **dilakukan di frontend** (backend hanya menerima daftar ID eksplisit) — keputusan pragmatis yang menghindari kompleksitas regex parsing markdown di server, konsisten dengan *Simplicity over Cleverness*.
3. Urutan pengecekan di `DMService.CreateOrFind` (Task 9.3.1) **wajib** cek block sebelum cek uniqueness — menutup risiko kebocoran informasi yang sudah diidentifikasi sejak API Specification §Risiko.
4. Reaction idempotent diimplementasikan di level **database** (`ON CONFLICT DO NOTHING`), bukan di application layer — lebih robust terhadap race condition dibanding cek-lalu-insert di kode Go.

## Trade-off yang Diterima

- Reaction removal oleh non-pemilik didesain sebagai no-op (`204` tanpa efek), bukan `403` — pilihan konsisten dengan sifat idempotent operasi, namun berarti user tidak mendapat feedback eksplisit bahwa aksinya "tidak berpengaruh"; dapat direvisi di frontend UX bila diperlukan tanpa mengubah backend.

## Risiko Arsitektur

- Task 9.3.1 (DMService.CreateOrFind) dan Task 9.5.1 (integration test) ditandai **Tinggi/Sedang-Tinggi** — kombinasi uniqueness constraint database + block enforcement + idempotency adalah logika paling kompleks di sprint ini, rawan edge case tak terduga (mis. race condition dua request create 1-on-1 bersamaan — partial unique index Database Design §2.3 menjadi pengaman terakhir, namun perlu test eksplisit untuk race ini bila waktu memungkinkan, dicatat sebagai potensi tambahan di Milestone 11).

## Technical Debt yang Sengaja Diterima

- Race condition test untuk `DMService.CreateOrFind` (dua request paralel create 1-on-1 bersamaan) belum eksplisit ditulis di Sprint 5 — mengandalkan partial unique index database sebagai pengaman, verifikasi test konkurensi eksplisit ditunda ke Milestone 11 (Optimization/Load Test).
- Flag permission baru `MENTION_EVERYONE` (Task 8.3.1) menambah daftar dari 8 menjadi 9 flag — pola penambahan bertahap ini terus berlanjut sesuai kebutuhan fitur (YAGNI, konsisten dengan catatan Sprint 3).

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah perilaku reaction removal oleh non-pemilik sebagai **no-op 204** (bukan 403) dapat diterima?
2. Dengan selesainya Sprint 4-5, **Release 2 (Messaging + DM) sudah terencana detail penuh**, menyisakan Upload (Sprint 6) dan Presence & Realtime Signal (Sprint 7) untuk menuntaskan Release 2 sepenuhnya. Lanjut ke **Sprint 6 (Upload)**, atau berhenti dulu menunggu Sprint 4-5 dieksekusi?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 5: 2 Epic (Messaging Advanced, Direct Message), 9 Feature, 13 task, menuntaskan sisa scope Messaging dan seluruh scope DM dari Release 2 |
