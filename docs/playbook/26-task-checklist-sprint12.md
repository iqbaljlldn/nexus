# Detailed Task Checklist — Sprint 12
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 12: Event Driven Migration — Milestone 12, Transisi ke Phase B)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `03-adr.md` (ADR-006 — Redis Streams), `07-hld.md` (§1.2 Phase B, §3 Event Catalog), `08-lld.md` (§2.4 Outbox Relay, §2.7 Idempotent Consumer), `09-database-design.md` (§2.8), `22-task-checklist-sprint8.md` (TODO yang harus diresolusi)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen, Prasyarat, dan Makna Sprint Ini

**Sprint ini adalah transisi arsitektur nyata**: setelah sprint ini selesai, proyek benar-benar berpindah dari **Phase A (Modular Monolith)** ke **Phase B (Event-Driven Modular Monolith)** — bukan lagi konsep di dokumen, tapi kondisi kode sungguhan (HLD §1.2).

**Prasyarat**: Sprint 11 (Optimization) selesai — fondasi monolith harus solid sebelum kompleksitas asynchronous ditambahkan (Learning Roadmap M11 rationale, dipatuhi urutan sprintnya).

**Sprint Goal** (persis Development Roadmap §2, Release 4): Outbox Pattern aktif untuk minimal 3 domain event kunci (**message, member, attachment**), consumer idempotent terverifikasi lewat test retry.

**Item wajib yang diresolusi sprint ini**: TODO eksplisit yang ditinggalkan sejak Sprint 8 —

```go
// TODO(release-4): ganti in-process call ini dengan Outbox event publish saat Milestone 12
```

di `MessageService.Send` (Task 13.4.1). Ini bukan opsional — sudah dijanjikan sejak Sprint 8 sebagai debt dengan rencana pelunasan eksplisit (Sprint 8 §Ringkasan Keputusan poin 3).

---

## EPIC 17: Event-Driven Infrastructure

### Feature 17.1: Migrasi Database — Outbox & Idempotency

#### Task 17.1.1: Migrasi Tabel `outbox_events`, `processed_events`

- **Deskripsi**: DDL sesuai Database Design §2.8 (sudah dirancang sejak Phase 5, baru sekarang benar-benar dimigrasikan — konsisten dengan pola proyek ini menyiapkan skema lebih awal dari fitur aktifnya).
- **Acceptance Criteria**: Index `idx_outbox_unpublished` (partial, `WHERE published_at IS NULL`) aktif — kritikal untuk performa relay worker (LLD §2.4).
- **Definition of Done**: `migrate up` sukses.
- **Dependency**: Sprint 11 selesai
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `outbox_events` (up & down)
- [ ] Tulis migrasi `processed_events` (up & down)
- [ ] Verifikasi index partial `idx_outbox_unpublished`

---

### Feature 17.2: Generic Event Bus Package

#### Task 17.2.1: `pkg/eventbus` — Publisher & Consumer Wrapper Redis Streams

- **Deskripsi**: Sesuai Playbook §3.1 — abstraksi generic di `pkg/`, tidak tahu apapun tentang domain (`message`, `member`, dll).
- **Acceptance Criteria**: `Publisher.Publish(ctx, streamKey string, event Event) error`; `Consumer.Subscribe(ctx, streamKey, consumerGroup string, handler HandlerFunc) error` — memakai Redis Streams consumer group (`XREADGROUP`), bukan Pub/Sub biasa (ADR-006).
- **Definition of Done**: Unit test (Redis test container): publish → consume oleh 1 consumer group; publish → 2 consumer group berbeda masing-masing menerima (memverifikasi karakteristik fan-out consumer group Redis Streams).
- **Dependency**: Task 17.1.1
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 4 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `pkg/eventbus/publisher.go` (`XADD`)
- [ ] Implementasi `pkg/eventbus/consumer.go` (`XREADGROUP`, `XACK`, `XPENDING` untuk retry pesan tertunda — sesuai Learning Roadmap M12)
- [ ] Auto-create consumer group bila belum ada (`XGROUP CREATE ... MKSTREAM`)
- [ ] Unit test: publish-consume single group, fan-out 2 group

---

### Feature 17.3: Outbox Relay Worker

#### Task 17.3.1: Implementasi `OutboxRelayWorker` (Persis LLD §2.4)

