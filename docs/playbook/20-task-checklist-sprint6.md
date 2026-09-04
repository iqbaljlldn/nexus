# Detailed Task Checklist — Sprint 6
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 6: Upload & Media Processing)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `03-adr.md` (ADR-007 v1.1 — MinIO), `06-srs.md` (§2.7 FR-UP), `07-hld.md` (§2.7, §2.11), `08-lld.md` (§2.10), `09-database-design.md` (§2.5), `10-api-specification.md` (§6), `12-deployment-architecture.md` (§8 Resource Allocation)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen & Prasyarat

Sesuai **Rolling Wave Planning**, dokumen ini mendetailkan **Sprint 6** — bagian terakhir sebelum Presence (Sprint 7) menuntaskan Release 2 penuh (Sprint Planning §2 Release 2: "M6 Upload, 2 minggu").

**Prasyarat**: Sprint 4-5 selesai (`MessageService.Send` sudah menerima `attachment_ids`, di-skip validasinya sementara — Task 7.2.1 Sprint 4 catatan TODO). Sprint 6 menuntaskan TODO tersebut.

**Sprint Goal**: User dapat mengunggah file hingga 1GB (image/video/audio/PDF/ZIP) sebagai lampiran pesan; file diproses asynchronous (thumbnail untuk gambar, ekstraksi metadata untuk video/audio) tanpa memblokir pengiriman pesan; client menerima notifikasi WS saat pemrosesan selesai.

**Catatan arsitektur penting**: Ini adalah domain dengan **profil resource paling berbeda** dari domain lain di proyek (CPU-intensive saat transcoding — HLD §5, Deployment Architecture §8) dan menjadi **kandidat kuat ekstraksi service** (urutan ke-4, HLD §5). Desain di sprint ini harus menjaga Media Processing **loosely coupled** dari `apps/api` utama (komunikasi via Asynq queue, bukan in-process call langsung) agar ekstraksi nanti berfriksi minimal.

---

## EPIC 10: Upload & Media Processing

### Feature 10.1: Migrasi Database

#### Task 10.1.1: Migrasi Tabel `attachments`, `media_processing_jobs`

- **Deskripsi**: DDL sesuai Database Design §2.5.
- **Acceptance Criteria**: `attachments.message_id` nullable (upload dapat mendahului pengiriman pesan final — HLD §2.7); `status` default `pending`.
- **Definition of Done**: `migrate up` sukses; index `idx_attachments_message_id` terverifikasi.
- **Dependency**: Sprint 5 selesai
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis migrasi `attachments` (up & down)
- [ ] Tulis migrasi `media_processing_jobs` (up & down)
- [ ] Verifikasi index & nullable `message_id`

#### Task 10.1.2: sqlc Setup — Domain Attachment

- **Deskripsi**: Query dasar CRUD attachment & job.
- **Acceptance Criteria**: `sqlc generate` sukses.
- **Definition of Done**: Kode ter-generate dapat dipanggil dari test sederhana.
- **Dependency**: Task 10.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Query `CreateAttachment`, `FindAttachmentByID`, `UpdateAttachmentStatus`, `LinkAttachmentToMessage`
- [ ] Query `CreateMediaJob`, `UpdateMediaJobStatus`
- [ ] `sqlc generate`, verifikasi

---

### Feature 10.2: MinIO Integration Setup

#### Task 10.2.1: MinIO Client Wrapper & Bucket Initialization

