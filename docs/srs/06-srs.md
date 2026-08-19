# Software Requirement Specification (SRS)
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 2 — Software Requirement Specification
**Versi:** 1.1.0
**Status:** Accepted (direvisi — ambiguitas DM & GeoIP diresolusi)
**Referensi Wajib:** `01-engineering-playbook.md` (v1.0.0), `02-vision-document.md` (v1.0.0), `03-adr.md` (v1.1.0), `04-learning-roadmap.md` (v1.0.0), `05-prd.md` (v1.1.0)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Posisi Dokumen Ini & Catatan Amandemen

SRS menerjemahkan **"apa"** (PRD) menjadi **"seberapa presisi"** — kondisi validasi, batasan numerik, target performa terukur, dan aturan bisnis eksplisit yang akan langsung dirujuk oleh HLD (Phase 3), Database Design (Phase 5), dan API Specification (Phase 6).

**Amandemen resmi berdasarkan keputusan Anda**: Provider SMTP untuk email notification (US-NOTIF-02) ditetapkan sebagai **Brevo** (dahulu Sendinblue). Keputusan ini mengikat §5.9 dan seluruh dokumen berikutnya (HLD, Deployment Architecture, Environment Variable Convention).

---

## 1. Ruang Lingkup

Dokumen ini mencakup Functional Requirements (FR) detail per domain dan Non-Functional Requirements (NFR) presisi, sesuai cakupan PRD v1.0.0. Detail skema database ada di Phase 5, detail kontrak endpoint ada di Phase 6 — SRS ini adalah **jembatan** antara keduanya: cukup presisi untuk memandu desain, namun belum berupa skema/kontrak final.

---

## 2. Functional Requirements

Format: `FR-<Domain>-<Nomor>`. Setiap FR mencakup deskripsi, aturan bisnis, dan kondisi error kunci. Field-level validation lengkap (regex, panjang string, dsb.) diselesaikan di API Specification (Phase 6) — di sini fokus pada **aturan bisnis dan batasan yang memengaruhi desain**.

### 2.1 Identity & Authentication

| ID | Requirement |
|---|---|
| FR-AUTH-01 | Sistem harus mendukung registrasi dengan email unik dan username unik (case-insensitive untuk pengecekan keunikan, mis. `JohnDoe` dan `johndoe` dianggap sama). |
| FR-AUTH-02 | Password di-hash menggunakan Argon2id dengan parameter minimal: memory 19 MiB, iterations 2, parallelism 1 (baseline OWASP) — parameter final divalidasi ulang saat Security Design (Phase 7) berdasarkan kapasitas server nyata. |
| FR-AUTH-03 | Login menerima identifier berupa email **atau** username; sistem menentukan tipe identifier otomatis berdasarkan format (mengandung `@` → email). |
| FR-AUTH-04 | Access token JWT berumur **15 menit**; Refresh token berumur **30 hari**, disimpan di database dengan status `active`/`revoked`, **dirotasi setiap dipakai** (refresh token lama langsung ditandai `revoked` saat dipakai untuk mendapat token baru). |
| FR-AUTH-05 | Sistem harus mendukung "logout dari semua device" — merevoke seluruh refresh token milik user dalam satu operasi. |
| FR-AUTH-06 | Sistem harus mencatat device/sesi aktif (user agent, IP, waktu login terakhir) dan menampilkannya ke user sebagai daftar yang dapat di-revoke individual. |
| FR-AUTH-07 | Percobaan login gagal dibatasi rate limit progresif (lihat §3.5) untuk mitigasi brute-force; pesan error login gagal bersifat generik (tidak membedakan "email tidak ditemukan" vs "password salah"). |
| FR-AUTH-08 | **(Amandemen)** Sistem harus menyediakan logout untuk **sesi saat ini saja** (device yang sedang dipakai), terpisah dari "logout dari semua device" (FR-AUTH-05). Sesi yang di-revoke ditentukan dari refresh token pada request itu sendiri (cookie), **bukan** dari `sessionId` yang harus diketahui client — ini celah yang sebelumnya terlewat di draft awal (hanya ada logout-all dan revoke-by-id). |

### 2.2 Workspace, Role, Permission, Category

