# Engineering Playbook
## Project: "Nexus" — Discord-like Realtime Platform (Project-Based Learning)

**Dokumen:** Phase 0 — Engineering Playbook
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Klasifikasi:** Internal Engineering Standard (Source of Truth)

---

## 0. Cara Membaca Dokumen Ini

Dokumen ini adalah **konstitusi teknis** proyek. Semua dokumen di fase berikutnya (Vision, ADR, PRD, SRS, HLD, LLD, dst.) **wajib tunduk** pada aturan di sini kecuali ada revisi eksplisit yang disetujui.

Prinsip penulisan playbook ini:

- Setiap aturan disertai **alasan (rationale)**, bukan sekadar perintah. Anda harus paham *mengapa*, bukan hanya *apa*.
- Setiap aturan punya **kondisi pelanggaran yang dapat diterima** (kapan boleh menyimpang) dan **kondisi yang memicu revisi**.
- Playbook ini akan berevolusi. Versi tercatat di header. Perubahan signifikan dicatat di bagian **Changelog** paling bawah.

Karena proyek ini akan berevolusi dari **Modular Monolith → Event-Driven Monolith → Hybrid → Microservices**, playbook ini ditulis agar valid di seluruh fase — bagian yang spesifik untuk fase tertentu ditandai eksplisit.

---

## 1. Repository Strategy: Monorepo

### 1.1 Keputusan

Proyek ini menggunakan **Monorepo tunggal** bernama `nexus` yang berisi seluruh kode: backend (Go), frontend (Nuxt), infrastruktur (IaC/compose/k8s), dan dokumentasi.

### 1.2 Alasan (Rationale)

| Faktor | Monorepo | Polyrepo | Keputusan |
|---|---|---|---|
| Learning value untuk memahami evolusi monolith→microservices | Tinggi — semua boundary domain terlihat dalam satu tempat, memudahkan refactor ekstraksi service | Rendah di awal — sudah terpisah sejak hari pertama, kehilangan pengalaman "merasakan sakitnya" distributed monolith | **Monorepo** |
| Atomic commit lintas domain (mis. ubah kontrak event dan consumer sekaligus) | Mudah, satu PR, satu review | Butuh koordinasi multi-repo, versioning package, sinkronisasi rilis | **Monorepo** |
| Overhead tooling di awal (single developer/small team) | Rendah — satu CI pipeline, satu dependency graph | Tinggi — perlu package registry privat, versi lintas repo | **Monorepo** |
| Ukuran tim | Proyek ini solo/learning, bukan tim besar dengan banyak service owner | Polyrepo unggul saat tim > 50 engineer dengan ownership service yang jelas | **Monorepo** |
| Kapan Polyrepo lebih unggul | — | Saat organisasi sudah masuk Phase D (Full Microservices) dengan tim terpisah per service, deployment cadence berbeda, dan kebutuhan isolasi akses repo | Dicatat sebagai **kemungkinan migrasi di masa depan**, bukan keputusan sekarang |

**Prinsip yang dipakai:** *Simplicity over Cleverness* dan *YAGNI*. Kompleksitas polyrepo (versioning lintas repo, private registry, release train) tidak memberi manfaat pada tahap belajar ini. Monorepo tidak menghalangi ekstraksi ke microservices — monorepo hanya soal *tempat penyimpanan kode*, bukan *arsitektur runtime*. Ini adalah kesalahan pemahaman umum yang harus dihindari: **Monorepo vs Polyrepo adalah keputusan source-control, Monolith vs Microservices adalah keputusan arsitektur runtime. Keduanya independen.**

### 1.3 Kapan Migrasi ke Polyrepo Dipertimbangkan

- Saat masing-masing service di Phase D memiliki tim terpisah dengan kebutuhan kontrol akses repo yang berbeda.
- Saat ukuran repo (git clone time, CI time) mulai jadi bottleneck nyata (indikator: `git clone` > 2 menit, CI end-to-end > 20 menit walau sudah pakai path-based trigger).
- **Tidak akan dilakukan hanya karena "microservices harus polyrepo"** — itu mitos. Google, Meta, dan banyak perusahaan microservices skala besar tetap memakai monorepo.

### 1.4 Struktur Direktori Top-Level

```
nexus/
├── apps/
│   ├── api/                     # Modular monolith Go (Phase A-C), lalu jadi gateway/BFF di Phase D
│   └── web/                     # Nuxt 4 frontend
├── services/                    # Kosong di Phase A, terisi progresif saat ekstraksi (Phase C-D)
│   ├── identity-svc/
│   ├── notification-svc/
│   └── ...
├── pkg/                         # Shared Go packages (lintas modul/service, TIDAK boleh bergantung pada domain apapun)
│   ├── logger/
│   ├── errors/
│   ├── validator/
│   ├── httpresponse/
│   ├── telemetry/
│   ├── eventbus/
│   └── config/
├── proto/                       # Kontrak gRPC & event schema (dipakai lintas service saat Phase C/D)
│   ├── events/
│   └── grpc/
├── migrations/                  # SQL migration per domain/service
├── deployments/
│   ├── docker-compose/
│   ├── traefik/
│   └── k8s/                     # Disiapkan sejak awal sbg dokumentasi, aktif dipakai di Phase D
├── docs/
│   ├── adr/
│   ├── playbook/
│   ├── prd/
│   ├── srs/
│   └── diagrams/
├── scripts/                     # Automasi lokal (migration runner, seed, codegen)
├── .github/
│   └── workflows/
├── go.work                      # Go workspace, menyatukan apps/api + services/* + pkg/*
├── package.json                 # Root workspace untuk frontend tooling (Biome, Husky, Commitlint)
└── README.md
```

**Rationale struktur:** `apps/` vs `services/` dipisah secara sengaja agar **niat evolusi arsitektur terlihat secara struktural**. Selama Phase A-B, `services/` kosong (atau berisi placeholder). Saat sebuah modul diekstraksi (Phase C), ia **berpindah folder** dari `apps/api/internal/module/xxx` ke `services/xxx-svc/`, bukan disalin. Perpindahan folder ini sendiri menjadi *artifact pembelajaran* yang bisa dilihat di git history.

`pkg/` menerapkan aturan ketat: **tidak boleh mengimpor apapun dari `apps/` atau `services/`**. Ini mencegah *circular dependency* dan menjaga `pkg/` sebagai genuinely shared kernel. Pelanggaran aturan ini dicek otomatis di CI (lihat §7).

---

## 2. Go Workspace & Dependency Management

### 2.1 Keputusan

