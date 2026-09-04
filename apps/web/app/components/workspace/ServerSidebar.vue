<template>
  <aside class="w-[72px] bg-gray-950 flex flex-col items-center py-3 gap-2 shrink-0 h-full select-none border-r border-gray-800/40">
    <!-- Home / Overview Link -->
    <NuxtLink
      to="/workspaces"
      class="relative group flex items-center justify-center w-12 h-12 rounded-3xl hover:rounded-2xl bg-gray-800 text-indigo-400 hover:bg-indigo-600 hover:text-white transition-all duration-200 shadow-md"
      title="Direct Messages & Dashboard"
    >
      <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
      </svg>
      <!-- Active Pill indicator -->
      <span
        v-if="$route.path === '/workspaces'"
        class="absolute -left-3 w-1.5 h-10 bg-white rounded-r-full"
      />
    </NuxtLink>

    <!-- Separator -->
    <div class="w-8 h-[2px] bg-gray-800/80 rounded my-1" />

    <!-- Workspace List -->
    <div class="flex-1 w-full flex flex-col items-center gap-2 overflow-y-auto no-scrollbar px-2">
      <div v-if="isLoading" class="flex flex-col gap-2">
        <div v-for="i in 3" :key="i" class="w-12 h-12 rounded-3xl bg-gray-800/60 animate-pulse" />
      </div>

      <div
        v-else
        v-for="ws in workspaces"
        :key="ws.id"
        class="relative group flex items-center justify-center w-full"
      >
        <!-- Active Pill -->
        <span
          v-if="activeStore.currentWorkspaceId === ws.id"
          class="absolute -left-3 w-1.5 h-10 bg-white rounded-r-full transition-all duration-200"
        />
        <span
          v-else
          class="absolute -left-3 w-1.5 h-0 group-hover:h-5 bg-white/70 rounded-r-full transition-all duration-200"
        />

        <button
          @click="selectWorkspace(ws)"
          :class="[
            'w-12 h-12 flex items-center justify-center font-bold text-base transition-all duration-200 shadow-md',
            activeStore.currentWorkspaceId === ws.id
              ? 'rounded-2xl bg-indigo-600 text-white'
              : 'rounded-3xl hover:rounded-2xl bg-gray-800 text-gray-200 hover:bg-indigo-600 hover:text-white'
          ]"
          :title="ws.name"
        >
          <img
            v-if="ws.icon_url"
            :src="ws.icon_url"
            :alt="ws.name"
            class="w-full h-full object-cover rounded-[inherit]"
          />
          <span v-else>{{ getInitials(ws.name) }}</span>
        </button>
      </div>

      <!-- Add Workspace Button -->
      <button
        @click="isModalOpen = true"
        class="group flex items-center justify-center w-12 h-12 rounded-3xl hover:rounded-2xl bg-gray-800 text-emerald-500 hover:bg-emerald-600 hover:text-white transition-all duration-200 shadow-md mt-1"
        title="Add a Workspace"
      >
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
      </button>
    </div>

    <!-- Create Workspace Modal -->
    <CreateWorkspaceModal
      :is-open="isModalOpen"
      @close="isModalOpen = false"
      @created="onWorkspaceCreated"
    />
  </aside>
</template>

<script setup lang="ts">
import { useQuery } from "@tanstack/vue-query";
import { computed, ref } from "vue";
import { navigateTo, useNuxtApp } from "#app";
import { useActiveWorkspaceStore } from "../../stores/activeWorkspace";
import CreateWorkspaceModal from "./CreateWorkspaceModal.vue";

const { $api } = useNuxtApp();
const activeStore = useActiveWorkspaceStore();
const isModalOpen = ref(false);

const { data: workspacesResponse, isLoading } = useQuery({
	queryKey: ["workspaces"],
	queryFn: async () => {
		const res: any = await $api("/workspaces", { method: "GET" });
		return res;
	},
});

const workspaces = computed(() => {
	if (!workspacesResponse.value) return [];
	// Envelope response: { success: true, data: { workspaces: [...] } } or { data: [...] }
	const data = workspacesResponse.value.data || workspacesResponse.value;
	return data.workspaces || data || [];
});

const getInitials = (name: string) => {
	if (!name) return "W";
	const words = name.trim().split(/\s+/);
	if (words.length >= 2) {
		return (words[0][0] + words[1][0]).toUpperCase();
	}
	return name.slice(0, 2).toUpperCase();
};

const selectWorkspace = (ws: any) => {
	activeStore.setWorkspace(ws.id);
	navigateTo(`/workspaces/${ws.id}`);
};

const onWorkspaceCreated = (ws: any) => {
	if (ws?.id) {
		activeStore.setWorkspace(ws.id);
		navigateTo(`/workspaces/${ws.id}`);
	}
};
</script>

<style scoped>
.no-scrollbar::-webkit-scrollbar {
  display: none;
}
.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
