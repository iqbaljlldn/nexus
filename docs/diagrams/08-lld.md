# Low Level Design (LLD)
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 4 — Low Level Design
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `01-engineering-playbook.md`, `02-vision-document.md`, `03-adr.md` (v1.1.0), `06-srs.md` (v1.1.0), `07-hld.md` (v1.0.0)
**Klasifikasi:** Internal — Source of Truth

---

## 0. Posisi Dokumen Ini

LLD menerjemahkan domain model (HLD) menjadi **kontrak kode konkret**: interface Go, struct, dan algoritma kunci yang menyelesaikan seluruh technical debt yang tercatat di dokumen sebelumnya (dynamic query filter sqlc, broadcast WebSocket multi-instance, resolusi permission, dsb.). Database Design (Phase 5) dan API Specification (Phase 6) akan merujuk langsung ke kontrak yang didefinisikan di sini.

---

## 1. Struktur Interface (Port) per Domain

Mengikuti Clean Architecture (Playbook §7.1): `domain` mendefinisikan interface (port), `infrastructure` mengimplementasikan (adapter). Berikut representasi untuk domain kunci (pola yang sama berlaku identik untuk domain lain yang tidak dijabarkan detail di sini — Workspace, Role, Attachment, dsb.).

### 1.1 Message Domain

```go
// internal/message/domain/message.go
package domain

type Message struct {
    ID            uuid.UUID
    ChannelID     uuid.UUID
    AuthorID      uuid.UUID
    ReplyToID     *uuid.UUID
    ThreadID      *uuid.UUID
    Content       string
    Mentions      []uuid.UUID
    Version       int32 // Optimistic Locking
    EditedAt      *time.Time
    DeletedAt     *time.Time
    CreatedAt     time.Time
}

// Repository port — diimplementasikan oleh sqlc-generated adapter
type MessageRepository interface {
    Create(ctx context.Context, msg *Message) error
    FindByID(ctx context.Context, id uuid.UUID) (*Message, error)
    // ListByChannel: cursor-based, lihat §2.2 untuk detail algoritma
    ListByChannel(ctx context.Context, channelID uuid.UUID, cursor Cursor, limit int) ([]*Message, Cursor, error)
    // UpdateWithVersion: mengembalikan ErrOptimisticLockConflict bila version tidak cocok
    UpdateWithVersion(ctx context.Context, msg *Message, expectedVersion int32) error
    SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type MessageService interface {
    Send(ctx context.Context, cmd SendMessageCommand) (*Message, error)
    Edit(ctx context.Context, cmd EditMessageCommand) (*Message, error)
    Delete(ctx context.Context, messageID, actorID uuid.UUID) error
}
```

### 1.2 Channel Domain (termasuk DM)

```go
// internal/channel/domain/channel.go
package domain

type ChannelType string

const (
    ChannelTypeText         ChannelType = "text"
    ChannelTypeVoice        ChannelType = "voice"
    ChannelTypeVideo        ChannelType = "video"
    ChannelTypeForum        ChannelType = "forum"
    ChannelTypeAnnouncement ChannelType = "announcement"
    ChannelTypeDM           ChannelType = "dm"
)

type Channel struct {
    ID          uuid.UUID
    WorkspaceID *uuid.UUID // NULL untuk tipe dm (FR-DM-01)
    CategoryID  *uuid.UUID
    Type        ChannelType
    Name        string
}

// ChannelAuthorizationService menggabungkan dua strategi otorisasi
// berbeda tergantung apakah channel bertipe workspace-scoped atau dm.
type ChannelAuthorizationService interface {
    CanRead(ctx context.Context, userID, channelID uuid.UUID) (bool, error)
    CanWrite(ctx context.Context, userID, channelID uuid.UUID) (bool, error)
}
```

Implementasi `ChannelAuthorizationService` (lihat §2.1 untuk algoritma lengkap resolusi permission workspace, dan §2.5 untuk logic DM):

