# Learning Roadmap
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 0 — Learning Roadmap
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `01-engineering-playbook.md` (v1.0.0), `02-vision-document.md` (v1.0.0), `03-adr.md` (v1.1.0)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Cara Menggunakan Roadmap Ini

Roadmap ini adalah dokumen terakhir Phase 0, menjembatani seluruh keputusan strategis (Playbook, Vision, ADR) menjadi **urutan belajar konkret**. Setiap milestone bukan sekadar daftar tugas — ini adalah **unit pembelajaran** dengan tujuan kognitif eksplisit.

Struktur tiap milestone mengikuti format konsisten:

- **Apa yang Dipelajari** — kemampuan/pola konkret yang dikuasai
- **Mengapa Penting** — keterkaitan dengan Vision & tujuan arsitektur keseluruhan
- **Konsep Computer Science yang Terlibat** — fondasi teori di baliknya
- **Referensi Resmi** — sumber otoritatif untuk pendalaman
- **Best Practice** — apa yang membedakan implementasi production-grade dari implementasi asal jalan
- **Kesalahan Umum** — jebakan yang paling sering dijumpai pemula-menengah
- **Trade-off** — keputusan yang harus disadari, bukan diambil default

Milestone diurutkan mengikuti **evolusi arsitektur** (Playbook §Architecture Strategy, ADR-010): Modular Monolith dulu, baru Event-Driven, baru Hybrid, baru Microservices — bukan urutan "fitur termudah dulu".

---

## Milestone 1 — Project Foundation

**Apa yang Dipelajari**
Setup monorepo (`go.work`, pnpm workspace), struktur folder Clean Architecture per domain, konfigurasi awal (Viper), logging dasar (Zap), health check endpoint, dan pipeline CI paling minimal (lint + test + build).

**Mengapa Penting**
Fondasi yang salah di tahap ini akan menular ke seluruh proyek. Kesalahan struktural yang tidak diperbaiki di Milestone 1 biasanya bertahan hingga refactor besar-besaran di kemudian hari — jauh lebih mahal daripada memperbaikinya sekarang.

**Konsep Computer Science yang Terlibat**
Separation of Concerns, Dependency Inversion Principle (fondasi Clean Architecture), Configuration Management.

**Referensi Resmi**
- Go Modules & Workspace: https://go.dev/ref/mod
- Effective Go: https://go.dev/doc/effective_go
- The Clean Architecture (Robert C. Martin, ringkasan konsep — bukan buku spesifik yang perlu diikuti kata per kata, tapi prinsip intinya)

**Best Practice**
- Struktur folder mengikuti konvensi §7.1 & §19 Engineering Playbook sejak commit pertama, bukan "dirapikan nanti".
- Health check endpoint (`/healthz`, `/readyz`) dibuat sejak awal meski belum ada dependency eksternal untuk dicek — kebiasaan ini penting agar tidak lupa saat dependency bertambah.

**Kesalahan Umum**
- Membuat folder `utils/` atau `common/` generik sejak awal sebagai "tempat sampah" kode yang belum jelas rumahnya — cikal bakal God Package (lihat §3.2 Playbook).
- Menunda setup CI sampai "proyek sudah jalan dulu" — padahal disiplin CI sejak commit pertama justru mencegah akumulasi masalah kualitas.

**Trade-off**
Investasi waktu di struktur/tooling di awal terasa lambat dibanding "langsung nulis fitur", tapi mencegah biaya refactor struktural yang jauh lebih mahal di kemudian hari.

---

## Milestone 2 — Authentication

**Apa yang Dipelajari**
Register/Login (email & username), Argon2id password hashing, JWT access token, Refresh Token rotation, session management, middleware autentikasi Gin.

**Mengapa Penting**
Authentication adalah domain pertama yang benar-benar diuji lintas layer (validation → service → repository → HTTP response) dan menjadi fondasi Authorization untuk seluruh domain lain.

**Konsep Computer Science yang Terlibat**
Cryptographic Hashing (Argon2id: memory-hard function, resisten terhadap GPU/ASIC brute-force), Stateless vs Stateful Authentication trade-off (JWT vs Session), Token Rotation untuk mitigasi token replay.

**Referensi Resmi**
- OWASP Password Storage Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
- Argon2 RFC 9106: https://www.rfc-editor.org/rfc/rfc9106
- JWT RFC 7519: https://www.rfc-editor.org/rfc/rfc7519

**Best Practice**
- Refresh token disimpan di database (bukan hanya di client) dengan status revoke, memungkinkan "logout dari semua device".
- Access token berumur pendek (15 menit), refresh token berumur panjang namun **rotasi setiap dipakai** (refresh token lama langsung invalid) — mitigasi terhadap refresh token yang dicuri.
- Argon2id parameter (memory, iteration, parallelism) dikonfigurasi sesuai kapasitas server, tidak memakai default library tanpa pertimbangan.

**Kesalahan Umum**
- Menyimpan JWT secret sebagai string hardcoded, bukan dari environment/secret manager.
- Tidak melakukan refresh token rotation, sehingga satu refresh token bocor = akses permanen bagi penyerang.
- Memakai bcrypt/MD5/SHA murni untuk password (bukan Argon2id) — rentan terhadap brute-force modern berbasis GPU.