- **Deskripsi**: Implementasi pseudocode LLD §2.4 apa adanya — polling ticker 500ms, batch 100, at-least-once delivery diterima secara sadar.
- **Acceptance Criteria**: **RULES.md §6 wajib dipatuhi**: event ditulis ke `outbox_events` dalam transaksi yang sama dengan perubahan data (bukan publish langsung); relay worker terpisah yang membaca & mempublikasikan.
- **Definition of Done**: Test: insert baris outbox manual (simulasi) → worker mem-publish ke Redis Streams dalam < 1 detik (mengingat ticker 500ms) → `published_at` terisi.
- **Dependency**: Task 17.2.1, Task 10.4.1 (Sprint 6 — reuse worker binary `cmd/worker`)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `internal/platform/outbox/relay_worker.go` persis LLD §2.4
- [ ] Wire sebagai goroutine tambahan di `cmd/worker` (reuse binary Sprint 6, bukan binary baru — worker sudah ada, relay adalah tanggung jawab tambahan di dalamnya)
- [ ] Stream key per `aggregate_type`: `stream:message:events`, `stream:member:events`, `stream:attachment:events` (Playbook §7.5 konvensi)
- [ ] Test: insert manual → ter-publish, `published_at` terisi
- [ ] Test: simulasikan Redis down saat publish → baris tetap `published_at = NULL`, dicoba lagi batch berikutnya (verifikasi resilience, bukan crash worker)

---

### Feature 17.4: Wiring Outbox — 3 Domain Event Kunci

#### Task 17.4.1: `message.MessageCreated` — Refactor dari In-Process Call

- **Deskripsi**: **Menuntaskan TODO Sprint 8** — mengganti panggilan langsung `MessageService.Send` → `NotificationDispatcher.Dispatch` dengan **insert ke outbox** dalam transaksi yang sama dengan `INSERT message`.
- **Acceptance Criteria**: Broadcast WebSocket realtime (Task 7.2.3 Sprint 4) **TETAP synchronous in-process** (HLD §3 — ini tidak berubah, hanya trigger notifikasi yang berpindah ke Outbox); hanya alur notifikasi yang berpindah dari in-process call ke event asynchronous.
- **Definition of Done**: Kirim pesan dengan mention → (a) broadcast WS ke channel tetap instan seperti sebelumnya (regression check terhadap Sprint 4), (b) baris baru muncul di `outbox_events` dengan `event_type = 'message.MessageCreated'` dalam transaksi yang sama dengan insert message, (c) **hapus** kode in-process call langsung ke `NotificationDispatcher` dari `MessageService.Send`.
- **Dependency**: Task 17.3.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Hapus panggilan langsung `NotificationDispatcher.Dispatch` dari `MessageService.Send` (dan hapus komentar TODO — resolved)
- [ ] Tambah `INSERT INTO outbox_events` dalam transaksi yang sama dengan `INSERT message` — payload mencakup seluruh data yang dulu dikirim langsung (nama pengirim, cuplikan, mentions, channel info)
- [ ] Regression test Sprint 4 (broadcast WS instan) — pastikan **tidak berubah**
- [ ] Test baru: verifikasi baris outbox dibuat dalam transaksi yang sama (rollback simulasi — bila insert message gagal, baris outbox juga tidak ada)

#### Task 17.4.2: `member.MemberJoined` — Outbox saat Redeem Invite

- **Deskripsi**: Perluas `InviteService.Redeem` (Sprint 3, Task 3.3.1) — tambah insert outbox dalam transaksi yang sama dengan insert `members`.
- **Acceptance Criteria**: Event `member.MemberJoined` di-publish, payload mencakup `workspace_id`, `user_id`, `joined_at`.
- **Definition of Done**: Test: redeem invite → baris outbox baru muncul dalam transaksi yang sama.
- **Dependency**: Task 17.3.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tambah `INSERT INTO outbox_events` di `InviteService.Redeem`, transaksi sama dengan insert `members`
- [ ] Test: redeem → outbox event tercatat

#### Task 17.4.3: `attachment.FileUploaded` — Outbox saat Upload Selesai Streaming

- **Deskripsi**: Perluas `UploadService` (Sprint 6, Task 10.3.1) — outbox event dipublikasikan setelah file berhasil masuk staging (memicu Media Processing, Sprint 6 Task 10.5.1/10.5.2 yang sebelumnya dipanggil langsung via Asynq enqueue — **tetap** via Asynq untuk task-nya sendiri, namun sekarang **trigger awal**-nya melalui Outbox agar pola konsisten dengan 2 domain event lain).

