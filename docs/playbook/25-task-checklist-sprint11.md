# Detailed Task Checklist — Sprint 11
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 11: Optimization — Milestone 11 Checkpoint)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `04-learning-roadmap.md` (Milestone 11), `13-development-roadmap.md` (Release 4), `08-lld.md` (seluruh debt bertanda "dituning di Milestone 11"), `12-deployment-architecture.md` (§8 Resource Allocation)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen & Sifat Sprint yang Berbeda

Sprint 11 **berbeda karakter** dari sprint sebelumnya — bukan membangun fitur baru, melainkan **checkpoint wajib** sebelum Event-Driven Migration (Learning Roadmap M11: "memastikan fondasi Modular Monolith solid sebelum kompleksitas event-driven ditambahkan"). Struktur task di sini mengikuti siklus **ukur → identifikasi → perbaiki → ukur ulang**, bukan "desain → implementasi → test" seperti sprint fitur.

**Prasyarat**: Release 1-3 selesai (Sprint 1-10) — sistem sudah punya cukup permukaan (auth, messaging, DM, upload, presence, notification, voice, video) untuk load test yang representatif.

**Sprint Goal** (persis Development Roadmap §2, Release 4): Load test mendekati target NFR awal (§3.1 SRS); **minimal 3 bottleneck teridentifikasi dan diperbaiki dengan bukti before/after benchmark** — bukan optimasi spekulatif.

**Debt yang wajib dituntaskan sprint ini** (dikumpulkan dari seluruh dokumen sebelumnya, bukan diciptakan baru):

| Sumber | Debt |
|---|---|
| LLD §2.9 | Ukuran buffer `sendCh` WebSocket (nilai awal 64) |
| Task 10.4.1 (Sprint 6) | Concurrency Asynq worker (nilai awal, belum divalidasi) |
| Sprint 3 Task 3.5.2 | Cache Permission Resolver (bila belum dikerjakan) |
| Sprint 9 revisi | Baris `voice_participants` menggantung (insert optimistic tanpa konfirmasi webhook) |
| Sprint 10 Task 15.3.2 | Batas partisipan video (nilai awal 10, asumsi) |
| Security Design §2 | Parameter Argon2id (perlu divalidasi ulang dengan benchmark nyata di server produksi-like) |
| Database Design §7 | Index GIN full-text search (belum dioptimasi composite/partial) |

---

## EPIC 16: Performance Optimization

### Feature 16.1: Load Testing Infrastructure

#### Task 16.1.1: Setup Tool Load Test (k6) & Skenario Dasar

- **Deskripsi**: Pilih tool load testing (k6 direkomendasikan — scriptable JavaScript, output metrik langsung dapat diproses).
- **Acceptance Criteria**: Skrip k6 untuk 3 alur kritikal: (a) login → kirim pesan berulang, (b) list message history (cursor pagination), (c) WebSocket connect + terima broadcast.
- **Definition of Done**: `k6 run` menghasilkan laporan p50/p95/p99 latency dan error rate untuk ketiga skenario terhadap environment staging-like.
- **Dependency**: Release 1-3 selesai
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Install & konfigurasi k6
- [ ] Skrip `k6/login-and-send-message.js`
- [ ] Skrip `k6/list-message-history.js`
- [ ] Skrip `k6/websocket-broadcast.js`
- [ ] Jalankan baseline (concurrent user rendah, mis. 50) untuk memastikan skrip berfungsi benar sebelum eskalasi beban

#### Task 16.1.2: Load Test Bertahap Menuju Target NFR

- **Deskripsi**: Eskalasi concurrent user secara bertahap (50 → 500 → 2.000 → 10.000) sesuai target NFR §3.1 SRS, mencatat titik di mana response time/error rate mulai melampaui target.
- **Acceptance Criteria**: Laporan berisi grafik latency vs concurrent user, titik degradasi teridentifikasi jelas (bukan "terasa lambat", tapi angka konkret: mis. "p95 melampaui 300ms pada 3.200 concurrent user").
- **Definition of Done**: Dokumen hasil load test (`docs/reports/sprint11-load-test-baseline.md`) berisi data mentah + interpretasi.
- **Dependency**: Task 16.1.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam (termasuk waktu tunggu eksekusi test bertahap)
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Jalankan load test bertahap, catat metrik tiap tahap
- [ ] Identifikasi titik degradasi per skenario (a/b/c)
- [ ] Tulis laporan baseline dengan data mentah

