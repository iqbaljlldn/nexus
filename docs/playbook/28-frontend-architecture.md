# Frontend Architecture Design
## Project: Nexus — Discord-like Realtime Platform

**Dokumen:** Phase 3b/4b — Frontend Architecture (High + Low Level Design)
**Versi:** 1.0.0
**Status:** Draft — Menunggu Persetujuan
**Referensi Wajib:** `01-engineering-playbook.md` (§10.3), `07-hld.md`, `08-lld.md`, `09-database-design.md`, `10-api-specification.md`, `11-security-design.md`
**Klasifikasi:** Internal — Source of Truth

---

## 0. Posisi Dokumen Ini & Latar Belakang

Dokumen ini menutup celah yang ditemukan setelah Sprint 13: seluruh dokumen Phase 0-11 sejauh ini nyaris 100% backend/infrastruktur, padahal spesifikasi awal proyek eksplisit menuntut stack frontend lengkap (Nuxt 4, Vue 3, TypeScript, Pinia, VueUse, TanStack Query, TailwindCSS, Floating UI, Vue Virtual Scroller, PWA) dan **"UI harus semirip mungkin dengan Discord"**.

Dokumen ini setara bobot dengan HLD (§3) + LLD (§4) backend — mencakup arsitektur komponen, strategi state management, routing, pola integrasi realtime, dan algoritma/pola kunci di level implementasi. Setelah dokumen ini disepakati, setiap Sprint Task Checklist (1-13, retroaktif) akan diamandemen dengan Epic Frontend paralel.

---

## 1. Prinsip Arsitektur Frontend

Mengikuti filosofi yang sama dengan backend (Playbook intro): *Simplicity over Cleverness, Explicit over Implicit, Composition over Inheritance, YAGNI*. Diterjemahkan ke konteks frontend:

- **Server state vs Client state dipisah tegas** (sudah diputuskan di Playbook §10.3): TanStack Query untuk **apapun yang berasal dari REST API** (cache, invalidation, retry, background refetch); Pinia **hanya** untuk state yang benar-benar global dan bukan hasil fetch server (sesi auth aktif, workspace yang sedang dibuka, peta presence realtime). Mencampur keduanya adalah anti-pattern yang harus dihindari sejak awal.
- **Composition API murni** (`<script setup lang="ts">`) — tidak ada Options API di manapun.
- **Komponen "dumb" sebisa mungkin** — logika bisnis (kapan menampilkan X, bagaimana memproses Y) tinggal di composable, komponen `.vue` fokus pada presentasi dan event binding.

---

## 2. Struktur Direktori (Perluasan Playbook §19.2)

```
apps/web/
├── app/
│   ├── components/
│   │   ├── ui/                    # Primitive re-usable (Button, Input, Modal, Avatar, Tooltip)
│   │   ├── workspace/              # ServerSidebar, ServerIcon, CategoryList
│   │   ├── channel/                 # ChannelList, ChannelHeader, ChannelSettingsModal
│   │   ├── message/                 # MessageList, MessageItem, MessageComposer, ReactionPicker
│   │   ├── voice/                   # VoiceParticipantGrid, VoiceControls
│   │   ├── presence/                # PresenceIndicator, TypingIndicator
│   │   └── admin/                   # AdminDashboard, AuditLogTable
│   ├── composables/
│   │   ├── useAuth.ts               # Session state, login/logout actions
│   │   ├── useWebSocket.ts          # Koneksi WS tunggal, reconnect logic (§5)
│   │   ├── usePermission.ts         # Cek permission client-side (UI hint only — §7 catatan keras)
│   │   ├── useMessages.ts           # TanStack Query wrapper untuk message list + mutation
│   │   ├── useVoiceRoom.ts          # LiveKit Client SDK wrapper
│   │   └── useInfiniteScroll.ts     # Wrapper Vue Virtual Scroller + cursor pagination (§6)
│   ├── pages/
│   │   ├── login.vue, register.vue
│   │   ├── workspaces/[id]/
│   │   │   ├── channels/[channelId].vue
│   │   │   └── settings/
│   │   └── dm/[channelId].vue
│   ├── layouts/
│   │   ├── default.vue              # Sidebar server + channel list + main content
│   │   └── auth.vue                 # Layout kosong untuk login/register
│   ├── stores/                      # Pinia — HANYA client state (§3)
│   │   ├── session.ts               # User yang login, access token in-memory
│   │   ├── activeWorkspace.ts       # Workspace/channel yang sedang dibuka
│   │   └── presence.ts              # Peta presence realtime (didorong dari useWebSocket)
│   └── plugins/
│       ├── api-client.ts            # Axios/ofetch instance dengan interceptor (§4)
│       └── websocket.client.ts      # Inisialisasi koneksi WS saat app mount
├── public/
├── nuxt.config.ts
└── package.json
```