**Trade-off**
JWT (stateless) memberi skalabilitas horizontal mudah tanpa shared session store, namun revocation menjadi lebih kompleks (butuh refresh token tracking di DB atau blacklist) dibanding session berbasis Redis murni yang secara inheren mudah di-revoke tapi butuh shared state.

---

## Milestone 3 — Workspace

**Apa yang Dipelajari**
Domain Workspace (Server), Role, Permission, Member, Category — pemodelan multi-tenancy dan Role-Based Access Control (RBAC) berbutir halus (granular permission per-channel, bukan hanya per-workspace).

**Mengapa Penting**
Ini adalah domain dengan kompleksitas relasi tertinggi di seluruh proyek (Workspace → Member → Role → Permission → Channel override). Kesalahan desain di sini berdampak luas ke seluruh fitur lain yang butuh authorization check.

**Konsep Computer Science yang Terlibat**
Access Control Models (RBAC vs ABAC), Graph-like relationship modeling dalam relational database, Bitmask/Bitwise permission representation (opsional, dibahas trade-off-nya).

**Referensi Resmi**
- NIST RBAC Model: https://csrc.nist.gov/projects/role-based-access-control
- PostgreSQL Row Level Security (opsional untuk pertimbangan lapisan tambahan): https://www.postgresql.org/docs/current/ddl-rowsecurity.html

**Best Practice**
- Permission direpresentasikan sebagai bitmask integer (mis. `int64`) untuk kombinasi permission yang efisien secara storage dan query, dibanding tabel many-to-many permission murni yang bisa membengkak — detail trade-off dibahas di Low Level Design.
- Authorization check selalu dilakukan di service layer (backend), tidak pernah hanya mengandalkan UI hiding tombol di frontend.

**Kesalahan Umum**
- Melakukan authorization check hanya di level UI (frontend menyembunyikan tombol) tanpa validasi ulang di backend — celah keamanan klasik.
- Permission model yang terlalu granular sejak awal tanpa kebutuhan nyata (YAGNI violation) — mulai dari beberapa permission inti (kelola channel, kelola member, kirim pesan, kelola role), bukan mendesain puluhan permission spekulatif di awal.

**Trade-off**
RBAC sederhana (role tetap) lebih mudah dipahami dan diimplementasikan dibanding ABAC (attribute-based, sangat fleksibel tapi kompleks) — RBAC dipilih karena cukup untuk kebutuhan Discord-like dan lebih sesuai Learning Objective bertahap, ABAC bisa jadi eksplorasi lanjutan opsional.

---

## Milestone 4 — Channel

**Apa yang Dipelajari**
Text, Voice, Video, Forum, Announcement channel — pemodelan polimorfik tipe channel dalam satu domain, permission override per-channel (berbeda dari permission default workspace).

**Mengapa Penting**
Melatih pemodelan domain yang punya variasi tipe (bukan satu tabel generik untuk semua) — keputusan desain di sini memengaruhi seberapa mudah menambah tipe channel baru di masa depan.

**Konsep Computer Science yang Terlibat**
Polymorphism dalam desain data relasional (Single Table Inheritance vs Class Table Inheritance), Permission Override/Inheritance chain resolution.

**Referensi Resmi**
- Martin Fowler, Patterns of Enterprise Application Architecture — konsep Single Table Inheritance/Class Table Inheritance (referensi konseptual).

**Best Practice**
- Satu tabel `channels` dengan kolom `type` (enum) untuk atribut umum, tabel terpisah untuk atribut spesifik tipe (mis. `voice_channel_settings` untuk bitrate/region) — menghindari tabel dengan puluhan kolom nullable.
- Resolusi permission override mengikuti urutan jelas: Workspace default → Role → Channel-specific override, didokumentasikan eksplisit di Low Level Design agar tidak ambigu.

**Kesalahan Umum**
- Satu tabel `channels` dengan seluruh kolom untuk semua tipe (banyak kolom NULL untuk tipe yang tidak relevan) — sulit dipelihara dan rawan bug logika kondisional yang rumit.
- Urutan resolusi permission yang tidak konsisten antar bagian kode — menyebabkan bug otorisasi yang sulit dilacak.

**Trade-off**
Class Table Inheritance (tabel terpisah per tipe channel) lebih bersih secara skema namun butuh JOIN tambahan; Single Table dengan kolom umum + tabel detail opsional dipilih sebagai kompromi seimbang (detail final di Database Design).

---

## Milestone 5 — Realtime Chat

**Apa yang Dipelajari**
WebSocket connection lifecycle (upgrade, ping/pong heartbeat, graceful close), broadcast pesan ke seluruh member channel, reply/thread/mention/reaction, markdown rendering, cursor-based pagination history pesan.

**Mengapa Penting**
Ini adalah fitur inti "rasa Discord" pertama yang benar-benar dirasakan pengguna, sekaligus titik pertama proyek benar-benar menguji WebSocket dan concurrency di Go secara nyata (banyak koneksi paralel per instance).

**Konsep Computer Science yang Terlibat**
Concurrency (goroutine per-koneksi WebSocket, dengan disiplin mencegah goroutine leak), Publish-Subscribe pattern in-memory (untuk broadcast dalam satu instance sebelum ke Redis di Milestone 12), Cursor-based Pagination (B-Tree index traversal).

