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
        <h3 class="text-xl font-bold tracking-wide">Invite Friends</h3>
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

      <!-- Generated Link Result -->
      <div v-if="inviteUrl" class="space-y-4">
        <p class="text-sm text-gray-300">
          Send this link to someone to grant them access to this workspace:
        </p>

        <div class="flex items-center gap-2 bg-gray-900 border border-gray-700/80 rounded-lg p-2">
          <input
            type="text"
            readonly
            :value="inviteUrl"
            class="bg-transparent text-sm text-gray-200 w-full focus:outline-none px-2 font-mono"
          />
          <button
            @click="copyUrl"
            class="bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold px-3 py-1.5 rounded-md transition-all shrink-0"
          >
            {{ copied ? 'Copied!' : 'Copy' }}
          </button>
        </div>

        <div class="pt-4 border-t border-gray-700/50 flex justify-end">
          <button
            @click="close"
            class="bg-gray-700 hover:bg-gray-600 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
          >
            Done
          </button>
        </div>
      </div>

      <!-- Invite Generation Form -->
      <form v-else @submit.prevent="handleSubmit" class="space-y-4">
        <div v-if="errorMessage" class="bg-red-500/10 border border-red-500/50 text-red-400 p-3 rounded-lg text-sm">
          {{ errorMessage }}
        </div>

        <div>
          <label for="max-uses" class="block text-xs font-semibold uppercase tracking-wider text-gray-300 mb-1.5">
            Max Uses <span class="text-gray-500 font-normal text-xs">(optional)</span>
          </label>
          <input
            id="max-uses"
            v-model.number="form.max_uses"
            type="number"
            min="1"
            placeholder="No limit"
            class="w-full bg-gray-900 border border-gray-700/70 rounded-lg px-3.5 py-2.5 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all"
          />
        </div>

        <div>
          <label for="expires-in" class="block text-xs font-semibold uppercase tracking-wider text-gray-300 mb-1.5">
            Expires In (Hours) <span class="text-gray-500 font-normal text-xs">(optional)</span>
          </label>
          <input
            id="expires-in"
            v-model.number="form.expires_in_hours"
            type="number"
            min="1"
            placeholder="Never"
            class="w-full bg-gray-900 border border-gray-700/70 rounded-lg px-3.5 py-2.5 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all"
          />
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
            <span v-if="isPending">Generating...</span>
            <span v-else>Generate Link</span>
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useMutation } from "@tanstack/vue-query";
import { reactive, ref } from "vue";
import { useNuxtApp } from "#app";

const props = defineProps<{
	isOpen: boolean;
	workspaceId: string;
}>();

const emit = defineEmits<(e: "close") => void>();

const { $api } = useNuxtApp();

const form = reactive({
	max_uses: null as number | null,
	expires_in_hours: null as number | null,
});

const inviteUrl = ref("");
const copied = ref(false);
const errorMessage = ref("");

const close = () => {
	form.max_uses = null;
	form.expires_in_hours = null;
	inviteUrl.value = "";
	copied.value = false;
	errorMessage.value = "";
	emit("close");
};

const copyUrl = async () => {
	if (inviteUrl.value) {
		await navigator.clipboard.writeText(inviteUrl.value);
		copied.value = true;
		setTimeout(() => {
			copied.value = false;
		}, 2000);
	}
};

const { mutate, isPending } = useMutation({
	mutationFn: (payload: any) =>
		$api(`/workspaces/${props.workspaceId}/invites`, {
			method: "POST",
			body: payload,
		}),
	onSuccess: (res: any) => {
		const data = res?.data || res;
		if (data?.url) {
			inviteUrl.value = data.url;
		} else if (data?.code) {
			inviteUrl.value = `${window.location.origin}/invite/${data.code}`;
		}
	},
	onError: (error: any) => {
		const errorData = error.response?._data;
		if (errorData?.error?.message) {
			errorMessage.value = errorData.error.message;
		} else {
			errorMessage.value = "Failed to generate invite. Please try again.";
		}
	},
});

const handleSubmit = () => {
	errorMessage.value = "";
	const body: any = {};
	if (form.max_uses) body.max_uses = form.max_uses;
	if (form.expires_in_hours) body.expires_in_hours = form.expires_in_hours;
	mutate(body);
};
</script>
