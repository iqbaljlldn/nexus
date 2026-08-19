# Detailed Task Checklist — Sprint 10
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 11 — Detailed Task Checklist (Sprint 10: Video)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `23-task-checklist-sprint9.md` (v1.1), `06-srs.md` (Learning Roadmap M10), `07-hld.md` (§2.12, revisi), `09-database-design.md` (§2.7, revisi), `10-api-specification.md` (§3), `12-deployment-architecture.md` (§8)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cakupan Dokumen, Prasyarat, dan Temuan Penting

**Prasyarat**: Sprint 9 (Voice) selesai, termasuk revisi v1.1 (skema `voice_participants` disederhanakan — channel = room permanen, tanpa entity sesi terpisah).

**Sprint Goal**: User dapat join video channel dengan kamera dan/atau screen share; bandwidth dikelola adaptif (simulcast) via konfigurasi LiveKit.

**Temuan penting saat perencanaan sprint ini**: berkat revisi Sprint 9 (channel = room langsung, tabel `voice_participants` tidak menyimpan apapun yang spesifik-tipe), **hampir seluruh infrastruktur Sprint 9 sudah generic untuk voice DAN video** — tidak ada perbedaan mendasar antara "join voice channel" dan "join video channel" dari sisi backend (`LiveKitTokenService`, webhook sync, `voice_participants` — semuanya sudah channel-type-agnostic). Perbedaan video hanya ada di **track yang dipublikasikan** (kamera/screen share, ditentukan client-side lewat LiveKit Client SDK) dan **konfigurasi resource** (simulcast, batas partisipan).

**Keputusan konsekuensi**: daripada membuat endpoint `/channels/{id}/video/join` terpisah yang isinya identik dengan `/channels/{id}/voice/join`, endpoint **digeneralisasi** menjadi `/channels/{id}/rtc/join` (RTC = Real-Time Communication, mencakup voice DAN video) — **amandemen API Specification**, mengurangi duplikasi kode nyata, bukan sekadar penamaan.

---

## EPIC 15: Video

### Feature 15.1: Generalisasi Endpoint Voice/Video (Amandemen API Spec)

#### Task 15.1.1: Rename & Generalisasi — `POST /channels/{id}/rtc/join`, `GET /channels/{id}/rtc/participants`

- **Deskripsi**: Refactor endpoint Sprint 9 (`/voice/join`, `/voice/participants`) menjadi `/rtc/join`, `/rtc/participants` — berfungsi identik untuk channel tipe `voice` maupun `video` (handler sudah tidak peduli tipe channel, hanya memvalidasi bahwa tipe termasuk salah satu dari keduanya).
- **Acceptance Criteria**: Endpoint lama (`/voice/join`) **dihapus** (bukan dipertahankan sebagai alias — proyek masih pra-rilis publik, tidak ada backward compatibility yang perlu dijaga, sesuai konteks pembelajaran). Validasi `channel.type IN ('voice', 'video')`.
- **Definition of Done**: Test regresi Sprint 9 (Task 14.5.1) di-update memakai path baru dan tetap lolos; test tambahan untuk channel tipe `video` dengan alur identik.
- **Dependency**: Sprint 9 selesai
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1.5 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Rename handler & route: `/voice/join` → `/rtc/join`, `/voice/participants` → `/rtc/participants`
- [ ] Update validasi tipe channel: terima `voice` DAN `video`
- [ ] Update `10-api-specification.md` — catat amandemen (endpoint lama dihapus, alasan dijelaskan)
- [ ] Update test Sprint 9 (Task 14.5.1) ke path baru, tambah skenario channel tipe `video`

---

### Feature 15.2: Screen Share Track Permission

#### Task 15.2.1: Permission Flag `SHARE_SCREEN` & Validasi di Token