**Referensi Resmi**
- Gorilla WebSocket: https://pkg.go.dev/github.com/gorilla/websocket
- RFC 6455 (The WebSocket Protocol): https://www.rfc-editor.org/rfc/rfc6455

**Best Practice**
- Setiap koneksi WebSocket punya goroutine read-loop dan write-loop terpisah, dikoordinasikan lewat channel Go — bukan satu goroutine yang menangani read dan write sekaligus (mencegah blocking).
- Ping/pong heartbeat dengan timeout eksplisit untuk mendeteksi koneksi mati (client crash tanpa proper close).
- Broadcast dilakukan lewat registry koneksi per-channel (map dengan mutex atau `sync.Map`) — bukan iterasi seluruh koneksi global.

**Kesalahan Umum**
- Menulis ke WebSocket connection dari banyak goroutine tanpa mutex — race condition pada level frame WebSocket (data corruption), harus dijaga single-writer per koneksi.
- Tidak menutup goroutine reader/writer saat koneksi terputus — goroutine leak yang lama-lama menghabiskan memori.
- Tidak membatasi ukuran pesan masuk — celah DoS lewat pesan raksasa.

**Trade-off**
Broadcast in-memory (Milestone ini) hanya bekerja untuk instance tunggal — pada Milestone 12 (Presence, multi-instance) perlu Redis Pub/Sub/Streams sebagai broadcast layer lintas instance; keterbatasan ini disengaja diterima dulu sesuai evolusi bertahap (tidak langsung membangun distributed broadcast sebelum kebutuhan single-instance benar-benar matang).

---

## Milestone 6 — Upload

**Apa yang Dipelajari**
Upload attachment (image, video, audio, PDF, ZIP) hingga 1 GB, validasi tipe file, integrasi MinIO (sesuai ADR-007), enqueue task pemrosesan via Asynq.

**Mengapa Penting**
Ini adalah titik pertama proyek benar-benar berinteraksi dengan Distributed Task Queue (Asynq) — melatih pola *fire against a queue, process asynchronously* yang menjadi fondasi Milestone-milestone berikutnya (Notification, Media processing).

**Konsep Computer Science yang Terlibat**
Streaming I/O (upload file besar tanpa memuat seluruhnya ke memori — `io.Reader`/`io.Copy` streaming ke MinIO), Task Queue & Worker Pool, Content-Type Sniffing untuk validasi keamanan (bukan hanya percaya ekstensi file).

**Referensi Resmi**
- MinIO Go Client: https://min.io/docs/minio/linux/developers/go/minio-go.html
- Asynq: https://github.com/hibiken/asynq

**Best Practice**
- Validasi tipe file berdasarkan **magic bytes** (`http.DetectContentType` atau library sejenis), bukan hanya ekstensi `.jpg`/`.png` yang mudah dipalsukan.
- Upload memakai streaming (chunked), tidak membaca seluruh file 1 GB ke memori sekaligus — krusial untuk mencegah OOM pada banyak upload paralel.
- Task Asynq untuk pemrosesan (thumbnail, transcoding) idempotent — bila task di-retry, tidak menghasilkan duplikasi file.

**Kesalahan Umum**
- Membaca seluruh file ke `[]byte` di memori sebelum upload — dengan 10.000 concurrent user dan file hingga 1 GB, ini adalah resep OOM crash.
- Validasi file hanya berdasarkan ekstensi, bukan konten aktual — celah upload file berbahaya menyamar sebagai gambar.

**Trade-off**
Streaming upload lebih kompleks diimplementasikan (perlu penanganan error mid-stream, progress tracking) dibanding baca-seluruhnya-lalu-upload, namun wajib untuk skala file besar sesuai NFR proyek.

---

## Milestone 7 — Presence

**Apa yang Dipelajari**
Status online/offline/idle/DND/invisible, deteksi disconnect (heartbeat timeout), penyimpanan state presence di Redis dengan TTL, broadcast presence update lintas koneksi.

**Mengapa Penting**
Presence adalah domain pertama yang benar-benar butuh Redis sebagai **shared state** lintas request/koneksi (bukan hanya cache biasa) — mempersiapkan pola yang sama untuk domain lain di Phase B.

**Konsep Computer Science yang Terlibat**
TTL-based expiry sebagai mekanisme deteksi "kematian" tanpa eksplisit disconnect signal (mirip Lease pattern di distributed systems), Eventual Consistency (presence yang sedikit delay antar client dapat diterima).

**Referensi Resmi**
- Redis Expire/TTL: https://redis.io/docs/latest/commands/expire/

**Best Practice**
- Presence disimpan sebagai key Redis dengan TTL pendek (mis. 30 detik), di-refresh (`EXPIRE`) tiap heartbeat dari client — jika client mati tanpa graceful disconnect, key otomatis expire tanpa perlu polling aktif.
- Perubahan presence di-broadcast hanya ke member yang **berbagi workspace** dengan user tersebut (bukan broadcast global) — pertimbangan skalabilitas untuk 100.000 member per server.