| ID | Requirement |
|---|---|
| FR-WS-01 | Satu user dapat memiliki/menjadi member dari banyak workspace (many-to-many melalui tabel membership). |
| FR-WS-02 | Setiap workspace memiliki role default `@everyone` yang tidak dapat dihapus, merepresentasikan permission dasar seluruh member. |
| FR-WS-03 | Permission direpresentasikan sebagai bitmask (`int64`), memungkinkan hingga 63 permission flag berbeda tanpa perubahan skema (lihat ADR Low Level Design lanjutan). |
| FR-WS-04 | Role memiliki urutan hierarki (`position: int`) yang menentukan resolusi konflik saat member memiliki banyak role — role dengan `position` lebih tinggi menang untuk permission yang saling bertentangan secara eksplisit (deny vs allow). |
| FR-WS-05 | Category adalah pengelompokan visual channel; channel dapat berada di luar kategori manapun (`category_id` nullable). |
| FR-WS-06 | Invite code memiliki atribut opsional: `max_uses` (nullable = unlimited), `expires_at` (nullable = tidak kedaluwarsa), dan bersifat idempotent saat redeem (redeem ganda oleh user yang sama tidak menghasilkan duplikasi membership, hanya mengembalikan status membership yang sudah ada). |
| FR-WS-07 | Permission override tingkat channel disimpan terpisah dari role default, mengikuti urutan resolusi: **Channel-specific member override → Channel-specific role override → Role default (berdasar `position`) → `@everyone`.** |
| FR-WS-08 | Kapasitas member per workspace mendukung hingga **100.000 member** (NFR) — implikasi: query member list wajib cursor-based pagination sejak awal, tidak boleh mengasumsikan "load semua member sekaligus" di level manapun (backend maupun frontend). |

### 2.3 Channel

| ID | Requirement |
|---|---|
| FR-CH-01 | Channel memiliki tipe tetap saat dibuat (`text`, `voice`, `video`, `forum`, `announcement`) — tipe tidak dapat diubah setelah dibuat (mengubah tipe channel adalah operasi destruktif yang secara sengaja tidak didukung, mendorong user membuat channel baru). |
| FR-CH-02 | Channel tipe `announcement` hanya dapat diposting oleh role dengan permission `MANAGE_ANNOUNCEMENT`, namun dapat dibaca oleh seluruh role yang memiliki akses baca ke channel tersebut. |
| FR-CH-03 | Channel tipe `voice`/`video` menampilkan daftar partisipan aktif secara realtime (bersumber dari state LiveKit room, disinkronkan ke client via WebSocket event, bukan polling). |
| FR-CH-04 | Channel tipe `forum` mewajibkan setiap post baru membentuk thread baru (bukan pesan linear seperti channel `text`). |
| FR-CH-05 | Sistem mendukung **unlimited channel per workspace** (NFR) — daftar channel di sidebar client wajib memakai virtual scrolling (Vue Virtual Scroller, sesuai stack frontend) untuk workspace dengan channel sangat banyak. |

### 2.4 Messaging

| ID | Requirement |
|---|---|
| FR-MSG-01 | Panjang pesan teks maksimal **4000 karakter** (nilai final dikonfirmasi di API Specification, dipakai sebagai baseline perencanaan skema). |
| FR-MSG-02 | Markdown yang didukung minimal: bold, italic, strikethrough, inline code, code block, blockquote, link. Rendering dilakukan di **frontend** (sanitasi HTML output wajib untuk mencegah XSS — lihat §3.9). |
| FR-MSG-03 | Reply menyimpan referensi `reply_to_message_id` (nullable); bila pesan yang direply telah dihapus (soft delete), UI tetap menampilkan indikator "membalas pesan yang telah dihapus" tanpa error. |
| FR-MSG-04 | Thread merupakan entitas turunan dari pesan induk, dengan `channel_id` sendiri secara internal (thread diperlakukan sebagai sub-channel sementara/permanen tergantung tipe channel induk) — detail model data di Database Design. |
| FR-MSG-05 | Mention mendukung mention user individual (`@username`), mention role (`@role_name`), dan mention `@everyone`/`@here` (dengan permission khusus `MENTION_EVERYONE` untuk mencegah abuse). |
| FR-MSG-06 | Reaction disimpan sebagai kombinasi unik `(message_id, user_id, emoji)` — satu user tidak dapat memberi reaksi emoji yang sama dua kali pada pesan yang sama. |
| FR-MSG-07 | Edit pesan hanya diizinkan oleh penulis asli; field `edited_at` (nullable, `timestamptz`) diisi saat edit pertama kali terjadi. |
| FR-MSG-08 | Delete pesan adalah **soft delete** (`deleted_at` diisi); pesan yang di-soft-delete tidak muncul di response API normal namun tetap ada di database untuk kebutuhan audit/moderasi. Moderator dengan permission `MANAGE_MESSAGES` dapat menghapus pesan siapapun di channel yang mereka kelola. |
| FR-MSG-09 | Sistem mendukung Optimistic Locking pada operasi edit pesan (kolom `version`) untuk mencegah race condition saat dua client mengedit pesan yang sama secara bersamaan — konflik dikembalikan sebagai HTTP 409 dengan kode `OPTIMISTIC_LOCK_CONFLICT` (sesuai §16.2 Engineering Playbook). |
| FR-MSG-10 | History pesan diambil dengan cursor-based pagination (§17.2 Playbook), default page size 50, maksimal 100 per request. |