- **Deskripsi**: Sesuai ADR-007 (MinIO self-hosted, S3-compatible). Buat wrapper generic di `pkg/objectstorage` (Playbook §3.1 — generic, tidak tahu domain).
- **Acceptance Criteria**: Bucket `nexus-attachments` dan `nexus-avatars` dibuat otomatis saat startup bila belum ada (idempotent).
- **Definition of Done**: `docker compose up` → bucket otomatis tersedia di MinIO console; unit test wrapper (upload/download/delete) lolos terhadap MinIO test instance.
- **Dependency**: Task 1.2.2 (Sprint 1 — MinIO service di Docker Compose)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `pkg/objectstorage/minio_client.go` (`minio-go` SDK) — `Put`, `Get`, `Delete`, `PresignedGetURL`
- [ ] Startup hook: `EnsureBucket(bucketName)` idempotent (cek exist dulu, buat bila belum)
- [ ] Konfigurasi env: `NEXUS_API_MINIO_ENDPOINT`, `NEXUS_API_MINIO_ACCESS_KEY`, `NEXUS_API_MINIO_SECRET_KEY` (Playbook §7.3 konvensi, ADR-007 amandemen)
- [ ] Unit test: upload → get → delete round-trip

---

### Feature 10.3: Endpoint Upload (Streaming)

#### Task 10.3.1: Validasi Magic Bytes & Streaming Upload ke Staging

- **Deskripsi**: FR-UP-01/02 — validasi tipe file berdasarkan konten aktual, **streaming** (bukan load seluruh file ke memori — LLD/Learning Roadmap M6 kesalahan umum yang dihindari).
- **Acceptance Criteria**: **RULES.md-consistent**: `io.Copy` streaming langsung dari request body ke MinIO staging, tidak pernah `ioutil.ReadAll` untuk file besar; file > 1GB ditolak sebelum streaming penuh selesai (cek `Content-Length` header dulu, plus limit reader sebagai pengaman kedua).
- **Definition of Done**: Test: upload file valid (magic bytes cocok) → sukses; upload file dipalsukan (ekstensi `.jpg` tapi isi bukan gambar) → ditolak `415`; upload > 1GB → ditolak `413` sebelum streaming selesai penuh (verifikasi memory usage tidak melonjak selama test, minimal secara observasional).
- **Dependency**: Task 10.2.1
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 3.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `POST /attachments` (`multipart/form-data`, streaming reader — bukan `c.FormFile` yang load ke memori/disk penuh dulu tanpa kontrol, evaluasi `MultipartForm` streaming API Gin/`net/http` yang tepat)
- [ ] Deteksi magic bytes dari beberapa byte pertama (`http.DetectContentType`) sebelum melanjutkan copy penuh
- [ ] `io.LimitReader` sebagai pengaman kedua terhadap `Content-Length` yang dipalsukan
- [ ] Streaming langsung ke MinIO bucket staging (`nexus-staging`, TTL 24 jam — FR-UP-05)
- [ ] Response `202 Accepted` dengan `attachment_id`, `status: pending`
- [ ] Test: file valid, file dipalsukan, file > 1GB, verifikasi tidak ada `ReadAll` di code path ini (code review checklist eksplisit)

#### Task 10.3.2: Rate Limiting Upload

- **Deskripsi**: SRS §3.5 — 20 file/jam per user, reuse `pkg/ratelimit` (Sprint 2).
- **Acceptance Criteria**: Percobaan upload ke-21 dalam 1 jam ditolak `429`.
- **Definition of Done**: Integration test: 20 upload sukses, upload ke-21 ditolak.
- **Dependency**: Task 2.8.1 (Sprint 2), Task 10.3.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Terapkan `pkg/ratelimit.Allow` key `upload:{user_id}`, window 1 jam, limit 20
- [ ] Test: batas terpenuhi, upload ke-21 ditolak

---

### Feature 10.4: Asynq Worker Infrastructure

#### Task 10.4.1: Setup Asynq Server dengan Prioritas Queue

- **Deskripsi**: Implementasi persis LLD §2.10 — queue `critical`/`default`/`low` dengan bobot berbeda.
- **Acceptance Criteria**: Worker server terpisah (`cmd/worker/main.go`, binary berbeda dari API server — mempersiapkan ekstraksi service Media di masa depan, HLD §5).
- **Definition of Done**: `docker compose up` menjalankan container worker terpisah; job di queue `critical` diproses lebih dulu dibanding `low` saat keduanya antre bersamaan (test observasional urutan pemrosesan).
- **Dependency**: Task 1.2.2 (Sprint 1 — Redis)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi `cmd/worker/main.go` — entrypoint terpisah dari `cmd/server`
- [ ] Konfigurasi `asynq.Config` persis LLD §2.10 (Concurrency, Queues weight, RetryDelayFunc)
- [ ] Tambahkan ke `docker-compose.yml` sebagai service terpisah (`worker`)
- [ ] Graceful shutdown untuk worker (LLD §3 pola, adaptasi untuk Asynq server)
- [ ] Test: job critical vs low diproses sesuai prioritas