---

### Feature 16.2: Profiling Infrastructure (pprof)

#### Task 16.2.1: Aktifkan `pprof` Endpoint (Terproteksi)

- **Deskripsi**: Sesuai Learning Roadmap — pprof sebagai standar profiling proyek.
- **Acceptance Criteria**: Endpoint `/debug/pprof/*` **tidak** diekspos publik (hanya bind ke port internal terpisah atau dilindungi network policy/IP allowlist — pelanggaran ini akan jadi celah keamanan nyata bila lolos ke production tanpa proteksi).
- **Definition of Done**: `go tool pprof` dapat mengambil profile CPU/memory/goroutine dari environment staging tanpa dapat diakses dari internet publik.
- **Dependency**: Tidak ada (independen)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Import `net/http/pprof` di binary terpisah/port terpisah (bukan port publik API)
- [ ] Verifikasi port pprof tidak ter-expose di Traefik routing (cek label Docker Compose)
- [ ] Test: `go tool pprof http://localhost:<port-internal>/debug/pprof/profile` berhasil dari jaringan internal

#### Task 16.2.2: Ambil Profile Saat Load Test Berjalan

- **Deskripsi**: Kombinasikan Task 16.1.2 dan 16.2.1 — ambil CPU/memory/goroutine profile **selama** load test pada beban mendekati titik degradasi.
- **Acceptance Criteria**: Profile flame graph dihasilkan, menunjukkan fungsi mana yang paling banyak menghabiskan CPU/alokasi memori.
- **Definition of Done**: Minimal 3 flame graph (CPU, memory, goroutine) tersimpan sebagai lampiran laporan Sprint 11.
- **Dependency**: Task 16.1.2, Task 16.2.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Jalankan load test pada beban mendekati titik degradasi, ambil profile paralel
- [ ] Generate flame graph (`go tool pprof -http`)
- [ ] Simpan sebagai lampiran laporan

---

### Feature 16.3: Benchmark Suite untuk Komponen Kritikal

#### Task 16.3.1: Benchmark `PermissionResolver.Resolve`

- **Deskripsi**: Komponen yang sudah ditandai risiko tertinggi sejak Sprint 3 (Task 3.5.1) — wajib dibuktikan performanya dengan data, bukan asumsi.
- **Acceptance Criteria**: `go test -bench=BenchmarkPermissionResolve` mengukur ns/op dan alokasi memori untuk kasus member dengan 1 role vs 10 role.
- **Definition of Done**: Hasil benchmark terdokumentasi; **keputusan eksplisit** dibuat: aktifkan cache (Task 3.5.2, bila belum) atau tidak, berdasarkan angka nyata (bukan asumsi "pasti perlu cache").
- **Dependency**: Task 3.5.1 (Sprint 3)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis `BenchmarkPermissionResolve` (kasus 1 role, 5 role, 10 role)
- [ ] Jalankan, catat ns/op dan `allocs/op`
- [ ] **Keputusan**: bila > threshold yang mengganggu (mis. > 1ms, mempengaruhi p95 endpoint kirim pesan secara signifikan) → implementasikan Task 3.5.2 (cache) sekarang; bila tidak → dokumentasikan keputusan "tidak perlu cache saat ini" dengan data pendukung

#### Task 16.3.2: Benchmark Cursor Pagination vs Skenario Offset (Pembuktian ADR)

