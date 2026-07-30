# Architecture Decision Record (ADR)
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 0 — Architecture Decision Record
**Versi:** 1.1.0
**Status:** Accepted (ADR-007 direvisi pada v1.1.0 — lihat Changelog)
**Referensi Wajib:** `01-engineering-playbook.md` (v1.0.0), `02-vision-document.md` (v1.0.0)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Format ADR yang Dipakai

Setiap ADR mengikuti struktur:

- **Status**: Proposed / Accepted / Superseded
- **Context**: Situasi dan kebutuhan yang memicu keputusan
- **Decision**: Keputusan final
- **Alternatives Considered**: Tabel perbandingan mendalam
- **Consequences**: Konsekuensi positif dan negatif yang diterima
- **Revisit Trigger**: Kondisi konkret yang harus memicu peninjauan ulang keputusan ini

Prinsip: **setiap ADR harus bisa dibaca berdiri sendiri** oleh orang yang tidak familiar dengan diskusi sebelumnya, dan harus menjelaskan trade-off secara jujur — termasuk kelemahan dari opsi yang dipilih, bukan hanya kelebihannya.

---

## ADR-001: Monorepo sebagai Strategi Source Control

**Status:** Accepted (formalisasi dari Engineering Playbook §1)

### Context
Proyek membutuhkan strategi source control yang mendukung visibilitas boundary domain penuh selama masa Modular Monolith, namun tidak menghalangi migrasi menuju Microservices di kemudian hari.

### Decision
Monorepo tunggal `nexus`, dengan Go Workspace untuk isolasi dependency antar module (detail lengkap di Engineering Playbook §1-§2).

### Alternatives Considered

| Opsi | Kelebihan | Kekurangan | Production Value | Learning Value |
|---|---|---|---|---|
| **Monorepo** (dipilih) | Atomic commit lintas domain, visibilitas penuh boundary, tooling sederhana untuk tim kecil | Butuh disiplin `depguard` untuk mencegah coupling tersembunyi; CI bisa melambat bila tidak ada path-filter saat service bertambah banyak | Tinggi — dipakai Google, Meta, Uber pada skala jauh lebih besar | Sangat tinggi — evolusi arsitektur terlihat literal dalam satu git history |
| **Polyrepo** | Isolasi akses/ownership jelas per tim, independent versioning alami | Overhead sinkronisasi kontrak lintas repo, butuh package registry privat, sulit melakukan atomic change lintas domain saat masih monolith | Tinggi untuk organisasi besar dengan banyak tim | Rendah untuk konteks belajar solo — kehilangan pengalaman "merasakan" proses ekstraksi |

### Consequences
- **Positif**: refactor besar (ekstraksi service) dapat dilakukan dalam satu PR yang terlacak jelas, memudahkan review dan rollback.
- **Negatif**: perlu investasi awal pada `depguard` dan path-based CI trigger (direncanakan aktif penuh mulai Phase C).

### Revisit Trigger
Migrasi ke Polyrepo dipertimbangkan ulang jika: (a) jumlah service di Phase D melebihi titik di mana CI end-to-end > 20 menit meski sudah path-filtered, atau (b) proyek berkembang menjadi kolaborasi multi-tim dengan kebutuhan kontrol akses berbeda per service.

---

## ADR-002: Web Framework — Gin vs Echo vs Fiber

**Status:** Accepted

### Context
Dibutuhkan HTTP web framework untuk REST API dan WebSocket upgrade endpoint, dengan performa baik, ekosistem middleware matang, dan learning curve yang wajar.

### Decision
**Gin** dipilih (sesuai spesifikasi proyek), dengan pemahaman trade-off berikut dicatat eksplisit agar keputusan ini bukan sekadar mengikuti instruksi tanpa pemahaman.

### Alternatives Considered

| Kriteria | Gin | Echo | Fiber |
|---|---|---|---|
| Performa (raw throughput) | Sangat baik, berbasis `httprouter`-style radix tree | Sangat baik, sebanding Gin | Tercepat secara benchmark murni, karena berbasis `fasthttp`, bukan `net/http` standar |
| Kompatibilitas ekosistem Go | Penuh — berbasis `net/http`, kompatibel dengan seluruh middleware/library standar (`net/http.Handler`, OpenTelemetry instrumentation resmi, dsb.) | Penuh, sama seperti Gin | **Terbatas** — `fasthttp` TIDAK kompatibel dengan `net/http.Handler` interface, sehingga banyak library observability/middleware pihak ketiga (termasuk OpenTelemetry instrumentation resmi) butuh adapter tambahan atau tidak didukung sama sekali |
| Kematangan & komunitas | Sangat matang, salah satu framework Go paling banyak dipakai di production | Matang, komunitas solid namun lebih kecil dari Gin | Matang untuk use-case tertentu, namun sering menimbulkan isu kompatibilitas library pihak ketiga |
| Learning value untuk proyek ini | Tinggi — mengajarkan pola middleware Go idiomatic yang portable ke proyek lain | Serupa Gin | Risiko mengajarkan pola yang tidak portable (karena `fasthttp` request/response object berbeda drastis dari `net/http`) |
| Kesesuaian dengan OpenTelemetry (learning objective wajib) | Native, didukung penuh oleh `otelgin` resmi | Didukung `otelecho` | Dukungan lebih terbatas/kurang resmi, meningkatkan friksi saat implementasi Observability (Learning Objective eksplisit) |

**Rationale keputusan:** Fiber unggul di benchmark murni, tapi ketidakcocokan dengan `net/http.Handler` menjadi masalah nyata ketika proyek ini punya Learning Objective eksplisit untuk **OpenTelemetry, Prometheus middleware, dan graceful shutdown** yang seluruh tooling referensinya dibangun di atas `net/http`. Memilih Fiber berarti menghadapi *impedance mismatch* yang tidak menambah learning value, hanya menambah friksi. Gin dan Echo setara secara teknis; Gin dipilih karena kematangan ekosistem middleware sedikit lebih luas dan lebih banyak dirujuk di materi pembelajaran Go production-grade.

### Consequences
- **Positif**: kompatibilitas penuh dengan seluruh tooling observability yang direncanakan.
- **Negatif**: melepas potensi throughput tertinggi (Fiber) — diterima karena bottleneck sistem ini kemungkinan besar bukan di HTTP routing layer, melainkan di database/network I/O (dibuktikan lebih lanjut lewat profiling di Performance Design).

### Revisit Trigger
Jika profiling (`pprof`) di production menunjukkan HTTP routing/serialization sebagai bottleneck nyata (bukan asumsi) — sangat tidak mungkin terjadi lebih dulu dibanding I/O database/network, tapi dicatat sebagai kondisi valid untuk revisit.

---

## ADR-003: Database Access Layer — sqlc vs Ent vs GORM vs Prisma Client Go

