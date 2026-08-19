# Detailed Task Checklist — Sprint 8
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 8: Notification)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `06-srs.md` (§2.6 FR-NOTIF, §5 Brevo), `07-hld.md` (§2.8), `08-lld.md` (§1.3), `09-database-design.md` (§2.6), `11-security-design.md` (§8 Secrets Management)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen, Prasyarat, dan Catatan Penomoran Ulang

**Catatan amandemen penomoran sprint**: Garis besar awal di `14-sprint-planning.md` §3 menyebut "Sprint 7: Notification" sebagai sprint pertama Release 3. Karena Release 2 (semula diperkirakan 3 sprint) ternyata membutuhkan **4 sprint** setelah didetailkan (Sprint 4-7 — WebSocket Infra, Messaging Core, Messaging Advanced+DM, Presence), seluruh penomoran Release 3 bergeser: **Notification kini Sprint 8** (bukan 7), Voice menjadi **Sprint 9**, Video menjadi **Sprint 10**. Ini adalah konsekuensi wajar dari Rolling Wave Planning — estimasi garis besar direvisi begitu detail nyata tersedia, bukan dipaksakan mengikuti angka awal.

**Prasyarat**: Release 2 selesai (Sprint 4-7) — khususnya `ConnectionRegistry.BroadcastToUsers` (dibuat Sprint 7, Task 11.3.1) yang akan dipakai untuk push notifikasi realtime, dan `mentions` table (dimigrasikan Sprint 4) yang menjadi sumber trigger notifikasi mention.

**Sprint Goal**: User menerima notifikasi realtime (WS) saat di-mention/reply/menerima DM; user yang offline > 5 menit menerima ringkasan email via Brevo; user dapat mengatur preferensi mute per channel/workspace.

**Catatan arsitektur penting**: Sesuai HLD §1.1 (Phase A masih Modular Monolith), Outbox Pattern **belum dibangun** hingga Release 4 (Milestone 12). Karena itu, trigger notifikasi di sprint ini memakai **in-process call langsung** dari `MessageService` ke `NotificationDispatchService` — **BUKAN** lewat event asynchronous sungguhan. Ini bukan penyimpangan, melainkan implementasi yang **sengaja disiapkan agar mudah dimigrasikan** ke Outbox/Redis Streams nanti: `NotificationDispatchService` didesain **tidak pernah query balik ke domain lain** (HLD §2.8, menerima payload lengkap) — persis kontrak yang sama yang akan dipakai saat notifikasi benar-benar menjadi event consumer di Release 4. Migrasi nanti murni mengganti *pemanggil* (in-process call → event consumer), bukan mendesain ulang `NotificationDispatchService` itu sendiri.

---

## EPIC 13: Notification

### Feature 13.1: Migrasi Database

#### Task 13.1.1: Migrasi Tabel `notification_preferences`, `notification_deliveries`

- **Deskripsi**: DDL sesuai Database Design §2.6.
- **Acceptance Criteria**: `notification_preferences` PK komposit `(user_id, scope_type, scope_id)`; `notification_deliveries` mencatat status pengiriman per channel (websocket/email).
- **Definition of Done**: `migrate up` sukses, index `idx_notif_deliveries_recipient` terverifikasi.
- **Dependency**: Release 2 selesai
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `notification_preferences` (up & down)
- [ ] Tulis migrasi `notification_deliveries` (up & down)
- [ ] Verifikasi index & constraint

#### Task 13.1.2: sqlc Setup — Domain Notification

- **Deskripsi**: Query dasar CRUD preference & delivery record.
- **Acceptance Criteria**: `sqlc generate` sukses.
- **Definition of Done**: Kode ter-generate dapat dipanggil dari test sederhana.
- **Dependency**: Task 13.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Query `UpsertNotificationPreference`, `GetPreference(userID, scopeType, scopeID)`
- [ ] Query `CreateNotificationDelivery`, `UpdateDeliveryStatus`, `CountRecentEmailDeliveries` (untuk rate limit batching, Task 13.7.1)
- [ ] `sqlc generate`, verifikasi

---

### Feature 13.2: NotificationDispatchService (Domain Inti)