```go
func (s *channelAuthzService) CanWrite(ctx context.Context, userID, channelID uuid.UUID) (bool, error) {
    ch, err := s.channelRepo.FindByID(ctx, channelID)
    if err != nil {
        return false, fmt.Errorf("find channel: %w", err)
    }
    if ch.Type == domain.ChannelTypeDM {
        return s.dmAuthz.CanWrite(ctx, userID, channelID) // §2.5
    }
    return s.permissionResolver.Resolve(ctx, userID, *ch.WorkspaceID, channelID, PermissionSendMessage) // §2.1
}
```

### 1.3 Notification Domain

```go
// internal/notification/domain/notification.go
package domain

type NotificationEvent struct {
    RecipientID uuid.UUID
    Type        NotificationType // mention, reply, dm, invite
    Payload     json.RawMessage  // payload lengkap dari event asal — TIDAK query balik ke domain lain
    ChannelID   uuid.UUID
    WorkspaceID *uuid.UUID
}

type NotificationDispatcher interface {
    Dispatch(ctx context.Context, event NotificationEvent) error
}
```

---

## 2. Algoritma Kunci

### 2.1 Permission Resolution Algorithm

Menyelesaikan FR-WS-04 dan FR-WS-07 (urutan resolusi: Channel-specific member override → Channel-specific role override → Role default berdasar `position` → `@everyone`).

```go
func (r *permissionResolver) Resolve(ctx context.Context, userID, workspaceID, channelID uuid.UUID, required PermissionFlag) (bool, error) {
    // 1. Channel-specific MEMBER override (prioritas tertinggi)
    if override, ok, err := r.channelOverrideRepo.FindMemberOverride(ctx, channelID, userID); err != nil {
        return false, err
    } else if ok {
        if override.Deny.Has(required) {
            return false, nil // deny eksplisit selalu menang di level ini
        }
        if override.Allow.Has(required) {
            return true, nil
        }
        // Tidak diatur di level ini → lanjut ke level berikutnya
    }

    // 2. Channel-specific ROLE override, diiterasi dari role dengan `position` tertinggi
    roles, err := r.roleRepo.FindMemberRolesSortedByPosition(ctx, workspaceID, userID) // DESC by position
    if err != nil {
        return false, err
    }
    for _, role := range roles {
        if override, ok, err := r.channelOverrideRepo.FindRoleOverride(ctx, channelID, role.ID); err != nil {
            return false, err
        } else if ok {
            if override.Deny.Has(required) {
                return false, nil
            }
            if override.Allow.Has(required) {
                return true, nil
            }
        }
    }

    // 3. Role default permission (bitmask), role tertinggi menang
    for _, role := range roles {
        if role.PermissionBitmask.Has(required) {
            return true, nil
        }
    }

    // 4. @everyone fallback
    everyone, err := r.roleRepo.FindEveryoneRole(ctx, workspaceID)
    if err != nil {
        return false, err
    }
    return everyone.PermissionBitmask.Has(required), nil
}
```

**Kompleksitas**: O(jumlah role yang dimiliki member), tipikal sangat kecil (< 10 role per member) — tidak menjadi bottleneck meski member per workspace besar (100.000), karena resolusi selalu di-scope ke satu member yang sedang request, bukan iterasi seluruh member.

**Caching**: Hasil resolusi permission per `(userID, channelID)` di-cache di Redis dengan TTL pendek (60 detik) dan invalidasi eksplisit saat role/override berubah (event `role.RolePermissionChanged` men-trigger cache invalidation) — detail cache invalidation dibahas di §2.6.

### 2.2 Cursor-Based Pagination

Menyelesaikan FR-MSG-10 dan §17.2 Engineering Playbook.

```go
type Cursor struct {
    LastID        uuid.UUID
    LastCreatedAt time.Time
}

func EncodeCursor(c Cursor) string {
    raw, _ := json.Marshal(c)
    return base64.URLEncoding.EncodeToString(raw)
}

func DecodeCursor(s string) (Cursor, error) {
    raw, err := base64.URLEncoding.DecodeString(s)
    if err != nil {
        return Cursor{}, fmt.Errorf("decode cursor: %w", err)
    }
    var c Cursor
    if err := json.Unmarshal(raw, &c); err != nil {
        return Cursor{}, fmt.Errorf("unmarshal cursor: %w", err)
    }
    return c, nil
}
```