### 2.5 Presence & Realtime Signal

| ID | Requirement |
|---|---|
| FR-PRES-01 | Status presence: `online`, `idle`, `dnd`, `invisible`, `offline`. Status `invisible` secara internal disimpan sebagai status nyata namun ditampilkan sebagai `offline` ke user lain. |
| FR-PRES-02 | Presence memakai TTL Redis 30 detik, di-refresh setiap heartbeat client (interval heartbeat: 15 detik, memberi margin aman 2x sebelum TTL habis). |
| FR-PRES-03 | Typing indicator memiliki timeout otomatis 5 detik di sisi client (bila tidak ada event "masih mengetik" baru, indikator hilang) — server tidak perlu menyimpan state typing secara persisten, cukup broadcast event dengan TTL implisit di sisi client. |
| FR-PRES-04 | Read receipt disimpan per-user per-channel (`last_read_message_id`), bukan per-pesan individual (menghindari eksplosi row untuk channel besar) — detail skema di Database Design. |

### 2.6 Notification

| ID | Requirement |
|---|---|
| FR-NOTIF-01 | Notifikasi realtime dikirim via WebSocket untuk: mention, reply langsung ke pesan user, direct message (bila fitur DM personal dipertimbangkan — dicatat sebagai klarifikasi di §6). |
| FR-NOTIF-02 | Notifikasi email dikirim untuk user yang **offline lebih dari 5 menit** dan menerima mention/DM, memakai **Brevo** sebagai SMTP/email API provider (amandemen §0), dieksekusi asynchronous via Asynq task dengan retry (maksimal 3x, exponential backoff). |
| FR-NOTIF-03 | Preferensi notifikasi per channel: `all`, `mentions_only`, `none` (mute). Preferensi per workspace sebagai default yang di-override oleh preferensi per channel. |
| FR-NOTIF-04 | Notification email dibatasi rate maksimal 1 email ringkasan per user per 10 menit (batching mention dalam window tersebut menjadi satu email) — mencegah spam email ke user yang sedang di-mention berkali-kali dalam waktu singkat. |

### 2.7 Upload

| ID | Requirement |
|---|---|
| FR-UP-01 | Ukuran file maksimal per upload: **1 GB**, divalidasi di level aplikasi (bukan hanya reverse proxy) untuk memberi pesan error yang jelas. |
| FR-UP-02 | Tipe file didukung: image (`jpg`, `png`, `gif`, `webp`), video (`mp4`, `webm`), audio (`mp3`, `wav`, `ogg`), `pdf`, `zip` — validasi berdasarkan magic bytes, bukan ekstensi. |
| FR-UP-03 | Upload gambar memicu generate thumbnail (beberapa resolusi: 128px, 512px) secara asynchronous via Asynq worker. |
| FR-UP-04 | Upload video/audio memicu ekstraksi metadata (durasi, resolusi/bitrate) secara asynchronous; transcoding penuh (mis. normalisasi codec) bersifat **best-effort**, tidak memblokir ketersediaan file asli untuk diunduh/diputar. |
| FR-UP-05 | File disimpan sementara di staging area sebelum diproses, dipindahkan ke bucket final MinIO (`nexus-attachments`) setelah pemrosesan selesai, staging dibersihkan otomatis (scheduled Asynq task, TTL staging 24 jam). |