#### Task 13.2.1: Domain & Interface — `NotificationEvent`, `NotificationDispatcher`

- **Deskripsi**: Implementasi persis LLD §1.3 — struct `NotificationEvent` membawa payload lengkap, interface `NotificationDispatcher`.
- **Acceptance Criteria**: **RULES.md §6 wajib dipatuhi**: `NotificationDispatchService` tidak melakukan query langsung ke tabel `messages`/`members`/`channels` — seluruh data yang dibutuhkan (nama pengirim, cuplikan pesan, nama channel) **disertakan dalam payload** oleh pemanggil (`MessageService`).
- **Definition of Done**: Code review checklist eksplisit: pastikan tidak ada import `internal/message/infrastructure` atau `internal/member/infrastructure` di dalam `internal/notification/`.
- **Dependency**: Task 13.1.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `internal/notification/domain/notification.go` (struct `NotificationEvent` — `RecipientID`, `Type`, `Payload json.RawMessage`, `ChannelID`, `WorkspaceID`)
- [ ] Implementasi interface `NotificationDispatcher` (`Dispatch(ctx, event) error`)
- [ ] Verifikasi manual: tidak ada import silang ke domain Message/Member (RULES.md §1)

#### Task 13.2.2: Implementasi `NotificationDispatchService.Dispatch`

- **Deskripsi**: Orkestrasi: cek preferensi (mute?) → cek status presence penerima (online/offline) → route ke WS dan/atau email sesuai kondisi.
- **Acceptance Criteria**: Bila preferensi `none` (mute) untuk scope terkait → **tidak** ada delivery sama sekali (WS maupun email); bila `mentions_only` → hanya proses `NotificationEvent.Type == mention`, tipe lain diabaikan.
- **Definition of Done**: Unit test (dengan mock `PresenceChecker` interface — bukan panggilan langsung ke domain Presence, tetap lewat port): 3 skenario preferensi (all/mentions_only/none) menghasilkan perilaku dispatch yang benar.
- **Dependency**: Task 13.2.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `internal/notification/application/dispatch_service.go`
- [ ] Definisikan port `PresenceChecker` interface (diimplementasikan adapter tipis ke `PresenceService` domain Presence — **satu-satunya** dependency domain lain yang diizinkan, dan itupun lewat interface, bukan akses langsung)
- [ ] Logic: cek preference → cek presence → route WS (selalu, bila tidak mute) dan/atau email (bila offline > 5 menit — FR-NOTIF-02)
- [ ] Unit test: 3 skenario preferensi

---

### Feature 13.3: Integrasi Brevo (Email)

#### Task 13.3.1: Brevo Email Client Wrapper

- **Deskripsi**: Implementasi generic di `pkg/email` (Playbook §3.1 — generic, tidak tahu domain notifikasi).
- **Acceptance Criteria**: `SendEmail(ctx, to, subject, htmlBody) error` memanggil Brevo Transactional Email API; API key dari `NEXUS_API_BREVO_API_KEY` (RULES.md §3 — tidak hardcode).
- **Definition of Done**: Unit test dengan mock HTTP client (bukan panggilan sungguhan ke Brevo saat test); integration test manual (sekali, terhadap Brevo sandbox/API key nyata) untuk verifikasi format request benar.
- **Dependency**: Task 11-Security Design §8 (env var Brevo sudah dicatat)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `pkg/email/brevo_client.go` — HTTP client ke Brevo Transactional Email API
- [ ] Konfigurasi `NEXUS_API_BREVO_API_KEY` via Viper
- [ ] Unit test dengan mock `http.RoundTripper`
- [ ] Test manual sekali (didokumentasikan hasilnya di PR, bukan bagian dari CI otomatis — menghindari dependency test ke layanan eksternal nyata)

#### Task 13.3.2: Task Handler Asynq — Kirim Email Notifikasi

