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
- [x] Tulis migrasi `workspaces` (up & down)
- [x] Tulis migrasi `categories` (up & down)
- [x] Tulis migrasi `invites` (up & down)
- [x] Verifikasi FK constraint via test insert invalid

#### Task 3.1.2: Migrasi Tabel `members`

- **Deskripsi**: Aggregate terpisah dari workspace (HLD §2.3), didesain untuk skala 100.000 row/workspace.
- **Acceptance Criteria**: Constraint unik `(workspace_id, user_id)`; index `idx_members_workspace_id` dan `idx_members_user_id` sesuai Database Design §2.2.
- **Definition of Done**: Insert member duplikat (workspace_id+user_id sama) ditolak database.
- **Dependency**: Task 3.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [x] Tulis migrasi `members` (up & down)
- [x] Verifikasi unique constraint via test insert duplikat
- [x] Verifikasi kedua index dengan `EXPLAIN` query dasar

#### Task 3.1.3: sqlc Setup — Domain Workspace & Member

- **Deskripsi**: Query dasar CRUD untuk workspace, category, invite, member.
- **Acceptance Criteria**: `sqlc generate` sukses untuk seluruh query baru.
- **Definition of Done**: Kode ter-generate dapat dipanggil dari test sederhana.
- **Dependency**: Task 3.1.1, 3.1.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [x] Query: `CreateWorkspace`, `ListWorkspacesByUserID` (join via members, cursor-based — Playbook §17.2)
- [x] Query: `CreateCategory`, `CreateInvite`, `FindInviteByCode`, `IncrementInviteUseCount`
- [x] Query: `CreateMember`, `FindMemberByWorkspaceAndUser`, `ListMembersByWorkspace` (cursor-based, SRS FR-WS-08)
- [x] `sqlc generate`, verifikasi tanpa error

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
- [x] Implementasi `internal/workspace/domain/workspace.go`
- [x] Implementasi `WorkspaceService.Create` dengan DB transaction (workspace + member + role + assignment)
- [x] Unit test dengan mock; integration test dengan DB nyata
- [x] Verifikasi rollback penuh bila salah satu langkah dalam transaksi gagal

#### Task 3.2.2: Handler — `POST /api/v1/workspaces`, `GET /api/v1/workspaces`

- **Deskripsi**: Sesuai API Specification §2.
- **Acceptance Criteria**: Create sesuai kontrak; List memakai cursor-based pagination, hanya menampilkan workspace milik user yang login (join via `members`).
- **Definition of Done**: Test HTTP: create → muncul di list; user lain tidak melihat workspace tersebut di list-nya.
- **Dependency**: Task 3.2.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [x] Handler `POST /workspaces` (proteksi Auth Middleware Sprint 2)
- [x] Handler `GET /workspaces` (cursor pagination, LLD §2.2 pola)
- [x] Test: isolasi antar user (workspace user A tidak muncul di list user B)

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
- [x] Implementasi `InviteService.Create` (max_uses, expires_at nullable)
- [x] Implementasi `InviteService.Redeem` — cek existing membership dulu sebelum insert (idempotent by design, bukan hanya mengandalkan `Idempotency-Key` header)
- [x] Validasi `expires_at`/`max_uses` sebelum redeem → error `BUSINESS_RULE_VIOLATION` sesuai kondisi
- [x] Integration test: redeem ganda, redeem kedaluwarsa, redeem max_uses tercapai

#### Task 3.3.2: Handler — `POST /workspaces/{id}/invites`, `POST /invites/{code}/redeem`

- **Deskripsi**: Sesuai API Specification §2, dengan header `Idempotency-Key` wajib untuk redeem (Playbook §17.4).
- **Acceptance Criteria**: Endpoint create invite memerlukan permission `MANAGE_INVITES` (bergantung Permission Resolver — Feature 3.5, dikerjakan paralel/setelahnya; untuk task ini, cek permission dapat memakai stub sementara bila Feature 3.5 belum selesai, ditandai TODO eksplisit dengan referensi task).
- **Definition of Done**: Test HTTP: create invite berhasil, redeem berhasil, redeem tanpa `Idempotency-Key` ditolak `400`.
- **Dependency**: Task 3.3.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [x] Handler `POST /workspaces/{id}/invites`
- [x] Handler `POST /invites/{code}/redeem` — validasi header `Idempotency-Key` ada
- [x] Test: create → redeem → verifikasi member baru; redeem tanpa idempotency key → 400

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
- [x] Tulis migrasi `roles` (up & down)
- [x] Tulis migrasi `member_role_assignments` (up & down, composite PK)
- [x] Verifikasi index `position DESC`