> **Catatan penting**: berbeda dari Task 17.4.1/17.4.2, di sini ada **dua lapis asynchronous**: Outbox→Redis Streams (untuk notifikasi konsumen lain yang mungkin tertarik pada "file diupload", mis. Search indexer masa depan) TETAP terpisah dari Asynq task queue (untuk pemrosesan thumbnail/metadata itu sendiri, sudah ada sejak Sprint 6). **Asynq enqueue untuk pemrosesan file TIDAK diganti** — itu tetap cara yang tepat untuk task-based processing (LLD §2.10 rationale masih berlaku). Outbox di sini murni untuk *event notification* ke konsumen lain di masa depan.

- **Acceptance Criteria**: Outbox event `attachment.FileUploaded` dipublikasikan terpisah dari (tidak menggantikan) Asynq enqueue thumbnail/metadata yang sudah ada.
- **Definition of Done**: Test: upload file → **baik** outbox event **maupun** Asynq task thumbnail tetap berjalan seperti sebelumnya (regression Sprint 6 tidak rusak).
- **Dependency**: Task 17.3.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tambah `INSERT INTO outbox_events` di `UploadService`, **tanpa mengubah** logic Asynq enqueue Sprint 6
- [ ] Regression test Sprint 6 (Task 10.8.1) tetap lolos
- [ ] Test baru: outbox event tercatat terpisah dari Asynq job

---

### Feature 17.5: Migrasi Notification Consumer ke Event-Driven

#### Task 17.5.1: `NotificationEventConsumer` — Subscribe ke `stream:message:events`

- **Deskripsi**: Implementasi persis LLD §2.7 (Idempotent Event Consumer) — mengganti pemanggil `NotificationDispatchService` dari in-process call (Task 17.4.1 sudah menghapusnya) menjadi event consumer.
- **Acceptance Criteria**: `NotificationDispatchService` (dibuat Sprint 8, Task 13.2.1-13.2.2) **tidak berubah sama sekali** — hanya *pemanggilnya* yang berganti dari HTTP handler in-process menjadi event consumer. Ini membuktikan desain "menerima payload lengkap, tidak query balik" (HLD §2.8) sejak awal memang mempersiapkan migrasi ini dengan mulus.
- **Definition of Done**: Integration test: publish event manual ke `stream:message:events` → consumer memproses → `notification_deliveries` tercatat, broadcast WS terkirim (SAMA seperti perilaku Sprint 8, hanya jalur pemicunya berbeda).
- **Dependency**: Task 17.2.1, Task 13.2.2 (Sprint 8)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `internal/notification/interface/eventconsumer/message_consumer.go`
- [ ] Cek `processed_events` sebelum proses (idempotency, LLD §2.7 persis)
- [ ] Panggil `NotificationDispatchService.Dispatch` (tidak diubah dari Sprint 8) dengan payload dari event
- [ ] `XACK` setelah sukses; **tidak** ack bila gagal (biar di-retry consumer group)
- [ ] Wire consumer sebagai goroutine di `cmd/worker`
- [ ] Test: publish manual → notifikasi terproses sama seperti Sprint 8 perilakunya

#### Task 17.5.2: `MemberEventConsumer` — Subscribe ke `stream:member:events`

- **Deskripsi**: Trigger notifikasi "member baru bergabung" (bila diinginkan sebagai fitur — dicatat opsional bila belum ada di scope PRD eksplisit, namun infrastrukturnya disiapkan sebagai bukti pola generik bekerja untuk event kedua).
- **Acceptance Criteria**: Minimal: consumer menerima event dan mencatat log (placeholder), **membuktikan pola konsumen bekerja untuk event kedua** tanpa harus membangun fitur notifikasi lengkap untuk member join (di luar scope Sprint Goal minimal 3 event *dipublikasikan*, bukan mengharuskan seluruhnya punya consumer fitur lengkap).
- **Definition of Done**: Test: publish `member.MemberJoined` → consumer menerima dan mencatat (idempotent check tetap diterapkan meski aksi minimal).
- **Dependency**: Task 17.2.1, Task 17.4.2
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Should

**Subtask & Checklist**:
- [ ] Implementasi consumer minimal (log + idempotency check, pola sama seperti Task 17.5.1 tapi lebih sederhana)
- [ ] Test: publish → consumer menerima, idempotent terhadap duplikasi

---

### Feature 17.6: Dead Letter Queue

#### Task 17.6.1: Dead Letter Handling — Setelah N Kali Retry Gagal