Query SQL (sqlc) yang mendasari — memakai **keyset pagination** dua kolom (`created_at`, `id`) untuk menangani kasus banyak pesan dengan `created_at` identik:

```sql
-- name: ListMessagesByChannel :many
SELECT * FROM messages
WHERE channel_id = $1
  AND deleted_at IS NULL
  AND (created_at, id) < (sqlc.arg(cursor_created_at), sqlc.arg(cursor_id))
ORDER BY created_at DESC, id DESC
LIMIT $2;
```

**Rationale**: dua kolom `(created_at, id)` sebagai composite cursor mencegah pesan terlewat/terduplikasi saat banyak pesan memiliki timestamp yang sama secara statistik (mis. import massal atau burst traffic tinggi).

### 2.2b Cursor Pagination dengan Dynamic Sort (Menyelesaikan Kombinasi Sort + Search)

Pola §2.2 mengasumsikan urutan tetap `(created_at, id) DESC`. Untuk endpoint yang butuh **sort dinamis** (mis. daftar channel diurutkan nama A-Z, atau hasil search diurutkan relevansi), pola itu tidak cukup — kolom yang dibandingkan di keyset comparison ikut berubah tergantung sort mode, dan `ORDER BY` tidak bisa di-parameterize lewat bind variable (mencoba melakukannya lewat string concatenation adalah **celah SQL injection**, dilarang RULES.md §2).

**Solusi: whitelist sort mode, satu query eksplisit per mode** — bukan satu query dengan `ORDER BY` dinamis.

```go
type Cursor struct {
    SortMode  string          `json:"sort_mode"`  // "newest" | "name_asc" | "relevance"
    SortValue json.RawMessage `json:"sort_value"` // nilai kolom urut: timestamp, string, atau rank score
    LastID    uuid.UUID       `json:"last_id"`    // tiebreaker, selalu ada di composite key manapun
}
```

Setiap sort mode dipetakan ke query sqlc terpisah, dengan `id` sebagai tiebreaker kedua:

```sql
-- name: ListChannelsByNewest :many
SELECT * FROM channels
WHERE workspace_id = $1
  AND (created_at, id) < (sqlc.arg(cursor_created_at), sqlc.arg(cursor_id))
ORDER BY created_at DESC, id DESC
LIMIT $2;

-- name: ListChannelsByNameAsc :many
SELECT * FROM channels
WHERE workspace_id = $1
  AND (name, id) > (sqlc.arg(cursor_name), sqlc.arg(cursor_id))
ORDER BY name ASC, id ASC
LIMIT $2;

-- name: SearchMessagesByRelevance :many
SELECT *, ts_rank(search_vector, query) AS rank FROM messages, plainto_tsquery('simple', sqlc.arg(q)) query
WHERE channel_id = ANY(sqlc.arg(channel_ids)::uuid[])
  AND search_vector @@ query
  AND (ts_rank(search_vector, query), id) < (sqlc.arg(cursor_rank), sqlc.arg(cursor_id))
ORDER BY rank DESC, id DESC
LIMIT $1;
```

Service layer memvalidasi `sort_mode` terhadap whitelist di awal (`ErrInvalidSortMode` bila di luar daftar yang didukung endpoint tersebut), lalu me-route ke query yang sesuai:

```go
func (s *ChannelService) List(ctx context.Context, workspaceID uuid.UUID, sortMode string, cursor *Cursor, limit int) ([]*Channel, *Cursor, error) {
    switch sortMode {
    case "newest":
        return s.repo.ListByNewest(ctx, workspaceID, cursor, limit)
    case "name_asc":
        return s.repo.ListByNameAsc(ctx, workspaceID, cursor, limit)
    default:
        return nil, nil, ErrInvalidSortMode
    }
}
```