#### Task 3.4.2: Definisi Permission Bitmask (Konstanta)

- **Deskripsi**: Definisikan flag permission sebagai konstanta bit (LLD §1.2 pola `PermissionFlag`).
- **Acceptance Criteria**: Minimal flag untuk Sprint 3: `MANAGE_WORKSPACE`, `MANAGE_ROLES`, `MANAGE_CHANNELS`, `MANAGE_INVITES`, `SEND_MESSAGES`, `MANAGE_MESSAGES`, `KICK_MEMBERS`, `BAN_MEMBERS` (flag lain ditambah sprint berikutnya sesuai kebutuhan fitur, bukan didefinisikan sekaligus semua di awal — YAGNI).
- **Definition of Done**: Konstanta di `internal/workspace/domain/permission.go`, masing-masing bit unik (unit test verifikasi tidak ada tabrakan bit).
- **Dependency**: Task 3.4.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [x] Definisikan `type PermissionFlag int64` + konstanta `1 << iota`
- [x] Method `Has(flag)`, `Add(flag)`, `Remove(flag)` pada bitmask
- [x] Unit test: tidak ada tabrakan bit, kombinasi flag bekerja benar

#### Task 3.4.3: Handler — `POST /workspaces/{id}/roles`, Assignment Role ke Member

- **Deskripsi**: Sesuai API Specification §2, FR-WS-02/FR-WS-04.
- **Acceptance Criteria**: Create role memerlukan permission `MANAGE_ROLES`; `PATCH /workspaces/{id}/members/{memberId}/roles` mengganti seluruh assignment (replace, bukan append — sesuai API Spec).
- **Definition of Done**: Test HTTP: create role, assign ke member, verifikasi `member_role_assignments` sesuai.
- **Dependency**: Task 3.4.2, Feature 3.5 (Permission Resolver — untuk cek `MANAGE_ROLES`)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [x] Handler `POST /workspaces/{id}/roles`
- [x] Handler `PATCH /workspaces/{id}/members/{memberId}/roles`
- [x] Test: create role custom, assign, verifikasi member memiliki role tersebut

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
- [x] Implementasi `internal/workspace/application/permission_resolver.go` sesuai pseudocode LLD §2.1
- [x] Query pendukung: `FindMemberOverride`, `FindRoleOverride`, `FindMemberRolesSortedByPosition`, `FindEveryoneRole`
- [x] Unit test skenario (a), (b), (c) di atas — minimal 3 test case eksplisit menguji urutan resolusi
- [x] Unit test tambahan: member dengan banyak role, role tertinggi (`position` terbesar) menang untuk role default

#### Task 3.5.2: Cache Permission (Redis) + Invalidation

- **Deskripsi**: Sesuai LLD §2.1 (caching) dan §2.6 (invalidation) — **opsional untuk Sprint 3** bila waktu terbatas, namun direkomendasikan dikerjakan agar Sprint 4 (Messaging, dengan volume permission check tinggi) tidak terbebani query berulang.
- **Acceptance Criteria**: Hasil resolusi di-cache TTL 60 detik per `(workspace_id, user_id, channel_id)`; cache diinvalidasi saat role/override berubah.
- **Definition of Done**: Test: setelah role diubah, permission check berikutnya mencerminkan perubahan (bukan cache basi) — bahkan dalam window TTL 60 detik.
- **Dependency**: Task 3.5.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: **Should** (dapat digeser ke Sprint 4 tanpa mengorbankan Sprint Goal Sprint 3 — Permission Resolver tanpa cache tetap fungsional benar, hanya lebih lambat)

