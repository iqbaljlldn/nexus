<template>
  <aside class="w-60 bg-gray-900 flex flex-col shrink-0 border-r border-gray-800/40 select-none h-full">
    <!-- Workspace Header -->
    <div class="h-14 border-b border-gray-800/60 px-4 flex items-center justify-between shadow-xs font-semibold text-white">
      <span class="truncate text-base tracking-tight font-bold">
        {{ workspaceName }}
      </span>
      <div class="flex items-center gap-1">
        <NuxtLink
          v-if="workspaceId"
          :to="`/workspaces/${workspaceId}/settings`"
          class="p-1.5 rounded-lg text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
          title="Workspace Settings"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        </NuxtLink>
      </div>
    </div>

    <!-- Channels List -->
    <div class="flex-1 overflow-y-auto px-2 py-3 space-y-4">
      <div>
        <div class="flex items-center justify-between px-2 mb-1.5 text-xs font-semibold text-gray-400 uppercase tracking-wider">
          <span>Text Channels</span>
          <button
            v-if="workspaceId"
            @click="isModalOpen = true"
            class="text-gray-400 hover:text-white transition-colors p-0.5 rounded hover:bg-gray-800"
            title="Create Channel"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
          </button>
        </div>

        <div v-if="isLoading" class="space-y-1">
          <div v-for="i in 3" :key="i" class="h-8 bg-gray-800/50 rounded-md animate-pulse" />
        </div>

        <div v-else-if="channels.length === 0" class="px-2 py-2 text-xs text-gray-500 italic">
          No channels yet
        </div>

        <nav v-else class="space-y-0.5">
          <button
            v-for="ch in channels"
            :key="ch.id"
            @click="selectChannel(ch)"
            :class="[
              'w-full flex items-center gap-2 px-2.5 py-1.5 rounded-md text-sm font-medium transition-colors text-left',
              activeStore.currentChannelId === ch.id
                ? 'bg-gray-800 text-white font-semibold'
                : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
            ]"
          >
            <span class="text-gray-500 font-bold text-base">#</span>
            <span class="truncate">{{ ch.name || 'unnamed-channel' }}</span>
          </button>
        </nav>
      </div>
    </div>

    <!-- User Footer Strip -->
    <div class="h-14 border-t border-gray-800/60 px-3 bg-gray-950/60 flex items-center justify-between">
      <div class="flex items-center gap-2.5 overflow-hidden">
        <div class="w-8 h-8 rounded-full bg-indigo-600 flex items-center justify-center font-bold text-xs text-white shrink-0">
          {{ userInitials }}
        </div>
        <div class="flex flex-col truncate">
          <span class="text-xs font-semibold text-white truncate">{{ session.user?.username || 'User' }}</span>
          <span class="text-[10px] text-gray-400 truncate">{{ session.user?.email }}</span>
        </div>
      </div>
    </div>

    <!-- Create Channel Modal -->
    <CreateChannelModal
      v-if="workspaceId"
      :is-open="isModalOpen"
      :workspace-id="workspaceId"
      @close="isModalOpen = false"
      @created="onChannelCreated"
    />
  </aside>
</template>

<script setup lang="ts">
import { useQuery } from "@tanstack/vue-query";
import { computed, ref, watch } from "vue";
import { navigateTo, useNuxtApp, useRoute } from "#app";
import { useActiveWorkspaceStore } from "~/stores/activeWorkspace";
import { useSessionStore } from "~/stores/session";
import CreateChannelModal from "./CreateChannelModal.vue";

const route = useRoute();
const { $api } = useNuxtApp();
const activeStore = useActiveWorkspaceStore();
const session = useSessionStore();
const isModalOpen = ref(false);

const workspaceId = computed(() => (route.params.id as string) || null);

// Fetch active workspace details (for name display)
const { data: workspacesResponse } = useQuery({
	queryKey: ["workspaces"],
	queryFn: async () => {
		const res: any = await $api("/workspaces", { method: "GET" });
		return res;
	},
});

const workspaceName = computed(() => {
	if (!workspaceId.value || !workspacesResponse.value)
		return "Select Workspace";
	const data = workspacesResponse.value.data || workspacesResponse.value;
	const list = data.workspaces || data || [];
	const found = list.find((w: any) => w.id === workspaceId.value);
	return found?.name || "Workspace";
});

// Fetch channels for the current workspace
const { data: channelsResponse, isLoading } = useQuery({
	queryKey: ["channels", workspaceId],
	queryFn: async () => {
		if (!workspaceId.value) return [];
		const res: any = await $api(`/workspaces/${workspaceId.value}/channels`, {
			method: "GET",
		});
		return res;
	},
	enabled: computed(() => !!workspaceId.value),
});

const channels = computed(() => {
	if (!channelsResponse.value) return [];
	const data = channelsResponse.value.data || channelsResponse.value;
	return Array.isArray(data) ? data : [];
});

const userInitials = computed(() => {
	const name = session.user?.username || "U";
	return name.slice(0, 2).toUpperCase();
});

const selectChannel = (ch: any) => {
	activeStore.setChannel(ch.id);
	if (workspaceId.value) {
		navigateTo(`/workspaces/${workspaceId.value}/channels/${ch.id}`);
	}
};

const onChannelCreated = (ch: any) => {
	if (ch?.id && workspaceId.value) {
		activeStore.setChannel(ch.id);
		navigateTo(`/workspaces/${workspaceId.value}/channels/${ch.id}`);
	}
};

// Sync active store with route params
watch(
	() => route.params,
	(params) => {
		if (params.id) {
			activeStore.setWorkspace(params.id as string);
		}
		if (params.channelId) {
			activeStore.setChannel(params.channelId as string);
		}
	},
	{ immediate: true },
);
</script>