**Kesalahan Umum**
- Mengandalkan hanya event `disconnect` WebSocket untuk mendeteksi offline — tidak menangani kasus client crash/network terputus tanpa close frame yang proper, sehingga status "online" nyangkut selamanya. TTL-based approach memitigasi ini.
- Broadcast presence update ke seluruh user aplikasi tanpa scoping — tidak scalable untuk NFR 100.000 member/server.

**Trade-off**
TTL-based presence sederhana dan robust terhadap disconnect tidak normal, namun ada delay presisi (idle-to-offline butuh menunggu TTL habis, bukan instan) — diterima karena presence secara inheren tidak butuh presisi real-time ketat.

---

## Milestone 8 — Notification

**Apa yang Dipelajari**
Notifikasi realtime (via WebSocket) dan email (via Asynq scheduled/delayed task), domain event pertama yang benar-benar dipublikasikan dan dikonsumsi lintas domain (mention pengguna → trigger notifikasi).

**Mengapa Penting**
Ini adalah **jembatan langsung menuju Phase B (Event-Driven)** — domain Notification secara alami butuh mendengarkan event dari domain lain (Message, Member) tanpa domain lain perlu tahu detail cara notifikasi bekerja (loose coupling).

**Konsep Computer Science yang Terlibat**
Observer Pattern (dalam bentuk domain event), Fan-out (satu event → banyak channel notifikasi: WebSocket + email), Rate Limiting untuk mencegah notification spam.

**Referensi Resmi**
- Martin Fowler, "What do you mean by Event-Driven": https://martinfowler.com/articles/201701-event-driven.html

**Best Practice**
- Notification service **tidak melakukan query langsung** ke tabel Message/Member — ia menerima payload lengkap dari event, menjaga boundary domain (persiapan langsung untuk ekstraksi service di Phase C).
- Preferensi notifikasi per-user (mute channel, mute workspace) dicek sebelum fan-out, bukan sesudah (menghindari kerja sia-sia).

**Kesalahan Umum**
- Notification service melakukan JOIN langsung ke tabel domain lain "demi kemudahan" — inilah awal mula Distributed Monolith yang harus dihindari sejak dalam monolith (lihat ADR-010, §2.3 Playbook).

**Trade-off**
Domain event murni (payload lengkap dikirim, tidak query balik) sedikit meningkatkan ukuran payload event, tapi memberi loose coupling yang jauh lebih berharga untuk ekstraksi service nanti.

---

## Milestone 9 — Voice

**Apa yang Dipelajari**
Integrasi LiveKit untuk voice channel, token-based room access, signaling dasar, konsep SFU (Selective Forwarding Unit) secara konseptual (ADR-005).

**Mengapa Penting**
Domain dengan karakteristik beban kerja paling berbeda dari domain lain (real-time media, CPU/bandwidth-bound) — mempersiapkan alasan konkret mengapa domain ini menjadi kandidat kuat ekstraksi service terpisah (beda resource profile).

**Konsep Computer Science yang Terlibat**
WebRTC fundamentals (ICE candidate gathering, STUN/TURN untuk NAT traversal, SDP negotiation) — dipahami secara konseptual karena diabstraksi LiveKit, Token-based Authorization untuk resource temporer (room access).

**Referensi Resmi**
- LiveKit Docs: https://docs.livekit.io/
- WebRTC for the Curious (referensi konseptual gratis): https://webrtcforthecurious.com/

**Best Practice**
- Token akses room LiveKit dibuat di backend dengan claim spesifik (room, identity, permission), bukan token statis — mencegah akses tak sah ke room manapun.
- Lifecycle room (dibuat saat member pertama join, dihapus saat kosong) dikelola eksplisit, bukan dibiarkan menumpuk.

**Kesalahan Umum**
- Meng-generate token LiveKit di frontend dengan API secret ter-embed — kebocoran credential fatal, token generation **wajib** di backend.

**Trade-off**
Menggunakan LiveKit (dibanding membangun SFU sendiri via mediasoup) berarti tidak mendapat pemahaman mendalam internal SFU — trade-off yang sudah dianalisis dan diterima di ADR-005.

---

## Milestone 10 — Video

**Apa yang Dipelajari**
Perluasan integrasi LiveKit untuk video channel (kamera + screen share), manajemen bandwidth adaptif (simulcast, konsep dipahami meski dikonfigurasi lewat LiveKit).

**Mengapa Penting**
Melatih penanganan beban kerja media paling berat dalam proyek — data konkret untuk keputusan scaling dan resource allocation di Deployment Architecture.

**Konsep Computer Science yang Terlibat**
Adaptive Bitrate Streaming (konsep simulcast: mengirim beberapa kualitas video sekaligus, SFU memilih kualitas sesuai bandwidth penerima).

**Referensi Resmi**
- LiveKit Simulcast: https://docs.livekit.io/home/client/tracks/

**Best Practice**
- Resource limit eksplisit per room (maksimal partisipan video sekaligus) untuk mencegah satu room video menghabiskan seluruh bandwidth server.

**Kesalahan Umum**
- Tidak membatasi jumlah video track aktif sekaligus per user — bandwidth dan CPU (encoding/decoding) bisa membengkak tak terkendali.

**Trade-off**
Kualitas video tinggi vs bandwidth cost — parameter simulcast perlu disesuaikan dengan kapasitas infrastruktur nyata proyek (VPS terbatas), bukan default LiveKit yang diasumsikan untuk skala cloud besar.

