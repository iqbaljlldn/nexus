# Detailed Task Checklist — Sprint 3
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 3: Workspace, Permission, Channel)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `14-sprint-planning.md` (Sprint 3), `06-srs.md` (§2.2-2.3), `07-hld.md` (§2.2-2.5), `08-lld.md` (§1.2, §2.1), `09-database-design.md` (§2.2-2.3), `10-api-specification.md` (§2-3)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen & Prasyarat

Sesuai **Rolling Wave Planning** (`14-sprint-planning.md` §0), dokumen ini mendetailkan **Sprint 3 saja** — sprint terakhir Release 1 (Foundation).

**Prasyarat**: Sprint 1 & Sprint 2 selesai — khususnya Auth Middleware (Task 2.5.2) yang akan dipakai untuk memproteksi seluruh endpoint di sprint ini, dan `AuthService`/`user_id` context yang sudah tersedia di setiap request.

**Sprint Goal** (dari `14-sprint-planning.md` Sprint 3): User dapat membuat workspace, invite anggota, mengatur role/permission, dan membuat channel — skenario end-to-end (buat workspace → invite user kedua → assign role custom → buat channel dengan override permission → verifikasi user kedua **tidak** bisa akses channel privat) lolos test otomatis.

**Catatan arsitektur penting untuk sprint ini**: Domain Workspace/Member/Role/Channel adalah domain dengan kompleksitas relasi tertinggi di proyek (HLD §2.2-2.5) dan **direkomendasikan tetap sebagai inti monolith** bahkan di Phase D (RULES.md §1) — jangan mendesain seolah-olah domain ini akan dipisah, boleh saling berinteraksi lebih erat dibanding domain lain (namun tetap lewat interface antar sub-domain, bukan akses tabel silang tanpa lapisan repository).

---

## EPIC 3: Workspace & Membership

### Feature 3.1: Migrasi Database — Workspace, Category, Invite, Member

#### Task 3.1.1: Migrasi Tabel `workspaces`, `categories`, `invites`

- **Deskripsi**: DDL sesuai Database Design §2.2.
- **Acceptance Criteria**: Migrasi up/down bersih; foreign key ke `users(id)` (owner_id, created_by) berfungsi.
- **Definition of Done**: `migrate up` sukses, constraint teruji (insert dengan `owner_id` tidak valid ditolak).
- **Dependency**: Sprint 1 selesai (tabel `users` ada)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `workspaces` (up & down)
- [ ] Tulis migrasi `categories` (up & down)
- [ ] Tulis migrasi `invites` (up & down)
- [ ] Verifikasi FK constraint via test insert invalid

#### Task 3.1.2: Migrasi Tabel `members`

- **Deskripsi**: Aggregate terpisah dari workspace (HLD §2.3), didesain untuk skala 100.000 row/workspace.
- **Acceptance Criteria**: Constraint unik `(workspace_id, user_id)`; index `idx_members_workspace_id` dan `idx_members_user_id` sesuai Database Design §2.2.
- **Definition of Done**: Insert member duplikat (workspace_id+user_id sama) ditolak database.
- **Dependency**: Task 3.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `members` (up & down)
- [ ] Verifikasi unique constraint via test insert duplikat
- [ ] Verifikasi kedua index dengan `EXPLAIN` query dasar

#### Task 3.1.3: sqlc Setup — Domain Workspace & Member

- **Deskripsi**: Query dasar CRUD untuk workspace, category, invite, member.
- **Acceptance Criteria**: `sqlc generate` sukses untuk seluruh query baru.
- **Definition of Done**: Kode ter-generate dapat dipanggil dari test sederhana.
- **Dependency**: Task 3.1.1, 3.1.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Query: `CreateWorkspace`, `ListWorkspacesByUserID` (join via members, cursor-based — Playbook §17.2)
- [ ] Query: `CreateCategory`, `CreateInvite`, `FindInviteByCode`, `IncrementInviteUseCount`
- [ ] Query: `CreateMember`, `FindMemberByWorkspaceAndUser`, `ListMembersByWorkspace` (cursor-based, SRS FR-WS-08)
- [ ] `sqlc generate`, verifikasi tanpa error

---

### Feature 3.2: Workspace CRUD

#### Task 3.2.1: Domain & Service — Workspace, Auto-Owner, Auto-@everyone

