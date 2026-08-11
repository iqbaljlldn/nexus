# Detailed Task Checklist — Sprint 1

## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 1: Project Foundation + Awal Authentication)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `14-sprint-planning.md` (Sprint 1), `01-engineering-playbook.md`, `09-database-design.md` (§2.1), `10-api-specification.md` (§1), `11-security-design.md` (§2)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen

Sesuai prinsip **Rolling Wave Planning** (Sprint Planning §0), dokumen ini **hanya mencakup Sprint 1**. Task Checklist untuk sprint berikutnya akan dibuat sebagai dokumen baru mendekati waktunya, dengan informasi ter-update dari hasil Sprint 1.

Struktur: **Epic → Feature → Task → Subtask → Checklist**, sesuai instruksi proyek.

---

## EPIC 1: Project Foundation

### Feature 1.1: Monorepo & Workspace Setup

#### Task 1.1.1: Inisialisasi Struktur Monorepo

- **Deskripsi**: Membuat struktur direktori monorepo sesuai Engineering Playbook §1.4 & §19.
- **Acceptance Criteria**: Struktur folder (`apps/`, `services/`, `pkg/`, `proto/`, `migrations/`, `deployments/`, `docs/`, `scripts/`, `.github/`) ada dan sesuai konvensi.
- **Definition of Done**: `tree -L 2` menunjukkan struktur sesuai Playbook §1.4; README.md root berisi deskripsi singkat proyek & cara menjalankan.
- **Dependency**: Tidak ada (task pertama proyek).
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Buat direktori top-level sesuai Playbook §1.4
- [x] Inisialisasi git repository, `.gitignore` (Node, Go, IDE, `.env`)
- [x] Buat `README.md` awal
- [x] Commit pertama: `chore: initialize monorepo structure`

#### Task 1.1.2: Setup Go Workspace

- **Deskripsi**: Konfigurasi `go.work` menyatukan `apps/api` sebagai module pertama.
- **Acceptance Criteria**: `go build ./...` berhasil dari root; `go.work` mendaftarkan `apps/api`.
- **Definition of Done**: Binary skeleton `apps/api` dapat di-build tanpa error.
- **Dependency**: Task 1.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] `go work init`
- [x] `cd apps/api && go mod init github.com/<org>/nexus/apps/api`
- [x] `go work use ./apps/api`
- [x] Buat `cmd/server/main.go` skeleton (`func main() { fmt.Println("nexus api") }`)
- [x] Verifikasi `go build ./...` sukses dari root

#### Task 1.1.3: Setup pnpm Workspace (Frontend)

- **Deskripsi**: Inisialisasi `apps/web` dengan Nuxt 4, dikonfigurasi sebagai pnpm workspace.
- **Acceptance Criteria**: `pnpm install` berhasil dari root; `pnpm --filter web dev` menjalankan Nuxt dev server.
- **Definition of Done**: Nuxt default page dapat diakses di `localhost:3000`.
- **Dependency**: Task 1.1.1
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Buat `pnpm-workspace.yaml` di root (`packages: ["apps/*"]`)
- [x] `pnpm dlx nuxi init apps/web` (Nuxt 4 + TypeScript)
- [x] Tambahkan TailwindCSS, Pinia, VueUse sesuai stack
- [x] Verifikasi dev server berjalan

#### Task 1.1.4: Setup Tooling Enforcement (golangci-lint, Biome, Husky, Commitlint)

- **Deskripsi**: Konfigurasi seluruh tooling wajib sesuai Playbook §21.
- **Acceptance Criteria**: `golangci-lint run` dan `biome check` berjalan tanpa error konfigurasi; commit dengan pesan tidak sesuai Conventional Commit ditolak Husky.
- **Definition of Done**: Percobaan commit dengan pesan salah format gagal secara otomatis; commit dengan format benar berhasil.
- **Dependency**: Task 1.1.2, 1.1.3
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Buat `.golangci.yml` (aktifkan `depguard`, `gosec`, `errcheck`, `govet`, `gofmt`)
- [x] Buat `biome.json` di `apps/web`
- [x] Install Husky (`pnpm dlx husky-init`), buat hook `commit-msg`
- [x] Install & konfigurasi `@commitlint/config-conventional`
- [x] Test: commit pesan salah format → ditolak; commit benar → diterima

---

### Feature 1.2: Docker Compose Environment

#### Task 1.2.1: Dockerfile untuk apps/api

- **Deskripsi**: Multi-stage Dockerfile (build stage Go, run stage minimal/distroless).
- **Acceptance Criteria**: Image ter-build berhasil, ukuran final < 50MB (distroless/alpine base).
- **Definition of Done**: `docker build -t nexus-api ./apps/api` sukses, `docker run` menjalankan binary tanpa error.
- **Dependency**: Task 1.1.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Stage 1: `golang:1.25` — `go build`
- [x] Stage 2: `gcr.io/distroless/static` atau `alpine` minimal — copy binary
- [x] Verifikasi image build & run