- **Deskripsi**: Validasi empiris keputusan LLD §2.2/Playbook §17.2 (cursor-based dipilih atas offset-based) — bukan sekadar diasumsikan benar dari teori.
- **Acceptance Criteria**: Benchmark `ListMessagesByChannel` (cursor) vs implementasi offset tandingan (dibuat khusus untuk perbandingan, tidak masuk kode produksi) pada dataset besar (mis. 1 juta baris pesan simulasi).
- **Definition of Done**: Data pembuktian nyata (bukan asumsi) bahwa cursor lebih baik pada skala besar — didokumentasikan sebagai lampiran ADR-final (opsional update `03-adr.md` dengan referensi data empiris).
- **Dependency**: Task 7.1.3 (Sprint 4)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Should

**Subtask & Checklist**:
- [ ] Seed database test dengan ~1 juta baris `messages` (script seeding terpisah)
- [ ] Benchmark cursor query pada offset ke-0, ke-500.000, ke-990.000
- [ ] Benchmark query offset-based tandingan pada posisi yang sama
- [ ] Dokumentasikan perbandingan (harus menunjukkan offset melambat signifikan di posisi jauh, cursor tetap konstan)

---

### Feature 16.4: Identifikasi & Perbaikan Bottleneck (Minimal 3 — Gerbang Sprint Goal)

#### Task 16.4.1: Bottleneck #1 — Analisis & Perbaikan Berdasarkan Profiling Nyata

- **Deskripsi**: **Tidak ditentukan di awal** — kandidat ditentukan dari hasil Task 16.2.2 (flame graph nyata), bukan diasumsikan sebelum data ada. Kandidat paling mungkin berdasarkan desain sistem sejauh ini: kontensi lock `sync.RWMutex` di `ConnectionRegistry` (LLD §2.9) saat broadcast volume tinggi.
- **Acceptance Criteria**: Benchmark before/after menunjukkan perbaikan terukur (mis. penurunan p95 latency broadcast, penurunan CPU time di flame graph).
- **Definition of Done**: PR berisi: (1) data before (flame graph/benchmark), (2) perubahan kode, (3) data after, (4) analisis mengapa perbaikan bekerja.
- **Dependency**: Task 16.2.2
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 4 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Identifikasi bottleneck #1 dari flame graph (jangan berasumsi sebelum data ada)
- [ ] Ukur baseline (benchmark/profile before)
- [ ] Implementasikan perbaikan
- [ ] Ukur ulang (benchmark/profile after), bandingkan
- [ ] Dokumentasikan di laporan Sprint 11

#### Task 16.4.2: Bottleneck #2 — Analisis & Perbaikan Berdasarkan Profiling Nyata

- **Deskripsi**: Kandidat kedua paling mungkin: query N+1 tersembunyi (mis. saat membangun response daftar member dengan role masing-masing, atau daftar pesan dengan reaction count).
- **Acceptance Criteria**: Sama seperti Task 16.4.1 — bukti before/after.
- **Definition of Done**: Sama seperti Task 16.4.1.
- **Dependency**: Task 16.2.2
- **Estimasi Kesulitan**: Tinggi
- **Estimasi Waktu**: 3.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Audit endpoint list (member, message dengan reaction, dsb.) untuk pola N+1 — cek log query (aktifkan query logging sementara di environment test)
- [ ] Ukur baseline
- [ ] Perbaiki (batch query / JOIN yang tepat, sesuai kasus yang ditemukan)
- [ ] Ukur ulang, dokumentasikan

#### Task 16.4.3: Bottleneck #3 — Analisis & Perbaikan Berdasarkan Profiling Nyata

- **Deskripsi**: Kandidat ketiga: ukuran/konfigurasi connection pool database (`pgxpool.MaxConns`) — apakah terlalu kecil (bottleneck antrian koneksi) atau terlalu besar (overhead tanpa manfaat, atau membebani PostgreSQL berlebihan).
- **Acceptance Criteria**: Sama seperti di atas — bukti before/after dengan variasi `MaxConns`.
- **Definition of Done**: Nilai `MaxConns` final ditentukan berdasarkan data, didokumentasikan sebagai konfigurasi resmi (bukan nilai default library).
- **Dependency**: Task 16.1.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Load test dengan variasi `MaxConns` (mis. 10, 25, 50, 100)
- [ ] Catat throughput & latency tiap variasi
- [ ] Tetapkan nilai final berdasarkan data, update `.env.example` dan Deployment Architecture §8