### 2.8 Search

| ID | Requirement |
|---|---|
| FR-SRCH-01 | Full-text search pesan memakai PostgreSQL `tsvector`/`tsquery` dengan konfigurasi bahasa (`simple` sebagai default netral bahasa, dipertimbangkan ulang di Database Design bila mayoritas konten berbahasa Indonesia — `indonesian` text search configuration). |
| FR-SRCH-02 | Search dibatasi scope: hasil pencarian pesan hanya mencakup channel yang dapat diakses user (permission check diterapkan **sebelum** query full-text, bukan filter setelahnya, untuk mencegah kebocoran informasi lewat timing/hasil parsial). |
| FR-SRCH-03 | Search user/channel/server memakai `ILIKE`/trigram index (`pg_trgm`) untuk pencarian substring yang toleran typo ringan. |

### 2.9 Direct Message (DM)

> **Amandemen (v1.1.0)**: Menyelesaikan ambiguitas §6 (versi draft) — DM dikonfirmasi masuk scope resmi.

| ID | Requirement |
|---|---|
| FR-DM-01 | DM dimodelkan sebagai Channel dengan `type = 'dm'` dan `workspace_id NULL` — seluruh logika Messaging (FR-MSG-*) berlaku identik pada channel tipe ini, tanpa duplikasi kode/skema. |
| FR-DM-02 | Channel `dm` 1-on-1 bersifat unik per pasangan user (tidak ada duplikasi channel DM untuk pasangan user yang sama) — dicapai lewat constraint unik pada kombinasi partisipan yang di-sort secara deterministik sebelum disimpan. |
| FR-DM-03 | Grup DM dibatasi maksimal **10 partisipan**; berbeda dari Workspace yang mendukung hingga 100.000 member — pembatasan ini membedakan tujuan desain (DM = privat/kecil, Workspace = komunitas/besar). |
| FR-DM-04 | User dapat memblokir user lain (`user_blocks` table: `blocker_id`, `blocked_id`); user yang diblokir tidak dapat membuat channel `dm` baru dengan blocker maupun mengirim pesan ke channel `dm` yang sudah ada bersama blocker (dicek di layer otorisasi setiap pengiriman pesan, bukan hanya saat pembuatan channel). |
| FR-DM-05 | Permission model Workspace (Role/Permission bitmask, FR-WS-*) **tidak berlaku** untuk channel `dm` — otorisasi DM murni berbasis keanggotaan partisipan (member of channel) dan status block, lebih sederhana dibanding model Workspace. |

### 2.10 Admin Panel

| ID | Requirement |
|---|---|
| FR-ADM-01 | Dashboard admin menampilkan metrik: total user aktif (harian/mingguan), total workspace, error rate 5xx (bersumber dari Prometheus), resource usage dasar (CPU/memory per service). |
| FR-ADM-02 | Admin dapat men-suspend (nonaktifkan sementara, dapat dipulihkan) atau men-ban (permanen) user tingkat platform; aksi ini dicatat di audit log dengan `actor_id`, `target_id`, `reason`, `timestamp`. |
| FR-ADM-03 | Audit log admin bersifat **append-only** (tidak dapat diedit/dihapus lewat aplikasi), mencakup minimal: perubahan role/permission, penghapusan workspace, suspend/ban user, perubahan konfigurasi sistem sensitif. |

---

## 3. Non-Functional Requirements (Detail)

### 3.1 Target Response Time

| Kategori Operasi | Target p95 | Target p99 | Catatan |
|---|---|---|---|
| Read sederhana (get channel list, get profile) | < 100 ms | < 250 ms | Diukur di level aplikasi (excl. network client) |
| Read kompleks (search, message history dengan filter) | < 300 ms | < 700 ms | Bergantung index yang tepat (Database Design Phase 5) |
| Write sederhana (kirim pesan, reaction) | < 150 ms | < 400 ms | Termasuk penulisan outbox event (Milestone 12) |
| Upload file (di luar durasi transfer file itu sendiri) | < 200 ms untuk inisiasi + enqueue | N/A | Pemrosesan asynchronous tidak dihitung dalam response time inisial |
| WebSocket message delivery (server-to-client broadcast) | < 100 ms dari publish ke delivery pada koneksi sehat | < 300 ms | Diukur dalam kondisi single-instance; multi-instance broadcast (Redis Streams) dievaluasi ulang saat Milestone 12/13 |