---

### Feature 10.5: Thumbnail Generation Job

#### Task 10.5.1: Task Handler — Generate Thumbnail (Image)

- **Deskripsi**: FR-UP-03 — thumbnail 128px & 512px untuk gambar.
- **Acceptance Criteria**: Job masuk queue `critical` (LLD §2.10 rationale — user menunggu preview segera).
- **Definition of Done**: Test: upload gambar → job diproses → 2 file thumbnail baru ada di bucket final `nexus-attachments` → `media_processing_jobs.status = completed`.
- **Dependency**: Task 10.4.1, Task 10.3.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi task handler `media:thumbnail:generate` (library image Go, mis. `disintegration/imaging` atau `nfnt/resize`)
- [ ] Download dari staging → generate 2 resolusi → upload ke bucket final → hapus dari staging
- [ ] Update `attachments.status = processed`, `media_processing_jobs.status = completed`
- [ ] Publish event internal (in-process untuk Phase A, akan jadi domain event Outbox di Milestone 12) → broadcast WS `media.ThumbnailGenerated` ke client pengunggah
- [ ] Test end-to-end: upload gambar → tunggu job → verifikasi 2 thumbnail ada di MinIO + WS event diterima

#### Task 10.5.2: Task Handler — Ekstraksi Metadata (Video/Audio)

- **Deskripsi**: FR-UP-04 — durasi, resolusi/bitrate, best-effort (tidak memblokir ketersediaan file asli).
- **Acceptance Criteria**: Job masuk queue `default`; kegagalan ekstraksi metadata **tidak** mengubah `attachments.status` menjadi `failed` (file asli tetap `processed`/dapat diunduh — metadata hanya "nice to have").
- **Definition of Done**: Test: upload video → metadata (durasi) tersimpan di `media_processing_jobs.result_metadata`; simulasi `ffprobe` gagal → `attachments.status` tetap `processed`, hanya `media_processing_jobs.status = failed` untuk job spesifik ini.
- **Dependency**: Task 10.4.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Implementasi task handler `media:metadata:extract` (`ffprobe` via `os/exec`, timeout eksplisit)
- [ ] Simpan hasil ke `media_processing_jobs.result_metadata` (JSONB)
- [ ] Pastikan kegagalan job ini TIDAK mempengaruhi `attachments.status` (best-effort, sesuai FR-UP-04)
- [ ] Test: sukses dan gagal (keduanya tidak mem-block ketersediaan file)

---

### Feature 10.6: Attachment Status & Linking ke Message

#### Task 10.6.1: Handler — `GET /attachments/{id}`

- **Deskripsi**: Client polling/cek status attachment.
- **Acceptance Criteria**: Sesuai API Specification §6 — permission baca message terkait (bila sudah linked); bila belum linked (`message_id IS NULL`), hanya uploader yang boleh akses.
- **Definition of Done**: Test: uploader dapat akses attachment miliknya sebelum linked; user lain ditolak.
- **Dependency**: Task 10.1.2
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Handler `GET /attachments/{id}` dengan otorisasi kondisional (linked vs belum)
- [ ] Test: 2 skenario otorisasi di atas

#### Task 10.6.2: Linking Attachment saat Send Message (Tuntaskan TODO Sprint 4)