---

## Milestone 11 — Optimization

**Apa yang Dipelajari**
Profiling dengan `pprof` (CPU profile, memory profile, goroutine profile), benchmarking (`go test -bench`), identifikasi dan perbaikan N+1 query, index tuning berdasarkan `EXPLAIN ANALYZE`.

**Mengapa Penting**
Ini adalah checkpoint wajib sebelum masuk Phase B — memastikan Modular Monolith benar-benar solid secara performa sebelum menambah kompleksitas event-driven di atasnya. Prinsip *Production First Mindset* dipraktikkan langsung di sini, bukan diasumsikan.

**Konsep Computer Science yang Terlibat**
Algorithmic Complexity Analysis (Big-O) diterapkan pada query nyata, Sampling Profiler mechanics (bagaimana `pprof` mengambil sampel stack trace), Amdahl's Law (memahami batas percepatan dari optimasi parsial).

**Referensi Resmi**
- pprof: https://pkg.go.dev/runtime/pprof
- Go Profiling Guide: https://go.dev/blog/pprof
- PostgreSQL EXPLAIN: https://www.postgresql.org/docs/current/sql-explain.html

**Best Practice**
- Profiling dilakukan **berdasarkan data nyata** (load test terlebih dahulu, baru profiling), bukan optimasi spekulatif ("kelihatannya lambat").
- Setiap optimasi diukur before/after dengan benchmark yang sama — tanpa angka, optimasi hanyalah asumsi.

**Kesalahan Umum**
- Melakukan optimasi mikro (micro-optimization) pada kode yang bukan bottleneck nyata ("premature optimization") — buang waktu tanpa dampak terukur.
- Menambah index database secara membabi buta tanpa memahami dampaknya terhadap write performance (setiap index memperlambat INSERT/UPDATE).

**Trade-off**
Index mempercepat read tapi memperlambat write dan menambah ukuran storage — setiap keputusan index harus mempertimbangkan rasio read:write aktual pada tabel tersebut (dibahas detail per tabel di Database Design).

---

## Milestone 12 — Event Driven (Transisi ke Phase B)

**Apa yang Dipelajari**
Outbox Pattern (menulis event ke tabel outbox dalam transaksi yang sama dengan perubahan data, lalu relay terpisah mempublikasikan ke Redis Streams), Idempotency Key pada consumer, retry dengan exponential backoff, Dead Letter strategy.

**Mengapa Penting**
Ini adalah **inti Learning Objective Event-Driven Architecture** — momen proyek benar-benar bertransisi dari Phase A ke Phase B secara nyata, bukan konseptual.

**Konsep Computer Science yang Terlibat**
Atomicity di luar transaksi database tunggal (Outbox Pattern menyelesaikan masalah *dual write problem*: menulis ke DB dan mempublikasikan event dalam satu langkah atomik yang genuinely reliable), At-Least-Once Delivery semantics, Idempotency sebagai properti matematis (`f(f(x)) = f(x)`).

**Referensi Resmi**
- Chris Richardson, Microservices Pattern — Transactional Outbox: https://microservices.io/patterns/data/transactional-outbox.html
- Redis Streams: https://redis.io/docs/latest/develop/data-types/streams/

**Best Practice**
- Tabel outbox berisi kolom `id`, `aggregate_id`, `event_type`, `payload`, `created_at`, `published_at` (nullable) — relay membaca baris dengan `published_at IS NULL`, mempublikasikan ke Redis Streams, lalu menandai `published_at`.
- Consumer menyimpan `processed_event_id` (atau memakai Redis Streams consumer group native acknowledgment) untuk memastikan event yang sudah diproses tidak diproses ulang meski di-retry.
- Retry memakai exponential backoff dengan jitter, bukan retry langsung tanpa jeda (mencegah thundering herd saat downstream service pulih dari gangguan).

**Kesalahan Umum**
- Mempublikasikan event **langsung** setelah commit database tanpa Outbox Pattern — rawan *dual write problem* (data tersimpan tapi event gagal terkirim karena crash di antara dua operasi, atau sebaliknya).
- Consumer yang tidak idempotent — event yang di-retry menyebabkan efek samping ganda (mis. notifikasi terkirim dua kali).

**Trade-off**
Outbox Pattern menambah latensi (event tidak instan, menunggu relay polling/membaca) dan kompleksitas (perlu proses relay terpisah) dibanding publish langsung — diterima karena garansi reliability (event tidak pernah hilang akibat crash) jauh lebih penting daripada latensi milidetik untuk kasus notification/indexing yang secara inheren asynchronous.

---

## Milestone 13 — Extract First Service (Transisi ke Phase C)

**Apa yang Dipelajari**
Ekstraksi modul pertama (sesuai Service Extraction Plan yang akan dibahas di HLD — kandidat awal: Notification atau Identity) menjadi service Go independen, komunikasi balik ke monolith lewat REST/gRPC, deployment independen di CI/CD.

**Mengapa Penting**
Ini adalah momen **paling krusial secara pedagogis** dalam seluruh proyek — praktik nyata "bagaimana rasanya" memisahkan modul yang sudah dirancang dengan boundary tegas sejak Phase A, membuktikan (atau membantah) asumsi bahwa disiplin domain boundary sejak awal benar-benar memudahkan ekstraksi.

