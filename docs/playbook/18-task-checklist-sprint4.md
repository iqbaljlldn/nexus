# Detailed Task Checklist — Sprint 4
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 4: WebSocket Infrastructure + Messaging Core)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `14-sprint-planning.md` (Release 2 overview), `06-srs.md` (§2.4 FR-MSG), `07-hld.md` (§2.6, §6.1), `08-lld.md` (§1.1, §2.2, §2.9), `09-database-design.md` (§2.4), `10-api-specification.md` (§4, §10), `11-security-design.md` (§7)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen & Prasyarat

Sesuai **Rolling Wave Planning**, dokumen ini adalah detail pertama untuk **Release 2** — sebelumnya hanya digambarkan garis besar di `14-sprint-planning.md` §3 ("Sprint 4-5: WebSocket infrastructure + Messaging inti + DM"). Dokumen ini **memecah** garis besar tersebut: **Sprint 4 fokus pada WebSocket Infrastructure + Messaging Core (kirim/list/edit/delete)**; reply/thread/mention/reaction/DM digeser ke **Sprint 5** (didetailkan terpisah setelah Sprint 4 selesai, konsisten dengan prinsip rolling wave itu sendiri).

**Prasyarat**: Release 1 selesai (Sprint 1-3) — khususnya Permission Resolver (Task 3.5.1) dan `ChannelAuthorizationService` yang akan dipakai untuk otorisasi kirim pesan.

**Sprint Goal**: User dapat mengirim, melihat riwayat (cursor pagination), mengedit (dengan optimistic locking), dan menghapus (soft delete) pesan di channel teks — pesan baru muncul **realtime** di client lain via WebSocket tanpa refresh.

---

## EPIC 6: WebSocket Infrastructure

### Feature 6.1: WebSocket Connection Manager

#### Task 6.1.1: WebSocket Upgrade Handler + Autentikasi Handshake

- **Deskripsi**: Endpoint `wss://.../ws?token=<access_token>` sesuai API Specification §10 dan Security Design §7 (token di query param, browser WS API tidak mendukung custom header saat handshake).
- **Acceptance Criteria**: Koneksi dengan token invalid/expired ditolak dengan custom close code `4001`; koneksi valid berhasil upgrade dan tersimpan di registry.
- **Definition of Done**: Test: koneksi tanpa token → ditolak; token valid → upgrade sukses, `user_id` tersimpan di context koneksi.
- **Dependency**: Task 2.5.1 (Sprint 2 — JWT verify utility)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `internal/platform/websocket/upgrader.go` (Gorilla WebSocket)
- [ ] Validasi token dari query param sebelum `upgrader.Upgrade()`
- [ ] Validasi header `Origin` terhadap domain aplikasi yang diizinkan (Security Design §7)
- [ ] Close code `4001` untuk auth gagal
- [ ] Test: token invalid, token expired, Origin tidak diizinkan, koneksi valid

#### Task 6.1.2: ConnectionRegistry — Single Writer per Connection

- **Deskripsi**: Implementasi persis sesuai LLD §2.9 — read-loop dan write-loop terpisah per koneksi, broadcast lewat channel buffered, slow-consumer protection.
- **Acceptance Criteria**: **RULES.md §4 wajib dipatuhi**: tidak ada penulisan langsung ke koneksi WS dari goroutine broadcast; single-writer-per-connection.
- **Definition of Done**: Test konkurensi (`go test -race`): broadcast ke banyak koneksi simultan tidak menghasilkan data race; koneksi dengan buffer penuh di-drop tanpa memblokir broadcast ke koneksi lain.
- **Dependency**: Task 6.1.1
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 4 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `Connection` struct: `sendCh chan []byte` (buffered, ukuran awal 64 — akan dituning Milestone 11), goroutine writer terpisah membaca dari `sendCh`
- [ ] Goroutine reader terpisah menangani pesan masuk (typing, presence, heartbeat)
- [ ] Implementasi `ConnectionRegistry` (`map[uuid.UUID]map[*Connection]struct{}`, `sync.RWMutex`)
- [ ] Method `Broadcast(channelID, msg)` — copy snapshot sebelum lepas lock, non-blocking send dengan `select`/`default` (LLD §2.9 persis)
- [ ] Test race detector: broadcast simultan dari 2 goroutine berbeda ke koneksi yang sama
- [ ] Test slow consumer: koneksi yang tidak membaca `sendCh` di-drop tanpa memblokir broadcast lain

