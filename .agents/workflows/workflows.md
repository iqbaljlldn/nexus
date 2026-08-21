---
description:
---

# WORKFLOWS.md — Nexus

> Alur kerja operasional harian: Git, CI/CD, migration, release, sprint. Untuk aturan keras yang tidak boleh dilanggar, lihat `RULES.md`. Untuk konteks proyek, lihat `AGENTS.md`. Detail lengkap tiap topik ada di `docs/01-engineering-playbook.md` dan `docs/12-deployment-architecture.md`.

---

## 1. Alur Kerja Fitur Baru (Feature Workflow)

```
1. Cek Task Checklist aktif (docs/15-task-checklist-sprintN.md) → pilih task
   ↓
2. Cek Definition of Ready:
   - Requirement ada di PRD/SRS? (docs/05-prd.md, docs/06-srs.md)
   - Desain ada di HLD/LLD/DB Design/API Spec? (docs/07-10)
   - Dependency task lain sudah selesai?
   ↓
3. Buat branch: feature/<domain>-<deskripsi-singkat>
   dari `main`
   ↓
4. Implementasi mengikuti urutan layer:
   domain/ → infrastructure/ (repository) → application/ (service) →
   interface/http/ (handler) → wire.go (DI)
   ↓
5. Tulis test seiring implementasi (bukan di akhir):
   - Unit test domain/application layer (mock repository)
   - Integration test infrastructure layer (test database)
   - HTTP test interface layer (httptest)
   ↓
6. Jalankan lokal: gofmt → golangci-lint → go test -race → go build
   ↓
7. Commit dengan Conventional Commit format (lihat §3)
   ↓
8. Buka PR menggunakan template (§4), isi checklist
   ↓
9. CI hijau penuh → self-review checklist → merge (Squash and Merge)
   ↓
10. Update Task Checklist: centang subtask/task yang selesai
```

---

## 2. Alur Kerja Perubahan Skema Database

```
1. Cek Database Design (docs/09-database-design.md) — apakah tabel/kolom sudah dirancang?
   Bila ya: implementasikan sesuai DDL yang ada.
   Bila tidak/perlu penyesuaian: catat sebagai amandemen, update docs/09-database-design.md
   ↓
2. Tulis migration baru:
   migrations/<timestamp>_<verb>_<object>.sql
   - Section up: perubahan skema
   - Section down: rollback lengkap
   ↓
3. Untuk tabel besar (messages, audit_logs) — WAJIB expand-contract:
   a. Migration 1: tambah kolom nullable
   b. Backfill data (batch, terpisah dari migration DDL)
   c. Migration 2: tambah constraint NOT NULL (setelah backfill selesai & diverifikasi)
   ↓
4. Test migration LOKAL: up → verifikasi skema → down → verifikasi rollback bersih
   ↓
5. Update file query .sql terkait (bila ada) → sqlc generate
   ↓
6. Sertakan estimasi waktu eksekusi migration di PR description bila tabel besar
   (bagian dari Checklist Release, docs/01-engineering-playbook.md §14)
```

---

## 3. Git & Commit Workflow

**Branch naming**: `<type>/<scope>-<deskripsi>` — contoh: `feature/message-reply-threading`, `fix/auth-refresh-token-expiry`.

**Commit format** (Conventional Commits, ditegakkan Husky+Commitlint):

```
<type>(<scope>): <description>

[body opsional]

[BREAKING CHANGE: ... — bila ada]
```

Type: `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `chore`, `ci`, `build`, `revert`.

**Merge strategy**: Squash and Merge ke `main` — satu PR = satu commit history yang bersih.

---

## 4. Pull Request Workflow

Template wajib (lihat `docs/01-engineering-playbook.md` §12.2 untuk versi lengkap):

```markdown
## Ringkasan

## Tipe Perubahan

## Domain/Module Terdampak

## Checklist

- [ ] Lolos lint & format lokal
- [ ] Test ditambahkan/diperbarui dan lolos (go test ./... -race)
- [ ] Tidak ada breaking change API/event tanpa versi baru
- [ ] Observability ditambahkan untuk operasi baru yang signifikan
- [ ] Tidak ada secret ter-hardcode
- [ ] Dokumentasi terkait diperbarui