**Status:** Accepted (dikonfirmasi final — sqlc sebagai satu-satunya database access layer, tidak ada pengecualian Ent untuk domain relasi kompleks)

### Context
Ini adalah salah satu keputusan **paling berdampak** dalam proyek karena memengaruhi seluruh Repository Pattern di setiap domain, kecepatan development, dan type safety di seluruh query database.

### Perbandingan Mendalam

| Kriteria | sqlc | Ent | GORM | Prisma Client Go |
|---|---|---|---|---|
| **Pendekatan** | Code generation dari SQL murni → Go struct & function type-safe | Code generation dari schema Go (graph-based ORM) | Runtime reflection-based ORM (Active Record-ish) | Code generation dari Prisma Schema (`.prisma`), engine Rust di-bundle sebagai binary terpisah |
| **Performance** | **Terbaik** — query SQL murni tanpa reflection overhead, prepared statement langsung | Baik — generated code, namun ada overhead graph traversal untuk eager loading kompleks | Cukup — reflection overhead nyata pada high-throughput, N+1 query mudah terjadi jika tidak hati-hati | Cukup-Kurang — overhead komunikasi ke Prisma engine binary (bukan native Go, ada IPC/FFI ke proses Rust) |
| **Type Safety** | Tinggi untuk query yang ditulis, tapi **developer menulis SQL secara eksplisit** (type safety adalah hasil generate dari SQL yang benar, bukan mencegah SQL salah ditulis) | **Tertinggi** — seluruh query dibangun lewat builder API type-safe termasuk relasi graph, kesalahan terdeteksi saat compile | Rendah — banyak menggunakan `interface{}`/`any` dan reflection, error query sering baru diketahui saat runtime | Tinggi, namun sebagian error tetap muncul di level engine binary (runtime), bukan compile-time Go murni |
| **Learning Value untuk proyek ini** | **Sangat tinggi** — memaksa developer memahami SQL secara langsung (index, join, query plan) yang krusial untuk Learning Objective terkait Query Optimization dan Database Design | Tinggi — mengajarkan pemodelan graph domain dan builder pattern, tapi mengabstraksi SQL asli sehingga pemahaman query plan lebih tidak langsung | Sedang — banyak dipakai industri, tapi mengajarkan kebiasaan yang justru sering menjadi anti-pattern di skala besar (lazy loading tersembunyi, N+1) | Sedang — konsep menarik (schema-first, engine terpisah) tapi ekosistem Go untuk Prisma masih kurang matang dibanding ekosistem TypeScript-nya |
| **Productivity (development speed)** | Sedang — perlu menulis SQL untuk tiap query, namun generate cepat dan iterasi jelas | Tinggi — builder API ekspresif untuk query kompleks dengan relasi | **Tertinggi** di awal — konvensi otomatis (migrasi otomatis dari struct, dsb.) mempercepat prototyping | Tinggi untuk skema sederhana, namun setup engine binary menambah kompleksitas deployment (image Docker lebih besar, perlu bundling binary Prisma engine) |
| **Scalability (kemampuan menangani query kompleks & performa di skala besar)** | **Terbaik** — karena SQL ditulis eksplisit, developer dapat mengoptimalkan query (index hint, CTE, window function) tanpa batasan abstraksi ORM | Baik — mendukung eager loading efisien lewat builder, namun query yang sangat kompleks/spesifik terkadang tetap butuh raw SQL escape hatch | Buruk pada skala besar tanpa disiplin ketat — reflection overhead dan kecenderungan N+1 menjadi masalah nyata di throughput tinggi | Sedang — dibatasi oleh kemampuan Prisma Query Engine menerjemahkan operasi kompleks ke SQL, kadang kurang optimal untuk query sangat spesifik PostgreSQL |
| **Maintainability** | Tinggi — SQL eksplisit di file `.sql` mudah di-review, di-diff, dan dioptimasi langsung oleh siapapun yang paham SQL | Tinggi — schema Go terpusat, migrasi otomatis terkelola oleh Ent | Sedang — struct tag dan konvensi implisit GORM kadang menyulitkan debugging perilaku query yang dihasilkan otomatis | Sedang — schema terpisah dari kode Go (`.prisma` file) menambah satu sumber kebenaran tambahan yang perlu disinkronkan |
| **Community & Ecosystem (Go)** | Besar dan terus tumbuh, didukung banyak perusahaan Go production (mis. dipakai luas di komunitas Go modern) | Cukup besar, didukung resmi sebagai bagian ekosistem Meta/Facebook open source | **Terbesar** di ekosistem Go — paling banyak tutorial, Stack Overflow, dan contoh production | Kecil untuk Go spesifik — mayoritas dokumentasi/komunitas Prisma berpusat di ekosistem Node.js/TypeScript, dukungan Go Client relatif baru dan kurang matang |
| **Production Readiness** | Sangat matang, dipakai luas di production skala besar | Matang, dipakai di production (termasuk oleh Facebook/Meta internal tools) | Sangat matang secara adopsi, namun rawan technical debt performa bila dipakai naif di skala besar | Kurang matang untuk Go — risiko dependency pada binary engine eksternal yang siklus rilisnya mengikuti ekosistem Prisma utama (TypeScript-first), bukan ekosistem Go |

### Decision

**sqlc dipilih.**

### Rationale Detail

1. **Kesesuaian tertinggi dengan Engineering Philosophy proyek** (Explicit over Implicit, Simplicity over Cleverness): sqlc tidak menyembunyikan apa yang sebenarnya terjadi di database — developer menulis SQL, sqlc hanya men-generate binding type-safe di sekitarnya. Ini kontras dengan GORM yang banyak "magic" implisit (lazy loading, auto migration berbasis struct tag) yang justru bertentangan langsung dengan filosofi proyek.
2. **Learning value tertinggi untuk Learning Objective "Query Optimization" dan "Database Design"**: karena SQL ditulis eksplisit, developer dipaksa memahami index, execution plan (`EXPLAIN ANALYZE`), dan trade-off desain skema secara langsung — bukan didelegasikan ke abstraksi ORM.
3. **Performa terbaik** tanpa reflection overhead — relevan untuk NFR 10.000 concurrent users.
4. **Prisma Client Go ditolak** karena kematangan ekosistem Go yang jauh di bawah tiga opsi lain, dan menambah kompleksitas deployment (bundling binary engine Rust) yang tidak sepadan dengan manfaatnya untuk skala proyek ini.
5. **Ent adalah kandidat kuat kedua** — khususnya untuk domain dengan relasi graph kompleks (mis. Role-Permission-Member yang saling terkait erat). Namun untuk konsistensi seluruh proyek dan agar seluruh domain memakai satu pendekatan yang sama (menghindari inkonsistensi arsitektur), sqlc dipilih sebagai standar tunggal. **Catatan penting**: pola query kompleks yang butuh dynamic query builder (mis. filter pencarian dengan banyak kombinasi kondisi opsional) akan memerlukan pendekatan tambahan (dibahas di Low Level Design Phase 4) karena ini adalah kelemahan nyata sqlc yang harus diakui secara jujur — sqlc kurang ekspresif untuk **query dinamis** dibanding query builder seperti Ent/GORM.