- **Deskripsi**: Perluas `MessageService.Send` (Sprint 4, Task 7.2.1) — validasi `attachment_ids` benar-benar milik pengirim dan belum ter-link ke pesan lain.
- **Acceptance Criteria**: Kirim pesan dengan `attachment_ids` milik user lain → ditolak; kirim dengan `attachment_id` yang sudah dipakai pesan lain → ditolak (mencegah reuse attachment lintas pesan tanpa sengaja).
- **Definition of Done**: Test: kirim pesan + attachment sukses (attachment ter-link, `message_id` terisi); kirim dengan attachment orang lain → `403`; kirim dengan attachment sudah terpakai → `422`.
- **Dependency**: Task 10.6.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Hapus TODO di `MessageService.Send` (Task 7.2.1 Sprint 4), implementasikan validasi penuh
- [ ] Query `LinkAttachmentToMessage` dengan cek `uploader_id` dan `message_id IS NULL` sebagai kondisi WHERE (atomik, mencegah race condition reuse)
- [ ] Test: 3 skenario di atas

---

### Feature 10.7: Staging Cleanup

#### Task 10.7.1: Scheduled Task — Bersihkan Staging Kedaluwarsa

- **Deskripsi**: FR-UP-05 — TTL staging 24 jam.
- **Acceptance Criteria**: Job terjadwal (Asynq periodic task atau cron sederhana) menghapus file staging yang belum diproses > 24 jam DAN attachment record terkait yang tidak pernah ter-link ke pesan manapun.
- **Definition of Done**: Test: buat attachment staging dengan `created_at` disimulasikan > 24 jam lalu → job cleanup menghapusnya dari MinIO dan database.
- **Dependency**: Task 10.4.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Should

**Subtask & Checklist**:
- [ ] Implementasi scheduled task Asynq (`asynq.PeriodicTaskManager` atau setara) — jalan setiap 1 jam
- [ ] Query attachment staging > 24 jam & `message_id IS NULL`
- [ ] Hapus dari MinIO staging bucket + hapus record database
- [ ] Test: simulasi data kedaluwarsa dibersihkan, data belum kedaluwarsa tidak tersentuh

---

### Feature 10.8: Integration Test End-to-End Sprint 6

#### Task 10.8.1: Skenario Penuh — Upload, Process, Link, Realtime Notify

- **Deskripsi**: Verifikasi Sprint Goal secara menyeluruh, sekaligus menandai **Release 2 selesai** bersama Sprint 7 (Presence) berikutnya.
- **Acceptance Criteria**: Alur: User A upload gambar → status `pending` → job thumbnail diproses (async, worker terpisah) → User A menerima WS event `media.ThumbnailGenerated` → User A kirim pesan dengan `attachment_id` tersebut → pesan ter-link dengan attachment → User B (koneksi WS di channel sama) menerima `message.created` dengan attachment info lengkap.
- **Definition of Done**: Test hijau konsisten 3x run berturut — perhatian khusus karena melibatkan proses asynchronous lintas 2 binary (`server` dan `worker`), test harus menunggu via polling/event dengan timeout eksplisit, bukan `sleep` sembarangan.
- **Dependency**: Seluruh task Epic 10
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 3.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Setup test harness: jalankan `server` + `worker` + dependencies (test container/Docker Compose test profile)
- [ ] Skenario upload → tunggu job selesai (polling `GET /attachments/{id}` dengan timeout, atau tunggu WS event) → verifikasi thumbnail ada
- [ ] Skenario kirim pesan dengan attachment → verifikasi broadcast ke User B mengandung attachment info
- [ ] Jalankan 3x berturut, pastikan tidak flaky
- [ ] Update `docs/AGENTS.md` §7 — Sprint 6 selesai

---

## EPIC 11: Frontend — Upload UI

### Feature 11.1: File Upload dengan Progress

#### Task 11.1.1: Drag-Drop & Pick File di Composer

