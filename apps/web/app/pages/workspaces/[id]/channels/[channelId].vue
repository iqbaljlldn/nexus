<template>
  <div class="flex-1 flex flex-col h-full bg-gray-800 text-white overflow-hidden">
    <!-- Channel Header -->
    <header class="h-14 border-b border-gray-700/60 px-6 flex items-center justify-between shrink-0 bg-gray-800/80 backdrop-blur-xs shadow-xs">
      <div class="flex items-center gap-2 font-bold text-lg">
        <span class="text-gray-400 font-normal">#</span>
        <span>{{ channelName }}</span>
      </div>
    </header>

    <!-- Messages Container -->
    <div class="flex-1 overflow-y-auto p-6 flex flex-col justify-end">
      <!-- Empty State / Sprint 4 Placeholder -->
      <div class="mb-6 space-y-2">
        <div class="w-16 h-16 rounded-full bg-gray-700/60 flex items-center justify-center mb-4">
          <span class="text-3xl font-bold text-gray-300">#</span>
        </div>
        <h1 class="text-3xl font-extrabold text-white">Welcome to #{{ channelName }}!</h1>
        <p class="text-gray-400 text-sm">
          This is the start of the <span class="font-semibold text-gray-300">#{{ channelName }}</span> channel.
        </p>
        <div class="pt-4 border-t border-gray-700/40">
          <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-indigo-500/10 border border-indigo-500/30 text-indigo-400">
            💬 Real-time messaging coming in Sprint 4
          </span>
        </div>
      </div>
    </div>

    <!-- Message Composer Bar Placeholder -->
    <div class="p-4 shrink-0 bg-gray-800 border-t border-gray-700/40">
      <div class="bg-gray-900 border border-gray-700/80 rounded-xl px-4 py-2.5 flex items-center gap-3">
        <input
          type="text"
          :placeholder="'Message #' + channelName"
          disabled
          class="w-full bg-transparent text-sm text-gray-400 placeholder-gray-500 focus:outline-none cursor-not-allowed"
        />
        <button
          disabled
          class="bg-indigo-600/50 text-white/50 text-xs font-semibold px-4 py-1.5 rounded-lg cursor-not-allowed"
        >
          Send
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useQuery } from "@tanstack/vue-query";
import { computed } from "vue";
import { useNuxtApp, useRoute } from "#app";

const route = useRoute();
const { $api } = useNuxtApp();

const workspaceId = computed(() => (route.params.id as string) || "");
const channelId = computed(() => (route.params.channelId as string) || "");

// Fetch channel list to find current channel name
const { data: channelsResponse } = useQuery({
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

const channelName = computed(() => {
	if (!channelsResponse.value) return "general";
	const data = channelsResponse.value.data || channelsResponse.value;
	const list = Array.isArray(data) ? data : [];
	const found = list.find((c: any) => c.id === channelId.value);
	return found?.name || "general";
});
</script>