---

## 3. Strategi State Management (Detail)

### 3.1 Pembagian Tegas

| State | Dikelola Oleh | Contoh |
|---|---|---|
| Data dari REST API (list, detail) | **TanStack Query** | Daftar channel, riwayat pesan, daftar member |
| Mutation ke REST API | **TanStack Query** (`useMutation`) | Kirim pesan, edit workspace, redeem invite |
| Sesi auth aktif (access token, user profile) | **Pinia** (`stores/session.ts`) | `session.user`, `session.isAuthenticated` |
| Workspace/channel yang sedang aktif di UI | **Pinia** (`stores/activeWorkspace.ts`) | `activeWorkspace.currentChannelId` |
| Presence realtime (didorong WS, bukan REST) | **Pinia** (`stores/presence.ts`) | `presence.statusByUserId` |
| State lokal komponen (form input, modal open/close) | **`ref`/`reactive` biasa** | Tidak pernah masuk Pinia |

### 3.2 Rationale Pemisahan

TanStack Query menangani **cache invalidation otomatis** berbasis query key — begitu mutation `sendMessage` sukses, query key `['messages', channelId]` di-invalidate, seluruh komponen yang subscribe otomatis refetch/update. Menaruh data ini di Pinia berarti membangun ulang mekanisme cache/invalidation secara manual — duplikasi kerja yang sudah disediakan library, melanggar DRY sekaligus menambah bug surface.

### 3.3 Contoh Pola — `useMessages` Composable

```typescript
// composables/useMessages.ts
export function useMessages(channelId: Ref<string>) {
  const query = useInfiniteQuery({
    queryKey: ['messages', channelId],
    queryFn: ({ pageParam }) => apiClient.get(`/channels/${channelId.value}/messages`, {
      params: { cursor: pageParam }
    }),
    getNextPageParam: (lastPage) => lastPage.meta.next_cursor,
  })

  const sendMutation = useMutation({
    mutationFn: (payload: SendMessagePayload) =>
      apiClient.post(`/channels/${channelId.value}/messages`, payload),
    // TIDAK perlu manual invalidate di sini — pesan baru datang lewat WS broadcast (§5),
    // bukan menunggu refetch REST. Mutation ini hanya untuk optimistic UI + error handling.
    onError: (err, variables, context) => rollbackOptimisticMessage(context),
    onMutate: async (payload) => addOptimisticMessage(payload), // optimistic update
  })

  return { messages: query, sendMessage: sendMutation }
}
```

**Poin penting**: pesan baru yang datang dari user lain **tidak** melalui TanStack Query refetch — itu datang lewat WebSocket broadcast (backend sudah didesain synchronous in-process, HLD §3) dan langsung disuntikkan ke query cache TanStack Query secara manual (`queryClient.setQueryData`), bukan trigger refetch REST. Ini penting untuk performa — refetch REST setiap ada pesan baru di channel ramai akan membebani backend tanpa perlu.

---

## 4. API Client & Auth Integration

### 4.1 HTTP Client dengan Interceptor