**Subtask & Checklist**:
- [x] Wrap `PermissionResolver.Resolve` dengan cache-aside pattern (Redis)
- [x] Implementasi `SCAN`-based invalidation (RULES.md §5 — **JANGAN** pakai `KEYS`)
- [x] Trigger invalidation di `RoleService.UpdatePermission`, `RoleService.AssignRole`
- [x] Test: ubah role → cache invalidated → resolusi berikutnya benar

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
- [x] Buat file migrasi up/down untuk tabel `channels`
- [x] Buat file migrasi up/down untuk tabel `channel_permission_overrides`
- [x] Verifikasi enum type ('text', 'voice', dll) dan constraint `chk_workspace_scoped_or_dm`
- [x] Jalankan `golang-migrate` dan pastikan sukses di lokal

#### Task 3.6.2: Handler — `POST /workspaces/{id}/channels` (Tipe Text)

- **Deskripsi**: FR-CH-01 — untuk Sprint 3, fokus tipe `text` dahulu (voice/video menyusul Release 3 sesuai Development Roadmap).
- **Acceptance Criteria**: Permission `MANAGE_CHANNELS` diperlukan; `type` immutable setelah dibuat (tidak ada endpoint update type).
- **Definition of Done**: Test HTTP: create channel text berhasil, user tanpa permission `MANAGE_CHANNELS` ditolak `403`.
- **Dependency**: Task 3.6.1, Task 3.5.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [x] Implementasi `ChannelService.Create` (validasi `category_id` milik workspace yang sama bila diisi)
- [x] Handler `POST /workspaces/{id}/channels`
- [x] Test: create sukses (Owner), create ditolak (member tanpa permission)

#### Task 3.6.3: Handler — `PATCH /channels/{id}/permission-overrides`

- **Deskripsi**: FR-WS-05 — permission override tingkat channel.
- **Acceptance Criteria**: Sesuai API Specification §3 — validasi XOR `role_id`/`member_id`; memerlukan permission `MANAGE_ROLES` di level workspace/channel.
- **Definition of Done**: Test HTTP: set override deny untuk role tertentu, verifikasi lewat `PermissionResolver` (Task 3.5.1) bahwa member dengan role tersebut kini ditolak akses.
- **Dependency**: Task 3.6.2, Task 3.5.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [x] Handler `PATCH /channels/{id}/permission-overrides`
- [x] Validasi request: HARUS salah satu dari `role_id`/`member_id`, tidak boleh keduanya/tidak ada
- [x] Test: set override → verifikasi via Permission Resolver hasil berubah sesuai

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
- [x] Tulis test: User A register & login → create workspace
- [x] User A create invite → User B register & login → redeem invite
- [x] User A create role "Restricted" (tanpa `SEND_MESSAGES`), assign ke User B
- [x] User A create channel privat, set permission override deny `SEND_MESSAGES`/read untuk role "Restricted"
- [x] Verifikasi: User B mencoba `GET /channels/{id}/messages` atau kirim pesan → `403 FORBIDDEN`
- [x] Verifikasi: User A (Owner, permission penuh) tetap dapat mengakses channel tersebut
- [x] Jalankan 3x berturut memastikan tidak flaky, pastikan CI hijau
- [x] Update `docs/AGENTS.md` §7 — tandai Release 1 selesai, Sprint aktif berpindah ke Sprint 4 (Release 2)

---

## EPIC 6: Frontend — Workspace, Channel, Layout Utama

### Feature 6.1: Layout Utama & Server Sidebar

#### Task 6.1.1: Layout `default.vue` — Server Sidebar + Channel Sidebar + Main Content

- **Deskripsi**: Struktur layout khas Discord (Frontend Architecture §2) — sidebar kiri (daftar workspace/server), sidebar tengah (kategori/channel), area konten kanan.
- **Acceptance Criteria**: Layout responsif dasar (tidak perlu sempurna mobile di sprint ini — cukup fungsional desktop-first, mobile responsiveness penuh dicatat sebagai debt untuk sprint UI-polish nanti bila ada).
- **Definition of Done**: E2E test: navigasi antar workspace via sidebar mengubah channel sidebar sesuai workspace aktif.
- **Dependency**: Task 3.4.1 (Sprint 2 — auto re-auth, layout hanya render setelah auth resolved)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `layouts/default.vue`
- [ ] Komponen `ServerSidebar.vue` (daftar workspace, icon)
- [ ] Komponen `ChannelSidebar.vue` (kategori + channel list, placeholder sebelum Task 6.2.2)
- [ ] E2E test navigasi dasar

