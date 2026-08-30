# RULES.md — Nexus

> Aturan keras (hard constraints). Berbeda dari `AGENTS.md` (konteks) dan `WORKFLOWS.md` (alur kerja), file ini berisi **larangan dan kewajiban non-negotiable** yang disintesis dari seluruh `docs/`. Setiap pelanggaran aturan di sini harus dianggap bug, bukan gaya penulisan alternatif. Sumber tiap aturan dicantumkan agar dapat ditelusuri.

---

## 1. Boundary Domain & Struktur Kode

- **JANGAN** import langsung antar `internal/<domain>` di `apps/api` — komunikasi antar domain wajib lewat interface (port) yang didefinisikan domain pemilik. *(Playbook §2.3, HLD §1.1)*
- **JANGAN** menambahkan tipe/struct domain (`User`, `Message`, dll.) ke dalam `pkg/` — `pkg/` hanya untuk kode generik yang tidak tahu apapun tentang domain bisnis. *(Playbook §3.1)*
- **JANGAN** membuat folder `utils/`/`common/` generik sebagai tempat sampah kode — bila kode dipakai 2+ domain tapi mengandung domain knowledge, itu sinyal boundary domain salah, bukan kandidat shared package. *(Playbook §3.2)*
- **JANGAN** mengisi folder `services/` sebelum Phase C (Hybrid Architecture) benar-benar dimulai secara sengaja. *(HLD §1.3)*
- Domain **Workspace/Member/Role/Channel** defaultnya **TETAP** sebagai inti monolith bahkan di Phase D, kecuali ada bukti konkret baru yang mengubah rekomendasi. *(HLD §5)*

## 2. Database & sqlc

- **WAJIB** memakai UUID v7 sebagai primary key untuk seluruh tabel baru. *(Playbook §7.6, ADR-final)*
- **WAJIB** memakai `TIMESTAMPTZ`, tidak pernah `TIMESTAMP` tanpa timezone. *(Playbook §7.6)*
- **JANGAN** menulis raw SQL string di kode Go — seluruh query lewat file `.sql` + `sqlc generate`. *(ADR-003)*
- **JANGAN** melakukan hard delete pada `messages`, `channels`, atau entitas lain yang punya kolom `deleted_at` — selalu soft delete. *(Playbook §7.6, SRS FR-MSG-08)*
- **JANGAN** mengedit file migration yang sudah pernah di-apply ke environment manapun (termasuk staging) — migration bersifat append-only. *(Playbook §18)*
- **WAJIB** memakai pola expand-contract untuk perubahan skema pada tabel besar (`messages`, `audit_logs`) — tidak boleh menambah kolom `NOT NULL` langsung tanpa backfill bertahap. *(Database Design §6)*
- **JANGAN** query `SELECT * FROM members WHERE workspace_id = ...` tanpa pagination — workspace bisa punya hingga 100.000 member. *(SRS FR-WS-08)*
- List endpoint apapun yang datanya berpotensi besar **WAJIB** cursor-based pagination, bukan offset-based. *(Playbook §17.2)*
- **JANGAN** membangun `ORDER BY` dari nama kolom yang dikirim client secara langsung (string concatenation) — **WAJIB** whitelist sort mode di service layer, tiap mode dipetakan ke query sqlc eksplisit dengan `id` sebagai tiebreaker. *(LLD §2.2b)*

## 3. Autentikasi & Keamanan

- **WAJIB** hash password dengan Argon2id, parameter final: memory 46 MiB, iterations 3, parallelism 2. **JANGAN** memakai bcrypt/MD5/SHA murni. *(Security Design §2)*
- **JANGAN** simpan refresh token sebagai plaintext di database — simpan sebagai hash. *(Database Design §2.1, Security Design §2)*
- **JANGAN** simpan refresh token di localStorage — WAJIB HttpOnly, Secure, SameSite=Strict Cookie. *(Security Design §2, dikonfirmasi final)*
- **WAJIB** refresh token rotation setiap dipakai (token lama langsung `revoked`). *(SRS FR-AUTH-04)*
- **WAJIB** CSRF protection (double-submit cookie) khusus endpoint `POST /auth/refresh` sebagai konsekuensi dari HttpOnly Cookie. *(Security Design §6)*
- **JANGAN** membedakan pesan error login antara "email tidak ditemukan" vs "password salah" — selalu generik untuk mencegah user enumeration. *(SRS FR-AUTH-03)*
- **WAJIB** authorization check di backend (service layer) untuk SETIAP operasi — tidak pernah hanya mengandalkan UI hiding di frontend. *(Security Design §3, Learning Roadmap M3)*
- **JANGAN** hardcode secret apapun (JWT key, DB password, Brevo API key, MinIO key, LiveKit secret) — selalu dari environment variable. *(Playbook §18, Security Design §8)*
- Validasi tipe file upload **WAJIB** berdasarkan magic bytes, **JANGAN** hanya berdasarkan ekstensi file. *(SRS FR-UP-02)*

## 4. Concurrency & Go

- **JANGAN** spawn goroutine tanpa mekanisme tunggu selesai (`WaitGroup`/`errgroup`) atau tanpa dikelola worker pool — goroutine leak dilarang. *(Playbook §10.1)*
- **JANGAN** menulis ke koneksi WebSocket dari lebih dari satu goroutine — single-writer-per-connection wajib, kirim lewat channel buffered internal. *(LLD §2.9)*
- **WAJIB** race detector aktif di setiap test run (`go test -race`), bukan opsional. *(Playbook §6.3)*
- **JANGAN** pakai `context.Background()` di dalam handler/service — context selalu di-propagate dari request, kecuali di `main.go`/goroutine background yang sengaja lepas (dengan komentar eksplisit alasan). *(Playbook §10.1)*
- **JANGAN** pakai `log.Fatal`/`os.Exit` di luar `main.go` — mencegah graceful shutdown berjalan. *(Playbook §15.1)*
- Setiap entrypoint (`cmd/*/main.go`) **WAJIB** implementasi graceful shutdown (tangkap SIGTERM/SIGINT, timeout 15 detik, tutup koneksi eksplisit). *(Playbook §10.1, LLD §3)*