- Backend menggunakan **Go 1.25.2** dengan **Go Workspace (`go.work`)**, bukan satu `go.mod` raksasa, dan bukan pula banyak module terpisah tanpa workspace.
- Setiap "unit yang bisa berdiri sendiri" (apps/api, tiap service di `services/`, tiap package di `pkg/` yang cukup besar) punya `go.mod` sendiri.
- Frontend menggunakan **pnpm workspace** untuk konsistensi lockfile dan hoisting yang lebih predictable dibanding npm/yarn (walau tidak eksplisit diminta, ini konsisten dengan filosofi *Convention over Configuration*; jika Anda ingin tetap npm, ini dapat direvisi — dicatat sebagai poin konfirmasi di akhir dokumen).

### 2.2 Rationale

| Opsi | Kelebihan | Kekurangan | Keputusan |
|---|---|---|---|
| Single `go.mod` untuk seluruh monorepo | Simpel di awal | Saat ekstraksi service di Phase C/D, service baru **tetap terikat** ke dependency graph monolith — bertentangan dengan tujuan pembelajaran independent deployability | Ditolak |
| Multi-module tanpa `go.work` | Isolasi penuh antar module | Development lintas module jadi menyakitkan (harus `replace` manual di setiap `go.mod`, mudah lupa revert sebelum commit) | Ditolak |
| **Go Workspace (`go.work`)** | Isolasi dependency antar module TETAP terjaga (masing-masing punya go.mod sendiri, sehingga saat diekstraksi ke service terpisah tidak ada "kejutan" dependency tersembunyi), namun development lokal tetap mulus tanpa `replace` manual | Sedikit tambahan konsep untuk dipelajari (tapi ini justru bagian dari learning objective) | **Dipilih** |

`go.work` **tidak di-commit ke git** dalam mode strict (opsional per developer) **KECUALI** kita memutuskan semua kontributor memakai environment identik — untuk proyek belajar solo, `go.work` **di-commit** agar reproducible, dengan catatan di README bahwa file ini murni alat development lokal dan tidak memengaruhi build produksi (build produksi tiap service dilakukan per-module dengan `go build` di dalam direktori module masing-masing, sehingga image Docker service benar-benar hanya membawa dependency miliknya sendiri).

### 2.3 Aturan Dependency Antar Modul Internal

1. `pkg/*` → tidak boleh depend ke `apps/*` atau `services/*`. (arah panah searah)
2. `apps/api/internal/<domain>` → tidak boleh depend langsung ke `apps/api/internal/<domain-lain>` melalui internal package private; komunikasi antar domain HARUS lewat **interface yang didefinisikan di domain pemilik** (lihat LLD Phase 4 untuk detail port/adapter). Ini adalah pencegahan dini terhadap **Distributed Monolith** — batas domain harus sudah tegas sejak dalam monolith, sebelum diekstraksi.
3. `services/*` satu sama lain **tidak boleh** saling import Go package secara langsung. Komunikasi wajib lewat REST/gRPC/Event (lihat HLD).

**Enforcement:** dicek dengan `depguard` (bagian dari `golangci-lint`) yang mendefinisikan allow-list import per package. Pelanggaran = CI merah, PR tidak bisa merge (lihat §12).

---

## 3. Shared Package Strategy

### 3.1 Klasifikasi Package

| Lokasi | Isi | Aturan |
|---|---|---|
| `pkg/` | Kode generik, tidak tahu apa-apa tentang domain bisnis (logger, error wrapper, HTTP response envelope, telemetry setup, config loader, event bus abstraction) | Boleh dipakai siapapun. Tidak boleh berisi tipe domain (`User`, `Message`, dll). |
| `apps/api/internal/shared/` (atau `services/<svc>/internal/shared/`) | Kode yang spesifik untuk aplikasi tersebut tapi dipakai lintas domain di dalamnya (mis. middleware auth, DB connection pool) | Hanya boleh dipakai di dalam module yang sama. |
| `apps/api/internal/<domain>/` | Kode domain spesifik | Tidak boleh diimpor module lain secara langsung. |

### 3.2 Rationale

Kesalahan umum dalam belajar DDD/Modular Monolith adalah menaruh "apa saja yang dipakai berulang" ke dalam satu folder `common/` atau `utils/` besar yang lama-lama menjadi **God Package** — semua orang import ini, sehingga secara efektif semua domain jadi coupled satu sama lain lewat backpintu.

**Aturan keras:** jika sebuah kode dipakai oleh 2 domain tapi mengandung *domain knowledge* (bukan generic utility), maka itu **bukan kandidat shared package** — itu adalah sinyal bahwa boundary domain Anda salah, atau butuh domain event/domain service baru untuk menjembatani, bukan shared code.

### 3.3 Kapan Membuat Shared Package Baru

Checklist sebelum membuat package baru di `pkg/`:

- [ ] Apakah kode ini benar-benar generic (tidak menyebut nama entity domain apapun)?
- [ ] Apakah kode ini dipakai (atau akan dipakai dalam waktu dekat, bukan spekulasi) oleh ≥ 2 module/service?
- [ ] Apakah mengekstraknya sekarang tidak menambah kompleksitas import yang tidak perlu untuk 1 use-case saja (YAGNI check)?

Jika salah satu jawaban "tidak", **jangan** buat shared package — duplikasi kecil (< 20 baris) lebih murah daripada abstraksi salah yang harus di-decouple ulang di kemudian hari (*Wrong Abstraction is costlier than Duplication* — prinsip yang lebih penting daripada DRY yang diterapkan naif).

---

## 4. Branching Strategy

### 4.1 Keputusan: Trunk-Based Development (Simplified) + Feature Branch Workflow

- Branch utama: **`main`** — selalu deployable, dilindungi (protected branch).
- Tidak ada branch `develop` terpisah (menghindari overhead Git Flow yang berlebihan untuk tim kecil/solo).
- Setiap pekerjaan dilakukan di **feature branch** pendek umur (idealnya < 3 hari hidup) dari `main`, digabungkan kembali via Pull Request setelah lolos checklist (§13).
- Rilis dikelola lewat **Git Tag** (`v0.1.0`, dst.), bukan branch rilis terpisah, kecuali saat proyek sudah masuk Phase D dan butuh hotfix paralel per service (di titik itu, `release/<service>/vX.Y` branch diperbolehkan per service, dijelaskan lebih lanjut nanti).

### 4.2 Rationale