- **Deskripsi**: Perluas `MessageComposer.vue` (Task 8.2.3 Sprint 4) — area drop file, tombol pilih file, preview sebelum kirim.
- **Acceptance Criteria**: Upload memakai `XMLHttpRequest`/`fetch` dengan progress event (`onUploadProgress` — `ofetch` tidak native mendukung ini, gunakan `axios` khusus untuk endpoint upload atau `XMLHttpRequest` manual) — progress bar menampilkan persentase real, bukan indikator indeterminate saja, penting untuk file besar hingga 1GB (FR-UP-01).
- **Definition of Done**: E2E test: upload gambar kecil → progress 0→100% → attachment_id diterima; upload file > 1GB (simulasi/mock) → error jelas ditampilkan sebelum mengirim penuh (validasi ukuran client-side sebagai UX cepat, backend tetap validasi ulang — RULES.md §3).
- **Dependency**: Task 8.2.3 (Sprint 4)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Area drag-drop + input file di `MessageComposer.vue`
- [ ] Validasi ukuran/tipe file client-side (UX cepat, bukan pengganti validasi backend Task 10.3.1 Sprint 6)
- [ ] Progress bar berbasis event upload real
- [ ] E2E test: upload sukses dengan progress, file terlalu besar ditolak client-side dengan pesan jelas

#### Task 11.1.2: Attachment Preview & Status Polling/WS

- **Deskripsi**: Setelah upload (status `pending`), tampilkan placeholder loading di composer; setelah pemrosesan selesai (`media.ThumbnailGenerated` via WS, Task 10.5.1 backend), tampilkan thumbnail final.
- **Acceptance Criteria**: **Prioritaskan WS event** (bukan polling) untuk update status — polling `GET /attachments/{id}` hanya sebagai fallback bila WS tidak tersedia/terputus saat itu.
- **Definition of Done**: E2E test: upload gambar → placeholder loading tampil → event WS diterima → thumbnail muncul tanpa reload.
- **Dependency**: Task 11.1.1, Task 8.1.2 (Sprint 4 — perluas event router dengan case `media.thumbnail_generated`)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Perluas event router (Task 8.1.2) dengan case `media.ThumbnailGenerated`
- [ ] Komponen `AttachmentPreview.vue` (loading state → final state)
- [ ] Fallback polling (interval wajar, mis. 3 detik, dengan batas maksimal percobaan) bila WS terputus
- [ ] E2E test: alur upload→thumbnail muncul via WS

#### Task 11.1.3: Tampilan Attachment di Pesan Terkirim

- **Deskripsi**: `MessageItem.vue` (Sprint 4) diperluas menampilkan attachment (gambar inline, ikon+nama file untuk PDF/ZIP, audio/video player untuk media).
- **Acceptance Criteria**: Gambar ditampilkan sebagai thumbnail yang dapat diklik untuk memperbesar (lightbox sederhana); tipe file lain ditampilkan sebagai kartu unduhan dengan ukuran file.
- **Definition of Done**: E2E test: kirim pesan dengan attachment gambar → tampil sebagai thumbnail; attachment PDF → tampil sebagai kartu unduhan.
- **Dependency**: Task 11.1.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Perluas `MessageItem.vue` — render attachment sesuai `mime_type`
- [ ] Lightbox sederhana untuk gambar
- [ ] Kartu unduhan untuk tipe file non-gambar
- [ ] E2E test: kedua tipe render

---

### Feature 11.2: Integration Test End-to-End Frontend Sprint 6

#### Task 11.2.1: Skenario Penuh — Mencerminkan Gerbang Backend (Task 10.8.1)

- **Deskripsi**: Versi frontend dari skenario end-to-end backend Sprint 6.
- **Acceptance Criteria**: Upload gambar via UI (drag-drop) → progress → thumbnail muncul via WS → kirim sebagai pesan → user lain (browser context lain) melihat pesan dengan attachment realtime.
- **Definition of Done**: Playwright test hijau konsisten 3x run berturut.
- **Dependency**: Seluruh task Epic 11
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Skenario Playwright penuh (upload → thumbnail → kirim → user lain menerima)
- [ ] Jalankan 3x berturut, pastikan tidak flaky
- [ ] Update `docs/AGENTS.md` §7 — Sprint 6 frontend selesai bersamaan backend

---

## Ringkasan Keputusan