## 5. Redis

- **JANGAN** pakai command `KEYS` untuk pencarian pattern di Redis — selalu `SCAN` cursor-based (mencegah blocking). *(LLD §2.6)*
- Rate limiting **WAJIB** atomik (Lua script `EVAL`), tidak boleh dipecah jadi beberapa command terpisah dari aplikasi (race condition). *(LLD §2.8)*
- Presence dan typing indicator **TIDAK** melalui Outbox Pattern — fire-and-forget dengan TTL, ini keputusan sadar bukan kelalaian. *(HLD §3)*

## 6. Event-Driven & Outbox

- **WAJIB** menulis event ke tabel `outbox_events` dalam transaksi yang sama dengan perubahan data — tidak pernah publish event langsung tanpa Outbox Pattern untuk event yang butuh durability. *(LLD §2.4)*
- **WAJIB** setiap event consumer bersifat idempotent (cek `processed_events` sebelum proses) — sistem menganut **at-least-once delivery**, bukan exactly-once. *(LLD §2.7)*
- Notification service dan Search indexer **JANGAN** query langsung ke tabel domain lain (Message, Member) — hanya boleh menerima payload lengkap dari event. *(HLD §2.8, §2.10)*
- Domain Admin adalah **satu-satunya pengecualian** yang boleh command lintas domain langsung (ke Identity untuk suspend/ban) — bukan pola yang boleh ditiru domain lain. *(HLD §2.13)*

## 7. Permission & Authorization Logic

- Urutan resolusi permission **WAJIB** mengikuti: Channel-specific member override → Channel-specific role override → Role default (by `position`) → `@everyone`. **JANGAN** mengubah urutan ini. *(SRS FR-WS-07, LLD §2.1)*
- Permission disimpan sebagai **bitmask `int64`**, bukan tabel many-to-many permission granular. *(SRS FR-WS-03)*
- Channel tipe `dm` **JANGAN** memakai model permission Workspace (Role/Bitmask) — otorisasi DM murni berbasis membership + status block. *(SRS FR-DM-05)*

## 8. Direct Message (DM)

- DM **WAJIB** memakai infrastruktur Channel/Message yang sama (tipe `dm`), **JANGAN** membuat domain/tabel Message terpisah untuk DM. *(PRD §6.9 rationale, LLD §1.2)*
- Uniqueness channel DM 1-on-1 **WAJIB** ditegakkan di level database (partial unique index `participant_key`), bukan hanya application-level check. *(Database Design §2.3, LLD §2.5)*
- **WAJIB** cek status block SEBELUM mengizinkan create/send DM. *(SRS FR-DM-04)*

## 9. API Contract

- **WAJIB** seluruh response REST memakai envelope `{success, data, meta}` atau `{success: false, error: {code, message, details}}`. *(Playbook §17.1)*
- **JANGAN** offset-based pagination untuk list endpoint volume besar — selalu cursor-based. *(Playbook §17.2)*
- Endpoint dengan efek samping kritikal (redeem invite, create DM) **WAJIB** mendukung header `Idempotency-Key`. *(Playbook §17.4, API Spec §2/§5)*
- Upload file **WAJIB** merespons `202 Accepted` (bukan `201`) karena pemrosesan asynchronous — merepresentasikan status jujur, bukan pura-pura selesai. *(API Spec §6)*

## 10. Testing & CI

- **JANGAN** merge PR dengan CI merah — tidak ada pengecualian "diperbaiki nanti setelah merge". *(Playbook §6.5)*
- **JANGAN** commit `TODO`/`FIXME` tanpa referensi ke issue/task tracker. *(Playbook §13)*
- **JANGAN** commit `console.log`/`fmt.Println` debug yang tertinggal — pakai logger terstruktur. *(Playbook §13)*

## 11. Copyright & Konten

- Aturan copyright compliance Claude (kutipan <15 kata, satu kutipan per sumber, tidak mereproduksi lirik/puisi) berlaku untuk seluruh konten yang dihasilkan dalam proyek ini bila mengutip sumber eksternal (dokumentasi API pihak ketiga, dsb.) — parafrase, jangan reproduksi verbatim panjang.

---

## Precedence (Urutan Prioritas Bila Aturan Tampak Bertentangan)

1. **Keamanan & integritas data** (§2, §3) — tidak pernah dikompromikan demi kecepatan.
2. **Boundary domain** (§1) — pelanggaran di sini menciptakan Distributed Monolith yang mahal diperbaiki kemudian.
3. **Konsistensi kontrak** (§6, §9) — perubahan di sini berdampak ke banyak konsumer (frontend, service lain).
4. **Style/konvensi** (§10) — penting untuk maintainability jangka panjang, namun paling mudah diperbaiki belakangan bila terlewat.

Bila sebuah task tampak mengharuskan pelanggaran salah satu aturan di atas: **berhenti, laporkan konflik, tunggu keputusan** — sama seperti pola yang dipakai konsisten di seluruh `docs/01-15` (lihat contoh resolusi konflik Cloudinary→MinIO di `docs/03-adr.md`).