```typescript
// plugins/api-client.ts
const apiClient = ofetch.create({
  baseURL: '/api/v1',
  credentials: 'include', // WAJIB — refresh token via HttpOnly Cookie (Security Design §2)
  onRequest({ options }) {
    const session = useSessionStore()
    if (session.accessToken) {
      options.headers.set('Authorization', `Bearer ${session.accessToken}`)
    }
  },
  async onResponseError({ response, request, options }) {
    if (response.status === 401) {
      // Access token expired — coba refresh SEKALI, retry request asli
      const refreshed = await useAuthRefresh()
      if (refreshed) return apiClient(request, options)
      // Refresh gagal → redirect login
      navigateTo('/login')
    }
  }
})
```

**Poin kritikal (RULES.md-consistent)**: access token disimpan **di memori** (Pinia store, bukan localStorage) — konsisten dengan keputusan Security Design §2 (mitigasi XSS). Refresh token murni di cookie HttpOnly, tidak pernah disentuh JavaScript sama sekali (browser mengirimnya otomatis via `credentials: 'include'`).

### 4.2 CSRF Token untuk Endpoint Refresh

```typescript
async function useAuthRefresh(): Promise<boolean> {
  const csrfToken = useCookie('csrf_token').value // cookie non-HttpOnly, readable
  try {
    const { data } = await apiClient('/auth/refresh', {
      method: 'POST',
      headers: { 'X-CSRF-Token': csrfToken },
    })
    useSessionStore().setAccessToken(data.access_token)
    return true
  } catch {
    return false
  }
}
```

Sesuai Security Design §6 (double-submit cookie) — frontend membaca cookie `csrf_token` (bukan HttpOnly, sengaja readable JS) dan mengirimkannya sebagai header.

---

## 5. Integrasi WebSocket (Client-Side)

### 5.1 Satu Koneksi Global, Bukan per Komponen

```typescript
// composables/useWebSocket.ts — singleton, diinisialisasi sekali di plugin
const ws = ref<WebSocket | null>(null)
const reconnectAttempts = ref(0)

function connect() {
  const session = useSessionStore()
  ws.value = new WebSocket(`${WS_URL}?token=${session.accessToken}`)

  ws.value.onmessage = (event) => {
    const envelope = JSON.parse(event.data)
    routeIncomingEvent(envelope) // §5.2
  }

  ws.value.onclose = (event) => {
    if (event.code === 4001) { // Auth gagal (Security Design §7) — jangan reconnect otomatis
      navigateTo('/login')
      return
    }
    scheduleReconnect() // exponential backoff
  }
}

function scheduleReconnect() {
  const delay = Math.min(1000 * 2 ** reconnectAttempts.value, 30000)
  setTimeout(() => { reconnectAttempts.value++; connect() }, delay)
}
```

**Rationale satu koneksi global**: mencegah N komponen masing-masing membuka koneksi WS sendiri (boros, dan melanggar batas 5 koneksi/user di Security Design §7 dari sisi lain — device yang sama membuka banyak tab wajar, tapi satu tab tidak boleh buka banyak koneksi).

### 5.2 Event Router — Menyuntik ke TanStack Query Cache & Pinia

```typescript
function routeIncomingEvent(envelope: WSEnvelope) {
  const queryClient = useQueryClient()
  switch (envelope.event) {
    case 'message.created':
      queryClient.setQueryData(['messages', envelope.payload.channel_id], (old) =>
        prependMessage(old, envelope.payload)) // manual cache update, BUKAN refetch (§3.3)
      break
    case 'presence.updated':
      usePresenceStore().setStatus(envelope.payload.user_id, envelope.payload.status)
      break
    case 'typing.updated':
      useTypingIndicator().show(envelope.payload.channel_id, envelope.payload.user_id) // auto-hide 5s, client-side timer (FR-PRES-03)
      break
    // ... event lain sesuai katalog API Spec §10
  }
}
```

**Konsistensi dengan backend**: daftar event yang di-route di sini **harus** persis sama dengan katalog Server→Client Events di `10-api-specification.md` §10 — setiap kali backend menambah event baru, frontend wajib menambah case yang sesuai (dicatat sebagai checklist wajib di setiap sprint yang menyentuh WebSocket, mulai amandemen retroaktif berikutnya).