#### Task 1.2.2: docker-compose.yml Lengkap

- **Deskripsi**: Definisikan seluruh service (Traefik, api, web, postgres, redis, minio) sesuai Deployment Architecture §1.
- **Acceptance Criteria**: `docker compose up` menjalankan seluruh service tanpa error, saling terhubung dengan benar (network internal).
- **Definition of Done**: `docker compose ps` menunjukkan seluruh service `healthy`/`running`; API dapat diakses via Traefik di `localhost/api`.
- **Dependency**: Task 1.2.1, 1.1.3
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Definisikan service `traefik` dengan label provider Docker
- [x] Definisikan service `postgres` (image `postgres:18`, volume persisten)
- [x] Definisikan service `redis`
- [x] Definisikan service `minio` (+ `mc` init container untuk membuat bucket awal — `nexus-attachments`, `nexus-avatars`)
- [x] Definisikan service `api` dengan label Traefik routing `/api`
- [x] Definisikan service `web` dengan label Traefik routing `/`
- [x] Buat `.env.example` mendokumentasikan seluruh environment variable (Playbook §7.3 konvensi `NEXUS_API_*`)
- [x] Verifikasi seluruh service `docker compose up` sukses

---

### Feature 1.3: CI Pipeline Dasar

#### Task 1.3.1: Workflow `ci.yml`

- **Deskripsi**: GitHub Actions workflow untuk format, lint, vulnerability scan, test, build (Playbook §6.3).
- **Acceptance Criteria**: PR ke `main` men-trigger seluruh tahap; PR dengan kode tidak terformat/lint error gagal CI.
- **Definition of Done**: PR percobaan (dengan sengaja melanggar format) menunjukkan CI merah pada tahap yang benar.
- **Dependency**: Task 1.1.4
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Job `format-check`: `gofmt -l .`, `biome check`
- [x] Job `lint`: `golangci-lint run`, `biome lint`
- [x] Job `security-scan`: `gosec ./...`, `govulncheck ./...`
- [x] Job `test`: `go test ./... -race -cover`
- [x] Job `build`: `go build ./...`
- [x] Branch protection rule: `main` wajib lolos seluruh job sebelum merge
- [x] Uji dengan PR percobaan (kode sengaja salah format) → CI merah pada job yang benar

---

### Feature 1.4: Health Check Endpoint

#### Task 1.4.1: Endpoint `/healthz` dan `/readyz`

- **Deskripsi**: Health check dasar (liveness) dan readiness check (mengecek koneksi DB/Redis).
- **Acceptance Criteria**: `/healthz` selalu 200 selama proses hidup; `/readyz` mengembalikan 503 bila DB/Redis tidak terhubung.
- **Definition of Done**: Simulasi mematikan koneksi PostgreSQL menyebabkan `/readyz` mengembalikan 503, `/healthz` tetap 200.
- **Dependency**: Task 1.2.2
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Implementasi handler `/healthz` (selalu 200, tanpa dependency check)
- [x] Implementasi handler `/readyz` (ping DB `SELECT 1`, ping Redis `PING`)
- [x] Wire ke router Gin
- [x] Test manual: matikan container `postgres`, verifikasi `/readyz` → 503

---

## EPIC 2: Authentication — Register & Fondasi

### Feature 2.1: Database Migration — Identity

#### Task 2.1.1: Migrasi Tabel `users` dan `sessions`

- **Deskripsi**: Menjalankan DDL sesuai Database Design §2.1.
- **Acceptance Criteria**: Migrasi `up` dan `down` berjalan tanpa error; ekstensi `CITEXT` dan fungsi `uuid_generate_v7()` tersedia.
- **Definition of Done**: `migrate up` sukses membuat tabel `users`, `sessions` dengan seluruh constraint & index; `migrate down` membersihkan tanpa sisa.
- **Dependency**: Task 1.2.2 (PostgreSQL berjalan)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Install/aktifkan extension `citext` dan fungsi UUID v7 (library `pg_uuidv7` atau fungsi custom PL/pgSQL)
- [x] Tulis migrasi `20260101000001_create_users_table.sql` (up & down)
- [x] Tulis migrasi `20260101000002_create_sessions_table.sql` (up & down)
- [x] Setup tool migration runner (`golang-migrate` atau setara) di `scripts/`
- [x] Verifikasi `migrate up` dan `migrate down` berjalan bersih di environment lokal

#### Task 2.1.2: Setup sqlc untuk Domain Identity