- **Deskripsi**: Sesuai HLD §3 Event Catalog (`dlq:message`, `dlq:member`, `dlq:attachment`) — event yang gagal diproses berkali-kali (dicek via `XPENDING`, idle time tinggi) dipindahkan ke dead-letter stream.
- **Acceptance Criteria**: Job terjadwal (reuse pola periodic task Sprint 6) memindahkan pesan dengan retry count melebihi threshold (mis. 5x, idle > 10 menit) ke stream `dlq:<domain>`.
- **Definition of Done**: Test: simulasikan consumer selalu gagal untuk satu event → setelah threshold, event berpindah ke `dlq:message`, tidak lagi mengganggu pemrosesan event lain di stream utama.
- **Dependency**: Task 17.2.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi scheduled task cek `XPENDING` per stream, identifikasi pesan idle > threshold
- [ ] Pindahkan ke `dlq:<domain>` stream (`XADD` ke DLQ + `XACK` dari stream asal agar tidak diproses ulang)
- [ ] Log Error saat pesan masuk DLQ (observability dasar — Milestone 15 belum tiba, tapi log wajib tetap ada sesuai Playbook §15)
- [ ] Test: simulasi kegagalan berulang → pesan berpindah ke DLQ

---

### Feature 17.7: Integration Test End-to-End — Idempotency & Retry

#### Task 17.7.1: Test Idempotency — Consumer Diproses Ulang, Tidak Ada Efek Ganda

- **Deskripsi**: **Gerbang kelulusan Sprint Goal** — "consumer idempotent terverifikasi lewat test retry" (persis kalimat Development Roadmap).
- **Acceptance Criteria**: Simulasikan event yang sama diproses 2x oleh consumer (mis. paksa ulang tanpa `XACK` pertama, atau panggil handler manual 2x dengan `event_id` sama) → efek samping (notification delivery, dsb.) **hanya terjadi 1x**, bukan 2x.
- **Definition of Done**: Test hijau, memverifikasi baris `processed_events` mencegah pemrosesan ganda secara nyata (bukan hanya asumsi dari membaca kode).
- **Dependency**: Task 17.5.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Test: proses event dengan `event_id` X → sukses, `processed_events` tercatat
- [ ] Proses ulang event `event_id` X yang sama → handler mendeteksi sudah diproses, langsung `XACK` tanpa efek samping ganda (LLD §2.7 pola persis)
- [ ] Verifikasi: hanya 1 baris `notification_deliveries` untuk event tersebut, bukan 2

#### Task 17.7.2: Skenario Penuh End-to-End — Sekaligus Regression Total

- **Deskripsi**: Verifikasi Sprint Goal secara menyeluruh + regression terhadap seluruh Sprint 1-11.
- **Acceptance Criteria**: Alur: User A kirim pesan dengan mention → outbox → relay → stream → consumer → notifikasi terkirim (WS + email bila offline) — **hasil akhir identik dengan perilaku Sprint 8**, hanya jalurnya sekarang asynchronous sungguhan (bukan in-process).
- **Definition of Done**: Test hijau konsisten 3x run berturut; **jalankan ulang seluruh regression suite Sprint 1-11** — pastikan migrasi arsitektur ini tidak merusak fitur manapun yang sudah ada (ini adalah risiko terbesar sprint ini, mengingat perubahan menyentuh jalur kritikal Messaging).
- **Dependency**: Seluruh task Epic 17
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 4 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis skenario end-to-end penuh (kirim pesan mention → tunggu via polling/event dengan timeout eksplisit → verifikasi notifikasi sampai)
- [ ] Jalankan **seluruh** test suite proyek (regression Sprint 1-11), pastikan tidak ada yang rusak
- [ ] Jalankan 3x berturut untuk skenario baru, pastikan tidak flaky
- [ ] Update `docs/AGENTS.md` §7 — **Sprint 12 selesai. Fase Arsitektur berubah dari "Phase A — Modular Monolith" menjadi "Phase B — Event-Driven Modular Monolith"** (update eksplisit, ini adalah perubahan status paling signifikan sejak proyek dimulai)

---

## Ringkasan Keputusan

