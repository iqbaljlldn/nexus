# AGENTS.md — Nexus

> Panduan konteks untuk AI coding agent (Claude Code, Cursor, dsb.) yang bekerja di repo ini. Disintesis dari 15 dokumen perencanaan lengkap (Phase 0-11) di `docs/`. **Baca file ini terlebih dahulu sebelum menyentuh kode apapun.**

---

## 1. Apa Proyek Ini

**Nexus** adalah platform komunikasi real-time bergaya Discord, dibangun sebagai **Project-Based Learning (PBL)** — tujuan utamanya adalah pembelajaran arsitektur software modern (bukan mengejar fitur produk semata). Proyek berevolusi secara sengaja: **Modular Monolith → Event-Driven Monolith → Hybrid → Full Microservices**.

**Implikasi penting untuk agent**: jangan menyarankan atau menerapkan pola arsitektur dari fase yang belum tiba (mis. jangan mengekstraksi service atau menambah gRPC bila proyek masih di Phase A) kecuali diminta eksplisit. Cek §7 dokumen ini untuk status fase saat ini.

Dokumen sumber kebenaran lengkap ada di `docs/`:

| Dokumen | Isi |
|---|---|
| `01-engineering-playbook.md` | Konvensi kode, struktur folder, Git/commit, CI/CD, naming |
| `02-vision-document.md` | Tujuan proyek, definisi sukses, exclusion list |
| `03-adr.md` | Keputusan teknologi & rationale (Gin, sqlc, MinIO, Redis Streams, dst.) |
| `04-learning-roadmap.md` | 18 milestone pembelajaran |
| `05-prd.md` | Requirement produk (user story, prioritas) |
| `06-srs.md` | Requirement teknis presisi (FR/NFR) |
| `07-hld.md` | Domain model, event catalog, service extraction plan |
| `08-lld.md` | Interface Go, algoritma kunci (permission resolver, outbox relay, dst.) |
| `09-database-design.md` | ERD, DDL, index, partitioning |
| `10-api-specification.md` | Kontrak endpoint REST & WebSocket |
| `11-security-design.md` | Threat model, auth flow final, OWASP mapping |
| `12-deployment-architecture.md` | Evolusi deployment 5 tahap |
| `13-development-roadmap.md` | 5 Release, estimasi waktu |
| `14-sprint-planning.md` | Breakdown sprint (Rolling Wave) |
| `15-task-checklist-sprint1.md` | Task-level checklist sprint aktif |

**Aturan emas**: dokumen di `docs/` adalah **Source of Truth**. Bila kode dan dokumen berkonflik, laporkan konflik — jangan diam-diam mengikuti salah satunya.

---

## 2. Tech Stack

| Layer | Teknologi | Alasan Singkat (detail: `03-adr.md`) |
|---|---|---|
| Backend Language | Go 1.25.2 | — |
| Web Framework | Gin | Kompatibel `net/http`, OpenTelemetry native (ADR-002) |
| DB Access | **sqlc** (satu-satunya, tanpa pengecualian Ent/GORM) | Explicit SQL, performa, learning value (ADR-003) |
| Database | PostgreSQL | UUID v7 sebagai PK di semua tabel |
| Cache/Event Backbone | Redis (cache + **Redis Streams** untuk event) | ADR-006 |
| Object Storage | **MinIO self-hosted** (BUKAN Cloudinary) | ADR-007 v1.1 |
| Realtime | Gorilla WebSocket (native, bukan Socket.IO) | ADR-004 |
| Voice/Video | LiveKit | ADR-005 |
| Task Queue | Asynq | Media processing, notification |
| Reverse Proxy | Traefik | ADR-008 |
| Email | Brevo | SRS §5 |
| Frontend | Nuxt 4, Vue 3, TypeScript, Pinia, TanStack Query, TailwindCSS | — |
| Package Manager Frontend | pnpm (workspace) | — |
| DI | Google Wire | — |
| Config | Viper | — |
| Logging | Zap (structured, wajib) | Playbook §15 |
| Observability | OpenTelemetry + Prometheus + Grafana | Milestone 15 |

---

## 3. Struktur Repo (Monorepo)

```
nexus/
├── apps/api/           # Modular monolith Go — SEMUA domain ada di sini selama Phase A/B
├── apps/web/           # Nuxt 4 frontend
├── services/           # KOSONG hingga Phase C — jangan buat isi di sini kecuali diminta ekstraksi
├── pkg/                # Shared generic packages — TIDAK BOLEH mengandung tipe domain
├── migrations/         # SQL migration, timestamp-based naming
├── deployments/        # docker-compose, traefik, k8s (k8s hanya skeleton hingga Phase D)
├── docs/               # 15 dokumen perencanaan — Source of Truth
└── go.work
```