### 3.2 Scalability Strategy

- **Vertical + Horizontal Read Scaling**: PostgreSQL primary-replica (read replica) dipertimbangkan saat Phase C/D bila read load menjadi bottleneck terukur (bukan dari awal — YAGNI).
- **Stateless Application Layer**: seluruh instance API tidak menyimpan state lokal yang tidak dapat direkonstruksi (session di JWT, presence di Redis) — memungkinkan horizontal scaling instance API tanpa sticky session, **kecuali** koneksi WebSocket yang secara inheren stateful per-koneksi (memerlukan strategi khusus, dibahas di HLD: Redis Pub/Sub sebagai broadcast layer lintas instance, atau sticky session di Traefik sebagai alternatif yang lebih sederhana namun kurang scalable — trade-off dianalisis di HLD Phase 3).
- **Database Partitioning**: tabel `messages` dipertimbangkan untuk **table partitioning** by `created_at` (range partitioning bulanan) begitu volume data mendekati puluhan juta baris — detail di Database Design Phase 5, tidak diaktifkan sejak awal (YAGNI, diaktifkan saat data riil mendekati threshold).

### 3.3 Availability Target

- Target awal (Phase A-B, deployment Docker Compose single VPS): **99.0%** uptime bulanan (~7.2 jam downtime/bulan dapat diterima, termasuk maintenance window terjadwal) — realistis untuk skala proyek belajar dengan infrastruktur terbatas.
- Target setelah Blue-Green Deployment matang (Phase C, Deployment Tahap 3): **99.5%** (~3.6 jam/bulan), downtime terjadwal (maintenance) tidak lagi dihitung sebagai downtime karena zero-downtime deployment.
- Target akhir (Phase D, Multi-Node): **99.9%** (~43 menit/bulan) sebagai aspirasi, bukan SLA kontraktual (proyek belajar, bukan komersial).

### 3.4 Disaster Recovery & Backup Strategy

- **RPO (Recovery Point Objective)**: maksimal kehilangan data 1 jam — backup PostgreSQL (`pg_basebackup` + WAL archiving) dilakukan kontinu, snapshot penuh harian.
- **RTO (Recovery Time Objective)**: maksimal 4 jam untuk restore penuh dari backup ke environment baru (target awal, diperketat seiring kematangan automasi).
- Backup MinIO: replikasi bucket ke storage sekunder (disk terpisah/off-site) terjadwal harian.
- **Wajib**: restore drill dilakukan berkala (minimal setiap milestone besar/quarterly) sesuai Learning Roadmap Milestone 18 — backup yang tidak pernah diuji restore dianggap tidak valid sebagai backup.

### 3.5 Rate Limiting

| Kategori | Batas | Window |
|---|---|---|
| Login attempt (per identifier) | 5 percobaan | 15 menit, lockout progresif (5 menit → 15 menit → 1 jam) |
| API umum (per user, authenticated) | 100 request | per menit |
| API umum (per IP, unauthenticated — mis. register) | 20 request | per jam |
| Kirim pesan (per user per channel) | 10 pesan | per 10 detik (mitigasi spam channel) |
| Upload file (per user) | 20 file | per jam |
| Search (per user) | 30 query | per menit (search lebih mahal secara komputasi) |

Implementasi: sliding window counter di Redis (memakai struktur data sorted set atau algoritma token bucket — detail teknis di Low Level Design Phase 4), diterapkan berlapis: Traefik middleware (proteksi kasar tingkat IP) + aplikasi (proteksi presisi tingkat user/aksi).

### 3.6 Audit Log