### Consequences
- **Positif**: developer memahami database secara mendalam, performa optimal, kode generated 100% type-safe untuk query yang sudah ditulis.
- **Negatif (diterima sebagai trade-off)**: setiap kombinasi query baru butuh menulis SQL eksplisit (tidak ada "generate otomatis" untuk query yang belum ditulis); butuh strategi tambahan untuk kasus dynamic query (mis. search dengan banyak filter opsional) — akan diselesaikan dengan pendekatan *query builder manual terbatas* atau *sqlc named parameter dengan `sqlc.narg()`* yang dijelaskan di Low Level Design.

### Revisit Trigger
Jika di kemudian hari mayoritas endpoint membutuhkan dynamic query builder yang sangat kompleks (bukan hanya beberapa filter opsional), pertimbangkan Ent untuk domain spesifik tersebut (bukan mengganti seluruh proyek) — didokumentasikan sebagai ADR terpisah bila terjadi.

---

## ADR-004: Realtime Communication Technology

**Status:** Accepted

### Context
Dibutuhkan mekanisme komunikasi realtime untuk: pesan baru, typing indicator, presence update, notifikasi live. Perlu dipilih teknologi yang tepat, dengan pemahaman bahwa **REST tetap menjadi communication layer utama** (sesuai spesifikasi proyek) — realtime channel hanya untuk kebutuhan yang benar-benar butuh push dari server ke client.

### Perbandingan Mendalam

| Kriteria | Native/Gorilla WebSocket | Socket.IO | Server-Sent Events (SSE) | gRPC Streaming |
|---|---|---|---|---|
| **Arah komunikasi** | Full-duplex (bidirectional) | Full-duplex (bidirectional), dengan abstraksi tambahan (rooms, namespace, automatic reconnect) | **Unidirectional** (server → client saja) | Full-duplex (bidirectional), termasuk client-streaming dan bidirectional-streaming |
| **Throughput** | Tinggi — overhead protokol minimal setelah handshake | Sedang — overhead tambahan dari layer protokol Socket.IO sendiri (framing tambahan di atas WebSocket/polling fallback) | Tinggi untuk push-only, namun terbatas oleh HTTP/1.1 connection limit per browser (~6 koneksi per domain) bila tidak pakai HTTP/2 | Tinggi, memanfaatkan HTTP/2 multiplexing secara native |
| **Latency** | Rendah | Rendah-Sedang (sedikit lebih tinggi dari native WS karena overhead framing) | Rendah untuk push, namun tidak relevan untuk kebutuhan client→server realtime (chat perlu client mengirim juga) | Rendah |
| **Skalabilitas horizontal** | Butuh strategi eksplisit (sticky session atau pub/sub broadcast lintas instance — dibahas di HLD) | Sama seperti WebSocket native + kompleksitas tambahan bila pakai Socket.IO adapter (mis. Redis adapter) | Baik untuk push-only, load balancer lebih sederhana karena unidirectional dan bisa lewat HTTP/1.1 biasa | Baik, namun butuh load balancer yang mendukung HTTP/2 secara penuh (bukan semua reverse proxy versi lama mendukung gRPC streaming dengan baik — Traefik modern mendukung) |
| **Learning Value** | **Sangat tinggi** — memahami WebSocket protocol secara langsung (upgrade handshake, ping/pong, frame), fundamental untuk memahami seluruh teknologi realtime lain | Sedang — mengajarkan abstraksi library, bukan protokol dasar; risiko "black box" tanpa memahami fallback mechanism di baliknya | Sedang — konsep penting untuk dipahami (kapan push-only cukup) tapi tidak cukup untuk chat bidirectional | Tinggi — mengajarkan HTTP/2 streaming dan Protocol Buffers, relevan untuk komunikasi internal antar service di Phase D |
| **Production Readiness** | Sangat matang (Gorilla WebSocket adalah standar de-facto Go) | Matang, namun Socket.IO server Go tidak seresmi/sematang versi Node.js aslinya | Matang, browser native support penuh | Matang untuk service-to-service; dukungan browser native untuk gRPC-Web butuh proxy tambahan (envoy/grpc-web) |
| **Kesesuaian kebutuhan proyek** | **Cocok** — chat butuh bidirectional (client kirim pesan/typing, server push pesan baru/presence) | Fallback mechanism (long-polling) tidak relevan karena target modern browser sudah mendukung WebSocket penuh — kompleksitas tambahan tanpa manfaat proporsional | **Tidak cukup sendirian** — hanya cocok sebagai pelengkap untuk notifikasi one-way sederhana, bukan pengganti utama chat | Sangat cocok untuk **komunikasi internal antar service** (Phase C/D), kurang cocok untuk client browser langsung tanpa proxy tambahan |

### Decision

- **Client ↔ Server (browser)**: **Gorilla WebSocket** (native, bukan Socket.IO) untuk seluruh kebutuhan realtime end-user: pesan baru, typing indicator, presence update, read receipt.
- **Service ↔ Service (internal, Phase C/D)**: **gRPC** dipertimbangkan untuk komunikasi sinkron antar service berlatensi rendah (dibahas lebih lanjut di ADR terpisah bila relevan saat HLD Phase C).
- **SSE tidak dipakai** sebagai mekanisme utama, namun dicatat sebagai opsi valid untuk kasus spesifik one-way notification stream sederhana di masa depan bila muncul kebutuhan (mis. live deployment log ke admin panel) — bukan untuk chat.

### Rationale Detail

Socket.IO ditolak meski populer, karena dua alasan: (1) fallback mechanism (long-polling) yang menjadi nilai jual utamanya sudah tidak relevan untuk target browser modern, (2) implementasi server Socket.IO di Go tidak seresmi dan sematang di Node.js, menambah risiko tanpa manfaat proporsional. Native Gorilla WebSocket dipilih karena **memaksa pemahaman protokol WebSocket secara langsung** — nilai pembelajaran yang jauh lebih tinggi, dan merupakan fondasi yang membuat pemahaman Socket.IO (bila suatu saat dibutuhkan) menjadi mudah (sebaliknya tidak berlaku).

SSE ditolak sebagai mekanisme utama karena chat pada dasarnya butuh client mengirim data (pesan, typing signal) secara realtime juga — unidirectional tidak mencukupi kebutuhan inti.