---

### Feature 16.5: Tuntaskan Debt Tuning (Tabel §0)

#### Task 16.5.1: Tuning Buffer WebSocket, Asynq Concurrency, Batas Partisipan Video

- **Deskripsi**: Menuntaskan seluruh item debt di tabel §0 yang eksplisit menunggu Milestone 11.
- **Acceptance Criteria**: Setiap nilai (buffer `sendCh`, Asynq concurrency, batas partisipan video) memiliki **justifikasi berbasis data** (dari load test/benchmark sprint ini), bukan sekadar dinaikkan/diturunkan tanpa dasar.
- **Definition of Done**: Nilai final terdokumentasikan di kode (komentar referensi ke laporan Sprint 11) dan di `12-deployment-architecture.md` §8 (update resource allocation bila berubah signifikan).
- **Dependency**: Task 16.4.1, 16.1.2
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tuning `sendCh` buffer size berdasarkan data broadcast load test
- [ ] Tuning Asynq `Concurrency` berdasarkan data pemrosesan media/notification load
- [ ] Validasi ulang batas partisipan video (10) — cukup atau perlu disesuaikan berdasarkan resource nyata
- [ ] Update `12-deployment-architecture.md` §8 bila estimasi resource awal meleset signifikan dari data nyata

#### Task 16.5.2: Validasi Ulang Parameter Argon2id

- **Deskripsi**: Security Design §2 mencatat parameter final "akan divalidasi ulang dengan benchmark nyata di Milestone 11".
- **Acceptance Criteria**: Benchmark waktu hashing (memory 46 MiB/iterations 3/parallelism 2) terhadap CPU produksi-like — pastikan tidak menjadi bottleneck nyata pada endpoint login dengan concurrent user tinggi.
- **Definition of Done**: Keputusan eksplisit: parameter dipertahankan (dengan data pendukung) atau disesuaikan.
- **Dependency**: Task 2.2.2 (Sprint 2)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Should

**Subtask & Checklist**:
- [ ] Benchmark `Hash`/`Verify` pada environment produksi-like
- [ ] Load test endpoint login dengan concurrent user tinggi, cek apakah Argon2id jadi bottleneck CPU nyata
- [ ] Dokumentasikan keputusan akhir (pertahankan/sesuaikan)

---

### Feature 16.6: Database Index Review

#### Task 16.6.1: `EXPLAIN ANALYZE` Query Kritikal

- **Deskripsi**: Sesuai Database Design §7 — verifikasi index yang sudah dirancang benar-benar dipakai query planner, terutama full-text search (debt eksplisit di §7 Database Design).
- **Acceptance Criteria**: Seluruh query di §7 Database Design (list message per channel, resolusi permission, search full-text, member list, DM uniqueness check) di-`EXPLAIN ANALYZE` terhadap data volume besar (hasil seeding Task 16.3.2).
- **Definition of Done**: Laporan berisi: query mana yang sudah optimal (Index Scan), query mana yang masih Seq Scan tak terduga (perlu index tambahan/composite), dengan perbaikan diterapkan untuk yang terakhir.
- **Dependency**: Task 16.3.2 (data seeding volume besar)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 3 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] `EXPLAIN ANALYZE` seluruh query di tabel Database Design §7
- [ ] Identifikasi Seq Scan tak terduga
- [ ] Tambah/sesuaikan index bila perlu (migration baru, sesuai konvensi Playbook §7.7)
- [ ] Verifikasi ulang setelah perubahan index

---

### Feature 16.7: Laporan & Dokumentasi Sprint 11

#### Task 16.7.1: Konsolidasi Laporan Optimization