#### Task 6.1.2: Pinia Store — `activeWorkspace.ts`

- **Deskripsi**: State client "workspace/channel mana yang sedang dibuka" sesuai Frontend Architecture §3.1.
- **Acceptance Criteria**: State ini **bukan** hasil fetch API (itu tanggung jawab TanStack Query) — murni penanda UI "sedang di mana".
- **Definition of Done**: Unit test store: set/get `currentWorkspaceId`/`currentChannelId`.
- **Dependency**: Task 6.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `stores/activeWorkspace.ts`
- [ ] Unit test dasar

---

### Feature 6.2: Workspace & Invite UI

#### Task 6.2.1: Halaman Buat/Lihat Workspace

- **Deskripsi**: UI untuk `POST/GET /workspaces` (Task 3.2.1/3.2.2 backend).
- **Acceptance Criteria**: Form buat workspace, daftar workspace milik user tampil di `ServerSidebar` (Task 6.1.1).
- **Definition of Done**: E2E test: buat workspace baru → langsung muncul di sidebar tanpa reload manual (TanStack Query invalidation, Frontend Architecture §3.2).
- **Dependency**: Task 6.1.1, Task 3.2.2 (backend)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Modal/halaman buat workspace dengan `useMutation`
- [ ] `useQuery` daftar workspace, wire ke `ServerSidebar`
- [ ] E2E test: buat → muncul otomatis

#### Task 6.2.2: UI Invite — Generate & Redeem

- **Deskripsi**: UI untuk `POST /workspaces/{id}/invites` dan `POST /invites/{code}/redeem` (Task 3.3.2 backend).
- **Acceptance Criteria**: Redeem invite via halaman terpisah (`pages/invite/[code].vue`), mengirim `Idempotency-Key` header (client generate UUID per percobaan submit, sesuai Playbook §17.4 — bukan backend yang generate).
- **Definition of Done**: E2E test: generate invite → link disalin → dibuka user lain → redeem berhasil → workspace muncul di sidebar user tersebut.
- **Dependency**: Task 6.2.1, Task 3.3.2 (backend)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Komponen generate invite (dengan opsi `max_uses`/`expires_in_hours`)
- [ ] Halaman redeem invite (`pages/invite/[code].vue`) dengan `Idempotency-Key` generated client-side
- [ ] E2E test alur penuh 2 user

---

### Feature 6.3: Role & Permission UI

#### Task 6.3.1: Halaman Kelola Role (Owner/Admin)

- **Deskripsi**: UI untuk `POST /workspaces/{id}/roles`, assignment role ke member (Task 3.4.3 backend).
- **Acceptance Criteria**: Form buat role dengan checkbox per permission flag (bukan input bitmask mentah — UX harus menerjemahkan flag jadi label manusiawi, mapping dilakukan di frontend berdasarkan konstanta yang **disinkronkan manual** dengan `internal/workspace/domain/permission.go` backend — dicatat sebagai titik yang wajib diperbarui bersamaan setiap kali flag baru ditambah, RULES.md prinsip yang sama diterapkan lintas bahasa).
- **Definition of Done**: E2E test: buat role custom → assign ke member → member tersebut menerima permission sesuai.
- **Dependency**: Task 6.2.1, Task 3.4.3 (backend)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Konstanta permission flag di frontend (`constants/permissions.ts`) — **disinkronkan manual** dengan backend, komentar eksplisit mengingatkan hal ini
- [ ] Form buat role (checkbox per flag)
- [ ] UI assignment role ke member
- [ ] E2E test alur penuh

#### Task 6.3.2: Permission-Aware UI Rendering — Konsumsi `viewer_permissions`