#### Task 6.1.3: Ping/Pong Heartbeat & Graceful Close

- **Deskripsi**: Deteksi koneksi mati (client crash tanpa close frame proper).
- **Acceptance Criteria**: Server mengirim ping setiap interval tetap; koneksi tanpa pong dalam timeout ditutup otomatis dan dihapus dari registry.
- **Definition of Done**: Test: simulasikan koneksi tidak merespons ping → koneksi otomatis dibersihkan dari `ConnectionRegistry` dalam waktu yang diharapkan.
- **Dependency**: Task 6.1.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Set `PongHandler`, kirim ping berkala dari writer goroutine
- [ ] Timeout: tutup koneksi & unregister dari `ConnectionRegistry` bila pong tidak diterima
- [ ] Integrasi dengan graceful shutdown aplikasi (LLD §3 — `wsHub.CloseAllGracefully`)
- [ ] Test: koneksi tanpa pong dibersihkan otomatis

#### Task 6.1.4: Payload Size Limit & Rate Limit Koneksi

- **Deskripsi**: Mitigasi DoS (Security Design §7) — batas ukuran frame masuk dan jumlah koneksi aktif per user.
- **Acceptance Criteria**: Frame > 8KB ditolak; user dengan > 5 koneksi aktif bersamaan, koneksi tertua di-drop atau koneksi baru ditolak (keputusan konkret: koneksi baru ditolak dengan close code khusus, lebih sederhana untuk diimplementasikan).
- **Definition of Done**: Test: kirim frame besar → ditolak; buka koneksi ke-6 dari user sama → ditolak.
- **Dependency**: Task 6.1.2
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Should

**Subtask & Checklist**:
- [ ] `conn.SetReadLimit(8192)`
- [ ] Hitung koneksi aktif per `user_id` di `ConnectionRegistry`, tolak koneksi ke-6+
- [ ] Test: frame besar, koneksi berlebih

---

### Feature 6.2: WebSocket Protocol — Event Envelope

#### Task 6.2.1: Definisi Event Envelope & Router Event Masuk

- **Deskripsi**: Format pesan WS standar `{ "event": "typing.start", "payload": {...} }` sesuai API Specification §10.
- **Acceptance Criteria**: Client→Server events (`typing.start`, `presence.set_status`, `heartbeat`) di-parse dan di-route ke handler masing-masing.
- **Definition of Done**: Unit test: pesan dengan `event` tidak dikenal diabaikan dengan log warning (bukan crash koneksi).
- **Dependency**: Task 6.1.2
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Struct `WSEnvelope{ Event string, Payload json.RawMessage }`
- [ ] Router sederhana (`map[string]HandlerFunc`) di reader goroutine
- [ ] Handler stub untuk `typing.start`, `presence.set_status` (implementasi penuh presence di Sprint mendatang — hanya routing dasar di sini)
- [ ] Handler `heartbeat` (no-op selain refresh, presence TTL menyusul)
- [ ] Test: event tidak dikenal tidak crash koneksi

---

## EPIC 7: Messaging Core

### Feature 7.1: Migrasi Database — Message & Reaction

#### Task 7.1.1: Migrasi Tabel `messages` (Partitioned) + Trigger Search Vector

- **Deskripsi**: DDL sesuai Database Design §2.4 — **termasuk `PARTITION BY RANGE (created_at)`** dan trigger `search_vector` (§4 Database Design), meski full-text search belum jadi fitur aktif hingga Milestone terkait Search (kolom & trigger disiapkan sekarang agar tidak perlu expand-contract nanti).
- **Acceptance Criteria**: Partisi bulan berjalan dibuat (mis. `messages_y2026m08`); index composite `idx_messages_channel_created_id` aktif; trigger mengisi `search_vector` otomatis saat insert.
- **Definition of Done**: `migrate up` sukses; insert pesan → `search_vector` terisi otomatis (verifikasi via query manual).
- **Dependency**: Task 3.6.1 (Sprint 3 — tabel `channels`)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `messages` (partitioned table, DDL persis Database Design §2.4)
- [ ] Buat minimal 2 partisi (bulan berjalan + bulan berikutnya)
- [ ] Buat index `idx_messages_channel_created_id`, `idx_messages_thread_root`, `idx_messages_search_vector`
- [ ] Buat trigger `messages_search_vector_update` (Database Design §4)
- [ ] Verifikasi insert manual: `search_vector` terisi