**Kombinasi dengan filter dinamis (§2.3)**: keduanya independen dan tetap dipakai bersamaan — filter opsional (author, date range, dst.) tetap lewat pola `(param IS NULL OR condition)` di `WHERE`, sort tetap lewat query terpisah per mode. Bila kombinasi filter × sort mode mulai terasa meledak jumlah query-nya, itu sinyal untuk membatasi sort mode yang benar-benar dibutuhkan produk (dalam praktik biasanya 2-3 mode masuk akal per endpoint — "terbaru", "nama", "relevansi" — bukan alasan memaksakan `ORDER BY` dinamis yang membuka celah keamanan).

**Kesalahan yang dihindari secara sengaja**: menerima nama kolom sort langsung dari query parameter client (`?sort=name`) lalu memakainya untuk membangun `ORDER BY name` via string formatting — client bisa mengirim `?sort=name; DROP TABLE users;--` atau memaksa sort ke kolom yang tidak diindex (DoS lewat full table sort). Whitelist di service layer menutup kedua risiko sekaligus.

### 2.3 Dynamic Query Filter Strategy untuk sqlc (Menyelesaikan Debt ADR-003)

Search dengan banyak kombinasi filter opsional (mis. filter by author, by date range, by has-attachment) sulit diekspresikan sebagai satu query sqlc statis. Strategi yang dipakai: **`sqlc.narg()` (nullable argument) dengan kondisi `OR` terkontrol**, bukan query builder dinamis penuh:

```sql
-- name: SearchMessages :many
SELECT * FROM messages
WHERE channel_id = ANY(sqlc.arg(channel_ids)::uuid[])
  AND deleted_at IS NULL
  AND (sqlc.narg('author_id')::uuid IS NULL OR author_id = sqlc.narg('author_id'))
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR created_at <= sqlc.narg('to_date'))
  AND (sqlc.narg('has_attachment')::boolean IS NULL OR
       (sqlc.narg('has_attachment') = true AND EXISTS (SELECT 1 FROM attachments a WHERE a.message_id = messages.id))
      )
  AND (sqlc.narg('query_text')::text IS NULL OR search_vector @@ plainto_tsquery('simple', sqlc.narg('query_text')))
ORDER BY created_at DESC
LIMIT $1;
```

**Rationale**: pola `(param IS NULL OR condition)` memungkinkan satu query statis menangani kombinasi filter opsional tanpa dynamic string building di aplikasi (yang berisiko SQL injection bila tidak hati-hati dan sulit di-maintain). PostgreSQL query planner cukup baik mengoptimalkan pola ini terutama dengan partial index yang sesuai (dibahas di Database Design Phase 5). **Batas pendekatan ini**: untuk kombinasi filter yang sangat banyak (> 8-10 parameter opsional), query mulai sulit dibaca — pada titik itu, pertimbangkan memecah menjadi beberapa query bertahap (progressive filtering) di service layer, bukan satu query raksasa.

### 2.4 Outbox Relay Worker

Menyelesaikan Milestone 12 (Learning Roadmap) secara konkret.

```go
func (w *OutboxRelayWorker) Run(ctx context.Context) error {
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return ctx.Err() // graceful shutdown dihormati
        case <-ticker.C:
            if err := w.relayBatch(ctx); err != nil {
                w.logger.Error("outbox relay batch failed", zap.Error(err))
                // Tidak return — worker tetap berjalan, error di-log untuk observability,
                // batch berikutnya akan retry baris yang sama (published_at masih NULL)
            }
        }
    }
}

func (w *OutboxRelayWorker) relayBatch(ctx context.Context) error {
    events, err := w.outboxRepo.FetchUnpublished(ctx, 100) // batch size 100
    if err != nil {
        return fmt.Errorf("fetch unpublished: %w", err)
    }
    for _, ev := range events {
        streamKey := fmt.Sprintf("stream:%s:events", ev.AggregateType)
        if err := w.streamPublisher.Publish(ctx, streamKey, ev); err != nil {
            w.logger.Error("publish failed, will retry next batch", zap.String("event_id", ev.ID.String()), zap.Error(err))
            continue // baris ini tetap published_at = NULL, akan dicoba lagi batch berikutnya
        }
        if err := w.outboxRepo.MarkPublished(ctx, ev.ID); err != nil {
            w.logger.Error("mark published failed — risiko duplikasi at-least-once", zap.Error(err))
            // Diterima: at-least-once delivery. Consumer WAJIB idempotent (lihat §2.7).
        }
    }
    return nil
}
```