**Konsep Computer Science yang Terlibat**
Bounded Context (DDD), Strangler Fig Pattern (migrasi bertahap tanpa big-bang rewrite), Network Fallacy ("the network is reliable" — asumsi yang tidak lagi valid begitu modul jadi service terpisah).

**Referensi Resmi**
- Martin Fowler, Strangler Fig Application: https://martinfowler.com/bliki/StranglerFigApplication.html
- Sam Newman, Building Microservices (referensi konseptual untuk service extraction).

**Best Practice**
- Sebelum ekstraksi, pastikan modul benar-benar tidak punya dependency langsung ke tabel domain lain (validasi ulang boundary, bukan asumsi).
- Ekstraksi dilakukan dengan **kontrak API/event stabil terlebih dahulu** — definisikan interface komunikasi sebelum memindahkan kode, bukan sebaliknya.
- Circuit breaker/timeout dipasang di sisi monolith saat memanggil service baru — mengakui bahwa network call bisa gagal, berbeda dari in-process function call yang selalu "berhasil" secara mekanis.

**Kesalahan Umum**
- Mengekstraksi modul yang ternyata masih punya banyak *hidden coupling* (foreign key lintas domain, shared transaction) — inilah momen paling umum ditemukannya Distributed Monolith yang sebenarnya sudah ada sejak dalam monolith tapi baru terlihat saat dipisah.
- Tidak menambahkan timeout/retry/circuit breaker saat memanggil service baru — kegagalan service terekstraksi bisa merambat (cascading failure) ke monolith.

**Trade-off**
Ekstraksi menambah latensi (network call vs in-process call) dan kompleksitas operasional (dua deployment unit, dua log stream) — diterima **hanya** bila manfaat independent scaling/deployment terbukti lebih besar dari cost ini (dievaluasi konkret di Service Extraction Plan, HLD).

---

## Milestone 14 — Hybrid Architecture

**Apa yang Dipelajari**
Mengelola sistem dengan sebagian modul masih monolith dan sebagian sudah service terpisah secara bersamaan — routing API Gateway (Traefik) untuk mengarahkan request ke tujuan yang benar, observability terpadu lintas monolith+service.

**Mengapa Penting**
Kondisi Hybrid adalah **kondisi paling realistis** yang dihadapi kebanyakan sistem produksi nyata (sangat sedikit sistem benar-benar 100% microservices) — Learning Objective di sini adalah mengelola kompleksitas transisi, bukan buru-buru menuju Full Microservices.

**Konsep Computer Science yang Terlibat**
API Gateway routing (path-based/host-based), Distributed Tracing context propagation lintas proses (trace_id yang sama harus mengalir dari monolith ke service terekstraksi).

**Referensi Resmi**
- OpenTelemetry Context Propagation: https://opentelemetry.io/docs/concepts/context-propagation/

**Best Practice**
- `trace_id` dan `request_id` di-propagate lewat HTTP header (`traceparent` sesuai W3C Trace Context) saat monolith memanggil service terekstraksi, memastikan satu request bisa ditelusuri end-to-end di Grafana/Jaeger.

**Kesalahan Umum**
- Trace terputus di boundary monolith→service karena header propagation tidak diimplementasikan — menyulitkan debugging lintas proses, salah satu masalah observability paling umum dalam sistem hybrid/microservices.

**Trade-off**
Hybrid architecture butuh dua "cara berpikir" sekaligus (in-process call untuk modul monolith, network call untuk service terekstraksi) — kompleksitas kognitif ini nyata dan harus diterima secara sadar selama masa transisi, bukan dianggap sementara yang bisa diabaikan kualitasnya.

---

## Milestone 15 — Observability

**Apa yang Dipelajari**
OpenTelemetry instrumentation menyeluruh (trace, metric, log terkorelasi), Prometheus metric exposition, dashboard Grafana, alerting dasar.

**Mengapa Penting**
"Observability by Default" adalah salah satu prinsip inti Engineering Philosophy proyek — milestone ini memastikan prinsip tersebut benar-benar terwujud secara teknis, bukan hanya slogan.

**Konsep Computer Science yang Terlibat**
Three Pillars of Observability (log, metric, trace), Cardinality management pada metric labels (label dengan cardinality tinggi seperti `user_id` bisa membuat Prometheus meledak memori-nya), Sampling strategy untuk tracing di skala tinggi.

**Referensi Resmi**
- OpenTelemetry Go: https://opentelemetry.io/docs/languages/go/
- Prometheus Best Practices: https://prometheus.io/docs/practices/naming/
- Google SRE Book, Monitoring Distributed Systems: https://sre.google/sre-book/monitoring-distributed-systems/

**Best Practice**
- Metric label **tidak pernah** memakai nilai dengan cardinality tinggi/tak terbatas (mis. `user_id`, `message_id`) sebagai label Prometheus — gunakan `trace_id` di log untuk drill-down ke level individual, metric untuk agregat.
- Dashboard Grafana dibangun berdasarkan **kebutuhan diagnosis nyata** (RED method: Rate, Errors, Duration untuk setiap service) bukan sekadar menampilkan semua metric yang ada.