#### Task 7.1.2: Migrasi Tabel `reactions`, `mentions`, `read_receipts`

- **Deskripsi**: DDL sesuai Database Design §2.4 — mentions/reactions disiapkan sekarang meski endpoint-nya baru Sprint 5, agar skema `messages` core (Sprint 4) tidak perlu direvisi lagi saat Sprint 5.
- **Acceptance Criteria**: Constraint `PRIMARY KEY (message_id, user_id, emoji)` pada reactions; CHECK XOR pada mentions.
- **Definition of Done**: `migrate up` sukses, constraint teruji.
- **Dependency**: Task 7.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `reactions`
- [ ] Tulis migrasi `mentions` (CHECK XOR user/role)
- [ ] Tulis migrasi `read_receipts`
- [ ] Verifikasi constraint via test insert invalid

#### Task 7.1.3: sqlc Setup — Domain Message

- **Deskripsi**: Query dasar untuk Sprint 4 (kirim, list cursor-based, update dengan version check, soft delete).
- **Acceptance Criteria**: Query `ListMessagesByChannel` persis mengikuti pola keyset pagination LLD §2.2 (composite `(created_at, id)`).
- **Definition of Done**: `sqlc generate` sukses.
- **Dependency**: Task 7.1.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Query `CreateMessage`
- [ ] Query `ListMessagesByChannel` (cursor `(created_at, id)`, `LIMIT`, filter `deleted_at IS NULL`)
- [ ] Query `FindMessageByID`
- [ ] Query `UpdateMessageWithVersion` — `WHERE id=$1 AND version=$2`, cek `RowsAffected()` untuk deteksi conflict (LLD §1.1)
- [ ] Query `SoftDeleteMessage`
- [ ] `sqlc generate`, verifikasi

---

### Feature 7.2: Kirim Pesan (Send Message)

#### Task 7.2.1: Domain & Service — MessageService.Send

- **Deskripsi**: Sesuai LLD §1.1, FR-MSG-01.
- **Acceptance Criteria**: Validasi `content` maksimal 4000 karakter, minimal 1 karakter ATAU ada attachment (attachment penuh baru Sprint 6 — untuk Sprint 4, validasi minimal 1 karakter cukup, item attachment_ids diterima tapi divalidasi longgar/di-skip dengan TODO eksplisit).
- **Definition of Done**: Unit test: pesan valid tersimpan, pesan > 4000 karakter ditolak, pesan kosong ditolak.
- **Dependency**: Task 7.1.3, Feature 3.5 (Permission Resolver, Sprint 3)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `internal/message/domain/message.go` (struct `Message`, sesuai LLD §1.1)
- [ ] Implementasi `MessageService.Send(ctx, cmd)` — cek otorisasi via `ChannelAuthorizationService.CanWrite` (Task 1.2 LLD, dibuat Sprint 3 untuk konteks workspace; untuk konteks DM di-stub Sprint 4, penuh Sprint 5)
- [ ] Validasi panjang & non-kosong
- [ ] Unit test: sukses, terlalu panjang, kosong, tanpa permission

#### Task 7.2.2: Rate Limiting Kirim Pesan

- **Deskripsi**: SRS §3.5 — 10 pesan/10 detik per user per channel, memakai `pkg/ratelimit` yang sudah dibuat Sprint 2 (Task 2.8.1) — **buktikan reusability shared package** sesuai rationale Sprint 2.
- **Acceptance Criteria**: Percobaan kirim ke-11 dalam 10 detik ditolak `429`.
- **Definition of Done**: Integration test: kirim 10 pesan cepat berturut → pesan ke-11 ditolak.
- **Dependency**: Task 2.8.1 (Sprint 2), Task 7.2.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Terapkan `pkg/ratelimit.Allow` dengan key `msgratelimit:{channel_id}:{user_id}`, window 10s, limit 10
- [ ] Test: 10 pesan sukses, pesan ke-11 → 429

#### Task 7.2.3: Handler + Broadcast Synchronous — `POST /channels/{id}/messages`