**Catatan penting**: Jeda antara `Publish` berhasil dan `MarkPublished` gagal adalah celah *at-least-once* yang disengaja diterima (bukan *exactly-once*, yang jauh lebih kompleks dan mahal untuk dijamin) — konsumen **wajib** idempotent (§2.7), bukan relay yang harus sempurna.

### 2.5 DM Authorization & Uniqueness (Menyelesaikan FR-DM-02, FR-DM-04)

```go
func (s *dmAuthzService) CanWrite(ctx context.Context, userID, channelID uuid.UUID) (bool, error) {
    isMember, err := s.channelMemberRepo.IsMember(ctx, channelID, userID)
    if err != nil || !isMember {
        return false, err
    }
    participants, err := s.channelMemberRepo.ListParticipants(ctx, channelID)
    if err != nil {
        return false, err
    }
    for _, p := range participants {
        if p.UserID == userID {
            continue
        }
        blocked, err := s.blockRepo.IsBlocked(ctx, p.UserID, userID) // p memblokir userID?
        if err != nil {
            return false, err
        }
        if blocked {
            return false, nil // FR-DM-04: user yang diblokir tidak dapat mengirim
        }
    }
    return true, nil
}

// Uniqueness 1-on-1 DM (FR-DM-02): partisipan di-sort deterministik sebelum dicari/dibuat
func BuildDMChannelKey(userIDs []uuid.UUID) string {
    sorted := make([]uuid.UUID, len(userIDs))
    copy(sorted, userIDs)
    slices.SortFunc(sorted, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
    var sb strings.Builder
    for _, id := range sorted {
        sb.WriteString(id.String())
    }
    return sb.String() // di-hash (SHA-256) dan disimpan sebagai unique constraint kolom `participant_key`
}
```

**Rationale**: constraint unik pada `participant_key` (hash dari partisipan yang di-sort) di level database mencegah race condition dua request bersamaan membuat 2 channel DM untuk pasangan user yang sama (dijamin database, bukan hanya application-level check yang rentan TOCTOU/race).

### 2.6 Cache Invalidation untuk Permission Resolver

```go
// Dipanggil oleh consumer event role.RolePermissionChanged / role.RoleAssignedToMember
func (s *permissionCacheInvalidator) OnRoleChanged(ctx context.Context, workspaceID uuid.UUID, affectedUserIDs []uuid.UUID) error {
    keys := make([]string, 0, len(affectedUserIDs))
    for _, uid := range affectedUserIDs {
        keys = append(keys, fmt.Sprintf("perm:%s:%s:*", workspaceID, uid))
    }
    return s.redisClient.DeleteByPattern(ctx, keys) // SCAN + DEL, bukan KEYS (menghindari blocking Redis)
}
```

**Kesalahan yang dihindari secara sengaja**: memakai perintah Redis `KEYS` untuk pencarian pattern (blocking, O(n) terhadap seluruh keyspace) — dipakai `SCAN` cursor-based sebagai gantinya, konsisten dengan prinsip *Production First Mindset*.

### 2.7 Idempotent Event Consumer