- **Deskripsi**: Konfigurasi `sqlc.yaml`, tulis query dasar untuk `users`/`sessions`, generate kode Go.
- **Acceptance Criteria**: `sqlc generate` menghasilkan kode type-safe tanpa error.
- **Definition of Done**: Kode hasil generate dapat dipanggil dari test sederhana (insert & select user).
- **Dependency**: Task 2.1.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Buat `sqlc.yaml` di `apps/api`
- [x] Tulis query `CreateUser`, `FindUserByEmailOrUsername`, `CreateSession` (`.sql` file)
- [x] `sqlc generate`, verifikasi kode ter-generate di `internal/identity/infrastructure/`
- [x] Test generate query dengan koneksi database lokal

---

### Feature 2.2: Endpoint Register

#### Task 2.2.1: Domain Layer — User Entity & Value Object

- **Deskripsi**: Implementasi struct `User`, value object `Email`/`Username`/`PasswordHash` sesuai LLD §1.1 pola.
- **Acceptance Criteria**: Value object memvalidasi format saat konstruksi (mis. `NewEmail(s string) (Email, error)`).
- **Definition of Done**: Unit test value object lolos untuk kasus valid & invalid.
- **Dependency**: Task 1.1.2
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Implementasi `internal/identity/domain/user.go`
- [x] Value object `Email` (validasi format)
- [x] Value object `Username` (3-32 karakter, alfanumerik+underscore)
- [x] Unit test seluruh value object (kasus valid & invalid)

#### Task 2.2.2: Argon2id Password Hashing

- **Deskripsi**: Implementasi hashing & verifikasi password sesuai parameter final Security Design §2 (memory 46 MiB, iterations 3, parallelism 2).
- **Acceptance Criteria**: `Hash(password)` menghasilkan encoded string; `Verify(password, hash)` mengembalikan true/false sesuai kecocokan.
- **Definition of Done**: Unit test hash-verify round-trip lolos; salah password mengembalikan false.
- **Dependency**: Task 2.2.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Implementasi `pkg/passwordhash` (generic, tidak bergantung domain — sesuai Playbook §3.2)
- [x] `Hash(password string) (string, error)` dengan parameter final
- [x] `Verify(password, encodedHash string) (bool, error)`
- [x] Unit test: hash-verify round-trip, verifikasi salah password gagal

#### Task 2.2.3: Repository Layer — UserRepository (sqlc adapter)

- **Deskripsi**: Implementasi `UserRepository` interface memakai kode sqlc hasil generate.
- **Acceptance Criteria**: `Create`, `FindByEmailOrUsername` berfungsi terhadap database nyata.
- **Definition of Done**: Integration test (memakai test database) lolos untuk create & find.
- **Dependency**: Task 2.1.2, 2.2.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Implementasi `internal/identity/infrastructure/postgres_user_repository.go`
- [x] Method `Create(ctx, user) error` — tangani error unique constraint → `ErrDuplicateEmail`/`ErrDuplicateUsername`
- [x] Method `FindByEmailOrUsername(ctx, identifier) (*User, error)`
- [x] Integration test dengan test database (docker container terpisah/testcontainers)

#### Task 2.2.4: Service Layer — AuthService.Register

- **Deskripsi**: Orkestrasi validasi, hashing, penyimpanan, sesuai FR-AUTH-01.
- **Acceptance Criteria**: Register sukses mengembalikan user tanpa password hash di response; register dengan email/username duplikat mengembalikan error yang tepat.
- **Definition of Done**: Unit test service (dengan mock repository) lolos untuk kasus sukses & duplikat.
- **Dependency**: Task 2.2.2, 2.2.3
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Implementasi `internal/identity/application/auth_service.go` — method `Register`
- [x] Mock `UserRepository` untuk unit test service
- [x] Unit test: register sukses, register email duplikat, register username duplikat

#### Task 2.2.5: HTTP Handler — `POST /api/v1/auth/register`

- **Deskripsi**: Gin handler, request DTO, validasi (`go-playground/validator`), wiring ke service.
- **Acceptance Criteria**: Sesuai kontrak API Specification §1 — response 201 dengan envelope standar, error 400/409 sesuai Error Code Catalog.
- **Definition of Done**: `curl` manual dan test HTTP (`httptest`) lolos untuk kasus sukses, validasi gagal, dan duplikat.
- **Dependency**: Task 2.2.4
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Implementasi `internal/identity/interface/http/register_handler.go`
- [x] Request DTO + validator tag
- [x] Mapping error domain → HTTP status code & error code (§16 Playbook, §0 API Spec)
- [x] Wire ke router Gin (`internal/platform/router.go`)
- [x] Test HTTP end-to-end (`httptest.NewServer` + test database)

#### Task 2.2.6: Wiring dengan Google Wire