Struktur domain di `apps/api/internal/<domain>/`:
```
<domain>/
├── domain/          # Entity, Value Object, Domain Event, Repository interface (port)
├── application/     # Service layer (business logic)
├── infrastructure/  # Repository implementation (sqlc adapter), event publisher
├── interface/http/  # Gin handler + DTO
└── wire.go
```

15 domain: Identity, Workspace, Member, Role, Permission, Channel (termasuk DM — lihat `07-hld.md` §2.14), Message, Attachment, Notification, Presence, Search, Media, Voice, Video, Admin.

---

## 4. Build, Test, Lint — Command Reference

```bash
# Build
go build ./...                              # dari root, via go.work

# Test (RACE DETECTOR WAJIB — bukan opsional)
go test ./... -race -cover

# Lint
golangci-lint run                           # termasuk depguard (cek boundary domain)

# Format
gofmt -l .

# Security scan
gosec ./...
govulncheck ./...

# sqlc
sqlc generate                               # setelah mengubah query .sql

# DI
wire ./...                                  # setelah mengubah provider set

# Migration
migrate -path migrations -database $NEXUS_API_DB_DSN up
migrate -path migrations -database $NEXUS_API_DB_DSN down 1

# Frontend
pnpm --filter web dev
pnpm --filter web lint                      # Biome

# Docker Compose (dev environment lengkap)
docker compose up -d
```

**Sebelum menganggap task selesai**, agent wajib menjalankan: `gofmt -l .` → `golangci-lint run` → `go test ./... -race` → `go build ./...`. Ini mencerminkan urutan CI (`01-engineering-playbook.md` §6.3).

---

## 5. Prinsip Arsitektur yang Tidak Boleh Dilanggar

Lihat `RULES.md` untuk daftar lengkap aturan keras. Ringkasan filosofi (detail: `01-engineering-playbook.md` intro):

Simplicity over Cleverness · Explicit over Implicit · YAGNI · Clean Architecture · DDD Lite · Production First Mindset · Observability by Default.

Setiap keputusan non-trivial (pilihan library baru, pola konkurensi baru, perubahan skema) harus bisa dijawab: *mengapa ini, apa alternatifnya, apa trade-off-nya* — konsisten dengan gaya seluruh dokumen `docs/`.

---

## 6. Bagaimana Agent Harus Bekerja di Repo Ini

1. **Sebelum implementasi fitur apapun**: cek `06-srs.md` untuk FR/NFR terkait, `08-lld.md` untuk interface/algoritma yang sudah didesain, `10-api-specification.md` untuk kontrak endpoint. **Jangan mendesain ulang dari nol** — desain sudah ada, tugas agent adalah mengimplementasikan sesuai desain, dan menandai bila desain ternyata tidak bisa diimplementasikan sesuai rencana.
2. **Ikuti Task Checklist aktif** (`15-task-checklist-sprint1.md` atau sprint terbaru) — kerjakan task sesuai urutan dependency yang tercantum, tandai checklist saat selesai.
3. **Setiap perubahan skema database**: tulis migration baru (append-only, timestamp naming, `up`+`down`), jangan edit migration lama yang sudah ada.
4. **Setiap query baru**: tulis di file `.sql`, jalankan `sqlc generate` — jangan menulis raw SQL string di kode Go.
5. **Setiap goroutine baru**: pastikan ada mekanisme tunggu selesai (`errgroup`/`WaitGroup`) — lihat `08-lld.md` §2.9-2.10 untuk pola yang sudah disetujui (WebSocket connection manager, worker pool).
6. **Bila menemukan ambiguitas/konflik** antara requirement dan implementasi yang diminta: laporkan eksplisit (seperti pola yang dipakai di seluruh `docs/` — lihat contoh resolusi Cloudinary→MinIO di `03-adr.md`), jangan diam-diam mengambil salah satu jalan.

---

## 7. Status Proyek Saat Ini

**Fase Arsitektur**: Phase A — Modular Monolith (belum ada service di `services/`).
**Sprint Aktif**: Sprint 4 — Real-time Messaging & Presence (`18-task-checklist-sprint4.md`). (Sprint 1, 2, dan 3 telah selesai. Release 1 selesai).
**Amandemen Terakhir**: MinIO menggantikan Cloudinary (ADR-007 v1.1), DM masuk scope resmi (PRD v1.1/SRS v1.1), Brevo sebagai email provider, refresh token via HttpOnly Cookie (Security Design).

> Update bagian ini setiap sprint/fase berpindah, agar agent yang membaca file ini selalu tahu konteks terkini tanpa harus membaca ulang seluruh 15 dokumen.

---

## 8. Referensi Cepat — Endpoint & Domain Event

Kontrak endpoint lengkap: `10-api-specification.md`. Event catalog lengkap (publisher/subscriber/retry/idempotency): `07-hld.md` §3.

Jangan berasumsi format response — selalu pakai envelope `{success, data, meta}` / `{success: false, error: {code, message}}` sesuai `01-engineering-playbook.md` §17.1.