```go
func (c *notificationConsumer) HandleMessageCreated(ctx context.Context, event StreamEvent) error {
    alreadyProcessed, err := c.processedEventRepo.Exists(ctx, event.ID)
    if err != nil {
        return fmt.Errorf("check processed: %w", err)
    }
    if alreadyProcessed {
        return c.streamClient.Ack(ctx, event.StreamKey, event.ID) // ack saja, tidak proses ulang
    }

    if err := c.dispatchNotification(ctx, event); err != nil {
        return fmt.Errorf("dispatch: %w", err) // TIDAK di-ack, akan di-retry consumer group
    }

    if err := c.processedEventRepo.MarkProcessed(ctx, event.ID); err != nil {
        return fmt.Errorf("mark processed: %w", err)
    }
    return c.streamClient.Ack(ctx, event.StreamKey, event.ID)
}
```

### 2.8 Rate Limiting — Sliding Window via Redis (Lua Script Atomik)

Menyelesaikan §3.5 SRS dengan detail implementasi konkret.

```lua
-- rate_limit.lua — dieksekusi atomik via EVAL
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local count = redis.call('ZCARD', key)
if count >= limit then
    return 0
end
redis.call('ZADD', key, now, now .. '-' .. math.random())
redis.call('EXPIRE', key, window)
return 1
```

**Rationale**: sliding window log via Redis Sorted Set dijalankan sebagai satu Lua script (atomik, mencegah race condition antara `ZCARD` check dan `ZADD` increment yang bisa terjadi bila dipisah menjadi 2 command Redis terpisah dari aplikasi Go).

### 2.9 WebSocket Connection Manager (Single-Instance, Phase A/B)

```go
type ConnectionRegistry struct {
    mu          sync.RWMutex
    byChannel   map[uuid.UUID]map[*Connection]struct{}
}

func (r *ConnectionRegistry) Broadcast(channelID uuid.UUID, msg []byte) {
    r.mu.RLock()
    conns := r.byChannel[channelID]
    snapshot := make([]*Connection, 0, len(conns))
    for c := range conns {
        snapshot = append(snapshot, c)
    }
    r.mu.RUnlock() // lock dilepas sebelum I/O agar tidak memblokir registrasi/unregistrasi koneksi lain

    for _, c := range snapshot {
        select {
        case c.sendCh <- msg: // non-blocking send ke channel buffered per-koneksi
        default:
            c.logger.Warn("send buffer full, dropping slow consumer connection")
            c.Close() // slow consumer protection — mencegah satu koneksi lambat menghambat broadcast ke semua
        }
    }
}
```

**Kesalahan yang dihindari secara sengaja**: menulis langsung ke `net.Conn`/WebSocket dari goroutine broadcast — ini akan menyebabkan race condition bila goroutine reader/writer koneksi yang sama juga menulis bersamaan. Setiap koneksi punya **satu** goroutine writer yang membaca dari `sendCh`, broadcast hanya mengirim ke channel tersebut (single-writer principle, §10.1 Playbook).

**Evolusi Multi-Instance (Phase C/D)**: `ConnectionRegistry` per-instance tetap ada (untuk koneksi lokal), namun broadcast lintas instance ditambahkan lewat subscriber Redis Streams `stream:<channel_id>:realtime` — setiap instance API subscribe ke stream yang relevan dengan koneksi aktif di instance tersebut, dan me-relay pesan ke `ConnectionRegistry` lokalnya. Detail penuh dan trade-off vs sticky-session Traefik dibahas sebagai keputusan wajib di awal Milestone 12/13 (dicatat sebagai risiko terbuka di HLD §Risiko).

### 2.10 Worker Pool untuk Asynq (Media Processing)

```go
func NewMediaWorkerServer(cfg WorkerConfig) *asynq.Server {
    return asynq.NewServer(
        asynq.RedisClientOpt{Addr: cfg.RedisAddr},
        asynq.Config{
            Concurrency: cfg.MaxConcurrentJobs, // dibatasi eksplisit — bukan unlimited goroutine
            Queues: map[string]int{
                "critical": 6, // thumbnail generation (cepat, prioritas tinggi)
                "default":  3, // metadata extraction
                "low":      1, // transcoding berat (video besar)
            },
            RetryDelayFunc: asynq.RetryDelayFunc(func(n int, err error, task *asynq.Task) time.Duration {
                return time.Duration(n*n) * time.Second // exponential backoff sederhana
            }),
        },
    )
}
```