| Strategi | Cocok untuk | Alasan Ditolak/Diterima |
|---|---|---|
| Git Flow (develop, release, hotfix branch) | Rilis terjadwal, tim besar, versi rilis eksplisit ke pelanggan enterprise | Terlalu berat untuk proyek continuous-learning yang mengutamakan iterasi cepat dan CI/CD zero-downtime. Overhead merge antar branch panjang jadi sumber konflik |
| **Trunk-Based + Feature Branch pendek** | Tim kecil-menengah, CI/CD matang, ingin belajar continuous deployment | **Dipilih.** Selaras dengan tujuan belajar Zero Downtime Deployment & Blue-Green — rilis sering dan kecil menurunkan risiko dibanding rilis besar jarang-jarang |
| GitHub Flow murni (tanpa versioning) | Aplikasi SaaS tanpa versi eksplisit | Ditolak sebagian — kita tetap butuh tag versi untuk korelasi observability (mis. `service_version` di metric/trace) |

### 4.3 Nama Branch

Format: `<type>/<scope>-<short-description>`

Contoh:
- `feature/auth-jwt-refresh-token`
- `fix/message-pagination-cursor`
- `chore/upgrade-go-1-25`
- `refactor/extract-notification-domain`
- `docs/adr-004-orm-selection`

`type` mengikuti daftar yang sama dengan Conventional Commit type (§9).

### 4.4 Aturan Proteksi `main`

- Wajib lolos CI (build, lint, unit test, vulnerability scan).
- Wajib minimal 1 approval review (self-review terstruktur diperbolehkan untuk proyek solo — lihat §11 — tapi checklist tetap wajib dijalankan, bukan diskip).
- Tidak boleh force-push ke `main`.
- Merge method: **Squash and Merge** (menjaga history `main` bersih, satu commit = satu unit kerja logis, memudahkan `git bisect` dan rollback).

---

## 5. Versioning Strategy

### 5.1 Keputusan

- **Semantic Versioning (SemVer 2.0.0)**: `MAJOR.MINOR.PATCH` untuk seluruh artifact yang punya "kontrak publik": REST API, event schema, dan shared package di `pkg/`.
- **API Versioning**: menggunakan **URI versioning** (`/api/v1/...`), bukan header-based versioning.
- **Event Schema Versioning**: setiap payload event membawa field `event_version` (integer), independen dari versi service.
- **Service Versioning** (Phase D): setiap service memiliki versi sendiri, di-tag terpisah (`identity-svc/v1.2.0`) menggunakan git tag berprefix nama service (mendukung independent release cadence antar service — salah satu tujuan utama microservices).

### 5.2 Rationale — URI Versioning vs Header Versioning

| Aspek | URI Versioning (`/v1/...`) | Header Versioning (`Accept: application/vnd.api+json;version=1`) |
|---|---|---|
| Learning value untuk pemula-menengah | Tinggi — eksplisit, terlihat di log, mudah di-debug, mudah di-routing oleh Traefik/API Gateway berdasarkan path | Rendah untuk konteks belajar — implisit, lebih sulit di-observe di log akses standar |
| Cache-ability | Baik — URL berbeda = cache key berbeda secara natural | Perlu konfigurasi `Vary` header tambahan |
| Routing di API Gateway (Traefik) | Native, path-based routing out of the box | Butuh custom middleware |
| Kesesuaian dengan prinsip *Explicit over Implicit* | Sangat sesuai | Kurang sesuai |
| Keputusan | **URI Versioning dipilih** | Ditolak untuk konteks proyek ini |

### 5.3 Kapan Menaikkan Versi

- **MAJOR**: breaking change pada kontrak (field dihapus/diganti tipe, endpoint dihapus, event field wajib berubah makna).
- **MINOR**: penambahan field/endpoint yang backward-compatible.
- **PATCH**: bug fix tanpa perubahan kontrak.

**Aturan keras:** breaking change pada event schema yang sudah punya subscriber aktif **wajib** melalui periode dual-publish (versi lama & baru berjalan bersamaan) minimal 1 sprint sebelum versi lama dimatikan — ini mempraktikkan pola *Expand-Contract Migration* yang relevan untuk pembelajaran Saga/Event-Driven.

---

## 6. CI/CD Strategy

### 6.1 Prinsip

CI/CD dirancang **evolusioner mengikuti fase arsitektur**, bukan dibuat sekali untuk selamanya:

| Fase | CI/CD yang dibutuhkan |
|---|---|
| Phase A (Modular Monolith) | Satu pipeline: lint → test → build → build image → deploy (Docker Compose, single VPS) |
| Phase B (Event-Driven Monolith) | Tambahan: contract test untuk event schema, test outbox/consumer |
| Phase C (Hybrid) | Path-based trigger (hanya build/deploy service yang berubah), pipeline mulai per-service |
| Phase D (Microservices) | Pipeline penuh per-service, independent deploy, canary/blue-green otomatis, service dependency graph untuk test integrasi |

Ini penting dijelaskan di awal karena **kesalahan umum** peserta belajar adalah membangun CI/CD serba lengkap (multi-stage canary, service mesh, dsb.) sejak Phase A padahal belum ada kebutuhan nyata — melanggar YAGNI dan justru mengaburkan pembelajaran evolusi arsitektur itu sendiri.

### 6.2 Tooling

**GitHub Actions**, dengan struktur:

```
.github/workflows/
├── ci.yml                # Trigger: pull_request → lint, test, vulnerability scan
├── build-and-push.yml    # Trigger: push ke main → build image, push ke registry
├── deploy-staging.yml    # Trigger: push ke main (auto) → deploy staging
├── deploy-production.yml # Trigger: manual approval / tag release → deploy production
└── path-filters.yml      # (Phase C+) Reusable workflow untuk deteksi perubahan per-service
```

### 6.3 Pipeline `ci.yml` (Wajib untuk Semua PR)

Urutan tahap (fail-fast, tahap lebih cepat duluan):

1. **Format Check** — `gofmt -l .` (backend), `biome check` (frontend). Gagal jika ada file tidak terformat.
2. **Lint** — `golangci-lint run` (termasuk `depguard` untuk aturan §2.3), `biome lint`.
3. **Static Security Scan** — `gosec` untuk Go, `npm audit` / `pnpm audit` untuk frontend, `govulncheck` untuk known vulnerability di dependency Go.
4. **Unit Test + Race Detector** — `go test ./... -race -cover`. Race detector **wajib**, bukan opsional, karena proyek ini banyak memakai goroutine/channel (worker pool, pipeline pattern) — bug race condition adalah kelas bug paling mahal untuk didebug di production.
5. **Build** — memastikan seluruh module/service dapat dikompilasi.
6. **(Phase B+) Contract Test** — validasi skema event terhadap `proto/events/`.

### 6.4 Rationale Urutan Tahap

Prinsip *fail fast, fail cheap*: tahap yang murah dan cepat (format, lint) dijalankan lebih dulu agar feedback loop developer secepat mungkin, sebelum menghabiskan waktu CI untuk test yang lebih berat.

### 6.5 Definition of "Green Pipeline"