- **Deskripsi**: Implementasi persis alur HLD §6.1 sequence diagram — response ke pengirim TIDAK menunggu Outbox Relay; broadcast WS terjadi **synchronous in-process** sebelum response dikirim.
- **Acceptance Criteria**: Setelah `INSERT message`, broadcast `message.created` ke `ConnectionRegistry` (Task 6.1.2) terjadi sebelum `201` dikembalikan; test dengan 2 client WS simulasi memverifikasi client kedua menerima event tanpa polling.
- **Definition of Done**: Test end-to-end: Client A kirim pesan via REST → Client B (koneksi WS aktif di channel sama) menerima event `message.created` dalam waktu singkat (< 1 detik di test lokal).
- **Dependency**: Task 7.2.1, 7.2.2, Task 6.1.2
- **Estimasi Kesulitan**: Sedang-Tinggi
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `POST /channels/{id}/messages`
- [ ] Setelah insert sukses, panggil `ConnectionRegistry.Broadcast(channelID, encodedEvent)`
- [ ] Response 201 dengan objek message lengkap
- [ ] Test end-to-end: 2 koneksi WS simulasi (test client), verifikasi Client B menerima broadcast setelah Client A POST pesan

---

### Feature 7.3: List Pesan (History)

#### Task 7.3.1: Handler — `GET /channels/{id}/messages`

- **Deskripsi**: FR-MSG-10, cursor-based pagination.
- **Acceptance Criteria**: Default `limit=50`, max `100`; `meta.next_cursor` dan `meta.has_more` sesuai kontrak API Spec.
- **Definition of Done**: Test: kirim 60 pesan, list pertama mengembalikan 50 + `has_more=true`, list dengan cursor mengembalikan 10 sisanya + `has_more=false`.
- **Dependency**: Task 7.1.3
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `GET /channels/{id}/messages` — parse `cursor`, `limit` (cap 100)
- [ ] Encode/decode cursor sesuai LLD §2.2 (`base64` dari `{last_id, last_created_at}`)
- [ ] Test: pagination penuh (60 pesan, 2 halaman), permission baca channel dicek

---

### Feature 7.4: Edit Pesan (Optimistic Locking)

#### Task 7.4.1: Handler — `PATCH /messages/{id}` dengan Optimistic Locking

- **Deskripsi**: FR-MSG-07/FR-MSG-09 — hanya penulis asli, conflict terdeteksi via `version`.
- **Acceptance Criteria**: **RULES.md §9 & Database Design**: edit dengan `expected_version` tidak cocok mengembalikan `409 OPTIMISTIC_LOCK_CONFLICT`.
- **Definition of Done**: Test: edit dengan version benar → sukses, `version` bertambah, `edited_at` terisi; edit dengan version salah (simulasi 2 client bersamaan) → `409`.
- **Dependency**: Task 7.1.3, 7.2.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `PATCH /messages/{id}` — cek `author_id == user_id`
- [ ] Panggil `UpdateMessageWithVersion`, mapping `RowsAffected==0` → `ErrOptimisticLockConflict` → HTTP 409
- [ ] Broadcast `message.updated` via WS setelah sukses
- [ ] Test: edit sukses, edit oleh bukan penulis (403), version conflict (409) — simulasikan dengan 2 request paralel terhadap version yang sama

---

### Feature 7.5: Hapus Pesan (Soft Delete)

#### Task 7.5.1: Handler — `DELETE /messages/{id}`

- **Deskripsi**: FR-MSG-08 — penulis asli ATAU permission `MANAGE_MESSAGES`.
- **Acceptance Criteria**: Soft delete (`deleted_at` terisi), pesan tidak lagi muncul di `ListMessagesByChannel` (filter `deleted_at IS NULL` sudah ada di query Task 7.1.3).
- **Definition of Done**: Test: delete oleh penulis sukses; delete oleh Moderator (`MANAGE_MESSAGES`) terhadap pesan orang lain sukses; delete oleh member biasa terhadap pesan orang lain → 403.
- **Dependency**: Task 7.1.3, Task 3.5.1 (Permission Resolver)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `DELETE /messages/{id}` — cek kepemilikan ATAU permission `MANAGE_MESSAGES`
- [ ] Broadcast `message.deleted` via WS
- [ ] Test: 3 skenario otorisasi di atas

---

### Feature 7.6: Integration Test End-to-End Sprint 4

#### Task 7.6.1: Skenario Penuh — Realtime Messaging Core