- **Deskripsi**: FR-WS-01/FR-WS-02 — pembuatan workspace otomatis menghasilkan Owner (member pertama) dan role `@everyone`.
- **Acceptance Criteria**: `WorkspaceService.Create(ctx, ownerID, name) (*Workspace, error)` dalam **satu transaksi** membuat: workspace, member (owner), role `@everyone` (bitmask default), assignment role owner→`@everyone`.
- **Definition of Done**: Unit test (dengan mock repository, lalu integration test dengan DB nyata): setelah create, query member & role `@everyone` mengembalikan data yang benar.
- **Dependency**: Task 3.1.3
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `internal/workspace/domain/workspace.go`
- [ ] Implementasi `WorkspaceService.Create` dengan DB transaction (workspace + member + role + assignment)
- [ ] Unit test dengan mock; integration test dengan DB nyata
- [ ] Verifikasi rollback penuh bila salah satu langkah dalam transaksi gagal

#### Task 3.2.2: Handler — `POST /api/v1/workspaces`, `GET /api/v1/workspaces`

- **Deskripsi**: Sesuai API Specification §2.
- **Acceptance Criteria**: Create sesuai kontrak; List memakai cursor-based pagination, hanya menampilkan workspace milik user yang login (join via `members`).
- **Definition of Done**: Test HTTP: create → muncul di list; user lain tidak melihat workspace tersebut di list-nya.
- **Dependency**: Task 3.2.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `POST /workspaces` (proteksi Auth Middleware Sprint 2)
- [ ] Handler `GET /workspaces` (cursor pagination, LLD §2.2 pola)
- [ ] Test: isolasi antar user (workspace user A tidak muncul di list user B)

---

### Feature 3.3: Invite & Idempotent Redeem

#### Task 3.3.1: Service — InviteService.Create & Redeem

- **Deskripsi**: FR-WS-06 — invite dengan `max_uses`/`expires_at` opsional, redeem idempotent.
- **Acceptance Criteria**: Redeem invite yang sudah dipakai user yang sama sebelumnya **tidak** membuat member duplikat, cukup mengembalikan membership yang sudah ada (bukan error).
- **Definition of Done**: Integration test: redeem 2x oleh user sama dengan `Idempotency-Key` sama → hasil identik, tidak ada baris `members` duplikat; redeem invite kedaluwarsa → `422 BUSINESS_RULE_VIOLATION`.
- **Dependency**: Task 3.1.3
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `InviteService.Create` (max_uses, expires_at nullable)
- [ ] Implementasi `InviteService.Redeem` — cek existing membership dulu sebelum insert (idempotent by design, bukan hanya mengandalkan `Idempotency-Key` header)
- [ ] Validasi `expires_at`/`max_uses` sebelum redeem → error `BUSINESS_RULE_VIOLATION` sesuai kondisi
- [ ] Integration test: redeem ganda, redeem kedaluwarsa, redeem max_uses tercapai

#### Task 3.3.2: Handler — `POST /workspaces/{id}/invites`, `POST /invites/{code}/redeem`

- **Deskripsi**: Sesuai API Specification §2, dengan header `Idempotency-Key` wajib untuk redeem (Playbook §17.4).
- **Acceptance Criteria**: Endpoint create invite memerlukan permission `MANAGE_INVITES` (bergantung Permission Resolver — Feature 3.5, dikerjakan paralel/setelahnya; untuk task ini, cek permission dapat memakai stub sementara bila Feature 3.5 belum selesai, ditandai TODO eksplisit dengan referensi task).
- **Definition of Done**: Test HTTP: create invite berhasil, redeem berhasil, redeem tanpa `Idempotency-Key` ditolak `400`.
- **Dependency**: Task 3.3.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `POST /workspaces/{id}/invites`
- [ ] Handler `POST /invites/{code}/redeem` — validasi header `Idempotency-Key` ada
- [ ] Test: create → redeem → verifikasi member baru; redeem tanpa idempotency key → 400

---

## EPIC 4: Role & Permission

### Feature 3.4: Migrasi & CRUD Role

#### Task 3.4.1: Migrasi Tabel `roles`, `member_role_assignments`

- **Deskripsi**: DDL sesuai Database Design §2.2, termasuk kolom `permission_bitmask BIGINT` dan `position INT`.
- **Acceptance Criteria**: Index `idx_roles_workspace_id_position` (DESC) sesuai LLD §2.1 (mendukung resolusi tanpa sort tambahan).
- **Definition of Done**: `migrate up` sukses, index terverifikasi via `\d roles` di psql.
- **Dependency**: Task 3.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `roles` (up & down)
- [ ] Tulis migrasi `member_role_assignments` (up & down, composite PK)
- [ ] Verifikasi index `position DESC`