**Kesalahan Umum**
- Menambahkan label metric dengan cardinality tinggi — menyebabkan Prometheus memori membengkak drastis (masalah production nyata yang sangat umum).
- Tracing tanpa sampling di volume tinggi — overhead tracing sendiri bisa menjadi bottleneck jika 100% request di-trace pada skala besar (walau untuk skala proyek ini, 100% sampling masih wajar; prinsip tetap dipahami).

**Trade-off**
Observability menambah overhead (CPU untuk instrumentasi, storage untuk metric/trace/log) — diterima karena tanpa ini, mendiagnosis masalah production pada sistem event-driven/distributed menjadi hampir mustahil ("jika tidak bisa diobservasi, maka belum production-ready" — Definition of Quality §8 Playbook).

---

## Milestone 16 — Production Deployment

**Apa yang Dipelajari**
Blue-Green Deployment, Zero Downtime Deployment, Rolling Update, graceful shutdown yang benar-benar teruji di bawah traffic nyata (bukan hanya kode `signal.Notify`).

**Mengapa Penting**
Ini adalah momen "teori bertemu kenyataan" untuk seluruh persiapan graceful shutdown yang sudah ditulis sejak Milestone 1 — diuji dengan deployment sungguhan, bukan hanya `kill -SIGTERM` di lokal.

**Konsep Computer Science yang Terlibat**
Connection Draining, Health Check-based Traffic Routing, Immutable Infrastructure principle.

**Referensi Resmi**
- Traefik Docs — Dynamic Configuration: https://doc.traefik.io/traefik/
- Kubernetes Rolling Update (referensi konseptual untuk Milestone 17): https://kubernetes.io/docs/tutorials/kubernetes-basics/update/update-intro/

**Best Practice**
- Readiness probe **dilepas dari traffic dulu** (Traefik/load balancer berhenti mengirim request baru) sebelum graceful shutdown context timeout dimulai — urutan ini penting agar tidak ada request baru masuk saat proses sedang menyelesaikan request lama.
- Blue-Green switch diuji dengan **traffic nyata paralel** (load test berjalan saat switch terjadi) untuk memvalidasi benar-benar zero downtime, bukan diasumsikan dari membaca dokumentasi.

**Kesalahan Umum**
- Graceful shutdown yang hanya menutup listener tapi tidak menunggu goroutine request in-flight selesai — request yang sedang diproses terputus paksa saat deployment.
- Tidak ada readiness probe terpisah dari liveness probe — traffic tetap dikirim ke instance yang belum siap menerima request (mis. koneksi DB belum ready) atau sedang shutdown.

**Trade-off**
Blue-Green Deployment butuh resource dua kali lipat sesaat (dua environment berjalan paralel selama transisi) dibanding Rolling Update yang lebih hemat resource namun window downtime-nya (jika ada) lebih sulit dikontrol presisi — dipilih sesuai kebutuhan nyata di Deployment Architecture (Phase 8), bukan satu ukuran untuk semua situasi.

---

## Milestone 17 — Microservices Migration (Transisi ke Phase D)

**Apa yang Dipelajari**
Melanjutkan Service Extraction Plan hingga seluruh modul yang direncanakan berdiri independen, API Gateway penuh, service discovery (konseptual — dibahas kapan benar-benar butuh service registry vs DNS-based discovery sederhana), gRPC untuk komunikasi antar service berlatensi rendah.

**Mengapa Penting**
Ini adalah pencapaian target akhir arsitektur sesuai Vision Document — namun **tetap dievaluasi** apakah benar-benar seluruh modul perlu diekstraksi, atau berhenti di Hybrid sesuai kondisi §7 Vision Document ("kapan sebaiknya tetap bertahan").

**Konsep Computer Science yang Terlibat**
Service Mesh concept (dipahami konseptual, tidak wajib diimplementasikan penuh untuk skala proyek ini), Protocol Buffers & gRPC code generation, CAP theorem trade-off saat data terdistribusi lintas service (Database-per-Service berarti melepas ACID transaction lintas domain, digantikan Saga Pattern).

**Referensi Resmi**
- gRPC Go: https://grpc.io/docs/languages/go/
- Chris Richardson, Saga Pattern: https://microservices.io/patterns/data/saga.html

**Best Practice**
- Saga Pattern (koreografi berbasis event, sesuai gaya Event-Driven yang sudah dipakai sejak Phase B) dipakai untuk transaksi lintas service (mis. hapus workspace → hapus semua channel, message, member di service berbeda), bukan mencoba distributed transaction (2PC) yang kompleks dan rapuh.
- Setiap service memiliki database sendiri (Database-per-Service) — tidak ada service yang mengakses tabel milik service lain secara langsung.

**Kesalahan Umum**
- Mencoba mempertahankan ACID transaction lintas service (distributed transaction/2PC) — anti-pattern yang dikenal luas menyebabkan sistem rapuh dan sulit di-scale; Saga (eventual consistency) adalah pendekatan yang secara sadar dipilih sebagai gantinya.
- Mengekstraksi service hanya karena "sudah waktunya" tanpa validasi ulang kebutuhan nyata (mengabaikan prinsip §7 Vision Document).