- **Deskripsi**: Verifikasi Sprint Goal secara menyeluruh.
- **Acceptance Criteria**: Alur: 2 client WS terhubung ke channel sama → Client A kirim pesan → Client B menerima realtime → Client A edit pesan (version benar) → Client B menerima update realtime → Client A hapus pesan → Client B menerima event delete → `GET messages` tidak lagi menampilkan pesan tersebut.
- **Definition of Done**: Test hijau konsisten (3x run berturut, tidak flaky — perhatian khusus karena melibatkan WebSocket async, rawan flaky bila tidak ada sinkronisasi test yang tepat).
- **Dependency**: Seluruh task Epic 6, 7
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 3.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Setup test harness: 2 WS test client + 1 HTTP test client
- [ ] Skenario kirim → verifikasi broadcast diterima (dengan timeout wajar, bukan `sleep` sembarangan — pakai channel/context timeout eksplisit)
- [ ] Skenario edit → verifikasi broadcast update
- [ ] Skenario delete → verifikasi broadcast delete + tidak muncul di list
- [ ] Jalankan 3x berturut memastikan tidak flaky
- [ ] Update `docs/AGENTS.md` §7 — Sprint 4 selesai

---

## EPIC 8: Frontend — WebSocket Client & Messaging UI

### Feature 8.1: WebSocket Client Infrastructure

#### Task 8.1.1: `useWebSocket` Composable — Koneksi Singleton

- **Deskripsi**: Implementasi persis Frontend Architecture §5.1 — satu koneksi global per tab, reconnect exponential backoff, penanganan close code `4001` (auth gagal, tidak reconnect otomatis).
- **Acceptance Criteria**: Koneksi diinisialisasi sekali di plugin (`websocket.client.ts`), bukan per komponen; reconnect otomatis dengan backoff saat koneksi putus tidak normal.
- **Definition of Done**: Test: simulasikan disconnect paksa → reconnect otomatis dalam waktu wajar; simulasikan close code 4001 → redirect login tanpa retry.
- **Dependency**: Task 3.4.1 (Sprint 2 — auth resolved sebelum WS connect), Task 6.1.1 (Sprint 3 — access token tersedia)
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 3.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `composables/useWebSocket.ts` persis §5.1
- [ ] Plugin `plugins/websocket.client.ts` — connect sekali saat app mount (setelah auth resolved)
- [ ] Reconnect exponential backoff (bukan retry langsung tanpa jeda — mencegah thundering herd, konsisten prinsip yang sama dipakai backend)
- [ ] Test: reconnect normal, close code 4001 tidak retry

#### Task 8.1.2: Event Router — `routeIncomingEvent`

- **Deskripsi**: Implementasi persis §5.2 — menyuntik event ke TanStack Query cache & Pinia store, bukan trigger refetch.
- **Acceptance Criteria**: Untuk Sprint 4, minimal handle `message.created`, `message.updated`, `message.deleted` (event lain ditambah bertahap seiring sprint yang memperkenalkannya — dicatat sebagai checklist wajib per sprint, Frontend Architecture §5.2 §Risiko).
- **Definition of Done**: Test: publish event WS manual (test harness) → cache TanStack Query ter-update tanpa network request tambahan (verifikasi via network tab/mock — tidak ada refetch terpicu).
- **Dependency**: Task 8.1.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `routeIncomingEvent` dengan case `message.created/updated/deleted`
- [ ] Wire ke `useWebSocket.onmessage`
- [ ] Test: event masuk → cache update tanpa refetch

---

### Feature 8.2: Message List (Virtual Scroll + Cursor Pagination)

#### Task 8.2.1: `useMessages` Composable

- **Deskripsi**: Implementasi persis Frontend Architecture §3.3 — `useInfiniteQuery` + `useMutation` dengan optimistic update.
- **Acceptance Criteria**: `sendMutation` **tidak** invalidate query cache secara manual (pesan baru datang lewat WS broadcast, Task 8.1.2, bukan refetch REST) — hanya optimistic update + rollback saat error.
- **Definition of Done**: Test: kirim pesan → muncul instan (optimistic) → dikonfirmasi WS broadcast (bukan refetch); simulasikan kirim gagal → optimistic message di-rollback.
- **Dependency**: Task 8.1.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `composables/useMessages.ts` persis §3.3
- [ ] Optimistic add + rollback on error
- [ ] Test: sukses (optimistic→confirmed via WS), gagal (rollback)

#### Task 8.2.2: `MessageList.vue` — Virtual Scroll