- **Deskripsi**: Sesuai FR-NOTIF-02 — dieksekusi asynchronous via Asynq (reuse worker infrastructure Sprint 6, Task 10.4.1), retry maksimal 3x exponential backoff.
- **Acceptance Criteria**: Job masuk queue `default` (bukan `critical` — email tidak seurgent thumbnail preview); kegagalan setelah 3x retry → `notification_deliveries.status = failed`, dicatat di log Error.
- **Definition of Done**: Test: simulasikan Brevo API gagal 2x lalu sukses di percobaan ke-3 → email akhirnya terkirim, `status = sent`; simulasikan gagal terus-menerus → setelah 3x, `status = failed`.
- **Dependency**: Task 13.3.1, Task 10.4.1 (Sprint 6 — worker infra)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi task handler `notification:email:send` di `cmd/worker`
- [ ] Wire retry policy (reuse `RetryDelayFunc` dari Task 10.4.1)
- [ ] Update `notification_deliveries.status` sesuai hasil
- [ ] Test: retry sukses di percobaan ke-3, retry gagal total

---

### Feature 13.4: Wiring Trigger Notifikasi ke Message Domain

#### Task 13.4.1: Trigger Notifikasi saat Mention (In-Process Call)

- **Deskripsi**: Perluas `MessageService.Send` (Sprint 4/5) — setelah pesan tersimpan & broadcast WS pesan terkirim (Task 7.2.3 Sprint 4), panggil `NotificationDispatcher.Dispatch` untuk setiap `mentioned_user_id` (dari tabel `mentions`, Sprint 5).
- **Acceptance Criteria**: Panggilan ini **setelah** response ke pengirim disiapkan secara logis namun **sebelum** actual response dikirim jika ingin tetap synchronous sederhana di Phase A (dicatat eksplisit: pada Phase A, panggilan ini masih menambah sedikit latensi ke response utama — ini adalah **trade-off yang diterima sementara**, akan hilang begitu Outbox Pattern datang di Release 4 dan panggilan ini digantikan event asynchronous sungguhan).
- **Definition of Done**: Integration test: kirim pesan dengan mention → `notification_deliveries` record baru dibuat untuk user yang di-mention; broadcast WS `notification.new` diterima user tersebut.
- **Dependency**: Task 13.2.2, Task 8.3.1 (Sprint 5 — mentions)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Perluas `MessageService.Send` — setelah insert message & mentions, panggil `NotificationDispatcher.Dispatch` per mentioned user
- [ ] Payload `NotificationEvent` membawa: nama pengirim, cuplikan pesan (maks 100 karakter), nama channel — **diambil dari data yang SUDAH ada di scope `MessageService`** (tidak ada query tambahan di sisi Notification)
- [ ] **Tandai jelas dengan komentar kode**: `// TODO(release-4): ganti in-process call ini dengan Outbox event publish saat Milestone 12`
- [ ] Test: mention → notification delivery + WS event

#### Task 13.4.2: Trigger Notifikasi saat Reply & DM

- **Deskripsi**: Perluas trigger yang sama untuk reply ke pesan sendiri (FR-NOTIF-01) dan pesan baru di channel `dm`.
- **Acceptance Criteria**: Reply ke pesan User A oleh User B → User A menerima notifikasi (bila bukan reply ke pesan sendiri); pesan baru di DM → seluruh partisipan lain (selain pengirim) menerima notifikasi.
- **Definition of Done**: Test: reply → notifikasi ke penulis asli; DM baru → notifikasi ke partisipan lain (bukan ke pengirim sendiri).
- **Dependency**: Task 13.4.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tambah kondisi trigger: `reply_to_id` diisi DAN `reply_to.author_id != sender_id` → dispatch ke `reply_to.author_id`
- [ ] Tambah kondisi trigger: `channel.type == dm` → dispatch ke seluruh partisipan selain pengirim (reuse `channel_members`, Sprint 5)
- [ ] Test: kedua skenario

---

### Feature 13.5: Notification Preferences

#### Task 13.5.1: Handler — `GET/PUT /api/v1/notifications/preferences`

- **Deskripsi**: FR-NOTIF-03, sesuai API Specification §8.
- **Acceptance Criteria**: Preferensi per channel meng-override preferensi per workspace (bila keduanya diatur, channel yang menang — logic resolusi sederhana 2 tingkat, bukan seketat Permission Resolver).
- **Definition of Done**: Test: set preferensi workspace `none`, override channel tertentu jadi `all` → notifikasi di channel tersebut tetap terkirim meski workspace di-mute.
- **Dependency**: Task 13.1.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `GET /notifications/preferences` — list seluruh preferensi user
- [ ] Handler `PUT /notifications/preferences` — upsert satu scope
- [ ] Implementasi resolusi 2 tingkat (channel override workspace) di `NotificationDispatchService.Dispatch` (Task 13.2.2, perluasan)
- [ ] Test: override channel menang atas workspace