---

## 6. Virtual Scrolling & Cursor Pagination (Frontend Side)

```typescript
// composables/useInfiniteScroll.ts — wrapper Vue Virtual Scroller + TanStack Query infinite
export function useChannelMessageList(channelId: Ref<string>) {
  const { messages, fetchNextPage, hasNextPage } = useMessages(channelId)

  // Vue Virtual Scroller butuh array flat, TanStack Query infinite mengembalikan array of pages
  const flatMessages = computed(() =>
    messages.value?.pages.flatMap(page => page.data) ?? [])

  function onScrollNearTop() {
    if (hasNextPage.value) fetchNextPage() // trigger load halaman berikutnya (cursor lama, §2.2 LLD)
  }

  return { flatMessages, onScrollNearTop }
}
```

**Rationale**: channel dengan riwayat sangat panjang (unlimited channel, SRS FR-CH-05) **wajib** virtual scrolling — merender seluruh DOM node untuk ribuan pesan akan membuat browser lag parah. Vue Virtual Scroller hanya me-render node yang terlihat di viewport.

---

## 7. Authorization di Frontend — Batasan Keras

**RULES.md §3 berlaku penuh di frontend**: `usePermission()` composable **hanya** untuk UI hint (menyembunyikan tombol yang user tidak punya akses, demi UX yang lebih bersih) — **bukan** mekanisme keamanan. Backend **selalu** melakukan pengecekan ulang (Security Design §3). Frontend tidak pernah boleh diberi tanggung jawab menegakkan otorisasi.

```typescript
// composables/usePermission.ts
export function useCanSendMessage(channelId: Ref<string>) {
  // Nilai ini di-fetch dari endpoint yang sama dengan channel detail (permission_bitmask
  // dikembalikan sebagai bagian response GET channel, backend yang menghitungnya via
  // Permission Resolver — LLD §2.1) — frontend TIDAK menghitung ulang logic resolusi permission.
  const { data: channel } = useChannelDetail(channelId)
  return computed(() => channel.value?.viewer_permissions?.can_send_message ?? false)
}
```

**Keputusan desain penting**: hasil resolusi permission (LLD §2.1, 4-tingkat) **dihitung sepenuhnya di backend**, dikirim sebagai field `viewer_permissions` di response API. Frontend **tidak pernah** mereplikasi algoritma resolusi permission — itu akan jadi duplikasi logic yang rawan drift (frontend dan backend bisa saja punya kesimpulan berbeda soal siapa boleh apa, sumber bug keamanan/UX yang membingungkan).

---

## 8. Voice/Video — LiveKit Client SDK Integration

```typescript
// composables/useVoiceRoom.ts
export function useVoiceRoom(channelId: Ref<string>) {
  const room = shallowRef<Room | null>(null) // LiveKit Room object — TIDAK reactive penuh (objek besar, mahal untuk Vue reactivity)

  async function join() {
    const { token, livekit_url } = await apiClient.post(`/channels/${channelId.value}/rtc/join`)
    room.value = new Room()
    await room.value.connect(livekit_url, token)
    room.value.on(RoomEvent.ParticipantConnected, syncParticipantList)
    room.value.on(RoomEvent.ParticipantDisconnected, syncParticipantList)
  }

  async function toggleScreenShare() {
    // Server sudah menolak token tanpa grant SHARE_SCREEN (Security Design + Sprint 10 Task 15.2.1) —
    // frontend hanya perlu MENCOBA, LiveKit SDK akan reject bila grant tidak ada
    await room.value?.localParticipant.setScreenShareEnabled(true)
  }

  return { room, join, toggleScreenShare }
}
```

**Catatan penting**: `shallowRef` (bukan `reactive`) dipakai untuk objek `Room` LiveKit — objek ini punya internal state kompleks (WebRTC connection, media tracks) yang **tidak boleh** dibuat reactive penuh oleh Vue (overhead proxy Vue pada objek sebesar ini signifikan, dan LiveKit SDK punya event system sendiri yang lebih tepat dipakai daripada Vue reactivity untuk update partisipan).