- **Deskripsi**: Implementasi persis §6 — Vue Virtual Scroller + `useInfiniteScroll`/`useChannelMessageList`.
- **Acceptance Criteria**: Scroll ke atas memicu `fetchNextPage` (load riwayat lama); hanya pesan dalam viewport yang di-render ke DOM (verifikasi via DevTools — jumlah DOM node jauh lebih sedikit dari jumlah total pesan pada channel dengan riwayat panjang).
- **Definition of Done**: Test manual/E2E: channel dengan 200+ pesan (seed data test) — scroll ke atas memuat halaman lama, DOM node tidak meledak.
- **Dependency**: Task 8.2.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `composables/useInfiniteScroll.ts` persis §6
- [ ] Komponen `MessageList.vue` dengan `RecycleScroller` (Vue Virtual Scroller)
- [ ] Komponen `MessageItem.vue` (render satu pesan)
- [ ] Test: scroll-load riwayat lama, verifikasi DOM node count wajar

#### Task 8.2.3: `MessageComposer.vue` — Kirim Pesan

- **Deskripsi**: Input area dengan submit ke `sendMutation` (Task 8.2.1).
- **Acceptance Criteria**: Rate limit backend (10 pesan/10 detik, Task 7.2.2) direspons dengan pesan error jelas di UI (bukan silent fail) bila `429` diterima.
- **Definition of Done**: E2E test: kirim pesan → muncul di list; kirim 11x cepat → pesan ke-11 menampilkan error rate limit.
- **Dependency**: Task 8.2.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Komponen `MessageComposer.vue` (textarea + submit)
- [ ] Handle error `429 RATE_LIMIT_EXCEEDED` dengan pesan UI jelas
- [ ] E2E test: kirim normal, kirim melebihi rate limit

---

### Feature 8.3: Edit & Delete Pesan (Optimistic Locking UX)

#### Task 8.3.1: UI Edit Pesan — Penanganan Conflict

- **Deskripsi**: UI untuk `PATCH /messages/{id}` (Task 7.4.1) — **wajib** menangani `409 OPTIMISTIC_LOCK_CONFLICT` dengan UX yang jelas, bukan generic error.
- **Acceptance Criteria**: Saat conflict terjadi (pesan sudah diubah pihak lain sejak dimuat), UI menawarkan reload versi terbaru sebelum user mencoba edit ulang — **bukan** menampilkan pesan error teknis mentah ke user.
- **Definition of Done**: Test: simulasikan 2 edit bersamaan (2 tab browser) → tab kedua menerima conflict UI yang jelas, bukan crash/silent fail.
- **Dependency**: Task 8.2.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Mode edit inline di `MessageItem.vue`, kirim `expected_version` dari data yang sedang ditampilkan
- [ ] Handler khusus `409 OPTIMISTIC_LOCK_CONFLICT` — tampilkan opsi "muat ulang versi terbaru"
- [ ] Test: 2 tab edit bersamaan, verifikasi UX conflict

#### Task 8.3.2: UI Hapus Pesan (Soft Delete)

- **Deskripsi**: UI untuk `DELETE /messages/{id}` (Task 7.5.1), permission-aware (reuse `usePermission`, Task 6.3.2 Sprint 3).
- **Acceptance Criteria**: Tombol hapus hanya muncul untuk penulis asli ATAU user dengan `viewer_permissions.can_manage_messages`.
- **Definition of Done**: E2E test: penulis melihat tombol hapus; member biasa tanpa permission tidak melihatnya untuk pesan orang lain.
- **Dependency**: Task 8.3.1, Task 6.3.2 (Sprint 3)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tombol hapus di `MessageItem.vue`, kondisional `usePermission`/kepemilikan
- [ ] Konfirmasi dialog sebelum hapus (UX standar, mencegah hapus tidak sengaja)
- [ ] E2E test: 2 skenario permission

---

### Feature 8.4: Integration Test End-to-End Frontend Sprint 4

#### Task 8.4.1: Skenario Penuh — Realtime Messaging via UI

- **Deskripsi**: Versi frontend dari Task 7.6.1 backend — 2 browser context.
- **Acceptance Criteria**: User A kirim pesan via UI → User B (browser context lain, channel sama) melihatnya muncul **tanpa refresh manual**; User A edit → User B melihat update realtime; User A hapus → User B melihat pesan hilang.
- **Definition of Done**: Playwright test hijau konsisten 3x run berturut.
- **Dependency**: Seluruh task Epic 8
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Skenario Playwright 2 context (2 user berbeda, channel sama)
- [ ] Jalankan 3x berturut, pastikan tidak flaky
- [ ] Update `docs/AGENTS.md` §7 — Sprint 4 frontend selesai bersamaan backend