PR hanya dapat di-merge jika **seluruh tahap** lolos. Tidak ada konsep "merge dengan CI merah lalu diperbaiki nanti" — ini melanggar Production First Mindset.

---

## 7. Naming Conventions

Konvensi penamaan adalah salah satu sumber inkonsistensi terbesar dalam proyek jangka panjang. Tabel berikut adalah **rujukan wajib**.

### 7.1 Go — Package & Folder

- Nama package: huruf kecil semua, satu kata, tanpa underscore/camelCase (`message`, bukan `messageService` atau `message_service`).
- Nama folder = nama package (kecuali folder container seperti `internal/`, `cmd/`).
- Domain module folder: `internal/<domain>/` — contoh: `internal/identity/`, `internal/channel/`.
- Di dalam domain module, sub-layer mengikuti Clean Architecture:
  ```
  internal/message/
  ├── domain/           # Entity, Value Object, Domain Event, Repository interface (port)
  ├── application/      # Use case / service layer (business logic orchestration)
  ├── infrastructure/   # Repository implementation (adapter), event publisher impl
  ├── interface/
  │   ├── http/         # Gin handler, DTO request/response
  │   └── websocket/    # WS handler (bila relevan untuk domain ini)
  └── wire.go           # Google Wire provider set untuk module ini
  ```

**Rationale nama layer:** `domain / application / infrastructure / interface` mengikuti terminologi standar Clean Architecture (Uncle Bob) yang paling banyak dirujuk di literatur — memudahkan korelasi antara kode dan konsep yang dipelajari di Learning Roadmap.

### 7.2 Go — Interface, Struct, Method

| Elemen | Konvensi | Contoh |
|---|---|---|
| Interface (port) | Nama benda/peran, TANPA prefix `I`, akhiran `-er` jika mendeskripsikan aksi tunggal, atau nama domain untuk repository | `MessageRepository`, `EventPublisher`, `PasswordHasher` |
| Struct implementasi (adapter) | `<Teknologi><InterfaceName>` atau deskriptif | `PostgresMessageRepository`, `RedisPresenceCache`, `AsynqTaskEnqueuer` |
| Method exported | PascalCase, verb-first | `CreateMessage`, `FindByID`, `MarkAsDelivered` |
| Method unexported (helper internal) | camelCase | `validatePayload`, `buildQuery` |
| Constructor | `New<StructName>` | `NewPostgresMessageRepository(db *sql.DB) *PostgresMessageRepository` |
| Getter | TANPA prefix `Get` (idiomatic Go) | `msg.Content()`, bukan `msg.GetContent()` |
| Context parameter | Selalu nama `ctx`, selalu parameter pertama | `func (s *MessageService) Create(ctx context.Context, cmd CreateMessageCommand) error` |

### 7.3 Variable & Konstanta

- Variable lokal: camelCase, sesingkat mungkin sesuai scope (`i` untuk index loop pendek, `msgRepo` bukan `messageRepositoryInstance` untuk variable lokal umum).
- Konstanta exported: PascalCase dengan grouping lewat `const` block + tipe custom bila representasi enum:
  ```go
  type PresenceStatus string

  const (
      PresenceOnline  PresenceStatus = "online"
      PresenceIdle    PresenceStatus = "idle"
      PresenceDND     PresenceStatus = "dnd"
      PresenceOffline PresenceStatus = "offline"
  )
  ```
- Environment variable name di kode Go (via Viper): SCREAMING_SNAKE_CASE, prefix per service: `NEXUS_API_<NAMA>`, contoh `NEXUS_API_DB_DSN`, `NEXUS_API_JWT_SECRET`, `NEXUS_NOTIFICATION_SVC_REDIS_ADDR` (Phase D). Prefix per-service **wajib** agar tidak terjadi tabrakan nama saat seluruh service digabung dalam satu orchestrator/secret manager.

### 7.4 Domain Event Naming

Format: `<Domain>.<Entity><PastTenseVerb>` — event SELALU past-tense karena event merepresentasikan **fakta yang sudah terjadi**, bukan perintah.

Contoh: `identity.UserRegistered`, `workspace.MemberJoined`, `channel.ChannelCreated`, `message.MessageDeleted`, `presence.PresenceUpdated`.

**Anti-pattern yang harus dihindari:** `message.DeleteMessage` (ini terlihat seperti command, bukan event — kesalahan yang sangat umum dan menyebabkan kebingungan CQRS/Event Sourcing bagi pemula).

### 7.5 Queue / Topic / Stream Naming (Asynq & Redis Streams/Pub-Sub)

Format: `<domain>:<subject>:<action>` (lowercase, colon separator — konvensi umum Redis key/topic).

Contoh task Asynq: `notification:email:send`, `media:video:transcode`, `search:index:rebuild`.

Contoh Redis Stream (Phase B, outbox relay): `stream:message:events`, `stream:notification:events`.

### 7.6 Database Naming

- Nama database: `nexus_<environment>` — `nexus_dev`, `nexus_staging`, `nexus_prod`. Pada Phase D, tiap service punya database sendiri: `nexus_identity_prod`, `nexus_message_prod` (Database-per-Service pattern).
- Nama tabel: snake_case, **plural**, tanpa prefix domain redundant (nama domain sudah jelas dari nama database/schema): `users`, `workspaces`, `channels`, `messages`, `message_reactions`.
- Primary key: selalu `id`, tipe **UUID v7** (bukan auto-increment integer) — rationale: UUID v7 time-ordered, aman untuk sistem terdistribusi (tidak ada konflik ID lintas service saat Phase D, dan tidak membocorkan jumlah row seperti auto-increment), sekaligus tetap index-friendly karena time-ordered (berbeda dari UUID v4 random yang buruk untuk B-Tree index).
- Foreign key: `<singular_referenced_table>_id`, contoh `workspace_id`, `channel_id`, `author_id` (bila mereferensi user sebagai penulis, nama kolom deskriptif, bukan generik `user_id` jika ada makna kontekstual).
- Kolom timestamp: `created_at`, `updated_at`, `deleted_at` (untuk soft delete), selalu `timestamptz`, bukan `timestamp` (menghindari ambiguitas timezone — kesalahan sangat umum).
- Kolom versi untuk Optimistic Locking: `version` (integer, default 0, increment tiap update).
- Index: `idx_<table>_<column(s)>`, contoh `idx_messages_channel_id_created_at`.
- Composite index: urutan kolom mengikuti pola query tersering (equality columns dulu, baru range/sort column) — dijelaskan detail per kasus di Database Design (Phase 5).
- Full text search column: `<table>_search_vector` bertipe `tsvector`, contoh `messages_search_vector`.

### 7.7 Migration Naming

