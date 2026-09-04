<template>
  <div class="flex-1 flex flex-col items-center justify-center p-6 text-center bg-gray-800 text-white">
    <div v-if="isLoading" class="flex flex-col items-center gap-3">
      <div class="w-10 h-10 border-4 border-indigo-500/30 border-t-indigo-500 rounded-full animate-spin" />
      <p class="text-sm text-gray-400">Loading channels...</p>
    </div>

    <div v-else class="max-w-md bg-gray-900/80 border border-gray-700/60 p-8 rounded-2xl shadow-xl flex flex-col items-center">
      <div class="w-14 h-14 rounded-full bg-indigo-600/20 text-indigo-400 flex items-center justify-center mb-4">
        <span class="text-2xl font-bold">#</span>
      </div>

      <h2 class="text-xl font-bold mb-2">No Channels in Workspace</h2>
      <p class="text-gray-400 text-sm mb-6">
        This workspace doesn't have any text channels yet. Create your first channel to start chatting!
      </p>

      <button
        @click="isModalOpen = true"
        class="bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-sm px-5 py-2.5 rounded-xl transition-all shadow-lg shadow-indigo-600/25 flex items-center gap-2"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span>Create Channel</span>
      </button>
    </div>

    <CreateChannelModal
      v-if="workspaceId"
      :is-open="isModalOpen"
      :workspace-id="workspaceId"
      @close="isModalOpen = false"
      @created="onChannelCreated"
    />
  </div>
</template>

<script setup lang="ts">
import { useQuery } from "@tanstack/vue-query";
import { computed, ref, watch } from "vue";
import { navigateTo, useNuxtApp, useRoute } from "#app";
import CreateChannelModal from "~/components/channel/CreateChannelModal.vue";
import { useActiveWorkspaceStore } from "~/stores/activeWorkspace";

const route = useRoute();
const { $api } = useNuxtApp();
const activeStore = useActiveWorkspaceStore();
const isModalOpen = ref(false);

const workspaceId = computed(() => (route.params.id as string) || "");

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

watch(
	channels,
	(list) => {
		if (list.length > 0 && workspaceId.value) {
			const firstCh = list[0];
			activeStore.setChannel(firstCh.id);
			navigateTo(`/workspaces/${workspaceId.value}/channels/${firstCh.id}`);
		}
	},
	{ immediate: true },
);

const onChannelCreated = (ch: any) => {
	if (ch?.id && workspaceId.value) {
		activeStore.setChannel(ch.id);
		navigateTo(`/workspaces/${workspaceId.value}/channels/${ch.id}`);
	}
};
</script>