- **Deskripsi**: Implementasi `usePermission()` composable persis Frontend Architecture §7 — mengonsumsi field `viewer_permissions` dari response API (amandemen API Spec).
- **Acceptance Criteria**: **RULES.md-consistent**: composable ini **tidak pernah** menghitung ulang logic resolusi permission — murni membaca field yang sudah dihitung backend. Tombol/aksi yang tidak diizinkan disembunyikan (UX), namun backend tetap jadi penjaga sesungguhnya (sudah ada sejak Sprint 3 backend, Task 3.5.1).
- **Definition of Done**: Test: user dengan `viewer_permissions.can_manage_channel = false` tidak melihat tombol "Kelola Channel"; verifikasi manual bahwa mencoba memanggil endpoint tersebut langsung via network tab tetap ditolak backend (membuktikan frontend hanya UI hint, bukan satu-satunya lapisan proteksi).
- **Dependency**: Task 6.3.1, amandemen API Spec §3 (`viewer_permissions`)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `composables/usePermission.ts` persis §7 Frontend Architecture
- [ ] Terapkan di komponen channel/workspace settings (sembunyikan aksi tanpa izin)
- [ ] Test: UI hint bekerja + verifikasi manual backend tetap menolak request langsung

---

### Feature 6.4: Channel UI

#### Task 6.4.1: Buat & Tampilkan Channel (Tipe Text)

- **Deskripsi**: UI untuk `POST /workspaces/{id}/channels` (Task 3.6.2 backend).
- **Acceptance Criteria**: Channel baru muncul di `ChannelSidebar` (Task 6.1.1) sesuai kategori; klik channel mengubah `activeWorkspace.currentChannelId` (Task 6.1.2) dan URL (`pages/workspaces/[id]/channels/[channelId].vue`).
- **Definition of Done**: E2E test: buat channel → muncul di sidebar → klik → URL berubah sesuai.
- **Dependency**: Task 6.1.2, Task 3.6.2 (backend)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Modal buat channel (permission-aware, reuse Task 6.3.2)
- [ ] `ChannelSidebar` menampilkan channel per kategori (reuse `useQuery`)
- [ ] Routing dinamis `pages/workspaces/[id]/channels/[channelId].vue`
- [ ] E2E test navigasi channel

---

### Feature 6.5: Integration Test End-to-End Frontend Sprint 3

#### Task 6.5.1: Skenario Penuh — Mencerminkan Gerbang Backend (Task 3.7.1)

- **Deskripsi**: Versi frontend dari skenario end-to-end backend Task 3.7.1 — memverifikasi UX-nya, bukan hanya API-nya.
- **Acceptance Criteria**: Alur via UI sungguhan (Playwright, bukan panggil API langsung): User A buat workspace → invite User B via UI → User B redeem via UI → User A buat role "Restricted" tanpa `SEND_MESSAGES` via UI → assign ke User B via UI → User A buat channel privat dengan override via UI → User B login, channel tersebut **tidak muncul dapat diakses** di sidebar-nya (atau muncul namun composer pesan disembunyikan sesuai `viewer_permissions`).
- **Definition of Done**: Test Playwright hijau konsisten 3x run berturut.
- **Dependency**: Seluruh task Epic 6
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 3.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis skenario Playwright penuh (2 browser context simulasi 2 user)
- [ ] Jalankan 3x berturut, pastikan tidak flaky
- [ ] Update `docs/AGENTS.md` §7 — Sprint 3 frontend selesai bersamaan backend

---

## Ringkasan Keputusan

1. Sprint 3 mencakup **5 Feature inti (3.1-3.4, 3.6) + 1 Feature krusial (3.5 Permission Resolver) + 1 Feature verifikasi (3.7)**, total 13 task backend, menuntaskan Release 1 secara penuh. *(Direvisi: ditambah Epic 6 — 5 Feature, 9 Task frontend, amandemen retroaktif — Layout utama, Workspace/Invite/Role/Channel UI, dan permission-aware rendering yang mengonsumsi `viewer_permissions`.)*
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
| 1.1.0 | Amandemen | Ditambahkan Epic 6: Frontend (Layout utama, Workspace/Invite/Role/Channel UI, permission-aware rendering via `viewer_permissions`) — amandemen retroaktif, menutup celah cakupan frontend |