**Trade-off**
Full Microservices memberi independent scaling & deployment penuh, namun melepas kemudahan ACID transaction lintas domain dan menambah kompleksitas operasional signifikan (lebih banyak deployment unit, lebih banyak titik kegagalan jaringan) — **keputusan mengekstraksi setiap modul dievaluasi individual**, bukan all-or-nothing.

---

## Milestone 18 — Production Hardening

**Apa yang Dipelajari**
Security hardening menyeluruh (OWASP Top 10 review), rate limiting matang, audit log lengkap, disaster recovery drill (uji restore backup sungguhan, bukan asumsi backup berhasil), load testing skala penuh (mendekati NFR 10.000 concurrent users).

**Mengapa Penting**
Ini adalah milestone penutup yang memvalidasi **seluruh** Definition of Quality (§8 Playbook) benar-benar terpenuhi di kondisi mendekati production sungguhan — bukan sekadar "kelihatannya sudah bagus".

**Konsep Computer Science yang Terlibat**
Threat Modeling (STRIDE), Chaos Engineering dasar (menguji sistem dengan kegagalan yang disengaja — mis. mematikan satu service secara sengaja untuk validasi graceful degradation).

**Referensi Resmi**
- OWASP Top 10: https://owasp.org/www-project-top-ten/
- Google SRE Book, Disaster Recovery: https://sre.google/sre-book/disaster-recovery/

**Best Practice**
- Disaster recovery **diuji nyata** secara berkala (restore backup ke environment terpisah, verifikasi data benar-benar utuh) — backup yang tidak pernah diuji restore-nya secara statistik sering ditemukan rusak/tidak lengkap saat benar-benar dibutuhkan.
- Load test dijalankan hingga mendekati batas NFR yang didefinisikan (10.000 concurrent users), bukan berhenti di angka yang "terasa cukup".

**Kesalahan Umum**
- Backup rutin dijalankan tapi **tidak pernah diuji proses restore-nya** — kesalahan produksi yang sangat umum dan baru disadari saat insiden nyata terjadi.
- Rate limiting hanya diterapkan di satu layer (mis. hanya di aplikasi) tanpa mempertimbangkan proteksi berlapis (Traefik middleware + aplikasi + database connection pool limit).

**Trade-off**
Hardening menyeluruh butuh waktu signifikan yang tidak menghasilkan fitur baru terlihat — namun inilah yang membedakan "aplikasi yang jalan" dari "sistem yang production-ready", selaras penuh dengan Vision Document tentang definisi kesuksesan proyek ini.

---

## Ringkasan Keputusan

1. Roadmap disusun mengikuti **evolusi arsitektur** (Modular Monolith → Event-Driven → Hybrid → Microservices), bukan urutan kemudahan fitur — konsisten dengan Vision Document dan ADR-010.
2. Setiap milestone memiliki tujuan kognitif eksplisit (konsep CS, best practice, kesalahan umum, trade-off) — bukan sekadar daftar tugas implementasi.
3. Milestone 11 (Optimization) sengaja ditempatkan sebagai **checkpoint wajib** sebelum Phase B, memastikan fondasi Modular Monolith solid sebelum kompleksitas event-driven ditambahkan.
4. Milestone 13 (Extract First Service) diberi penekanan khusus sebagai momen paling krusial secara pedagogis dalam keseluruhan proyek.

## Trade-off yang Diterima

- Roadmap ini cukup panjang (18 milestone) dan tidak semua akan terasa "cepat menghasilkan fitur terlihat" (khususnya Milestone 11, 15, 18) — diterima karena milestone tersebut adalah inti dari Learning Objective "Production Readiness" yang menjadi pembeda utama proyek ini dari tutorial biasa.

## Risiko Arsitektur

- Risiko terbesar dalam eksekusi roadmap ini adalah tergoda melompati Milestone 11/15/18 karena "tidak menghasilkan fitur baru" — mitigasi: rujuk kembali ke Vision Document §2 tentang definisi kesuksesan proyek yang eksplisit bukan soal fitur.

## Technical Debt yang Sengaja Diterima

- Roadmap ini belum menyertakan estimasi waktu per milestone — akan dibahas di Development Roadmap (Phase 9) dan Sprint Planning (Phase 10), disengaja dipisah agar Learning Roadmap murni fokus pada tujuan pembelajaran, bukan tercampur perencanaan waktu yang bisa berubah-ubah.

## Hal yang Perlu Dikonfirmasi Sebelum Lanjut ke Fase Berikutnya

Dengan selesainya dokumen ini, **seluruh Phase 0 (Engineering Playbook, Vision Document, ADR, Learning Roadmap) telah lengkap**. Sebelum lanjut ke **Phase 1 — Product Requirement Document (PRD)**, mohon konfirmasi:

1. Apakah urutan dan kedalaman 18 milestone di atas sudah sesuai ekspektasi Anda, atau ada milestone yang ingin ditambah/digabung/dipecah?
2. Lanjut ke **Phase 1 — PRD**?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen pertama, melengkapi seluruh Phase 0. Merujuk penuh ke Engineering Playbook v1.0.0, Vision Document v1.0.0, dan ADR v1.1.0 (termasuk keputusan final MinIO, sqlc, Redis Streams) |