- **Deskripsi**: Satu dokumen ringkas yang menjadi rujukan hasil sprint ini.
- **Acceptance Criteria**: Mencakup: hasil load test baseline vs setelah optimasi, 3 bottleneck yang diperbaiki (before/after), keputusan tuning (§16.5), hasil index review.
- **Definition of Done**: `docs/reports/sprint11-optimization-report.md` selesai, ditinjau, dan **menjadi bukti** bahwa Sprint Goal Development Roadmap tercapai (bukan klaim tanpa data).
- **Dependency**: Seluruh task Epic 16
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis laporan konsolidasi
- [ ] Update `docs/AGENTS.md` §7 — Sprint 11 selesai, sistem siap masuk Sprint 12 (Event Driven Migration, Milestone 12)

---

## Ringkasan Keputusan

1. Sprint 11 mencakup **1 Epic, 7 Feature, 13 task**, dengan sifat berbeda dari sprint sebelumnya: siklus ukur-identifikasi-perbaiki-ukur ulang, bukan bangun fitur baru.
2. **Kandidat bottleneck sengaja tidak ditentukan di awal** (Task 16.4.1-16.4.3) — harus ditemukan dari data profiling nyata (Task 16.2.2), konsisten dengan prinsip "jangan optimasi spekulatif" (Learning Roadmap M11 Best Practice).
3. Seluruh debt tuning yang sudah dikumpulkan sejak Sprint 3, 6, 9, 10 (tabel §0) **wajib** dituntaskan dengan data di sprint ini — bukan dibiarkan menumpuk lebih jauh.
4. Benchmark cursor vs offset pagination (Task 16.3.2) memberi **pembuktian empiris** terhadap keputusan arsitektur yang sebelumnya hanya berbasis teori (Playbook §17.2) — memperkuat kredibilitas ADR dengan data nyata.

## Trade-off yang Diterima

- Task 16.3.2 dan 16.5.2 berprioritas *Should* — dapat digeser bila kapasitas sprint tidak cukup, namun sangat direkomendasikan dikerjakan karena memperkuat kepercayaan terhadap keputusan arsitektur sebelumnya dengan data, bukan asumsi.

## Risiko Arsitektur

- Bila hasil load test (Task 16.1.2) menunjukkan sistem **jauh** di bawah target NFR (bukan hanya bottleneck kecil), ini adalah sinyal bahwa keputusan arsitektur fundamental (bukan hanya tuning) perlu ditinjau ulang — skenario ini memerlukan perpanjangan Sprint 11 di luar estimasi 2 minggu awal (Development Roadmap), dan harus dilaporkan sebagai revisi Development Roadmap, bukan dipaksakan selesai sesuai jadwal.

## Technical Debt yang Sengaja Diterima

- Bila hanya 3 bottleneck (minimum Sprint Goal) yang sempat ditangani padahal profiling menemukan lebih banyak kandidat, sisanya dicatat eksplisit di laporan sebagai backlog optimasi lanjutan — tidak semua temuan harus diselesaikan di satu sprint (YAGNI diterapkan pada optimasi itu sendiri: prioritaskan yang berdampak nyata terhadap NFR).

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah tool **k6** dapat diterima untuk load testing, atau ada preferensi tool lain (mis. Vegeta, Gatling)?
2. Apakah pendekatan "kandidat bottleneck tidak ditentukan di awal, murni dari data profiling" dapat diterima — meski ini berarti Task 16.4.1-16.4.3 punya ketidakpastian lebih tinggi dibanding task-task sebelumnya yang sudah jelas scope-nya sejak awal?
3. Dengan Sprint 11 selesai didetailkan, ini adalah **sprint terakhir sebelum Event-Driven Migration (Milestone 12, Release 4 lanjutan)**. Lanjut ke **Sprint 12**, atau berhenti dulu — mengingat checkpoint "Solid Checkpoint" (Development Roadmap §3) baru benar-benar tercapai setelah Sprint 12 (Outbox Pattern) selesai?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 11: 1 Epic, 7 Feature, 13 task. Mengkonsolidasikan seluruh technical debt tuning dari Sprint 3-10, menerapkan siklus ukur-identifikasi-perbaiki-ukur ulang untuk minimal 3 bottleneck sesuai Sprint Goal Development Roadmap |