---

## Ringkasan Keputusan

1. Garis besar Release 2 (Sprint Planning §3) dipecah: **Sprint 4 = WebSocket Infra + Messaging Core**, **Sprint 5 = Reply/Thread/Mention/Reaction/DM** — pemecahan ini sendiri adalah instance dari Rolling Wave Planning yang diterapkan berulang, bukan hanya sekali di level Release. *(Direvisi: ditambah Epic 8 — 4 Feature, 9 Task frontend, amandemen retroaktif — WebSocket client singleton, virtual-scroll message list, optimistic UI kirim/edit pesan.)*
2. Skema `messages`, `reactions`, `mentions` disiapkan **sekaligus** di Sprint 4 (Task 7.1.1-7.1.2) meski sebagian fiturnya (reaction, mention) baru datang Sprint 5 — menghindari migrasi tambahan yang tidak perlu, konsisten dengan pendekatan proaktif di Sprint 3 untuk skema `channels`.
3. Broadcast WebSocket **synchronous in-process** (bukan lewat Outbox) untuk pesan real-time — implementasi persis mengikuti keputusan HLD §3 dan §6.1 (dibedakan tegas dari notifikasi/indexing yang asynchronous, baru datang di Release 4).
4. Task 6.1.2 (ConnectionRegistry) dan Task 7.6.1 (integration test) ditandai **Estimasi Kesulitan Tinggi** — dua titik paling berisiko meleset waktu di sprint ini karena melibatkan konkurensi dan test asynchronous.

## Trade-off yang Diterima

- Task 6.1.4 (payload/connection limit) dan permission check DM di Task 7.2.1 di-stub/prioritas Should — tidak memblokir Sprint Goal inti (chat teks realtime dalam konteks Workspace), diselesaikan penuh saat DM didetailkan di Sprint 5.
- Attachment pada pesan (`attachment_ids`) diterima di request tapi validasi penuh ditunda ke Sprint 6 (Upload) — field disiapkan di kontrak API sekarang agar tidak perlu breaking change nanti.

## Risiko Arsitektur

- Test end-to-end WebSocket (Task 7.6.1) berisiko flaky bila sinkronisasi test tidak didisiplinkan (memakai `time.Sleep` sembarangan alih-alih channel/context dengan timeout) — wajib pakai pola sinkronisasi eksplisit, dicatat sebagai catatan implementasi keras di subtask.
- `ConnectionRegistry` single-instance (Task 6.1.2) akan perlu direvisi total saat Horizontal Scaling (Deployment Architecture Tahap 4) — ini bukan bug, tapi keterbatasan yang disengaja diterima sesuai evolusi arsitektur (HLD §Risiko), dicatat ulang di sini agar tidak terlupa saat Release 5 tiba.

## Technical Debt yang Sengaja Diterima

- Ukuran buffer `sendCh` (64) adalah nilai awal, akan dituning berdasarkan benchmark nyata di Milestone 11 (Release 4), bukan dianggap final.
- Presence (`typing.start`, `presence.set_status`) baru di-routing dasar (Task 6.2.1), implementasi penuh menyusul di sprint domain Presence (Release 2 lanjutan, belum didetailkan — rolling wave).

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah pemecahan Sprint 4 (WS Infra + Messaging Core) vs Sprint 5 (Reply/Thread/Mention/Reaction/DM) sudah sesuai ekspektasi Anda, atau ada preferensi pengelompokan lain?
2. Apakah estimasi kesulitan **Tinggi** untuk Task 6.1.2 dan 7.6.1 (masing-masing 4 jam dan 3.5 jam) realistis, atau perlu buffer waktu tambahan mengingat ini pengalaman pertama tim dengan konkurensi WebSocket di proyek ini?
3. Lanjut menyiapkan **Sprint 5**, atau berhenti dulu menunggu Sprint 4 dieksekusi?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 4: 2 Epic (WebSocket Infrastructure, Messaging Core), 6 Feature, 15 task, memecah garis besar Release 2 dari Sprint Planning menjadi detail penuh |
| 1.1.0 | Amandemen | Ditambahkan Epic 8: Frontend (WebSocket client, virtual-scroll message list, optimistic UI, conflict handling edit) — amandemen retroaktif |
