# API Specification
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 6 — API Specification
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `01-engineering-playbook.md` (§17 API Convention), `06-srs.md` (v1.1.0), `08-lld.md`, `09-database-design.md`
**Klasifikasi:** Internal — Source of Truth

---

## 0. Konvensi Global (Recap & Detail Presisi)

- **Base path**: `/api/v1`
- **Auth header**: `Authorization: Bearer <access_token>`
- **Response envelope**: sesuai §17.1 Playbook (`success`, `data`, `meta` / `error`)
- **Pagination**: cursor-based, query param `cursor` & `limit` (default 50, max 100), response `meta.next_cursor` & `meta.has_more`
- **Idempotency**: header `Idempotency-Key` (UUID) untuk endpoint dengan efek samping (create invite redeem, create DM)
- **Rate limit headers** (setiap response): `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

### Error Code Catalog (Global)

| HTTP Status | Code | Kapan Dipakai |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Field request tidak valid, `details` berisi list field & pesan |
| 401 | `UNAUTHORIZED` | Token tidak ada/invalid/expired |
| 403 | `FORBIDDEN` | Otentikasi valid, namun permission tidak cukup |
| 404 | `RESOURCE_NOT_FOUND` | Resource tidak ditemukan (atau soft-deleted) |
| 409 | `OPTIMISTIC_LOCK_CONFLICT` | Version tidak cocok saat update (FR-MSG-09) |
| 409 | `DUPLICATE_RESOURCE` | Pelanggaran unique constraint (mis. username sudah dipakai) |
| 422 | `BUSINESS_RULE_VIOLATION` | Melanggar aturan bisnis (mis. invite sudah kedaluwarsa, grup DM melebihi 10 partisipan) |
| 429 | `RATE_LIMIT_EXCEEDED` | Melebihi rate limit (§3.5 SRS) |
| 500 | `INTERNAL_ERROR` | Kegagalan tak terduga, di-log Error level |

---

## 1. Authentication

### `POST /api/v1/auth/register`

- **Auth**: Tidak perlu
- **Rate Limit**: 20/jam per IP
- **Request**:
```json
{ "email": "user@example.com", "username": "johndoe", "display_name": "John Doe", "password": "********" }
```
- **Validasi**: `email` format valid & unik; `username` 3-32 karakter alfanumerik+underscore, unik (case-insensitive); `password` minimal 8 karakter mengandung huruf & angka.
- **Response 201**:
```json
{ "success": true, "data": { "id": "uuid", "username": "johndoe", "email": "user@example.com" } }
```
- **Error**: `409 DUPLICATE_RESOURCE` (email/username sudah dipakai), `400 VALIDATION_ERROR`.

### `POST /api/v1/auth/login`

- **Auth**: Tidak perlu
- **Rate Limit**: 5/15 menit per identifier (lockout progresif, §3.5 SRS)
- **Request**: `{ "identifier": "johndoe atau user@example.com", "password": "********" }`
- **Response 200**: `{ "data": { "access_token": "...", "refresh_token": "...", "expires_in": 900 } }`
- **Error**: `401 UNAUTHORIZED` (pesan generik, tidak membedakan sebab — FR-AUTH-03).

### `POST /api/v1/auth/refresh`

- **Auth**: Refresh token (body, atau cookie HttpOnly bila diaktifkan — Security Design Phase 7)
- **Response 200**: access & refresh token baru (rotated — FR-AUTH-04); refresh token lama otomatis `revoked`.
- **Error**: `401 UNAUTHORIZED` (refresh token invalid/revoked/expired).

### `POST /api/v1/auth/logout` *(amandemen — FR-AUTH-08, diimplementasikan)*

- **Auth**: Wajib
- **Request**: kosong — refresh token diambil dari cookie HttpOnly pada request ini sendiri (bukan dari body/path), sehingga client tidak perlu tahu/menyimpan `sessionId`.
- **Response 204**: hanya sesi yang terkait refresh token cookie tersebut yang di-revoke; sesi lain (device lain) tetap aktif.
- **Perbedaan dengan `logout-all`**: endpoint ini adalah aksi "Logout" biasa (device ini saja); `logout-all` adalah aksi eksplisit terpisah ("Logout dari semua device").

### `POST /api/v1/auth/logout-all`

- **Auth**: Wajib
- **Response 204**: seluruh sesi/refresh token user direvoke (FR-AUTH-05).

### `GET /api/v1/auth/sessions`

- **Auth**: Wajib
- **Response 200**: daftar sesi aktif (device, IP, last_active) — FR-AUTH-06.

### `DELETE /api/v1/auth/sessions/{sessionId}`

- **Auth**: Wajib, hanya pemilik sesi
- **Response 204**

---

## 2. Workspace, Role, Permission, Category

### `POST /api/v1/workspaces`

- **Auth**: Wajib
- **Request**: `{ "name": "My Server", "icon_url": "..." }`
- **Response 201**: workspace baru, pembuat otomatis jadi Owner + role `@everyone` dibuat otomatis (FR-WS-01/02).

### `GET /api/v1/workspaces`

- **Auth**: Wajib — daftar workspace milik user (join via `members`)
- **Pagination**: cursor-based

### `POST /api/v1/workspaces/{workspaceId}/invites`

- **Auth**: Permission `MANAGE_INVITES`
- **Request**: `{ "max_uses": 10, "expires_in_hours": 24 }` (keduanya opsional/nullable — FR-WS-06)
- **Response 201**: `{ "code": "abc123", "url": "https://nexus.app/invite/abc123" }`

### `POST /api/v1/invites/{code}/redeem`

- **Auth**: Wajib
- **Idempotency**: Wajib header `Idempotency-Key` (§17.4 Playbook) — redeem ganda mengembalikan status membership yang sudah ada (FR-WS-06).
- **Response 200**: `{ "workspace_id": "...", "member_id": "..." }`
- **Error**: `422 BUSINESS_RULE_VIOLATION` (invite kedaluwarsa/max_uses tercapai), `404 RESOURCE_NOT_FOUND`.

### `POST /api/v1/workspaces/{workspaceId}/roles`

- **Auth**: Permission `MANAGE_ROLES`
- **Request**: `{ "name": "Moderator", "permission_bitmask": 12, "position": 5 }`

### `PATCH /api/v1/workspaces/{workspaceId}/members/{memberId}/roles`

- **Auth**: Permission `MANAGE_ROLES`
- **Request**: `{ "role_ids": ["uuid1", "uuid2"] }` (replace seluruh assignment)

### `POST /api/v1/workspaces/{workspaceId}/categories`

- **Auth**: Permission `MANAGE_CHANNELS`
- **Request**: `{ "name": "General", "position": 0 }`

---

## 3. Channel

### `POST /api/v1/workspaces/{workspaceId}/channels`

- **Auth**: Permission `MANAGE_CHANNELS`
- **Request**: `{ "type": "text", "name": "general", "category_id": "uuid-or-null" }`
- **Validasi**: `type` immutable setelah dibuat (FR-CH-01).
- **Response 201**

### `PATCH /api/v1/channels/{channelId}/permission-overrides`

- **Auth**: Permission `MANAGE_ROLES` di level channel (atau permission workspace setara)
- **Request**: `{ "role_id": "uuid-or-null", "member_id": "uuid-or-null", "allow_bitmask": 4, "deny_bitmask": 0 }`
- **Validasi**: XOR `role_id`/`member_id` (Database Design §2.3).

### `GET /api/v1/channels/{channelId}/voice/participants`

- **Auth**: Permission baca channel voice
- **Response 200**: daftar partisipan aktif (bersumber dari state LiveKit via `voice_sessions`).

---

## 4. Messaging

### `POST /api/v1/channels/{channelId}/messages`

- **Auth**: Permission `SEND_MESSAGES` (via `ChannelAuthorizationService`, LLD §1.2) — untuk channel `dm`, dicek via `BlockService` (FR-DM-04).
- **Rate Limit**: 10 pesan/10 detik per user per channel (§3.5 SRS)
- **Request**:
```json
{ "content": "Halo semua!", "reply_to_id": "uuid-or-null", "mentions": ["uuid1"], "attachment_ids": ["uuid-attach"] }
```
- **Validasi**: `content` maksimal 4000 karakter (FR-MSG-01), minimal 1 karakter ATAU minimal 1 attachment (pesan tidak boleh benar-benar kosong).
- **Response 201**: objek message lengkap.
- **Error**: `403 FORBIDDEN` (tidak punya permission, atau diblokir untuk DM), `429 RATE_LIMIT_EXCEEDED`.

### `GET /api/v1/channels/{channelId}/messages`

- **Auth**: Permission baca channel
- **Pagination**: cursor-based (`cursor`, `limit` max 100) — FR-MSG-10
- **Response 200**: array message + `meta.next_cursor`

### `PATCH /api/v1/messages/{messageId}`

- **Auth**: Hanya penulis asli
- **Request**: `{ "content": "...", "expected_version": 3 }`
- **Error**: `409 OPTIMISTIC_LOCK_CONFLICT` bila `expected_version` tidak cocok dengan version terkini (FR-MSG-09).

### `DELETE /api/v1/messages/{messageId}`

- **Auth**: Penulis asli ATAU permission `MANAGE_MESSAGES`
- **Response 204**: soft delete (FR-MSG-08).

### `PUT /api/v1/messages/{messageId}/reactions/{emoji}`

- **Auth**: Permission baca channel
- **Response 204**: idempotent (menambah reaksi yang sama dua kali tidak error, hanya no-op — FR-MSG-06).

### `DELETE /api/v1/messages/{messageId}/reactions/{emoji}`

- **Auth**: Hanya milik reaksi sendiri
- **Response 204**

### `POST /api/v1/messages/{messageId}/threads`

- **Auth**: Permission `SEND_MESSAGES` di channel induk
- **Response 201**: thread baru dengan `thread_root_id = messageId`.

---

## 5. Direct Message (DM)

### `POST /api/v1/dm`

- **Auth**: Wajib
- **Idempotency**: Wajib header `Idempotency-Key`
- **Request**: `{ "participant_ids": ["uuid1", "uuid2"] }` (2 = 1-on-1, 3-10 = grup DM — FR-DM-03)
- **Validasi**: Tidak ada partisipan yang saling memblokir (FR-DM-04); untuk 1-on-1, bila channel sudah ada (dicek via `participant_key`), mengembalikan channel yang sudah ada (bukan membuat duplikat — FR-DM-02).
- **Response 200/201**: objek channel tipe `dm`.
- **Error**: `422 BUSINESS_RULE_VIOLATION` (partisipan > 10, atau ada yang saling blokir).

### `POST /api/v1/users/{userId}/block`

- **Auth**: Wajib
- **Response 204**

### `DELETE /api/v1/users/{userId}/block`

- **Auth**: Wajib
- **Response 204**

---

## 6. Upload

### `POST /api/v1/attachments`

- **Auth**: Wajib
- **Rate Limit**: 20 file/jam per user
- **Request**: `multipart/form-data`, field `file` (maks 1 GB — FR-UP-01)
- **Validasi**: magic bytes sesuai tipe didukung (FR-UP-02).
- **Response 202 Accepted**: `{ "attachment_id": "uuid", "status": "pending" }` — pemrosesan asynchronous (FR-UP-03/04), client polling status via `GET /attachments/{id}` atau menunggu WS event `media.ThumbnailGenerated`.
- **Error**: `413 Payload Too Large` (ukuran > 1GB), `415 Unsupported Media Type`.

### `GET /api/v1/attachments/{attachmentId}`

- **Auth**: Permission baca message terkait
- **Response 200**: status & URL final (bila `processed`).

---

## 7. Search

### `GET /api/v1/search/messages`

- **Auth**: Wajib
- **Rate Limit**: 30 query/menit (§3.5 SRS)
- **Query Params**: `q` (wajib), `channel_ids` (opsional, comma-separated), `author_id` (opsional), `from_date`/`to_date` (opsional), `has_attachment` (opsional boolean)
- **Response 200**: hasil pencarian, hanya channel yang dapat diakses user (FR-SRCH-02, permission check sebelum query — LLD §2.3 pola dynamic filter).

### `GET /api/v1/search/users`, `GET /api/v1/search/channels`, `GET /api/v1/search/workspaces`

- **Auth**: Wajib
- **Query Params**: `q` (wajib, min 2 karakter)
- **Response 200**: hasil trigram search (FR-SRCH-03).

---

## 8. Notification

### `GET /api/v1/notifications/preferences`

- **Auth**: Wajib
- **Response 200**: daftar preferensi per scope (workspace/channel).

### `PUT /api/v1/notifications/preferences`

- **Auth**: Wajib
- **Request**: `{ "scope_type": "channel", "scope_id": "uuid", "level": "mentions_only" }` (FR-NOTIF-03)

---

## 9. Admin

### `GET /api/v1/admin/dashboard`

- **Auth**: Role Platform Admin (permission khusus tingkat platform, bukan workspace)
- **Response 200**: metrik agregat (FR-ADM-01).

### `POST /api/v1/admin/users/{userId}/suspend`

- **Auth**: Platform Admin
- **Request**: `{ "reason": "..." }`
- **Response 204**: dicatat ke `audit_logs` (FR-ADM-02/03).

### `GET /api/v1/admin/audit-logs`

- **Auth**: Platform Admin
- **Pagination**: cursor-based
- **Query Params**: `actor_id`, `target_type`, `from_date`/`to_date` (opsional filter)

---

## 10. WebSocket Protocol

**Endpoint**: `wss://api.nexus.app/ws?token=<access_token>`