### Consequences
- **Positif**: kontrol penuh atas protokol, learning value maksimal, tidak ada dependency tambahan untuk fallback mechanism yang tidak dibutuhkan.
- **Negatif**: perlu membangun sendiri mekanisme reconnect, heartbeat/ping-pong, dan message acknowledgment di level aplikasi (yang otomatis disediakan Socket.IO) — diterima karena ini justru bagian dari Learning Objective.

### Revisit Trigger
Jika di kemudian hari dibutuhkan dukungan browser sangat lama tanpa WebSocket (skenario sangat tidak mungkin untuk target proyek ini), atau jika tim berkembang dan butuh abstraksi client library siap pakai lintas platform — revisit Socket.IO.

---

## ADR-005: Voice & Video Infrastructure — LiveKit vs mediasoup vs Janus vs WebRTC Murni

**Status:** Accepted

### Context
Dibutuhkan infrastruktur voice/video channel. WebRTC sebagai protokol dasar bersifat peer-to-peer secara default, namun untuk voice channel multi-partisipan (bukan 1-on-1 call), dibutuhkan **SFU (Selective Forwarding Unit)** agar tidak membebani setiap client mengirim stream ke semua partisipan lain (mesh topology tidak scalable).

### Perbandingan Mendalam

| Kriteria | WebRTC Murni (mesh, tanpa SFU) | Janus | mediasoup | LiveKit |
|---|---|---|---|---|
| **Kompleksitas implementasi** | Rendah untuk 1-on-1, namun **naik drastis secara kuadratik** untuk multi-partisipan (setiap client harus koneksi ke semua client lain — N×(N-1) koneksi) | Tinggi — Janus adalah general-purpose WebRTC gateway berbasis plugin C, butuh pemahaman mendalam arsitektur plugin dan sinyal SIP-like | Tinggi — mediasoup adalah SFU low-level (Node.js/C++ core) yang memberi kontrol penuh tapi developer harus membangun signaling layer, room management, dan recording sendiri dari nol | **Rendah-Sedang** — LiveKit adalah SFU modern dengan signaling, room management, recording, dan SDK client (termasuk untuk Vue/React) sudah tersedia out-of-the-box |
| **Skalabilitas** | Sangat buruk untuk > 4-5 partisipan (bandwidth client naik linear per partisipan tambahan) | Baik, terbukti di banyak deployment production (Janus dipakai luas untuk broadcasting) | Sangat baik, dirancang untuk skala tinggi dengan performa native C++ | Baik, dibangun di atas prinsip SFU modern, mendukung horizontal scaling multi-node dengan built-in mechanism |
| **Learning Value** | Tinggi untuk **memahami dasar WebRTC** (ICE, STUN/TURN, SDP negotiation) — **wajib dipahami konsepnya** meski tidak dipakai langsung sebagai solusi akhir | Tinggi — mengajarkan arsitektur SFU dan plugin-based gateway | Tinggi — mengajarkan SFU dari sisi yang lebih "close to the metal", termasuk router/transport/producer/consumer model | Sedang-Tinggi — mengajarkan konsep SFU modern dan integrasi praktis, namun signaling detail lebih tersembunyi di balik SDK dibanding mediasoup |
| **Production Readiness** | Tidak layak production untuk voice channel > beberapa orang | Matang, dipakai production luas (mis. untuk broadcasting/streaming) | Sangat matang, dipakai perusahaan besar untuk video conferencing skala tinggi | Sangat matang, open-source dengan traksi cepat, dipakai banyak startup modern, mendukung self-hosted maupun cloud |
| **Effort setup untuk fitur lengkap (room, recording, client SDK)** | N/A (tidak dipakai) | Tinggi — perlu konfigurasi plugin manual, signaling server terpisah | **Sangat tinggi** — hampir seluruh signaling, state management room, dan reconnection logic harus dibangun sendiri | **Rendah** — signaling server, room API, dan client SDK (JS/TS termasuk Vue-friendly) sudah disediakan, tinggal diintegrasikan |
| **Kesesuaian dengan proyek belajar dengan waktu terbatas** | Hanya sebagai bahan pemahaman konsep dasar, bukan solusi | Overhead setup tinggi untuk manfaat yang tidak proporsional dibanding LiveKit untuk kasus voice/video channel chat app | Overhead signaling layer custom terlalu besar untuk scope PBL ini — risiko waktu belajar habis untuk "plumbing" bukan untuk konsep inti | **Paling seimbang** — cukup banyak yang harus dipelajari (SFU concept, room lifecycle, token-based auth) namun tidak sampai harus membangun signaling dari nol |

### Decision

**LiveKit dipilih**, sesuai spesifikasi proyek dan dikonfirmasi tepat melalui analisis di atas.

### Rationale Detail

Pemahaman **konsep WebRTC dasar (ICE/STUN/TURN/SDP) tetap menjadi bagian wajib Learning Roadmap** (dipelajari secara konseptual, bukan diimplementasikan dari nol), karena LiveKit sendiri dibangun di atas fondasi ini — memahami apa yang "disembunyikan" LiveKit sama pentingnya dengan memakainya. Namun untuk **implementasi produksi**, membangun SFU dari nol (mediasoup) atau general gateway (Janus) tidak memberi learning value tambahan yang proporsional terhadap waktu yang dihabiskan untuk plumbing signaling — waktu tersebut lebih baik dialokasikan untuk mempelajari domain lain (Event-Driven, Microservices extraction) yang menjadi fokus utama proyek ini.

### Consequences
- **Positif**: implementasi voice/video channel dapat berjalan dengan effort proporsional, memungkinkan proyek tetap fokus pada Learning Objective arsitektur backend secara keseluruhan.
- **Negatif**: pemahaman **internal SFU** (bagaimana LiveKit benar-benar meneruskan RTP packet) tidak didapat sedalam bila membangun mediasoup dari nol — diterima sebagai trade-off yang disengaja (didokumentasikan sebagai *conceptual learning* saja untuk bagian ini, bukan *hands-on build*).

### Revisit Trigger
Jika Learning Objective berubah untuk secara spesifik fokus pada "membangun SFU dari nol" (bukan "menggunakan realtime infrastructure di sistem Discord-like"), revisit mediasoup sebagai proyek pembelajaran terpisah.

---

## ADR-006: Message Broker / Event Backbone — Redis Pub/Sub vs Redis Streams vs NATS vs Kafka

**Status:** Accepted (dikonfirmasi final)

### Context
Dibutuhkan mekanisme event backbone untuk Event-Driven Architecture (Phase B ke atas): Outbox Relay, notifikasi asynchronous, dan (di Phase C/D) komunikasi event lintas service.

### Perbandingan Mendalam

