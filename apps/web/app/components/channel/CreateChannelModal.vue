<template>
  <div
    v-if="isOpen"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-xs transition-opacity duration-200"
    @click.self="close"
  >
    <div
      class="bg-gray-800 border border-gray-700/60 rounded-xl shadow-2xl w-full max-w-md p-6 transform transition-all scale-100 text-white"
    >
      <div class="flex items-center justify-between pb-4 border-b border-gray-700/50 mb-5">
        <h3 class="text-xl font-bold tracking-wide">Create Channel</h3>
        <button
          @click="close"
          class="text-gray-400 hover:text-gray-200 transition-colors p-1 rounded-lg hover:bg-gray-700/50"
          aria-label="Close modal"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div v-if="errorMessage" class="bg-red-500/10 border border-red-500/50 text-red-400 p-3 rounded-lg text-sm">
          {{ errorMessage }}
        </div>

        <div>
          <label for="ch-name" class="block text-xs font-semibold uppercase tracking-wider text-gray-300 mb-1.5">
            Channel Name <span class="text-red-400">*</span>
          </label>
          <div class="relative flex items-center">
            <span class="absolute left-3 text-gray-400 text-base font-bold">#</span>
            <input
              id="ch-name"
              v-model="form.name"
              type="text"
              required
              maxlength="100"
              placeholder="new-channel"
              class="w-full bg-gray-900 border border-gray-700/70 rounded-lg pl-8 pr-3.5 py-2.5 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all"
            />
          </div>
        </div>

        <div class="flex items-center justify-end gap-3 pt-4 border-t border-gray-700/50 mt-6">
          <button
            type="button"
            @click="close"
            class="px-4 py-2 text-sm font-medium text-gray-300 hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            :disabled="isPending"
            class="bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium px-5 py-2 rounded-lg transition-all shadow-md shadow-indigo-600/20 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            <span v-if="isPending">Creating...</span>
            <span v-else>Create Channel</span>
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useMutation, useQueryClient } from "@tanstack/vue-query";
import { reactive, ref } from "vue";
import { useNuxtApp } from "#app";

const props = defineProps<{
	isOpen: boolean;
	workspaceId: string;
}>();

const emit = defineEmits<{
	(e: "close"): void;
	(e: "created", channel: any): void;
}>();

const { $api } = useNuxtApp();
const queryClient = useQueryClient();

const form = reactive({
	name: "",
});

const errorMessage = ref("");

const close = () => {
	form.name = "";
	errorMessage.value = "";
	emit("close");
};

const sanitizeChannelName = (val: string) => {
	return val
		.toLowerCase()
		.replace(/\s+/g, "-")
		.replace(/[^a-z0-9_-]/g, "");
};

const { mutate, isPending } = useMutation({
	mutationFn: (channelName: string) =>
		$api(`/workspaces/${props.workspaceId}/channels`, {
			method: "POST",
			body: {
				name: channelName,
				type: "text",
			},
		}),
	onSuccess: (res: any) => {
		queryClient.invalidateQueries({
			queryKey: ["channels", props.workspaceId],
		});
		const createdCh = res?.data || res;
		emit("created", createdCh);
		close();
	},
	onError: (error: any) => {
		const errorData = error.response?._data;
		if (errorData?.error?.message) {
			errorMessage.value = errorData.error.message;
		} else {
			errorMessage.value = "Failed to create channel. Please try again.";
		}
	},
});

const handleSubmit = () => {
	errorMessage.value = "";
	const cleanName = sanitizeChannelName(form.name);
	if (!cleanName) {
		errorMessage.value = "Please enter a valid channel name.";
		return;
	}
	mutate(cleanName);
};
</script>