### Client → Server Events

| Event | Payload | Keterangan |
|---|---|---|
| `typing.start` | `{ channel_id }` | Timeout otomatis 5 detik di client (FR-PRES-03) |
| `presence.set_status` | `{ status: "online"\|"idle"\|"dnd"\|"invisible" }` | |
| `heartbeat` | `{}` | Setiap 15 detik, me-refresh TTL presence (FR-PRES-02) |

### Server → Client Events

| Event | Payload | Keterangan |
|---|---|---|
| `message.created` | Objek message lengkap | Broadcast synchronous (HLD §3) |
| `message.updated` | Objek message | |
| `message.deleted` | `{ message_id, channel_id }` | |
| `presence.updated` | `{ user_id, status }` | Di-scope ke member yang berbagi workspace (FR-PRES §Best Practice) |
| `typing.updated` | `{ channel_id, user_id }` | |
| `notification.new` | Objek notifikasi | |
| `voice.participant_joined` / `voice.participant_left` | `{ channel_id, user_id }` | |

---

## Ringkasan Keputusan

1. Seluruh endpoint mengikuti envelope, pagination, dan error code konsisten sesuai Playbook §17.
2. Endpoint dengan efek samping kritikal (redeem invite, create DM) **wajib** memakai `Idempotency-Key`.
3. Upload memakai pola **202 Accepted** (bukan 201 langsung final) untuk mencerminkan pemrosesan asynchronous secara jujur di level kontrak API.
4. WebSocket protocol didefinisikan eksplisit (client→server dan server→client event), menjadi kontrak tunggal untuk implementasi frontend & backend.