---

## 9. PWA Configuration

- `@vite-pwa/nuxt` sebagai module, service worker strategy: `NetworkFirst` untuk API call (data selalu diprioritaskan fresh), `CacheFirst` untuk asset statis (font, icon).
- Manifest: nama aplikasi, ikon, `display: standalone` (terasa seperti native app).
- **Push Notification** (Web Push API) **di luar scope** MVP — SRS FR-NOTIF hanya menyebut WS + email, bukan Web Push; dicatat sebagai extension point masa depan, bukan dibangun sekarang (YAGNI).

---

## 10. Testing Strategy Frontend (Ringkas, Detail di Sprint Checklist)

| Level | Tool | Cakupan |
|---|---|---|
| Unit (composable logic) | Vitest | `useMessages`, `usePermission`, cursor pagination logic |
| Component | Vitest + `@vue/test-utils` | Rendering kondisional (mis. `MessageItem` menampilkan tombol edit hanya untuk penulis) |
| E2E (alur kritikal) | Playwright | Login → kirim pesan → terima realtime (2 browser context simulasi 2 user) |

---

## Ringkasan Keputusan

1. Server state (TanStack Query) dan client state (Pinia) dipisah tegas — mencegah duplikasi mekanisme cache.
2. Satu koneksi WebSocket global per tab, event di-route langsung ke TanStack Query cache (manual injection) dan Pinia store — bukan trigger refetch REST.
3. Access token di memori (Pinia), refresh token murni HttpOnly Cookie — tidak pernah disentuh JavaScript, konsisten dengan Security Design.
4. Resolusi permission **sepenuhnya** di backend, dikirim sebagai field di response API — frontend tidak pernah mereplikasi logic otorisasi, hanya memakainya sebagai UI hint.
5. LiveKit `Room` object memakai `shallowRef`, bukan `reactive` — pertimbangan performa terhadap objek WebRTC kompleks.

## Trade-off yang Diterima

- Optimistic UI untuk kirim pesan (§3.3) menambah kompleksitas (rollback logic saat gagal) dibanding menunggu response server dulu — diterima demi UX responsif yang jadi ekspektasi standar chat app modern.
- Satu koneksi WS global berarti seluruh state realtime "hidup" di level plugin/composable singleton, bukan scoped ke komponen — sedikit mengurangi lokalitas kode, diterima demi efisiensi koneksi.

## Risiko Arsitektur

- Event router WebSocket (§5.2) rawan **drift** dari katalog backend bila tidak didisiplinkan — setiap sprint yang menambah event WS baru di backend wajib menambah case yang sesuai di frontend pada sprint yang sama, dicatat sebagai checklist wajib di amandemen retroaktif berikutnya.

## Technical Debt yang Sengaja Diterima

- Push Notification (Web Push API) tidak dibangun — hanya WS + email sesuai SRS. Dicatat sebagai extension point, bukan kelalaian.
- Component testing coverage belum ditargetkan angka spesifik — akan dikalibrasi bersamaan Milestone 11 (Optimization), sejalan dengan pendekatan backend.

## Hal yang Perlu Dikonfirmasi Sebelum Melanjutkan

1. Apakah pembagian state TanStack Query vs Pinia (§3.1) sudah sesuai model mental Anda?
2. Apakah keputusan "permission resolution sepenuhnya di backend, frontend cuma pakai hasilnya" (§7) dapat diterima — ini berarti response API channel/workspace perlu menyertakan field `viewer_permissions`, sebuah **amandemen kecil** ke `10-api-specification.md` yang perlu saya tambahkan?
3. Selanjutnya saya akan **retrofit Epic Frontend ke Sprint 1-13** secara bertahap (per beberapa sprint sekaligus agar tidak terlalu panjang per giliran). Mulai dari Sprint 1-3 dulu, atau Anda ingin urutan lain?

---

## Changelog

| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | Draft awal | Dokumen pertama — menutup celah cakupan frontend yang ditemukan pasca-Sprint 13, setara bobot HLD+LLD backend |