#### Task 3.4.2: Definisi Permission Bitmask (Konstanta)

- **Deskripsi**: Definisikan flag permission sebagai konstanta bit (LLD §1.2 pola `PermissionFlag`).
- **Acceptance Criteria**: Minimal flag untuk Sprint 3: `MANAGE_WORKSPACE`, `MANAGE_ROLES`, `MANAGE_CHANNELS`, `MANAGE_INVITES`, `SEND_MESSAGES`, `MANAGE_MESSAGES`, `KICK_MEMBERS`, `BAN_MEMBERS` (flag lain ditambah sprint berikutnya sesuai kebutuhan fitur, bukan didefinisikan sekaligus semua di awal — YAGNI).
- **Definition of Done**: Konstanta di `internal/workspace/domain/permission.go`, masing-masing bit unik (unit test verifikasi tidak ada tabrakan bit).
- **Dependency**: Task 3.4.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Definisikan `type PermissionFlag int64` + konstanta `1 << iota`
- [ ] Method `Has(flag)`, `Add(flag)`, `Remove(flag)` pada bitmask
- [ ] Unit test: tidak ada tabrakan bit, kombinasi flag bekerja benar

#### Task 3.4.3: Handler — `POST /workspaces/{id}/roles`, Assignment Role ke Member

- **Deskripsi**: Sesuai API Specification §2, FR-WS-02/FR-WS-04.
- **Acceptance Criteria**: Create role memerlukan permission `MANAGE_ROLES`; `PATCH /workspaces/{id}/members/{memberId}/roles` mengganti seluruh assignment (replace, bukan append — sesuai API Spec).
- **Definition of Done**: Test HTTP: create role, assign ke member, verifikasi `member_role_assignments` sesuai.
- **Dependency**: Task 3.4.2, Feature 3.5 (Permission Resolver — untuk cek `MANAGE_ROLES`)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `POST /workspaces/{id}/roles`
- [ ] Handler `PATCH /workspaces/{id}/members/{memberId}/roles`
- [ ] Test: create role custom, assign, verifikasi member memiliki role tersebut

---

### Feature 3.5: Permission Resolver (Komponen Kunci Sprint Ini)

#### Task 3.5.1: Implementasi Permission Resolver 4-Tingkat

- **Deskripsi**: Implementasi algoritma persis sesuai LLD §2.1 — urutan: Channel member override → Channel role override → Role default (by position) → `@everyone`.
- **Acceptance Criteria**: Fungsi `Resolve(ctx, userID, workspaceID, channelID, required PermissionFlag) (bool, error)` mengikuti urutan resolusi TANPA penyimpangan (RULES.md §7 — urutan ini tidak boleh diubah).
- **Definition of Done**: **Unit test wajib mencakup skenario multi-level override** (bukan hanya kasus sederhana): (a) role default allow tapi channel override deny → hasil deny; (b) role default deny tapi member-specific override allow → hasil allow; (c) tidak ada override sama sekali → fallback ke `@everyone`.
- **Dependency**: Task 3.4.2
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 4 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `internal/workspace/application/permission_resolver.go` sesuai pseudocode LLD §2.1
- [ ] Query pendukung: `FindMemberOverride`, `FindRoleOverride`, `FindMemberRolesSortedByPosition`, `FindEveryoneRole`
- [ ] Unit test skenario (a), (b), (c) di atas — minimal 3 test case eksplisit menguji urutan resolusi
- [ ] Unit test tambahan: member dengan banyak role, role tertinggi (`position` terbesar) menang untuk role default

#### Task 3.5.2: Cache Permission (Redis) + Invalidation

- **Deskripsi**: Sesuai LLD §2.1 (caching) dan §2.6 (invalidation) — **opsional untuk Sprint 3** bila waktu terbatas, namun direkomendasikan dikerjakan agar Sprint 4 (Messaging, dengan volume permission check tinggi) tidak terbebani query berulang.
- **Acceptance Criteria**: Hasil resolusi di-cache TTL 60 detik per `(workspace_id, user_id, channel_id)`; cache diinvalidasi saat role/override berubah.
- **Definition of Done**: Test: setelah role diubah, permission check berikutnya mencerminkan perubahan (bukan cache basi) — bahkan dalam window TTL 60 detik.
- **Dependency**: Task 3.5.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: **Should** (dapat digeser ke Sprint 4 tanpa mengorbankan Sprint Goal Sprint 3 — Permission Resolver tanpa cache tetap fungsional benar, hanya lebih lambat)