| Kriteria | Redis Pub/Sub | Redis Streams | NATS (Core + JetStream) | Kafka |
|---|---|---|---|---|
| **Durability (persistence event)** | **Tidak ada** — pesan hilang bila tidak ada subscriber aktif saat dikirim (fire-and-forget) | Ada — event tersimpan di stream, consumer dapat membaca ulang dari posisi tertentu (consumer group model mirip Kafka) | Core NATS: tidak persistent (mirip Pub/Sub). JetStream: persistent, mendukung replay | **Sangat kuat** — didesain untuk retensi jangka panjang, replay penuh, dan throughput sangat tinggi |
| **Kompleksitas operasional** | Sangat rendah — sudah tersedia karena Redis sudah dipakai untuk cache | Rendah-Sedang — masih dalam Redis yang sama, tidak butuh infrastruktur tambahan | Sedang — butuh deployment cluster NATS terpisah (walau ringan dibanding Kafka) | **Tinggi** — butuh Zookeeper/KRaft, partisi, replikasi, tuning kompleks; overkill untuk skala proyek ini |
| **Throughput & Latency** | Sangat rendah latency, throughput baik untuk skala kecil-menengah | Baik, sedikit overhead dibanding Pub/Sub murni karena persistence | Sangat tinggi throughput, latency sangat rendah (dirancang untuk cloud-native messaging) | Sangat tinggi throughput untuk skala besar, namun latency per-message sedikit lebih tinggi karena batching/replication overhead |
| **Learning Value untuk Outbox Pattern & Idempotency** | Rendah — tanpa persistence, tidak bisa benar-benar mendemonstrasikan "at-least-once delivery" dan retry yang menjadi inti Learning Objective | **Tinggi** — consumer group, acknowledgment (`XACK`), pending entries list (`XPENDING`) adalah mekanisme nyata untuk mempelajari retry, dead-letter, dan idempotency | Tinggi (dengan JetStream) — konsep durable consumer dan ack/nack serupa | Tinggi, namun kompleksitas operasional tambahan mengalihkan fokus belajar dari konsep event-driven itu sendiri ke "cara mengoperasikan Kafka" |
| **Kesesuaian skala proyek** | Terlalu sederhana untuk kebutuhan Outbox Pattern yang butuh durability | **Pas** — sudah cukup untuk skala NFR (10.000 concurrent user), dan Redis sudah menjadi dependency wajib proyek (cache) sehingga tidak menambah infrastruktur baru | Baik, namun menambah satu komponen infrastruktur baru yang perlu dioperasikan & di-monitor terpisah | Berlebihan (overkill) untuk skala 10.000 concurrent users — kompleksitas operasional tidak sepadan dengan kebutuhan aktual |
| **Production Readiness** | Matang tapi bukan untuk use-case durable messaging | Matang, dipakai production untuk skala menengah | Sangat matang, dipakai production skala besar (cloud-native) | Sangat matang, standar industri untuk skala sangat besar |

### Decision

**Redis Streams dipilih** sebagai event backbone utama untuk Phase B (Event-Driven Modular Monolith) dan tetap dipertahankan hingga Phase C, dengan evaluasi ulang eksplisit di Phase D.

### Rationale Detail

Redis sudah menjadi dependency wajib proyek ini (cache, session, rate limiting). Redis Streams memberi **durability dan consumer-group semantics** yang cukup untuk mendemonstrasikan seluruh Learning Objective terkait Outbox Pattern, retry strategy, dan idempotency **tanpa menambah komponen infrastruktur baru**, konsisten dengan prinsip *Simplicity over Cleverness* dan *YAGNI* — Kafka/NATS akan menambah kompleksitas operasional yang tidak proporsional dengan skala NFR proyek ini (10.000 concurrent users, bukan skala jutaan event/detik yang menjadi target asli Kafka).

Redis Pub/Sub murni ditolak karena **tidak memberikan durability**, sehingga tidak bisa mendemonstrasikan pola Outbox/retry/dead-letter secara jujur (event akan hilang begitu saja jika consumer down — bertentangan langsung dengan Learning Objective "Idempotency" dan "Retry Strategy" yang mengasumsikan pesan tidak hilang).

### Consequences
- **Positif**: tidak ada komponen infrastruktur baru, cukup untuk seluruh Learning Objective terkait event-driven pada skala proyek ini.
- **Negatif**: Redis Streams tidak memiliki fitur *partitioning* dan *log compaction* setara Kafka — pada Phase D dengan volume event sangat tinggi lintas banyak service, ini bisa menjadi batasan nyata.

### Revisit Trigger
Pada Phase D, jika volume event lintas service dan kebutuhan replay jangka panjang (retensi berbulan-bulan, audit log skala besar) menjadi kebutuhan nyata (bukan hipotetis) — evaluasi migrasi ke **NATS JetStream** (lebih ringan dibanding Kafka, cloud-native) sebagai prioritas pertama, Kafka sebagai opsi berikutnya hanya jika skala benar-benar mendekati jutaan event/detik.

**Catatan penting terkait spesifikasi proyek:** sesuai instruksi awal, *"Jangan gunakan queue sebagai komunikasi antar microservice kecuali memang diperlukan dan dijelaskan alasannya"* — pada Phase C/D, Redis Streams **hanya** dipakai untuk domain event yang secara inheren asynchronous (notifikasi, indexing search, dsb.), sementara komunikasi yang butuh respons langsung antar service tetap memakai REST/gRPC (lihat ADR-004 dan HLD).

---

## ADR-007: Object Storage — MinIO (Self-Hosted)

**Status:** Accepted (v1.1.0 — REVISI dari keputusan awal Cloudinary)

> **Catatan revisi:** Draft awal ADR ini (v1.0.0) melaporkan konflik antara spesifikasi awal proyek (Cloudinary) dan Learning Objective "Asynchronous Processing"/domain "Media". Anda telah menyetujui rekomendasi untuk beralih **penuh** ke MinIO self-hosted. Keputusan ini **menggantikan Cloudinary di seluruh dokumen proyek** — lihat catatan dampak di §"Dampak Perubahan terhadap Spesifikasi Proyek" di bawah.

### Context
Dibutuhkan object storage untuk upload attachment (gambar, video, audio, PDF, ZIP) hingga 1 GB per file, yang selaras dengan Learning Objective "Asynchronous Processing" dan domain "Media" (pipeline transcoding/processing dibangun sendiri, bukan didelegasikan ke pihak ketiga).

### Perbandingan Mendalam