- **Deskripsi**: Screen share adalah track tambahan (bukan endpoint terpisah) — LiveKit Client SDK yang menangani publish track dari sisi client, backend hanya perlu memastikan **grant token** mengizinkan publish track video (kamera dan/atau screen share memakai grant yang sama di level LiveKit — `CanPublish`), namun proyek ingin **membedakan** siapa boleh screen share (mis. dibatasi permission khusus, bukan otomatis semua yang bisa join video juga boleh screen share).
- **Acceptance Criteria**: Tambah flag permission baru `SHARE_SCREEN` (perluasan `PermissionFlag`, Task 3.4.2 Sprint 3 — YAGNI pattern yang sama, flag ditambah sesuai kebutuhan fitur). Token yang di-generate menyertakan grant `CanPublishSources` yang mencakup screen share **hanya** bila user punya permission ini.
- **Definition of Done**: Test: user dengan `SHARE_SCREEN` mendapat token dengan grant screen share; user tanpa permission tersebut mendapat token yang secara eksplisit menolak publish source screen share (LiveKit menegakkan ini di level SFU, bukan hanya UI hiding tombol — konsisten dengan prinsip "authorization selalu di backend", di sini diperkuat karena LiveKit sendiri yang menegakkan).
- **Dependency**: Task 14.3.1 (Sprint 9 — `LiveKitTokenService`), Task 3.4.2 (Sprint 3 — permission flags)
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tambah flag `SHARE_SCREEN` ke `internal/workspace/domain/permission.go`
- [ ] Perluas `LiveKitTokenService` — cek permission `SHARE_SCREEN` sebelum menyertakan grant `CanPublishSources` untuk screen share di token
- [ ] Test: token dengan & tanpa grant screen share sesuai permission

---

### Feature 15.3: Simulcast & Resource Limit

#### Task 15.3.1: Konfigurasi Simulcast di LiveKit Room

- **Deskripsi**: Learning Roadmap M10 — adaptive bitrate streaming, dikonfigurasi lewat LiveKit (bukan diimplementasikan sendiri).
- **Acceptance Criteria**: Konfigurasi simulcast (jumlah layer kualitas) diatur di level LiveKit server config (`livekit.yaml`), bukan per-request — konsisten dengan prinsip "abstraksi LiveKit dipakai sepenuhnya, bukan direimplementasikan" (ADR-005).
- **Definition of Done**: Verifikasi manual: room video dengan simulcast aktif menunjukkan multiple layer kualitas di LiveKit dashboard/log.
- **Dependency**: Task 14.1.1 (Sprint 9 — LiveKit server config)
- **Estimasi Kesulitan**: Mudah
- **Estimasi Waktu**: 1 jam
- **Prioritas**: Should

**Subtask & Checklist**:
- [ ] Update `livekit.yaml` — aktifkan simulcast dengan layer sesuai kapasitas infrastruktur proyek (bukan default cloud-scale — Deployment Architecture §8 mengingatkan resource terbatas)
- [ ] Verifikasi manual via LiveKit dashboard

#### Task 15.3.2: Batas Partisipan Video per Channel

- **Deskripsi**: Mitigasi bandwidth/CPU membengkak tak terkendali (Learning Roadmap M10 — kesalahan umum yang dihindari).
- **Acceptance Criteria**: Channel video memiliki batas partisipan aktif dengan video track menyala (nilai awal: 10, dapat dikonfigurasi via env var `NEXUS_API_MAX_VIDEO_PARTICIPANTS`) — melebihi batas, join berikutnya ditolak `422 BUSINESS_RULE_VIOLATION` (dapat tetap join sebagai audio-only, bukan ditolak total — namun audio-only untuk channel video di luar scope Sprint 10 minimal, dicatat sebagai debt bila tidak sempat).
- **Definition of Done**: Test: join ke-11 dengan video (setelah 10 aktif) → ditolak dengan pesan jelas.
- **Dependency**: Task 15.1.1
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Cek jumlah `voice_participants` aktif (`left_at IS NULL`) untuk channel sebelum generate token, terhadap `NEXUS_API_MAX_VIDEO_PARTICIPANTS`
- [ ] Response `422 BUSINESS_RULE_VIOLATION` dengan pesan jelas bila terlampaui
- [ ] Test: batas terpenuhi, join ke-11 ditolak

---

### Feature 15.4: Integration Test End-to-End Sprint 10

#### Task 15.4.1: Skenario Penuh — Video Join, Screen Share Grant, Resource Limit