- Seluruh aksi sensitif berikut **wajib** tercatat: perubahan role/permission, kick/ban member, hapus channel, hapus workspace, suspend/ban user platform, perubahan invite code, login dari device baru.
- Format log audit: `actor_id`, `action`, `target_type`, `target_id`, `metadata (JSONB)`, `ip_address`, `created_at` — disimpan di tabel terpisah `audit_logs`, append-only, tanpa `updated_at`/`deleted_at` (secara desain tidak dapat diubah).
- Retensi audit log: minimal 1 tahun (dapat diarsipkan ke storage dingin setelah periode aktif, detail di Deployment Architecture).

### 3.7 Device Management

- Setiap login berhasil mencatat entri sesi baru (device fingerprint dasar: user agent + IP), ditampilkan ke user di halaman "Sesi Aktif".
- User dapat me-revoke sesi individual atau seluruh sesi selain sesi saat ini.
- Login dari device/lokasi baru (heuristik: IP berbeda signifikan dari histori) memicu notifikasi email keamanan (via Brevo).

### 3.8 Enkripsi

- **In-transit**: seluruh komunikasi (REST, WebSocket, gRPC internal Phase D) wajib TLS — TLS termination di Traefik dengan sertifikat otomatis (Let's Encrypt, sesuai ADR-008).
- **At-rest**: password di-hash (bukan dienkripsi — Argon2id, satu arah); refresh token disimpan dalam bentuk hash (bukan plaintext) di database, hanya dibandingkan hash-nya saat validasi; data sensitif lain (mis. jika ada integrasi payment di masa depan — saat ini tidak ada dalam scope) akan memakai enkripsi kolom spesifik bila diperlukan.
- Koneksi ke MinIO dan PostgreSQL antar service internal (Phase D) melalui jaringan privat/VPC-like (Docker network internal atau Kubernetes NetworkPolicy), tidak diekspos publik.

### 3.9 Content Security Policy (CSP) & Proteksi Frontend

- CSP header membatasi source script/style/img hanya ke domain aplikasi sendiri dan domain MinIO (untuk attachment) — mencegah XSS lewat injeksi script eksternal.
- Rendering markdown pesan **wajib** melalui sanitizer (mis. DOMPurify di frontend) sebelum di-render sebagai HTML — mencegah stored XSS lewat konten pesan.

### 3.10 CSRF Protection

- Karena autentikasi memakai JWT di header `Authorization` (bukan cookie otomatis dikirim browser untuk request API), risiko CSRF klasik pada endpoint API berkurang signifikan. Namun, bila refresh token disimpan sebagai **HttpOnly cookie** (opsi lebih aman dibanding localStorage untuk mitigasi XSS-based token theft — keputusan final di Security Design Phase 7), maka endpoint refresh token **wajib** dilindungi CSRF token (double-submit cookie pattern) karena endpoint tersebut memang dipicu otomatis oleh cookie.

### 3.11 Spam Protection

- Rate limiting kirim pesan (§3.5).
- Deteksi pesan berulang identik dalam window singkat (mis. > 3 pesan identik dalam 10 detik dari user yang sama) memicu soft-block sementara (throttle, bukan ban otomatis).
- CAPTCHA (mekanisme final ditentukan di Security Design) dipertimbangkan pada endpoint registrasi bila terdeteksi pola registrasi otomatis (bot) — tidak diaktifkan default di awal (YAGNI), diaktifkan reaktif bila diperlukan.

---

## 4. Data Requirements (Ringkasan — Detail Penuh di Phase 5)

- Seluruh entitas menggunakan UUID v7 sebagai primary key (ADR-final, Playbook §7.6).
- Retensi soft-deleted data (pesan, channel) minimal 30 hari sebelum dipertimbangkan hard-delete permanen (kebijakan final di Security/Compliance, dicatat sebagai item konfirmasi bila relevan dengan regulasi privasi tertentu).

---

## 5. External Interface Requirements

| Sistem Eksternal | Kebutuhan Interface |
|---|---|
| PostgreSQL | Koneksi via connection pool (`pgxpool` atau setara), TLS untuk koneksi non-lokal |
| Redis | Cache, Presence store, Rate limiting counter, Redis Streams (event backbone) |
| MinIO | S3-compatible API (`minio-go`), bucket terpisah per keperluan |
| LiveKit | Server SDK Go untuk token generation & room management, Client SDK JS untuk frontend |
| **Brevo** | Email API (transactional email) untuk notifikasi email dan security alert (login device baru) — endpoint dan autentikasi API key dikonfigurasi via environment variable `NEXUS_API_BREVO_API_KEY` |
| Traefik | Reverse proxy, TLS termination, dynamic routing berbasis label |

---

## 6. Klarifikasi — Status Resolusi (v1.1.0)

Kedua ambiguitas yang dilaporkan di draft v1.0.0 telah **diresolusi**:

1. ✅ **Direct Message (DM) personal**: dikonfirmasi **masuk scope resmi**. Detail requirement lengkap ada di §2.9 (FR-DM-01 s.d. FR-DM-05), selaras dengan PRD v1.1.0 §6.9.
2. ✅ **Threshold device/lokasi baru**: dikonfirmasi memakai **heuristik IP sederhana tanpa GeoIP** (sesuai rekomendasi YAGNI) — tidak ada dependency baru yang ditambahkan ke sistem.

---

## Ringkasan Keputusan

1. Seluruh target performa (response time, availability, RPO/RTO) ditetapkan dengan angka konkret dan terukur, bukan pernyataan kualitatif — menjadi baseline objektif untuk Performance Design dan Load Testing (Milestone 11, 18).
2. Rate limiting dirancang berlapis (Traefik + aplikasi) dengan angka spesifik per kategori operasi.
3. **Brevo** dikonfirmasi sebagai provider email, mengikat environment variable convention dan External Interface Requirements.
4. Permission direpresentasikan sebagai bitmask `int64` — keputusan teknis yang memengaruhi langsung Database Design (Phase 5).

## Trade-off yang Diterima

- Availability target dibuat bertahap (99.0% → 99.5% → 99.9%) mengikuti evolusi deployment, bukan target tunggal sejak awal — realistis untuk infrastruktur proyek belajar.
- CAPTCHA dan GeoIP lookup tidak diaktifkan sejak awal (YAGNI) — berisiko menjadi celah spam/abuse sebelum diaktifkan reaktif.

## Risiko Arsitektur

- Broadcast WebSocket lintas instance (multi-node, Phase D) belum final desainnya di dokumen ini — dicatat sebagai keputusan yang wajib dituntaskan di HLD Phase 3 (opsi: Redis Pub/Sub/Streams vs sticky session Traefik).
- Model DM sebagai varian Channel (`workspace_id` nullable) menambah satu kondisi percabangan (`IF workspace_id IS NULL`) di banyak query/otorisasi Channel — perlu didisiplinkan di Low Level Design agar tidak menjadi sumber bug tersembunyi (mis. permission check Workspace yang tidak sengaja tetap dijalankan untuk channel `dm`).

## Technical Debt yang Sengaja Diterima

- Parameter final Argon2id (FR-AUTH-02) masih baseline OWASP, belum divalidasi dengan benchmark kapasitas server nyata — akan diuji ulang di Milestone 11 (Optimization)/Security Design.
- Kebijakan retensi hard-delete data (§4) belum final — dicatat sebagai item yang perlu keputusan eksplisit bila ada pertimbangan regulasi privasi di masa depan.

## Hal yang Perlu Dikonfirmasi Sebelum Lanjut ke Fase Berikutnya

Seluruh ambiguitas telah diresolusi dan target NFR diterima sebagai baseline pembelajaran. Satu-satunya hal yang tersisa:

1. Lanjut ke **Phase 3 — High Level Design (HLD)**?

---

## Changelog

| Versi | Tanggal | Perubahan |
| 1.2.0 | Amandemen | Ditambahkan FR-AUTH-08: endpoint logout sesi-saat-ini-saja (terpisah dari logout-all), menutup celah desain yang ditemukan pasca-Sprint 2 |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen pertama Phase 2. Brevo dikonfirmasi sebagai SMTP/email provider (amandemen dari keputusan Anda). 2 ambiguitas ditemukan dan dilaporkan (§6) untuk konfirmasi sebelum HLD. |
| 1.1.0 | Revisi | Ambiguitas diresolusi: (1) DM dikonfirmasi masuk scope, ditambahkan §2.9 FR-DM-01 s.d. FR-DM-05; (2) threshold device/lokasi baru dikonfirmasi heuristik IP sederhana tanpa GeoIP. |