**Rationale prioritas queue**: thumbnail generation (`critical`) diprioritaskan karena user menunggu preview gambar segera; transcoding video besar (`low`) tidak seharusnya memblokir resource worker dari tugas yang lebih kecil/cepat — mencegah head-of-line blocking.

---

## 3. Graceful Shutdown — Implementasi Konkret

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
    defer stop()

    srv := &http.Server{Addr: cfg.Addr, Handler: router}
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("server error", zap.Error(err))
        }
    }()

    <-ctx.Done() // menunggu SIGTERM/SIGINT
    logger.Info("shutdown signal received, draining connections")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if err := srv.Shutdown(shutdownCtx); err != nil {
        logger.Error("forced shutdown", zap.Error(err))
    }
    wsHub.CloseAllGracefully(shutdownCtx) // kirim close frame ke seluruh koneksi WS aktif
    dbPool.Close()
    redisClient.Close()
    logger.Info("shutdown complete")
}
```

---

## Ringkasan Keputusan

1. Seluruh technical debt yang tercatat di ADR (dynamic query filter sqlc) dan SRS (broadcast multi-instance) memiliki **solusi konkret** di dokumen ini — bukan lagi debt terbuka untuk aspek desain, hanya tersisa keputusan konfigurasi/tuning di fase implementasi.
2. Resolusi permission memakai algoritma 4-tingkat eksplisit dengan caching Redis + invalidation berbasis event, bukan query berulang di setiap request.
3. Outbox Relay menerapkan **at-least-once delivery** secara sadar — konsumen wajib idempotent, bukan mengandalkan relay yang sempurna.
4. WebSocket Connection Manager menerapkan single-writer-per-connection dan slow-consumer protection sebagai pola wajib.

## Trade-off yang Diterima

- Cache permission resolver menambah kompleksitas invalidation (perlu event listener khusus) demi menghindari query resolusi berulang di setiap request — dianggap sepadan mengingat frekuensi permission check jauh lebih tinggi dari frekuensi perubahan role.
- Dynamic query filter dengan pola `(param IS NULL OR condition)` kurang optimal dibanding query builder murni untuk kombinasi filter sangat kompleks — diterima sebagai kompromi konsisten dengan keputusan sqlc di ADR-003.

## Risiko Arsitektur

- Slow consumer protection pada WebSocket (§2.9) menyebabkan koneksi di-drop paksa saat buffer penuh — perlu dipantau di Milestone 11 apakah threshold buffer sudah tepat (terlalu kecil = drop tidak perlu, terlalu besar = memory bloat saat banyak slow consumer).
- Broadcast multi-instance (evolusi Phase C/D) masih berupa desain prinsip, belum detail penuh — wajib dituntaskan sebelum Milestone 12/13 dieksekusi.

## Technical Debt yang Sengaja Diterima

- Parameter konkret (ukuran buffer `sendCh`, batch size Outbox Relay, jumlah worker concurrency Asynq) ditulis sebagai nilai awal yang wajar namun **akan dituning berdasarkan hasil benchmark nyata** di Milestone 11, bukan dianggap final di sini.

## Hal yang Perlu Dikonfirmasi Sebelum Lanjut ke Fase Berikutnya

1. Apakah pendekatan **cache permission dengan TTL 60 detik + event-based invalidation** dapat diterima, atau Anda lebih memilih tanpa cache (query langsung tiap request, lebih sederhana namun berpotensi lebih lambat)?
2. Apakah strategi **at-least-once delivery** (bukan exactly-once) untuk Outbox Relay sudah sesuai ekspektasi Anda tentang tingkat garansi yang wajar dipelajari?
3. Lanjut ke **Phase 5 — Database Design**?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen pertama Phase 4, menyelesaikan seluruh technical debt desain dari ADR/SRS/HLD dengan interface dan algoritma konkret |