- **Deskripsi**: Verifikasi Sprint Goal.
- **Acceptance Criteria**: Alur: User A (punya `SHARE_SCREEN`) join channel video → token dengan grant screen share; User B (tanpa permission) join channel sama → token tanpa grant screen share; simulasikan 10 partisipan aktif → partisipan ke-11 ditolak.
- **Definition of Done**: Test hijau konsisten 3x run berturut.
- **Dependency**: Seluruh task Epic 15
- **Estimasi Kesulitan**: Sedang
- **Estimasi Waktu**: 2 jam
- **Prioritas**: Must

**Subtask & Checklist**:
- [ ] Tulis skenario penuh (reuse test harness Sprint 9, path `/rtc/join` baru)
- [ ] Jalankan 3x berturut, pastikan tidak flaky
- [ ] **Jalankan ulang regression test Sprint 9** (dengan path baru) — pastikan tidak ada yang rusak akibat rename endpoint
- [ ] Update `docs/AGENTS.md` §7 — Sprint 10 selesai, **Release 3 (Engagement) selesai penuh** (Notification+Voice+Video)

---

## Ringkasan Keputusan

1. Sprint 10 mencakup **1 Epic, 4 Feature, 6 task** — jauh lebih ringkas dari sprint-sprint sebelumnya, karena **temuan penting**: revisi Sprint 9 (channel = room permanen) membuat infrastruktur voice sudah generic untuk video tanpa perubahan besar.
2. **Amandemen API Specification**: endpoint `/voice/join`, `/voice/participants` digeneralisasi menjadi `/rtc/join`, `/rtc/participants` — mengurangi duplikasi kode nyata (bukan sekadar rename kosmetik), berlaku untuk channel tipe `voice` maupun `video`.
3. Screen share **bukan endpoint terpisah** — cukup grant tambahan di token (`SHARE_SCREEN` permission), penegakan aktual dilakukan LiveKit sendiri di level SFU (bukan hanya UI hiding), konsisten dengan prinsip authorization backend-enforced.
4. Simulcast dikonfigurasi di level **LiveKit server config**, bukan diimplementasikan sendiri — konsisten dengan rationale ADR-005 (LiveKit dipilih agar tidak membangun SFU dari nol).

## Trade-off yang Diterima

- Task 15.3.2 (batas partisipan) menolak total join video ke-11 (bukan fallback ke audio-only) — audio-only fallback untuk channel video dicatat sebagai debt, bukan diimplementasikan penuh di sprint ini (kompleksitas tambahan yang tidak proporsional untuk MVP video).
- Endpoint lama (`/voice/join`) dihapus tanpa periode transisi/alias — dapat diterima karena proyek belum rilis publik dengan konsumen API eksternal.

## Risiko Arsitektur

- Batas partisipan video (Task 15.3.2, nilai awal 10) adalah **asumsi**, belum divalidasi terhadap kapasitas infrastruktur nyata (VPS spesifik) — akan dikalibrasi ulang di Milestone 11 berdasarkan load test nyata, bukan dianggap final.

## Technical Debt yang Sengaja Diterima

- Audio-only fallback saat batas partisipan video terlampaui belum diimplementasikan — user yang ditolak join video harus mencoba lagi nanti atau bergabung sebagai penonton tanpa track (belum ada UX untuk ini, dicatat sebagai follow-up bila diperlukan).

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah generalisasi endpoint `/voice/*` → `/rtc/*` dapat diterima sebagai amandemen API Specification?
2. Apakah nilai awal batas partisipan video (10, dikonfigurasi via env var) masuk akal sebagai baseline?
3. Dengan Sprint 10 selesai, **Release 3 (Engagement — Notification, Voice, Video) sudah terencana detail penuh**, menyisakan Release 4 (Optimization + Event Driven Migration) dan Release 5 (Distributed System). Lanjut ke **Sprint 11** (Milestone 11 — Optimization, checkpoint wajib sebelum Event-Driven), atau berhenti dulu?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen Sprint 10: 1 Epic, 4 Feature, 6 task. Menemukan dan memanfaatkan reusability tinggi dari revisi Sprint 9, menggeneralisasi endpoint voice/video menjadi `/rtc/*` |