- **Deskripsi**: Setup dependency injection module Identity memakai Google Wire.
- **Acceptance Criteria**: `wire_gen.go` ter-generate tanpa error, `main.go` memakai provider set hasil Wire.
- **Definition of Done**: Aplikasi start tanpa manual wiring eksplisit yang berantakan.
- **Dependency**: Task 2.2.5
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [x] Buat `wire.go` dengan provider set (`NewUserRepository`, `NewAuthService`, `NewRegisterHandler`, dst.)
- [x] `wire` generate → `wire_gen.go`
- [x] `main.go` memanggil `InitializeApp()` hasil Wire
- [x] Verifikasi aplikasi start & endpoint register berfungsi end-to-end

---

### Feature 2.3: Testing & Observability Dasar

#### Task 2.3.1: Structured Logging untuk Auth Flow

- **Deskripsi**: Instrumentasi Zap logger pada titik penting (register berhasil, register gagal validasi/duplikat).
- **Acceptance Criteria**: Log terstruktur dengan field sesuai Playbook §15.2 (tanpa password/sensitive data).
- **Definition of Done**: Review manual output log memastikan tidak ada kebocoran data sensitif.
- **Dependency**: Task 2.2.5
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [ ] Setup Zap logger di `pkg/logger`
- [ ] Tambahkan log `Info` untuk register sukses (tanpa password)
- [ ] Tambahkan log `Warn` untuk register gagal validasi/duplikat
- [ ] Review manual: pastikan tidak ada password/hash ter-log

#### Task 2.3.2: Test Suite Lengkap Sprint 1

- **Deskripsi**: Konsolidasi seluruh test (unit + integration) Sprint 1, pastikan coverage memadai.
- **Acceptance Criteria**: `go test ./... -race -cover` lolos 100% tanpa flaky test; coverage domain Identity minimal 70%.
- **Definition of Done**: CI hijau penuh untuk seluruh Sprint 1 di branch `main`.
- **Dependency**: Seluruh task Epic 2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:

- [ ] Jalankan seluruh test suite lokal, perbaiki test flaky bila ada
- [ ] Cek coverage report (`go test -coverprofile`)
- [ ] Pastikan CI `main` hijau penuh
- [ ] Tag rilis internal `v0.1.0-sprint1` (opsional, untuk penanda milestone)

---

## Ringkasan Keputusan

1. Sprint 1 dipecah menjadi **2 Epic, 7 Feature, 16 Task**, masing-masing dengan Acceptance Criteria dan Definition of Done yang **dapat diverifikasi secara objektif** (bukan subjektif "kelihatannya selesai").
2. Urutan task mengikuti dependency eksplisit (Foundation → Migration → Domain → Repository → Service → Handler → Wiring → Testing) — Clean Architecture diterapkan bahkan dalam urutan pengerjaan task, bukan hanya struktur kode.
3. Total estimasi granular Sprint 1: ± 37 jam kerja aktif, mendukung validasi/kalibrasi terhadap asumsi 2 minggu di Sprint Planning.

## Trade-off yang Diterima

- Estimasi per task dalam jam (bukan story point abstrak) — lebih konkret untuk konteks solo learning, namun kurang fleksibel bila kecepatan kerja bervariasi signifikan hari ke hari; akan dikalibrasi ulang di Sprint Retrospective.

## Risiko Arsitektur

- Total estimasi granular (±37 jam) perlu dibandingkan dengan kapasitas riil 2 minggu — bila kapasitas riil jauh di bawah 37 jam, Sprint 1 perlu diperpanjang atau dipangkas scope (Should-priority item seperti Task Device Management dapat ditunda ke sprint berikutnya tanpa mengorbankan Definition of Done inti).

## Technical Debt yang Sengaja Diterima

- Test coverage domain Identity ditargetkan 70% (bukan 100%) — cukup untuk memvalidasi business logic kritikal tanpa over-investasi waktu testing di Sprint pertama; target dapat dinaikkan di sprint berikutnya berdasarkan pengalaman.

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

Dengan selesainya dokumen ini, **seluruh rangkaian dokumen perencanaan awal (Phase 0-11) telah lengkap** untuk Sprint 1. Proyek Nexus siap dieksekusi.

1. Apakah breakdown task Sprint 1 sudah cukup granular dan actionable untuk langsung mulai coding?
2. Apakah Anda ingin saya membantu memulai implementasi kode (mis. Task 1.1.1 — inisialisasi struktur monorepo), atau Anda akan mengerjakan sendiri dan kembali untuk Sprint Planning/Task Checklist berikutnya (Sprint 2) setelah Sprint 1 selesai?

---

## Changelog

| Versi | Tanggal    | Perubahan                                                                                                                                          |
| ----- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1.0.0 | Draft awal | Dokumen pertama Phase 11, mencakup Sprint 1 (Project Foundation + Awal Authentication) dengan struktur Epic→Feature→Task→Subtask→Checklist lengkap |