## Referensi

## Catatan Technical Debt (bila ada)
```

**Ukuran PR ideal**: < 400 baris diff. Pecah PR besar per layer logis bila memungkinkan.

---

## 5. CI/CD Pipeline

```
PR dibuka → ci.yml (format → lint → security scan → test -race → build)
   ↓ (lolos semua, merge ke main)
build-and-push.yml → image di-build & di-push ke registry
   ↓
deploy-staging.yml → otomatis deploy ke staging
   ↓
[Manual approval]
   ↓
deploy-production.yml:
   - Tahap 1-2 (Docker Compose/Single VPS): pull + up -d
   - Tahap 3+ (Blue-Green): deploy Green → health check → switch Traefik label →
                              grace period 5 menit → matikan Blue
   - Tahap 5 (Kubernetes): kubectl apply / helm upgrade, Rolling Update native
```

Detail lengkap evolusi 5 tahap deployment: `docs/12-deployment-architecture.md`.

---

## 6. Sprint Workflow (Rolling Wave Planning)

```
1. Sprint dimulai → buka docs/14-sprint-planning.md + docs/15-task-checklist-sprintN.md
   ↓
2. Kerjakan task sesuai urutan dependency (lihat kolom Dependency tiap task)
   ↓
3. Sprint Review: demo terhadap Acceptance Criteria (rujuk FR-ID di SRS)
   ↓
4. Sprint Retrospective: catat kalibrasi estimasi (waktu aktual vs estimasi)
   ↓
5. Sprint berikutnya BELUM didetailkan (Rolling Wave) →
   buat docs/15-task-checklist-sprint<N+1>.md baru berdasarkan:
   - Breakdown garis besar di docs/14-sprint-planning.md §3
   - Pembelajaran nyata dari sprint sebelumnya (velocity, scope adjustment)
```

**Jangan** mendetailkan sprint jauh ke depan sekaligus — ini melanggar prinsip Rolling Wave yang disepakati di `docs/14-sprint-planning.md` §0.

---

## 7. Dokumentasi — Kapan Harus Diupdate

| Perubahan                           | Dokumen yang Wajib Diupdate                                                                |
| ----------------------------------- | ------------------------------------------------------------------------------------------ |
| Keputusan teknologi baru/berubah    | `docs/03-adr.md` — tambah ADR baru atau revisi (naikkan versi, catat di Changelog)         |
| Requirement baru/berubah            | `docs/05-prd.md` dan/atau `docs/06-srs.md`                                                 |
| Perubahan desain domain/event       | `docs/07-hld.md`                                                                           |
| Perubahan interface/algoritma kunci | `docs/08-lld.md`                                                                           |
| Perubahan skema tabel               | `docs/09-database-design.md`                                                               |
| Endpoint baru/berubah               | `docs/10-api-specification.md`                                                             |
| Selesai sprint                      | Task checklist sprint berikutnya dibuat baru; `docs/AGENTS.md` §7 (Status Proyek) diupdate |

**Aturan konsistensi** (dari `docs/02-vision-document.md`): dokumen fase sebelumnya adalah referensi wajib fase berikutnya. Bila ditemukan konflik antara dokumen dan implementasi/permintaan baru: **jelaskan konflik → jelaskan dampak → beri rekomendasi → tunggu keputusan** sebelum mengubah dokumen source-of-truth.

---

## 8. Alur Kerja Insiden/Bug Production (dipersiapkan sejak Milestone 15+)

```
1. Alert dari Prometheus/Grafana atau laporan manual
   ↓
2. Cek dashboard RED method (Rate, Errors, Duration) per service
   ↓
3. Trace lintas proses via trace_id (OpenTelemetry, W3C Trace Context)
   ↓
4. Perbaikan → PR dengan tag `fix:` → CI → deploy via Blue-Green (rollback instan bila perlu)
   ↓
5. Post-mortem singkat: root cause, mengapa lolos test sebelumnya, aksi pencegahan
```
