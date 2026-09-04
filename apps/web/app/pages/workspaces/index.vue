<template>
  <div class="flex-1 flex flex-col items-center justify-center p-6 text-center bg-gray-800 text-white">
    <div v-if="isLoading" class="flex flex-col items-center gap-3">
      <div class="w-12 h-12 border-4 border-indigo-500/30 border-t-indigo-500 rounded-full animate-spin" />
      <p class="text-sm text-gray-400">Loading workspaces...</p>
    </div>

    <div v-else-if="workspaces.length === 0" class="max-w-md bg-gray-900/80 border border-gray-700/60 p-8 rounded-2xl shadow-xl flex flex-col items-center">
      <div class="w-16 h-16 rounded-full bg-indigo-600/20 text-indigo-400 flex items-center justify-center mb-4">
        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5m0 0h4m-4 0v-4m0 4h4" />
        </svg>
      </div>

      <h2 class="text-2xl font-bold mb-2">Welcome to Nexus</h2>
      <p class="text-gray-400 text-sm mb-6">
        You don't belong to any workspace yet. Create your first workspace to start chatting and collaborating!
      </p>

      <button
        @click="isModalOpen = true"
        class="bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-sm px-6 py-2.5 rounded-xl transition-all shadow-lg shadow-indigo-600/25 flex items-center gap-2"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span>Create Workspace</span>
      </button>
    </div>

    <CreateWorkspaceModal
      :is-open="isModalOpen"
      @close="isModalOpen = false"
      @created="onCreated"
    />
  </div>
</template>

<script setup lang="ts">
import { useQuery } from "@tanstack/vue-query";
import { computed, ref, watch } from "vue";
import { navigateTo, useNuxtApp } from "#app";
import CreateWorkspaceModal from "~/components/workspace/CreateWorkspaceModal.vue";
import { useActiveWorkspaceStore } from "~/stores/activeWorkspace";

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
	const data = workspacesResponse.value.data || workspacesResponse.value;
	return data.workspaces || data || [];
});

// Auto redirect to first workspace if list is not empty
watch(
	workspaces,
	(list) => {
		if (list.length > 0) {
			const first = list[0];
			activeStore.setWorkspace(first.id);
			navigateTo(`/workspaces/${first.id}`);
		}
	},
	{ immediate: true },
);

const onCreated = (ws: any) => {
	if (ws?.id) {
		activeStore.setWorkspace(ws.id);
		navigateTo(`/workspaces/${ws.id}`);
	}
};
</script>