1. Sprint 12 mencakup **1 Epic, 7 Feature, 12 task**, menandai **transisi arsitektur nyata pertama** dalam proyek — Phase A → Phase B.
2. TODO yang sengaja ditinggalkan sejak Sprint 8 (Task 13.4.1) **diresolusi tuntas** di Task 17.4.1 — bukti bahwa debt yang didokumentasikan dengan rencana pelunasan eksplisit benar-benar dilunasi, bukan dilupakan.
3. `NotificationDispatchService` (dibuat Sprint 8) **tidak berubah sama sekali** saat migrasi ke event-driven (Task 17.5.1) — memvalidasi keputusan desain HLD §2.8 ("menerima payload lengkap, tidak query balik") yang sejak awal memang dirancang untuk memudahkan transisi ini.
4. Broadcast WebSocket realtime pesan (Task 7.2.3 Sprint 4) **sengaja TIDAK diubah** menjadi asynchronous — tetap synchronous in-process, konsisten dengan HLD §3 (dibedakan tegas dari notifikasi yang memang asynchronous).
5. Attachment (Task 17.4.3) memperkenalkan pola "dua lapis asynchronous" (Asynq untuk task processing + Outbox untuk event notification) — dijelaskan eksplisit agar tidak membingungkan, keduanya punya tujuan berbeda dan tidak saling menggantikan.

## Trade-off yang Diterima

- Task 17.5.2 (`MemberEventConsumer`) diimplementasikan minimal (log + idempotency check) tanpa fitur notifikasi lengkap — cukup untuk membuktikan pola bekerja untuk event kedua, sesuai Sprint Goal ("minimal 3 event **dipublikasikan**", bukan seluruhnya harus punya consumer fitur lengkap).
- At-least-once delivery (bukan exactly-once) tetap menjadi model yang diterima — Task 17.7.1 membuktikan idempotency di level aplikasi menutup celah ini secara memadai.

## Risiko Arsitektur

- Task 17.7.2 (regression total) adalah task **paling berisiko** di sprint ini — perubahan menyentuh jalur kritikal `MessageService.Send` yang sudah dipakai sejak Sprint 4 dan diperluas di Sprint 5, 6, 8. Kesalahan kecil di sini berpotensi merusak fitur yang sudah berfungsi baik di 8 sprint sebelumnya. **Rekomendasi eksekusi**: kerjakan Task 17.4.1 di awal sprint (bukan akhir), agar ada waktu cukup mendeteksi regresi sebelum sprint berakhir.
- Consumer group Redis Streams (Task 17.2.1) menambah kompleksitas operasional baru (consumer group harus di-monitor agar tidak ada pesan menumpuk di `XPENDING` tanpa DLQ yang berfungsi) — Task 17.6.1 memitigasi ini, namun perlu diverifikasi benar-benar berjalan sebagai scheduled task yang reliable, bukan sekadar kode yang ada tapi tidak pernah dieksekusi.

## Technical Debt yang Sengaja Diterima

- Search Indexer consumer (disebut di HLD §3 Event Catalog) **belum** diimplementasikan Sprint 12 — akan datang bersamaan dengan fitur Search itu sendiri (belum didetailkan, rolling wave), memanfaatkan infrastruktur `pkg/eventbus` yang sudah dibangun sprint ini.
- Dashboard/visibility untuk memonitor kesehatan consumer group (lag, pending count) belum ada — akan datang bersamaan Milestone 15 (Observability), untuk sekarang cukup log dan DLQ sebagai pengaman dasar.

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah pola "dua lapis asynchronous" untuk Attachment (Task 17.4.3 — Asynq tetap ada, Outbox ditambah terpisah) sudah cukup jelas rasionalnya, atau Anda ingin penyederhanaan lebih lanjut (mis. Outbox attachment ditunda saja sampai benar-benar ada consumer yang membutuhkannya — YAGNI lebih ketat)?
2. Task 17.5.2 (`MemberEventConsumer` minimal) — dikerjakan sesuai scope minimal ini, atau Anda ingin sekalian membangun fitur notifikasi "member baru bergabung" secara penuh di sprint ini?
3. Dengan Sprint 12 selesai, proyek akan resmi berstatus **Phase B — Event-Driven Modular Monolith**, menandai **"Solid Checkpoint"** tercapai (Development Roadmap §3). Lanjut menyiapkan **Sprint 13** (Extract First Service — awal Phase C/Hybrid Architecture), atau berhenti dulu di checkpoint solid ini?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 12: 1 Epic, 7 Feature, 12 task. Meresolusi TODO Sprint 8, mengimplementasikan Outbox Pattern penuh untuk 3 domain event kunci, menandai transisi resmi Phase A → Phase B |