## Trade-off yang Diterima

- Endpoint list yang didetailkan di dokumen ini adalah **representative sample** per domain (bukan ekshaustif untuk setiap kombinasi CRUD) — pola yang sama (envelope, pagination, error code) berlaku konsisten untuk endpoint yang tidak dijabarkan detail (mis. `PATCH /workspaces/{id}`, `DELETE /channels/{id}`), didokumentasikan lebih lengkap di OpenAPI spec saat implementasi (di luar scope dokumen Markdown ini).

## Risiko Arsitektur

- Endpoint `POST /dm` menggabungkan pengecekan uniqueness (partial unique index) dan blokir dalam satu request — perlu memastikan urutan pengecekan (blokir dulu, baru cek uniqueness) konsisten untuk menghindari kebocoran informasi (mis. mengungkap keberadaan channel DM sebelum memvalidasi blokir).

## Technical Debt yang Sengaja Diterima

- OpenAPI/Swagger spec formal (`.yaml`) belum dibuat di fase ini — dokumen ini menjadi rujukan tekstual, konversi ke OpenAPI dapat dilakukan sebagai task terpisah saat implementasi dimulai (Sprint Planning Phase 10).

## Hal yang Perlu Dikonfirmasi Sebelum Lanjut ke Fase Berikutnya

1. Apakah pola **202 Accepted untuk upload** (bukan 201) dapat diterima sebagai representasi jujur dari pemrosesan asynchronous?
2. Apakah cakupan **representative sample** endpoint (bukan ekshaustif) di dokumen ini sudah cukup, atau ada endpoint kritikal tertentu yang ingin dijabarkan lebih detail?
3. Lanjut ke **Phase 7 — Security Design**?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen pertama Phase 6, mencakup endpoint representative seluruh domain (termasuk DM) dan WebSocket protocol lengkap |