**Subtask & Checklist**:
- [ ] Wrap `PermissionResolver.Resolve` dengan cache-aside pattern (Redis)
- [ ] Implementasi `SCAN`-based invalidation (RULES.md §5 — **JANGAN** pakai `KEYS`)
- [ ] Trigger invalidation di `RoleService.UpdatePermission`, `RoleService.AssignRole`
- [ ] Test: ubah role → cache invalidated → resolusi berikutnya benar

---

## EPIC 5: Channel

### Feature 3.6: Migrasi & CRUD Channel

#### Task 3.6.1: Migrasi Tabel `channels`, `channel_permission_overrides`

- **Deskripsi**: DDL sesuai Database Design §2.3 — **termasuk kolom `participant_key` dan constraint untuk tipe `dm`** meski fitur DM baru dikerjakan di Sprint 4/5 (skema disiapkan sekarang agar migrasi tidak perlu expand-contract nanti).
- **Acceptance Criteria**: Constraint `chk_workspace_scoped_or_dm` aktif; partial unique index `uidx_channels_dm_participant_key` dibuat (meski belum dipakai aktif sampai fitur DM datang).
- **Definition of Done**: `migrate up` sukses; test insert channel `type='text'` tanpa `workspace_id` DITOLAK oleh constraint (validasi constraint bekerja).
- **Dependency**: Task 3.1.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `channels` lengkap dengan CHECK constraint (Database Design §2.3)
- [ ] Tulis migrasi `channel_permission_overrides` dengan CHECK constraint XOR (`role_id`/`member_id`)
- [ ] Test: insert channel invalid (text tanpa workspace_id) → ditolak; insert channel dm dengan workspace_id → ditolak

#### Task 3.6.2: Handler — `POST /workspaces/{id}/channels` (Tipe Text)

- **Deskripsi**: FR-CH-01 — untuk Sprint 3, fokus tipe `text` dahulu (voice/video menyusul Release 3 sesuai Development Roadmap).
- **Acceptance Criteria**: Permission `MANAGE_CHANNELS` diperlukan; `type` immutable setelah dibuat (tidak ada endpoint update type).
- **Definition of Done**: Test HTTP: create channel text berhasil, user tanpa permission `MANAGE_CHANNELS` ditolak `403`.
- **Dependency**: Task 3.6.1, Task 3.5.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `ChannelService.Create` (validasi `category_id` milik workspace yang sama bila diisi)
- [ ] Handler `POST /workspaces/{id}/channels`
- [ ] Test: create sukses (Owner), create ditolak (member tanpa permission)

#### Task 3.6.3: Handler — `PATCH /channels/{id}/permission-overrides`

- **Deskripsi**: FR-WS-05 — permission override tingkat channel.
- **Acceptance Criteria**: Sesuai API Specification §3 — validasi XOR `role_id`/`member_id`; memerlukan permission `MANAGE_ROLES` di level workspace/channel.
- **Definition of Done**: Test HTTP: set override deny untuk role tertentu, verifikasi lewat `PermissionResolver` (Task 3.5.1) bahwa member dengan role tersebut kini ditolak akses.
- **Dependency**: Task 3.6.2, Task 3.5.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `PATCH /channels/{id}/permission-overrides`
- [ ] Validasi request: HARUS salah satu dari `role_id`/`member_id`, tidak boleh keduanya/tidak ada
- [ ] Test: set override → verifikasi via Permission Resolver hasil berubah sesuai

---

### Feature 3.7: Integration Test End-to-End Sprint 3

#### Task 3.7.1: Skenario Penuh — Sprint Goal Verification