1. Sprint 6 mencakup **1 Epic besar, 8 Feature, 12 task** backend, menuntaskan seluruh scope Upload & Media Processing sesuai FR-UP-01 s.d. 05. *(Direvisi: ditambah Epic 11 — 2 Feature, 4 Task frontend, amandemen retroaktif — drag-drop upload dengan progress real, status update prioritas via WS bukan polling.)*
2. Worker Asynq dijalankan sebagai **binary/container terpisah** (`cmd/worker/main.go`) sejak sprint ini — bukan goroutine di dalam `cmd/server` — secara sengaja mempersiapkan ekstraksi service Media (urutan ke-4, HLD §5) agar friksi ekstraksi nanti minimal.
3. Task 10.3.1 (streaming upload) ditandai **Tinggi** dan diberi porsi waktu terbesar — ini titik paling kritikal untuk NFR (mencegah OOM pada file hingga 1GB dengan banyak upload paralel).
4. Ekstraksi metadata video/audio (Task 10.5.2) didesain **best-effort** — kegagalannya tidak boleh menghalangi ketersediaan file asli, konsisten dengan FR-UP-04.

## Trade-off yang Diterima

- Task 10.7.1 (Staging Cleanup) berprioritas *Should* — dapat digeser ke sprint berikutnya tanpa mengorbankan Sprint Goal inti (upload & processing tetap berfungsi tanpa cleanup, hanya storage staging akan menumpuk lebih lama dari 24 jam yang diharapkan).
- Video transcoding penuh (bukan hanya ekstraksi metadata) **tidak** termasuk scope Sprint 6 — sesuai FR-UP-04, transcoding "ringan" bersifat best-effort dan dapat ditunda ke sprint optimasi berikutnya bila diperlukan, tanpa mengubah kontrak API (attachment tetap dapat diunduh dalam format asli).

## Risiko Arsitektur

- Task 10.3.1 dan 10.8.1 memiliki risiko waktu meleset tertinggi di sprint ini (masing-masing melibatkan streaming I/O presisi dan test lintas-proses asynchronous) — prioritaskan lebih awal dalam sprint untuk memberi buffer waktu.
- Worker terpisah (Task 10.4.1) menambah satu binary baru yang perlu di-maintain paralel dengan `cmd/server` — perlu dipastikan `go.work`/CI mencakup build & test untuk `cmd/worker` juga, bukan hanya `cmd/server` (catatan eksplisit untuk update `ci.yml` bila belum mencakup path ini).

## Technical Debt yang Sengaja Diterima

- Video transcoding penuh (normalisasi codec) ditunda — hanya ekstraksi metadata yang diimplementasikan Sprint 6 (sesuai FR-UP-04, ini bukan penyimpangan tapi memang scope resmi requirement).
- Presigned URL untuk akses langsung MinIO (disebut di Security Design §4 sebagai opsi) belum diimplementasikan — attachment saat ini diakses lewat backend proxy/redirect sederhana; presigned URL dapat ditambahkan sebagai optimasi di Milestone 11 bila diperlukan.

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah Task 10.7.1 (Staging Cleanup, prioritas *Should*) dikerjakan di Sprint 6 ini, atau digeser ke sprint berikutnya?
2. Apakah keputusan menjalankan worker sebagai **binary/container terpisah sejak sekarang** (bukan goroutine dalam `cmd/server`) dapat diterima, mengingat ini menambah sedikit kompleksitas Docker Compose tapi mempersiapkan ekstraksi service di masa depan?
3. Dengan Sprint 6 selesai didetailkan, tersisa **Sprint 7 (Presence & Realtime Signal)** untuk menuntaskan Release 2 sepenuhnya. Lanjut ke Sprint 7, atau berhenti dulu?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 6: 1 Epic, 8 Feature, 12 task, menuntaskan Upload & Media Processing dengan worker terpisah mempersiapkan ekstraksi service masa depan |
| 1.1.0 | Amandemen | Ditambahkan Epic 11: Frontend (drag-drop upload dengan progress, attachment preview via WS, tampilan attachment di pesan) — amandemen retroaktif |