Format: `<timestamp>_<verb>_<object>.sql`, contoh:
```
20260101120000_create_users_table.sql
20260102093000_add_version_column_to_messages.sql
20260103150000_create_message_reactions_table.sql
```

Timestamp (bukan sequential integer) dipilih agar **tidak terjadi konflik nomor migrasi** ketika bekerja di banyak feature branch paralel — masalah klasik yang sering dialami pemula memakai sequential numbering.

### 7.8 API Endpoint Naming

- Resource-based, plural noun, kebab-case bila multi-kata: `/api/v1/workspaces`, `/api/v1/workspaces/{workspaceId}/channels`, `/api/v1/message-reactions` (dihindari sebisa mungkin — reaction sebaiknya nested: `/api/v1/messages/{messageId}/reactions`).
- Verb HANYA untuk aksi yang bukan CRUD murni: `/api/v1/auth/login`, `/api/v1/messages/{id}/pin` (POST), `/api/v1/invites/{code}/redeem` (POST).
- Query parameter: snake_case, contoh `?cursor=xxx&limit=50&sort=created_at_desc`.

Detail lengkap tiap endpoint dibahas di API Specification (Phase 6); di sini hanya konvensi penamaan.

---

## 8. Definition of Quality

Sebuah unit kerja (fitur/modul) dianggap **berkualitas** jika memenuhi seluruh kriteria berikut. Ini bukan checklist administratif — ini adalah standar yang membedakan "berfungsi" dari "production-ready", sesuai *Production First Mindset*.

1. **Correctness** — memenuhi acceptance criteria, termasuk edge case (empty state, boundary value, concurrent access).
2. **Testability** — logic bisnis dapat diuji tanpa memerlukan database/network nyata (berkat Repository Pattern + DI).
3. **Observability** — setiap operasi penting menghasilkan log terstruktur, metric, dan (bila lintas boundary) trace span. "Jika tidak bisa di-observe di production, maka belum selesai."
4. **Idempotency (bila relevan)** — operasi yang bisa di-retry (event consumer, task queue, API dengan efek samping finansial/state kritikal) aman dijalankan berulang.
5. **Graceful Failure** — error ditangani eksplisit, tidak ada `panic` yang bocor ke response API, tidak ada goroutine leak.
6. **Security Baseline** — input tervalidasi, authorization dicek di layer yang benar (bukan hanya di frontend), tidak ada secret ter-hardcode.
7. **Performance Baseline** — tidak ada N+1 query, tidak ada operasi O(n²) yang jelas dapat dihindari untuk data besar, dipertimbangkan terhadap NFR (10.000 concurrent users).
8. **Documentation** — perubahan kontrak publik (API/event) terdokumentasi; keputusan non-trivial dicatat sebagai ADR bila relevan.

**Kapan kriteria ini boleh dilonggarkan:** untuk *spike/prototype* eksploratif yang eksplisit ditandai tidak akan masuk `main` (dikerjakan di branch terpisah yang tidak akan di-PR), kriteria observability/idempotency boleh diskip. Tapi begitu kode akan masuk `main`, seluruh kriteria berlaku penuh — tanpa pengecualian "nanti diperbaiki belakangan" (technical debt harus **dicatat eksplisit**, bukan diam-diam diskip — lihat §14 Checklist Merge).

---

## 9. Commit Convention — Conventional Commits

### 9.1 Format

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### 9.2 Type yang Diizinkan

| Type | Kapan dipakai |
|---|---|
| `feat` | Penambahan fitur/kapabilitas baru |
| `fix` | Perbaikan bug |
| `refactor` | Perubahan struktur kode tanpa mengubah behavior |
| `perf` | Perubahan yang murni untuk meningkatkan performa |
| `test` | Menambah/memperbaiki test tanpa mengubah kode produksi |
| `docs` | Perubahan dokumentasi saja |
| `chore` | Maintenance (upgrade dependency, konfigurasi tooling) |
| `ci` | Perubahan pipeline CI/CD |
| `build` | Perubahan build system/Dockerfile |
| `revert` | Membatalkan commit sebelumnya |

### 9.3 Scope

Scope = nama domain/module yang terdampak: `feat(message): add reply threading`, `fix(auth): correct refresh token expiry calculation`, `refactor(presence): extract redis client to shared pkg`.

### 9.4 Breaking Change

Ditandai dengan `!` setelah type/scope, dan wajib ada footer `BREAKING CHANGE: <penjelasan>`:

```
feat(api)!: change pagination response envelope

BREAKING CHANGE: field `next_page` diganti menjadi `next_cursor` di seluruh endpoint list.
```

### 9.5 Enforcement

**Commitlint** (via **Husky** `commit-msg` hook) menolak commit yang tidak sesuai format di atas — mencegah pelanggaran sampai masuk history, bukan dikoreksi setelahnya.

### 9.6 Rationale

Conventional Commits dipilih karena tiga manfaat konkret: (1) **automated changelog generation** per rilis, (2) **automated semantic version bump** (feat→minor, fix→patch, BREAKING CHANGE→major) bila di masa depan proyek ingin otomasi rilis penuh, (3) **searchability history** — `git log --grep="^fix(auth)"` langsung memberi riwayat bug fix domain tertentu.

---

## 10. Coding Guideline

### 10.1 Go — Prinsip Umum