---

### Feature 13.6: Rate Limiting & Batching Email

#### Task 13.6.1: Batching Email — Maksimal 1 Ringkasan per 10 Menit

- **Deskripsi**: FR-NOTIF-04 — mencegah spam email saat user di-mention berkali-kali dalam waktu singkat.
- **Acceptance Criteria**: Bila user menerima > 1 trigger notifikasi email dalam window 10 menit, **hanya email pertama** yang benar-benar dikirim segera; trigger berikutnya dalam window yang sama **diakumulasikan** dan dikirim sebagai satu email ringkasan di akhir window (bukan langsung ditolak seperti rate limiting API biasa — perilaku berbeda dari `pkg/ratelimit` Sprint 2 yang menolak, di sini yang terjadi adalah **batching/deferral**).
- **Definition of Done**: Test: trigger 5x mention dalam 2 menit untuk user sama → hanya 1-2 email benar-benar terkirim (pertama + ringkasan akhir window), bukan 5 email terpisah.
- **Dependency**: Task 13.3.2, Task 13.1.2 (`CountRecentEmailDeliveries`)
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 3.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi buffer akumulasi per user di Redis (list/hash sementara dengan TTL 10 menit) — **bukan** memakai `pkg/ratelimit` Sprint 2 (tujuannya beda: batching bukan reject)
- [ ] Scheduled Asynq task (reuse pola periodic task Sprint 6, Task 10.7.1) — setiap akhir window, kirim email ringkasan berisi akumulasi trigger yang tertunda, kosongkan buffer
- [ ] Logic: trigger pertama dalam window → kirim langsung + mulai window baru; trigger berikutnya dalam window sama → akumulasi, tidak kirim langsung
- [ ] Test: skenario 5x trigger dalam window, verifikasi jumlah email actual terkirim sesuai ekspektasi

---

### Feature 13.7: Broadcast Realtime Notification

#### Task 13.7.1: Broadcast `notification.new` via WebSocket

- **Deskripsi**: Sesuai API Specification §10.
- **Acceptance Criteria**: Reuse `ConnectionRegistry.BroadcastToUsers` (Sprint 7, Task 11.3.1) — broadcast hanya ke `RecipientID` spesifik, bukan ke channel/workspace.
- **Definition of Done**: Test: dispatch notifikasi → user penerima (bila punya koneksi WS aktif) menerima event `notification.new` dalam waktu singkat.
- **Dependency**: Task 13.2.2, Task 11.3.1 (Sprint 7)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Wire `NotificationDispatchService` → `ConnectionRegistry.BroadcastToUsers([recipientID], event)`
- [ ] Test: broadcast diterima user dengan koneksi aktif; user tanpa koneksi aktif tidak error (silent, WS bukan satu-satunya channel)

---

### Feature 13.8: Integration Test End-to-End Sprint 8

#### Task 13.8.1: Skenario Penuh — Notification

- **Deskripsi**: Verifikasi Sprint Goal.
- **Acceptance Criteria**: Alur: (a) User A online, User B mention User A → User A terima WS notification realtime, TIDAK ada email (karena online); (b) User A offline (simulasi > 5 menit) → User B mention lagi → User A terima email via Brevo (mock/test double untuk CI, bukan API sungguhan); (c) User A set preferensi channel tertentu jadi `none` → mention di channel itu tidak menghasilkan notifikasi apapun; (d) 5x mention beruntun dalam window singkat → batching bekerja (Task 13.6.1).
- **Definition of Done**: Test hijau konsisten 3x run berturut.
- **Dependency**: Seluruh task Epic 13
- **Estimasi Kesulitan**: Sedang-Tinggi
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Setup test double untuk Brevo client (interface `EmailSender` di-mock, bukan panggilan API sungguhan di CI)
- [ ] Skenario (a)-(d) sebagai test suite terstruktur
- [ ] Jalankan 3x berturut, pastikan tidak flaky
- [ ] Update `docs/AGENTS.md` §7 — Sprint 8 selesai