- **Deskripsi**: Menguji **persis** skenario yang didefinisikan sebagai Definition of Done Release 1 di Development Roadmap: *buat workspace → invite user kedua → assign role custom → buat channel dengan override permission → verifikasi user kedua tidak bisa akses channel privat*.
- **Acceptance Criteria**: Seluruh langkah berjalan lewat HTTP API (bukan manipulasi database langsung), mencerminkan alur pengguna nyata.
- **Definition of Done**: Test hijau konsisten di CI; ini adalah **gerbang kelulusan Release 1** (Development Roadmap §2) — bila test ini lolos, Release 1 dianggap selesai.
- **Dependency**: Seluruh task Epic 3, 4, 5
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis test: User A register & login → create workspace
- [ ] User A create invite → User B register & login → redeem invite
- [ ] User A create role "Restricted" (tanpa `SEND_MESSAGES`), assign ke User B
- [ ] User A create channel privat, set permission override deny `SEND_MESSAGES`/read untuk role "Restricted"
- [ ] Verifikasi: User B mencoba `GET /channels/{id}/messages` atau kirim pesan → `403 FORBIDDEN`
- [ ] Verifikasi: User A (Owner, permission penuh) tetap dapat mengakses channel tersebut
- [ ] Jalankan 3x berturut memastikan tidak flaky, pastikan CI hijau
- [ ] Update `docs/AGENTS.md` §7 — tandai Release 1 selesai, Sprint aktif berpindah ke Sprint 4 (Release 2)

---

## Ringkasan Keputusan

1. Sprint 3 mencakup **5 Feature inti (3.1-3.4, 3.6) + 1 Feature krusial (3.5 Permission Resolver) + 1 Feature verifikasi (3.7)**, total 13 task, menuntaskan Release 1 secara penuh.
2. Skema `channels` disiapkan **lengkap dengan constraint tipe `dm`** sejak sprint ini (Task 3.6.1) meski fitur DM baru datang Sprint 4/5 — menghindari migrasi expand-contract yang tidak perlu nanti (konsisten dengan RULES.md §2 dan semangat proaktif Database Design).
3. Task 3.5.1 (Permission Resolver) ditandai **Estimasi Kesulitan: Tinggi** dan diberi porsi waktu terbesar (4 jam) — ini komponen paling kritikal secara arsitektur di sprint ini karena dipakai oleh hampir seluruh domain berikutnya.
4. Task 3.7.1 dijadikan **gerbang kelulusan Release 1** — bukan sekadar test biasa, tapi validasi langsung terhadap Development Roadmap.

## Trade-off yang Diterima

- Task 3.5.2 (Cache Permission) diberi prioritas *Should*, boleh digeser ke Sprint 4 — Permission Resolver tanpa cache tetap benar secara fungsional, hanya berpotensi lebih lambat; ini trade-off yang aman untuk skala data Sprint 3 (masih sedikit).
- Task 3.3.2 mencatat kemungkinan stub sementara untuk permission check bila Feature 3.5 belum selesai saat Feature 3.3 dikerjakan — trade-off pragmatis untuk paralelisasi kerja dalam sprint yang sama, dengan TODO eksplisit sebagai pengingat wajib.

## Risiko Arsitektur

- Task 3.5.1 adalah **titik kegagalan tunggal** yang mempengaruhi kebenaran seluruh sistem otorisasi — bug di sini akan merambat ke seluruh fitur mendatang (Messaging, Voice, dll). Disiplin test coverage untuk task ini harus lebih tinggi dari task lain di sprint ini (minimal 3 skenario eksplisit, bukan hanya 1 happy path).
- Constraint CHECK untuk channel `dm` disiapkan sejak sekarang, namun belum benar-benar diuji dengan data DM nyata (baru diuji constraint-nya menolak data invalid) — validasi penuh menunggu Sprint 4/5.

## Technical Debt yang Sengaja Diterima

- Flag permission (Task 3.4.2) baru mencakup 8 flag dasar — flag tambahan (mis. `MANAGE_ANNOUNCEMENT`, `MENTION_EVERYONE`) akan ditambah saat fitur terkait dikerjakan (Sprint 4+), bukan didefinisikan sekaligus di awal (YAGNI, sesuai catatan di Task 3.4.2).

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah Task 3.5.2 (Cache Permission) dikerjakan di Sprint 3 ini, atau digeser ke Sprint 4 sesuai catatan trade-off di atas?
2. Dengan selesainya breakdown ini, **Release 1 (Foundation) sudah terencana detail penuh (Sprint 1-3)**. Apakah Anda ingin saya mulai menyiapkan **Sprint 4** (awal Release 2 — Realtime Chat + DM), atau berhenti dulu di sini menunggu Sprint 1-3 benar-benar dieksekusi?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 3: 13 task lengkap mencakup Workspace, Role/Permission Resolver, Channel, menuntaskan Release 1 dengan gerbang kelulusan end-to-end test |