- Ikuti [Effective Go](https://go.dev/doc/effective_go) dan [Google Go Style Guide](https://google.github.io/styleguide/go/) sebagai baseline; playbook ini hanya menambahkan aturan spesifik proyek.
- **Error wrapping wajib** menggunakan `fmt.Errorf("...: %w", err)` agar `errors.Is`/`errors.As` tetap berfungsi lintas layer.
- **Custom error** didefinisikan sebagai sentinel error (`var ErrMessageNotFound = errors.New(...)`) untuk domain error yang perlu dibedakan penanganannya di layer atas (mis. mapping ke HTTP 404 vs 500).
- **Context selalu di-propagate**, tidak pernah `context.Background()` di dalam handler/service (hanya boleh di `main.go`/entrypoint atau goroutine background yang sengaja lepas dari request lifecycle, dengan komentar eksplisit menjelaskan alasan).
- **Goroutine tanpa pengawasan dilarang**: setiap goroutine yang di-spawn wajib punya mekanisme untuk menunggu selesai (`sync.WaitGroup`, `errgroup.Group`) atau di-manage oleh worker pool — mencegah goroutine leak.
- **Graceful Shutdown wajib** di setiap entrypoint (`cmd/*/main.go`): menangkap `SIGTERM`/`SIGINT`, memberi context timeout untuk menyelesaikan request in-flight (default 15 detik), menutup koneksi DB/Redis/message broker secara eksplisit sebelum proses exit.

### 10.2 Konkurensi — Kapan Memakai Goroutine/Channel

Sesuai instruksi: goroutine/channel **hanya dipakai bila ada manfaat nyata**, bukan default gaya penulisan. Panduan keputusan:

| Situasi | Pakai Goroutine/Channel? | Alasan |
|---|---|---|
| Fan-out ke beberapa dependency I/O independen (mis. ambil data user + workspace + presence sekaligus untuk 1 response) | Ya, dengan `errgroup` | Latency total = max(latency), bukan sum(latency) — manfaat nyata dan terukur |
| Worker Pool untuk pemrosesan task background (mis. transcoding, thumbnail generation) | Ya | Membatasi concurrency agar tidak membanjiri resource (CPU/memori/koneksi eksternal) |
| Pipeline pattern untuk streaming pemrosesan data besar (mis. export data) | Ya, jika data cukup besar hingga blocking single-threaded jadi bottleneck terukur | Perlu dibuktikan dengan benchmark, bukan asumsi |
| Single request handler biasa (CRUD) | Tidak | Tidak ada manfaat, hanya menambah kompleksitas debugging tanpa gain performa |
| Update field sederhana yang tidak I/O-bound | Tidak | Overhead goroutine scheduling > gain |

### 10.3 Frontend (Nuxt/Vue) — Prinsip Umum

- **Composition API** dengan `<script setup lang="ts">` sebagai standar, tidak memakai Options API (konsistensi, lebih baik untuk type inference TypeScript).
- **Pinia store** hanya untuk state yang benar-benar global/lintas komponen (auth session, active workspace, presence map). State lokal komponen tetap pakai `ref`/`reactive` biasa — menghindari over-centralization state yang jadi anti-pattern umum pemula Vue.
- **TanStack Query** untuk seluruh data-fetching dari REST API (caching, invalidation, retry, background refetch) — Pinia **tidak** dipakai untuk menyimpan hasil fetch server state (pemisahan **server state vs client state** adalah prinsip modern yang harus dipahami, bukan mencampur keduanya di satu store).
- **VueUse** dipakai untuk composable umum (debounce, resize observer, dsb.) daripada menulis ulang.

---

## 11. Code Review Guideline

### 11.1 Tujuan Review

Code review bukan gerbang administratif, tapi mekanisme **knowledge sharing** dan **quality gate**. Untuk proyek belajar solo, review dilakukan sebagai **self-review terstruktur** menggunakan checklist yang sama — mensimulasikan disiplin tim nyata, sekaligus melatih kemampuan membaca kode sendiri secara kritis (skill arsitek yang penting).

### 11.2 Yang Diperiksa Reviewer

1. **Kebenaran logika** — apakah edge case tertangani.
2. **Kesesuaian arsitektur** — apakah boundary domain dilanggar (lihat §2.3), apakah layer yang tepat dipakai (business logic tidak bocor ke handler HTTP).
3. **Konsistensi konvensi** — penamaan, struktur folder sesuai §7.
4. **Observability** — apakah log/metric/trace ditambahkan pada titik yang tepat.
5. **Keamanan** — validasi input, authorization check, tidak ada secret/PII di log.
6. **Test** — apakah test yang ditambahkan benar-benar menguji behavior, bukan sekadar menaikkan angka coverage.
7. **Dampak performa** — apakah ada query baru yang berpotensi N+1 atau full table scan.

### 11.3 Etika Review

- Kritik ditujukan ke **kode**, bukan ke penulis.
- Reviewer wajib memberi **alasan**, bukan hanya "ubah ini" — selaras filosofi playbook ini secara keseluruhan (selalu ada rationale).
- Beri label pada komentar: `[blocking]` (wajib diperbaiki sebelum merge), `[suggestion]` (opsional, boleh diabaikan dengan alasan), `[question]` (klarifikasi, bukan tuntutan perubahan).

---

## 12. Pull Request Guideline

### 12.1 Ukuran PR

- PR ideal: **< 400 baris diff** (di luar file generated/lockfile). PR besar dipecah per lapisan logis (mis. PR 1: domain + repository, PR 2: service layer + handler, PR 3: frontend integration) bila memungkinkan.
- Rationale: PR besar menurunkan kualitas review secara drastis (reviewer fatigue) dan menyulitkan `git bisect` bila terjadi regresi.

### 12.2 Template PR (Wajib)

```markdown
## Ringkasan
<Apa yang diubah dan mengapa — 2-3 kalimat>

## Tipe Perubahan
- [ ] feat
- [ ] fix
- [ ] refactor
- [ ] docs
- [ ] chore

## Domain/Module Terdampak
<mis. message, channel>

## Checklist
- [ ] Lolos lint & format lokal
- [ ] Test ditambahkan/diperbarui dan lolos (`go test ./... -race`)
- [ ] Tidak ada breaking change pada API/event TANPA dokumentasi versi baru
- [ ] Observability (log/metric/trace) ditambahkan untuk operasi baru yang signifikan
- [ ] Tidak ada secret/kredensial ter-hardcode
- [ ] Dokumentasi terkait diperbarui (bila relevan)

## Referensi
<Link ke ADR/task/issue terkait>

## Catatan Technical Debt (bila ada)
<Jelaskan debt yang sengaja diambil beserta rencana pelunasannya>
```

### 12.3 Draft PR

Untuk pekerjaan besar yang butuh visibilitas dini (feedback arah desain sebelum selesai), gunakan **Draft PR** — bukan menunggu selesai 100% baru dibuka PR.

---

## 13. Checklist Sebelum Merge

- [ ] CI Green (§6.3) seluruh tahap.
- [ ] Tidak ada `TODO`/`FIXME` tanpa referensi ke issue/task tracker.
- [ ] Tidak ada `console.log`/`fmt.Println` debug yang tertinggal (harus pakai logger terstruktur).
- [ ] Migration (bila ada) sudah diuji `up` **dan** `down`.
- [ ] Tidak ada perubahan schema database yang breaking tanpa strategi backward-compatible (expand-contract, lihat §5.3).
- [ ] Environment variable baru (bila ada) sudah didokumentasikan di `.env.example` dan README konfigurasi.
- [ ] Self-review checklist §11.2 sudah dijalankan.

---

## 14. Checklist Sebelum Release

- [ ] Seluruh item Checklist Merge (§13) sudah terpenuhi untuk seluruh PR dalam rilis ini.
- [ ] Changelog dihasilkan (otomatis dari Conventional Commit, atau manual bila belum ada automasi).
- [ ] Versi di-tag sesuai SemVer (§5).
- [ ] Migration database sudah dijalankan di staging dan diverifikasi tidak ada data loss.
- [ ] Dashboard Grafana/alert Prometheus sudah mencakup metric baru (bila ada penambahan signifikan).
- [ ] Rencana rollback dituliskan eksplisit (bagaimana cara mundur bila rilis bermasalah — image sebelumnya, migration down script).
- [ ] Technical debt yang diambil pada rilis ini dicatat di dokumen tracking debt (lihat Vision Document, Phase 0 berikutnya).
- [ ] Load/smoke test dasar dijalankan di staging untuk perubahan yang berpotensi berdampak performa signifikan.

---

## 15. Logging Convention (Zap)

### 15.1 Prinsip

- **Structured logging wajib** — tidak ada `log.Printf` string bebas format. Semua log memakai Zap `SugaredLogger`/`Logger` dengan field terstruktur.
- Setiap log request HTTP/event wajib menyertakan **`trace_id`** dan **`request_id`** (dipropagasi lewat context, terhubung dengan OpenTelemetry span — lihat Observability).
- **Level log**:
  - `Debug`: detail teknis untuk debugging lokal, **tidak aktif di production** secara default.
  - `Info`: event bisnis penting (user registered, message created) dan lifecycle aplikasi (startup, shutdown).
  - `Warn`: kondisi tidak normal tapi masih tertangani (retry terjadi, fallback dipakai, rate limit mendekati batas).
  - `Error`: kegagalan yang mempengaruhi hasil operasi, wajib menyertakan error yang di-wrap lengkap.
  - `Fatal`: HANYA dipakai di `main.go` untuk kegagalan startup yang membuat aplikasi tidak bisa berjalan sama sekali (mis. gagal konek DB saat boot). **Tidak pernah** dipakai di dalam business logic/handler (karena `Fatal` memanggil `os.Exit`, yang berarti tidak ada graceful shutdown).

### 15.2 Field Wajib

```go
logger.Info("message created",
    zap.String("trace_id", traceID),
    zap.String("request_id", requestID),
    zap.String("actor_id", userID),
    zap.String("workspace_id", workspaceID),
    zap.String("channel_id", channelID),
    zap.String("message_id", messageID),
)
```

### 15.3 Larangan Keras

- **Tidak boleh** log password, token JWT/refresh token mentah, isi payload PII sensitif tanpa masking.
- **Tidak boleh** log seluruh body request/response secara membabi-buta (risiko kebocoran data & noise log berlebihan) — hanya field relevan.

---

## 16. Error Handling Convention

### 16.1 Klasifikasi Error

| Kategori | Contoh | Penanganan |
|---|---|---|
| **Domain Error (Expected)** | `ErrMessageNotFound`, `ErrInsufficientPermission`, `ErrOptimisticLockConflict` | Sentinel error, di-map eksplisit ke HTTP status code di layer HTTP handler (mis. 404, 403, 409). Tidak di-log sebagai `Error` level (ini bagian normal alur bisnis), cukup `Info`/`Warn` bila perlu. |
| **Infrastructure Error (Unexpected)** | Koneksi DB putus, timeout Redis | Di-wrap dengan konteks, di-log `Error`, di-map ke HTTP 500/503, memicu alert bila melebihi threshold (lihat Observability). |
| **Validation Error** | Input tidak sesuai skema | Ditangani di layer validasi (`go-playground/validator`) sebelum masuk business logic, response 400 dengan detail field yang salah. |
| **Panic (Programmer Error)** | Nil pointer dereference, index out of range | Ditangkap oleh `recover()` di middleware Gin paling luar, dicatat sebagai `Error` dengan stack trace lengkap, response 500 generik ke client (tidak membocorkan stack trace ke client). |

### 16.2 Struktur Custom Error untuk Domain

```go
type DomainError struct {
    Code    string // mis. "MESSAGE_NOT_FOUND"
    Message string // pesan aman ditampilkan ke user
    Err     error  // error asli untuk logging/wrapping
}

func (e *DomainError) Error() string { return e.Message }
func (e *DomainError) Unwrap() error { return e.Err }
```

**Rationale:** field `Code` machine-readable dipisah dari `Message` human-readable memungkinkan frontend melakukan penanganan spesifik (mis. menampilkan modal khusus untuk `OPTIMISTIC_LOCK_CONFLICT`) tanpa parsing string pesan yang rapuh.

### 16.3 API Error Response Envelope

Lihat §17.3 untuk format lengkap response error yang konsisten di seluruh endpoint.

---

## 17. API Convention

### 17.1 Response Envelope

Semua response REST API mengikuti envelope konsisten:

```json
{
  "success": true,
  "data": { },
  "meta": { }
}
```

Error:

```json
{
  "success": false,
  "error": {
    "code": "MESSAGE_NOT_FOUND",
    "message": "Pesan yang diminta tidak ditemukan.",
    "details": []
  }
}
```

### 17.2 Pagination

**Cursor-based pagination** dipakai untuk seluruh list endpoint yang berpotensi data besar (messages, members) — **bukan** offset-based, karena:

| Aspek | Offset-based | Cursor-based |
|---|---|---|
| Performa pada dataset besar | Menurun seiring page bertambah (`OFFSET` besar tetap harus di-scan DB) | Konsisten (`WHERE id > cursor LIMIT n`, memanfaatkan index) |
| Konsistensi saat data berubah real-time (chat message terus bertambah) | Rawan duplikasi/skip data antar page saat data baru masuk | Stabil karena berbasis posisi relatif, bukan index absolut |
| Keputusan | Ditolak untuk list yang high-write-throughput | **Dipilih untuk seluruh list endpoint** |

Format response list:

```json
{
  "success": true,
  "data": [ ],
  "meta": {
    "next_cursor": "eyJpZCI6...",
    "has_more": true
  }
}
```

### 17.3 Rate Limiting Response

HTTP 429 dengan header `Retry-After`, body mengikuti error envelope dengan `code: "RATE_LIMIT_EXCEEDED"`. Detail strategi rate limiting dibahas di Security Design (Phase 7).

### 17.4 Idempotency Key

Endpoint dengan efek samping yang tidak boleh terduplikasi bila di-retry client (mis. create invite, redeem invite) wajib mendukung header `Idempotency-Key`, disimpan sementara di Redis dengan TTL, memetakan ke response yang sudah pernah dihasilkan.

---

## 18. Git Convention (Tambahan di Luar Branching)

- `.gitignore` terpusat di root, mencakup seluruh sub-project (Go build artifact, `node_modules`, `.env*` kecuali `.env.example`).
- **Tidak pernah** commit file `.env` asli — hanya `.env.example` dengan placeholder.
- **Tidak pernah** commit binary besar (video/audio sample) ke repo — dipakai `git-lfs` bila benar-benar dibutuhkan untuk asset test, atau disimpan di object storage terpisah.
- Setiap file SQL migration bersifat **append-only** — tidak pernah mengedit migration yang sudah pernah di-apply ke environment manapun (termasuk staging). Perubahan lanjutan = migration baru.

---

## 19. Folder Structure Convention (Detail per Aplikasi)

### 19.1 `apps/api/` (Modular Monolith, Phase A-C)

```
apps/api/
├── cmd/
│   └── server/
│       └── main.go              # Entrypoint, wiring, graceful shutdown
├── internal/
│   ├── identity/                # Domain module (lihat §7.1 untuk sub-layer)
│   ├── workspace/
│   ├── member/
│   ├── role/
│   ├── channel/
│   ├── message/
│   ├── attachment/
│   ├── notification/
│   ├── presence/
│   ├── search/
│   ├── media/
│   ├── admin/
│   ├── platform/                # Middleware lintas domain, router setup, health check
│   └── config/                  # Viper config loader spesifik aplikasi ini
├── migrations/
├── go.mod
└── wire_gen.go                  # Hasil generate Google Wire
```

### 19.2 `apps/web/` (Nuxt 4)

```
apps/web/
├── app/
│   ├── components/
│   ├── composables/
│   ├── pages/
│   ├── layouts/
│   ├── stores/                  # Pinia
│   └── plugins/
├── public/
├── nuxt.config.ts
└── package.json
```

Detail lengkap struktur frontend akan diperluas di Low Level Design (Phase 4) sesuai kebutuhan komponen UI Discord-like (server list, channel sidebar, message pane, dsb.).

---

## 20. Documentation Convention

- Seluruh dokumen fase (Vision, ADR, PRD, dst.) disimpan di `docs/` dalam format Markdown, memakai Mermaid untuk diagram (native render di GitHub).
- ADR mengikuti format standar: **Title, Status, Context, Decision, Consequences, Alternatives Considered** — detail template ada di dokumen ADR (dokumen selanjutnya).
- Setiap dokumen fase menyertakan **Changelog** dan **Versi** di header, sama seperti dokumen ini.
- README tiap module/service minimal menjawab: apa tanggung jawabnya, cara menjalankan lokal, dependency eksternal apa yang dibutuhkan.

---

## 21. Tooling Enforcement Summary

| Tooling | Fungsi | Wajib/Opsional |
|---|---|---|
| `golangci-lint` (dengan `depguard`, `gosec`, `errcheck`, `govet`) | Lint & boundary enforcement Go | Wajib, blocking CI |
| `gofmt` / `goimports` | Format Go | Wajib, blocking CI |
| `govulncheck` | Vulnerability scan dependency Go | Wajib, blocking CI |
| Biome | Lint & format frontend | Wajib, blocking CI |
| Husky | Git hook (`commit-msg`, `pre-push`) | Wajib di lingkungan development |
| Commitlint | Validasi format Conventional Commit | Wajib, via Husky |
| `go test -race` | Unit test + race detector | Wajib, blocking CI |

---

## Ringkasan Keputusan (Executive Summary)

1. **Monorepo** dipilih untuk memaksimalkan learning value evolusi arsitektur dan meminimalkan overhead di skala tim kecil/solo.
2. **Go Workspace (`go.work`)** dipakai agar isolasi dependency antar module tetap terjaga tanpa menyakiti developer experience — krusial untuk mempersiapkan ekstraksi service yang mulus.
3. **Trunk-Based Development + Feature Branch pendek** dipilih dibanding Git Flow untuk mendukung continuous delivery dan mengurangi risiko rilis besar.
4. **URI-based API versioning** dan **cursor-based pagination** dipilih atas pertimbangan observability, cache-ability, dan kesesuaian dengan skala data chat real-time.
5. **UUID v7** sebagai primary key seluruh tabel untuk kesiapan distributed system sejak awal.
6. **Conventional Commits** sebagai standar commit, ditegakkan otomatis via Husky+Commitlint.
7. **Boundary domain ditegakkan otomatis** lewat `depguard` untuk mencegah Distributed Monolith sejak dalam fase monolith.

## Trade-off yang Diterima

- Kompleksitas awal `go.work` + multi-module lebih tinggi dibanding single `go.mod` — diterima demi kesiapan ekstraksi service di kemudian hari.
- Self-review terstruktur (bukan peer-review sungguhan) untuk konteks solo learning — risiko blind spot lebih tinggi, dimitigasi dengan checklist eksplisit dan disiplin tinggi terhadap Definition of Quality (§8).
- Cursor-based pagination lebih kompleks diimplementasikan dibanding offset-based — diterima karena krusial untuk NFR skala chat real-time.

## Risiko Arsitektur

- Disiplin menjaga boundary domain (§2.3) bergantung pada konsistensi penggunaan `depguard`; bila aturan ini tidak dijaga ketat sejak awal, risiko Distributed Monolith saat ekstraksi service (Phase C/D) meningkat signifikan.
- Monorepo dengan banyak service di Phase D berpotensi memperlambat CI bila tidak segera diterapkan path-based trigger — perlu direncanakan sebelum jumlah service bertambah banyak.

## Technical Debt yang Sengaja Diterima

- Belum ada automasi penuh semantic-release berbasis Conventional Commit (changelog masih semi-manual di awal) — akan dipertimbangkan saat frekuensi rilis meningkat.
- `k8s/` manifest disiapkan sebagai skeleton dokumentasi sejak Phase A namun **tidak dipakai aktif** hingga Phase D — ini debt yang disengaja untuk menghindari kompleksitas prematur (YAGNI), bukan kelalaian.

## Hal yang Perlu Dikonfirmasi Sebelum Lanjut ke Fase Berikutnya

1. Apakah nama proyek **"Nexus"** dapat diterima sebagai working name, atau Anda ingin nama lain? (dipakai konsisten di seluruh dokumen berikutnya, termasuk prefix environment variable dan nama database).
2. Apakah pemilihan **pnpm** untuk frontend package manager disetujui, atau tetap ingin **npm**? (§2.1)
3. Apakah struktur top-level `apps/` + `services/` (§1.4) sudah sesuai ekspektasi, atau ada preferensi struktur lain?
4. Apakah kebijakan **UUID v7 sebagai primary key** dapat diterima, mengingat ini akan menjadi keputusan mendasar yang sulit diubah setelah Database Design (Phase 5) dibuat?
5. Apakah Anda ingin melanjutkan ke dokumen **Vision Document** berikutnya (masih bagian Phase 0), atau ada revisi terhadap Engineering Playbook ini terlebih dahulu?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen pertama, mencakup seluruh scope Phase 0 - Engineering Playbook sesuai project instructions |