| Kriteria | Cloudinary | **MinIO (self-hosted)** | AWS S3 |
|---|---|---|---|
| **Kemudahan integrasi** | Sangat tinggi — SDK matang, transformasi media built-in via URL parameter | Sedang — API S3-compatible (SDK Go resmi `minio-go` tersedia), perlu deployment & maintenance sendiri | Tinggi — SDK matang, ekosistem terbesar |
| **Biaya** | Free tier terbatas, biaya naik cepat untuk file besar & bandwidth transformasi | **Gratis** — self-hosted, hanya biaya infrastruktur VPS/disk yang sudah ada | Bayar sesuai pemakaian, ada biaya nyata untuk proyek non-komersial |
| **Built-in media processing** | Sangat kuat, tapi **menyembunyikan** proses yang seharusnya dipelajari | Tidak ada bawaan — **inilah yang diinginkan**: memaksa membangun sendiri pipeline transcoding via Asynq worker + `ffmpeg`/image library Go | Tidak ada bawaan, butuh Lambda/MediaConvert terpisah |
| **Learning Value** | Sedang — mengajarkan integrasi third-party, tapi tidak melatih Asynchronous Processing | **Tertinggi** — melatih langsung Learning Objective "Distributed Task Queue" (Asynq), pipeline pemrosesan file besar, dan API S3 (protokol dipahami luas di industri, skill portable ke S3 asli kapan saja) | Tinggi juga, namun ada biaya yang tidak perlu |
| **Kedaulatan/kontrol data** | Rendah — data disimpan di infrastruktur pihak ketiga | **Tinggi** — data sepenuhnya berada di infrastruktur sendiri, relevan untuk mempelajari operasional storage (bucket policy, lifecycle, retention) secara langsung | Rendah — tetap di infrastruktur AWS |
| **Kesesuaian dengan filosofi proyek** ("Production First Mindset", "bangun untuk belajar") | Bertentangan — mendelegasikan konsep inti yang seharusnya dipelajari | **Paling selaras** | Selaras secara teknis, tapi ada biaya |

### Decision

**MinIO (self-hosted) dipilih sebagai satu-satunya object storage**, menggantikan Cloudinary sepenuhnya.

### Rationale Detail

1. MinIO API sepenuhnya **S3-compatible**, sehingga kode yang ditulis (`minio-go` SDK atau AWS SDK dengan custom endpoint) portable ke AWS S3 asli kapan saja tanpa perubahan arsitektur — keputusan ini tidak mengorbankan opsi migrasi ke cloud storage komersial di masa depan.
2. Pipeline upload wajib melalui alur: **upload sementara → validasi (tipe file, ukuran, scanning dasar) → enqueue task pemrosesan via Asynq → worker memproses (thumbnail generation untuk gambar, transcoding ringan untuk video/audio via `ffmpeg`, ekstraksi metadata) → simpan hasil akhir ke bucket MinIO → update record attachment dengan URL final**. Alur ini secara langsung melatih Learning Objective "Asynchronous Processing" dan "Distributed Task Queue" yang sebelumnya berisiko tidak tercapai dengan Cloudinary.
3. MinIO dijalankan sebagai container tambahan di Docker Compose (Phase A-C) dan sebagai StatefulSet/Operator di Kubernetes (Phase D) — konsisten dengan evolusi deployment di ADR-009, tanpa menambah dependency layanan eksternal berbayar.
4. Bucket dipisah per keperluan: `nexus-attachments` (attachment umum), `nexus-avatars` (avatar user/workspace), dengan lifecycle policy berbeda (dibahas detail di Database/Storage Design fase berikutnya).

### Dampak Perubahan terhadap Spesifikasi Proyek

Karena spesifikasi awal proyek (pesan pertama Anda) secara eksplisit menyebut Cloudinary di bagian **BACKEND → Object Storage**, perubahan ini dicatat sebagai **amandemen resmi terhadap spesifikasi proyek**, berlaku efektif sejak ADR ini (v1.1.0) dan mengikat seluruh dokumen berikutnya:

- **Seterusnya**: setiap referensi "Object Storage" di PRD, SRS, HLD, LLD, Database Design, API Specification, Security Design, dan Deployment Architecture akan memakai **MinIO**, bukan Cloudinary.
- **Environment variable** terkait akan memakai penamaan `NEXUS_API_MINIO_ENDPOINT`, `NEXUS_API_MINIO_ACCESS_KEY`, `NEXUS_API_MINIO_SECRET_KEY`, `NEXUS_API_MINIO_BUCKET_ATTACHMENTS`, dst., mengikuti konvensi §7.3 Engineering Playbook (bukan `NEXUS_API_CLOUDINARY_*`).
- **Deployment**: `deployments/docker-compose/` akan menyertakan service MinIO sejak Docker Compose tahap awal (bagian dari Deployment Architecture Phase 8).
- Dokumen `01-engineering-playbook.md` dan `02-vision-document.md` **tidak menyebut Cloudinary secara eksplisit**, sehingga tidak memerlukan revisi tambahan — perubahan ini murni terlokalisir di ADR dan akan diteruskan otomatis ke dokumen-dokumen fase berikutnya yang belum dibuat.

### Consequences
- **Positif**: learning value maksimal untuk domain Media & Asynchronous Processing, tanpa biaya layanan pihak ketiga, skill yang portable (S3-compatible API).
- **Negatif**: menanggung sendiri beban operasional storage (backup, replication, disk capacity planning) yang sebelumnya didelegasikan ke Cloudinary — ini **diterima secara sadar** karena selaras dengan Learning Objective "Production Readiness" dan "Backup Strategy" yang memang menjadi bagian NFR proyek.
- **Negatif**: tidak ada CDN bawaan seperti Cloudinary — bila performa delivery attachment ke end-user menjadi perhatian di Phase D, perlu dipertimbangkan reverse-proxy caching layer (Traefik + cache middleware) atau CDN eksternal murni sebagai lapisan tambahan (bukan mengganti MinIO, hanya menambah caching di depannya).

### Revisit Trigger
Jika beban operasional self-hosted storage (disk capacity, backup) menjadi tidak proporsional dengan skala pembelajaran yang tersisa, atau jika kebutuhan CDN global menjadi prioritas eksplisit — pertimbangkan menambahkan CDN caching layer di depan MinIO (bukan mengganti MinIO itu sendiri, kecuali ada keputusan eksplisit baru).

---

## ADR-008: Reverse Proxy / API Gateway — Traefik vs Nginx vs Caddy

**Status:** Accepted

### Context
Dibutuhkan reverse proxy sekaligus API Gateway yang menangani routing, TLS termination, load balancing, dan mendukung kebutuhan Blue-Green Deployment serta (di Phase D) service discovery dinamis.

### Perbandingan Mendalam