---

## Ringkasan Keputusan

1. Sprint 8 mencakup **1 Epic besar, 8 Feature, 12 task**, menuntaskan seluruh scope Notification (FR-NOTIF-01 s.d. 04).
2. **Penomoran sprint direvisi** dari garis besar awal (Sprint Planning) — Notification bergeser dari "Sprint 7" menjadi **Sprint 8**, konsekuensi wajar Release 2 yang ternyata butuh 4 sprint, bukan 3. Ini dicatat eksplisit sebagai contoh nyata Rolling Wave Planning bekerja sesuai desain.
3. Trigger notifikasi (Task 13.4.1/13.4.2) memakai **in-process call sementara** (bukan event asynchronous sungguhan) karena Outbox Pattern belum dibangun hingga Release 4 — namun `NotificationDispatchService` didesain dengan kontrak yang **identik** dengan yang akan dipakai saat menjadi event consumer nanti, meminimalkan rework di Milestone 12.
4. Batching email (Task 13.6.1) adalah task **paling kompleks** di sprint ini (ditandai Tinggi) — logic deferral/akumulasi berbeda signifikan dari rate limiting reject biasa yang sudah dibangun Sprint 2.

## Trade-off yang Diterima

- Trigger notifikasi in-process (Task 13.4.1) menambah sedikit latensi ke response `POST /channels/{id}/messages` di Phase A — diterima sebagai trade-off sementara, akan hilang otomatis begitu Release 4 (Event-Driven Migration) datang dan panggilan ini digantikan publish event asynchronous.
- Test Brevo (Task 13.3.1) memakai mock untuk CI, dengan satu kali test manual terhadap API sungguhan yang didokumentasikan manual (bukan otomatis) — trade-off untuk menghindari CI yang bergantung pada layanan eksternal (flaky, butuh API key di CI secret).

## Risiko Arsitektur

- **TODO eksplisit** di Task 13.4.1 (`// TODO(release-4): ganti in-process call...`) adalah penanda penting yang **wajib** ditindaklanjuti nyata saat Milestone 12 tiba — risiko: bila terlupa, sistem tetap berfungsi (tidak error) namun kehilangan manfaat decoupling yang menjadi tujuan Event-Driven Architecture. Catat di Task Checklist Sprint 10-11 (Release 4, belum didetailkan) sebagai item wajib diperiksa.
- Task 13.6.1 (Batching) memperkenalkan state tambahan di Redis (buffer akumulasi) yang perlu dibersihkan dengan benar — kegagalan scheduled task bisa menyebabkan email ringkasan tidak pernah terkirim (silent failure); pastikan ada log Error yang jelas bila scheduled task gagal, agar terdeteksi lewat observability (walau Milestone 15 belum tiba, log dasar tetap wajib sesuai Playbook §15).

## Technical Debt yang Sengaja Diterima

- Trigger notifikasi in-process (bukan event) adalah **debt yang disengaja dan terdokumentasi**, dengan rencana pelunasan eksplisit di Milestone 12 (Release 4) — ini adalah instance konkret dari kebijakan "technical debt harus dicatat eksplisit" (Playbook §8 Definition of Quality).

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah pendekatan **in-process call sementara** untuk trigger notifikasi (dengan TODO eksplisit untuk migrasi Release 4) dapat diterima, dibanding menunda seluruh fitur Notification sampai Outbox Pattern selesai dibangun?
2. Apakah desain batching email (Task 13.6.1 — kirim pertama langsung, sisanya diakumulasi ke ringkasan akhir window) sudah sesuai ekspektasi UX yang Anda inginkan?
3. Lanjut ke **Sprint 9** (Voice, LiveKit integration), atau berhenti dulu?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 8 (penomoran direvisi dari "Sprint 7" di garis besar awal): 1 Epic, 8 Feature, 12 task, menuntaskan Notification dengan trigger in-process sementara yang didesain siap migrasi ke event-driven di Release 4 |