| Kriteria | Traefik | Nginx | Caddy |
|---|---|---|---|
| **Dynamic Configuration (service discovery)** | **Native** — mendeteksi otomatis container/service baru lewat Docker labels atau Kubernetes CRD tanpa reload manual | Butuh reload konfigurasi manual (atau tooling tambahan seperti `nginx-gen`/`consul-template`) saat service berubah | Otomatis untuk beberapa kasus, namun ekosistem plugin untuk service discovery dinamis kurang semendalam Traefik |
| **Kesesuaian dengan Docker Compose → Kubernetes evolution** | **Sangat sesuai** — konfigurasi berbasis label bekerja identik secara konsep di Docker Compose maupun Kubernetes (Ingress Controller Traefik tersedia), memudahkan transisi deployment (Learning Objective evolusi deployment) | Perlu rekonfigurasi signifikan saat pindah dari Compose ke Kubernetes (walau Ingress-Nginx tersedia, filosofi konfigurasinya berbeda dari static config file) | Baik untuk kesederhanaan, namun dukungan Kubernetes Ingress kurang semendalam Traefik/Nginx |
| **TLS Otomatis (Let's Encrypt)** | Built-in otomatis | Butuh Certbot terpisah | Built-in otomatis (bahkan paling sederhana konfigurasinya) | 
| **Kesesuaian untuk Blue-Green Deployment** | **Sangat baik** — mendukung weighted round-robin dan dynamic backend switching tanpa downtime, relevan langsung untuk Learning Objective Blue-Green/Zero-Downtime | Mendukung, namun butuh scripting tambahan (reload konfigurasi) untuk switch backend secara mulus | Mendukung dasar, namun ekosistem otomasi Blue-Green kurang sematang Traefik |
| **Dashboard & Observability bawaan** | Dashboard built-in menampilkan routing real-time, metric Prometheus native | Butuh modul tambahan (`nginx-prometheus-exporter`) | Metric tersedia namun dashboard visual bawaan tidak sekomprehensif Traefik |
| **Learning Value** | **Tinggi** — mengajarkan konsep modern cloud-native reverse proxy yang relevan langsung untuk transisi ke Kubernetes | Tinggi juga — Nginx adalah skill yang sangat portable dan dipakai luas di industri secara umum (bukan hanya cloud-native context) | Sedang — konfigurasi sangat sederhana (Caddyfile) baik untuk belajar dasar reverse proxy, namun kurang relevan untuk Learning Objective spesifik proyek ini (Service Discovery, Blue-Green otomatis) |
| **Production Readiness** | Matang, dipakai luas terutama di lingkungan container-native | Sangat matang, standar industri selama puluhan tahun | Matang untuk skala kecil-menengah |

### Decision

**Traefik dipilih**, sesuai spesifikasi awal, dan analisis di atas menunjukkan pilihan ini memang paling selaras dengan Learning Objective proyek (Service Discovery, Blue-Green Deployment, evolusi Docker Compose → Kubernetes).

### Consequences
- **Positif**: konfigurasi berbasis label/CRD memberikan pengalaman belajar yang konsisten sepanjang evolusi deployment (Tahap 1-5 di Engineering Playbook §Deployment Evolution).
- **Negatif**: skill Nginx (yang jauh lebih portable secara umum di industri, termasuk untuk peran DevOps non-cloud-native) relatif kurang terlatih — diterima sebagai trade-off karena fokus proyek adalah cloud-native evolution.

### Revisit Trigger
Tidak ada pemicu revisit spesifik — Traefik cocok di seluruh fase deployment yang direncanakan (Phase 1-5 Deployment Evolution).

---

## ADR-009: Deployment Platform — Docker Compose vs Kubernetes

**Status:** Accepted (Evolusioner, bukan pilihan tunggal)

### Context
Sesuai Vision Document, proyek berevolusi melalui beberapa tahap deployment. Keputusan bukan "pilih salah satu selamanya", melainkan **kapan transisi terjadi**.

### Perbandingan & Rationale per Tahap

| Kriteria | Docker Compose | Kubernetes |
|---|---|---|
| Kompleksitas operasional | Rendah — satu file YAML, cocok single-node | Tinggi — butuh pemahaman Pod, Service, Deployment, Ingress, ConfigMap, Secret, HPA, dsb. |
| Skalabilitas | Terbatas pada satu mesin (kecuali Docker Swarm, yang tidak direncanakan dipakai) | Dirancang native untuk multi-node horizontal scaling |
| Learning Value tahap awal | Tinggi untuk memahami container fundamentals tanpa distraksi orkestrasi kompleks | Rendah bila dipakai terlalu dini — kompleksitas orkestrasi mengalihkan fokus dari Learning Objective inti (domain design, event-driven, dsb.) di Phase A/B |
| Learning Value tahap lanjut (Phase D) | Tidak cukup — tidak ada mekanisme native untuk service discovery lintas node, rolling update terbatas | **Wajib** untuk benar-benar mempraktikkan Rolling Update, Horizontal Scaling, dan Multi-Node sesuai Learning Objective eksplisit |
| Kesesuaian dengan NFR 10.000 concurrent users | Bisa dicapai di satu VPS besar dengan tuning yang tepat (tergantung resource) | Lebih sesuai untuk scaling horizontal yang genuine |

### Decision

**Evolusi bertahap** (detail lengkap dibahas di Deployment Architecture Phase 8, di sini hanya keputusan prinsip):

- **Docker Compose** dipakai pada Deployment Tahap 1-3 (Docker Compose lokal, Single VPS, Blue-Green sederhana) — selaras Phase A/B/C arsitektur.
- **Kubernetes** diperkenalkan mulai Deployment Tahap 4-5 (Horizontal Scaling, Multi-Node) — selaras Phase C akhir/D arsitektur, saat kebutuhan orkestrasi multi-service benar-benar nyata.

### Rationale Detail

Memperkenalkan Kubernetes terlalu dini (sebelum ada lebih dari satu service yang benar-benar butuh di-scale independen) akan melanggar YAGNI dan mengalihkan energi belajar dari Learning Objective arsitektur inti ke sekadar "belajar YAML Kubernetes". Manifest `k8s/` **disiapkan sebagai skeleton sejak awal** (dicatat sebagai technical debt disengaja di Engineering Playbook §Ringkasan) namun **tidak diaktifkan** hingga benar-benar dibutuhkan.

### Consequences
- **Positif**: setiap tahap deployment punya kompleksitas yang proporsional dengan kebutuhan riil saat itu.
- **Negatif**: penundaan pengenalan Kubernetes berarti skill tersebut baru benar-benar dipraktikkan di fase akhir proyek — diterima karena selaras filosofi evolusioner keseluruhan proyek.

### Revisit Trigger
Transisi ke Kubernetes dipicu bukan oleh waktu, melainkan oleh kondisi: minimal 3 service independen di Phase C/D yang butuh scaling terpisah, ATAU kebutuhan nyata mempraktikkan Rolling Update/Multi-Node sebagai Learning Objective yang belum tercapai lewat cara lain.

---

## ADR-010: Modular Monolith sebagai Starting Point (bukan Microservices sejak awal)

**Status:** Accepted (formalisasi dari Vision Document §7)

### Context
Target akhir proyek adalah Full Microservices. Perlu keputusan eksplisit mengapa **tidak dimulai langsung** dari microservices.

### Perbandingan

| Kriteria | Modular Monolith (starting point) | Microservices (starting point) |
|---|---|---|
| Learning value untuk memahami *kapan* microservices dibutuhkan | **Tinggi** — merasakan langsung titik sakit (scaling terbatas, deployment coupling) yang memotivasi ekstraksi | Rendah — microservices sejak awal berarti asumsi kebutuhan yang belum tentu benar, tanpa pernah membuktikan trade-off-nya |
| Kompleksitas operasional awal | Rendah — satu deployment unit | Tinggi — butuh service discovery, distributed tracing, network reliability handling sejak hari pertama, padahal domain logic belum matang |
| Risiko Distributed Monolith (microservices yang saling coupled erat, kehilangan seluruh manfaat microservices tapi tetap menanggung kompleksitasnya) | Risiko dimitigasi lewat disiplin boundary domain sejak dalam monolith (`depguard`, §2.3 Playbook) sebelum ekstraksi | **Risiko sangat tinggi** bila boundary domain belum matang — kesalahan pemula paling umum dalam mempelajari microservices |
| Kecepatan development fitur baru di tahap awal | Cepat — tidak ada overhead network call antar domain | Lambat — setiap perubahan lintas domain butuh koordinasi API contract antar service yang belum stabil |

### Decision

**Modular Monolith** sebagai starting point (Phase A), dengan disiplin boundary domain ketat sejak awal agar ekstraksi ke microservices (Phase C/D) dapat dilakukan dengan **friksi minimal** ketika saatnya tiba.

### Rationale Detail

Ini adalah keputusan yang selaras penuh dengan **seluruh filosofi proyek**: memahami *mengapa* dan *kapan*, bukan mengikuti tren arsitektur. Memulai dari microservices tanpa pernah merasakan monolith akan menghasilkan pemahaman yang dangkal tentang trade-off nyata — persis kelemahan yang ingin ditutup proyek ini (lihat Vision Document §1.1).

### Consequences
- **Positif**: fondasi domain model yang matang sebelum menanggung kompleksitas distributed system.
- **Negatif**: Learning Objective terkait microservices (service discovery, distributed tracing penuh, dsb.) baru benar-benar dipraktikkan di paruh kedua proyek — diterima karena urutan ini justru yang paling edukatif.

### Revisit Trigger
Tidak ada — ini adalah keputusan fondasional yang sudah selaras dengan Vision Document dan tidak diharapkan berubah.

---

## Ringkasan Keputusan

| ADR | Keputusan |
|---|---|
| 001 | Monorepo |
| 002 | Gin (ditolak: Fiber karena inkompatibilitas `net/http`, khususnya untuk OpenTelemetry) |
| 003 | sqlc (ditolak: GORM karena implicit magic, Prisma Go karena ekosistem belum matang; Ent sebagai kandidat kuat kedua) |
| 004 | Gorilla WebSocket native untuk client-server; SSE/Socket.IO ditolak sebagai mekanisme utama |
| 005 | LiveKit (ditolak: mediasoup/Janus karena overhead signaling custom tidak proporsional) |
| 006 | Redis Streams — **dikonfirmasi final** (ditolak: Kafka karena overkill, Pub/Sub karena tidak durable) |
| 007 | **MinIO self-hosted — dikonfirmasi final**, menggantikan Cloudinary sepenuhnya (amandemen resmi spesifikasi proyek) |
| 008 | Traefik |
| 009 | Docker Compose → Kubernetes evolusioner, bukan pilihan tunggal |
| 010 | Modular Monolith sebagai starting point |

## Trade-off yang Diterima

- Melepas throughput maksimal (Fiber) demi kompatibilitas observability tooling (Gin).
- Melepas ekspresivitas dynamic query builder (Ent/GORM) demi performa dan explicit SQL (sqlc) — butuh strategi tambahan di LLD untuk kasus dynamic filter.
- Menunda Kubernetes hingga benar-benar dibutuhkan, menerima bahwa skill tersebut baru terlatih di akhir proyek.
- Menanggung beban operasional storage sendiri (backup, replication, capacity planning) dengan beralih ke MinIO, demi learning value Asynchronous Processing yang lebih tinggi dan bebas biaya layanan pihak ketiga.

## Risiko Arsitektur

- Redis Streams (ADR-006) memiliki batas skalabilitas dibanding Kafka — perlu dipantau saat volume event bertambah di Phase D.
- MinIO self-hosted (ADR-007) tidak memiliki CDN bawaan seperti Cloudinary — berpotensi menjadi bottleneck delivery attachment di skala besar; dimitigasi lewat caching layer tambahan bila diperlukan di kemudian hari (bukan mengganti MinIO).
- MinIO menambah satu komponen infrastruktur stateful yang harus dikelola sendiri (backup, disk capacity) sejak Docker Compose tahap awal — perlu direncanakan strategi backup sejak Deployment Architecture (Phase 8), bukan ditunda.

## Technical Debt yang Sengaja Diterima

- Belum ada keputusan final mekanisme dynamic query filter untuk sqlc (search dengan banyak kombinasi filter opsional) — akan diselesaikan di Low Level Design.
- `k8s/` manifest disiapkan sebagai skeleton namun tidak divalidasi berjalan hingga Phase D benar-benar tiba.
- Strategi backup & disaster recovery untuk data MinIO belum didetailkan di dokumen ini — wajib dibahas eksplisit di Non-Functional Requirements (SRS) dan Deployment Architecture (Phase 8), dicatat sebagai item wajib agar tidak terlewat.

## Hal yang Perlu Dikonfirmasi Sebelum Lanjut ke Fase Berikutnya

Seluruh keputusan teknologi utama pada dokumen ini (ADR-001 s.d. ADR-010) telah **dikonfirmasi final** oleh Anda, termasuk revisi ADR-007 ke MinIO self-hosted. Satu-satunya hal yang perlu dikonfirmasi sebelum lanjut:

1. Lanjut ke dokumen **Learning Roadmap** berikutnya (dokumen terakhir Phase 0), atau ada revisi lebih lanjut terhadap ADR ini?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | 10 ADR pertama mencakup seluruh perbandingan teknologi wajib sesuai project instructions; 1 konflik ditemukan dan dilaporkan (ADR-007) |
| 1.1.0 | Revisi | ADR-007 direvisi total: Cloudinary digantikan penuh oleh **MinIO self-hosted** atas persetujuan eksplisit, dengan pipeline processing custom (Asynq + ffmpeg) sebagai bagian wajib alur upload. Ditandai sebagai amandemen resmi terhadap spesifikasi awal proyek, mengikat seluruh dokumen fase berikutnya. ADR-003 (sqlc) dan ADR-006 (Redis Streams) dikonfirmasi final tanpa perubahan. |
