<template>
  <div class="min-h-screen bg-gray-900 flex items-center justify-center p-6 text-white">
    <div class="bg-gray-800 border border-gray-700/60 rounded-2xl shadow-2xl p-8 max-w-md w-full text-center">
      <div class="w-16 h-16 rounded-full bg-indigo-600/20 text-indigo-400 flex items-center justify-center mx-auto mb-4">
        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" />
        </svg>
      </div>

      <h2 class="text-2xl font-bold mb-2">You've Been Invited!</h2>
      <p class="text-gray-400 text-sm mb-6">
        Click below to join this workspace and connect with members.
      </p>

      <div v-if="errorMessage" class="bg-red-500/10 border border-red-500/50 text-red-400 p-3 rounded-lg text-sm mb-6">
        {{ errorMessage }}
      </div>

      <button
        @click="handleRedeem"
        :disabled="isPending"
        class="w-full bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-sm py-3 px-6 rounded-xl transition-all shadow-lg shadow-indigo-600/25 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
      >
        <span v-if="isPending">Accepting Invite...</span>
        <span v-else>Accept Invite</span>
      </button>

      <p class="mt-4 text-xs text-gray-500">
        Invite Code: <span class="font-mono text-gray-300">{{ code }}</span>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useMutation, useQueryClient } from "@tanstack/vue-query";
import { computed, ref } from "vue";
import { navigateTo, useNuxtApp, useRoute } from "#app";
import { useActiveWorkspaceStore } from "~/stores/activeWorkspace";

definePageMeta({
	layout: "auth",
});

const route = useRoute();
const { $api } = useNuxtApp();
const queryClient = useQueryClient();
const activeStore = useActiveWorkspaceStore();

const code = computed(() => (route.params.code as string) || "");
const errorMessage = ref("");

const { mutate, isPending } = useMutation({
	mutationFn: () => {
		const idempotencyKey = crypto.randomUUID();
		return $api(`/invites/${code.value}/redeem`, {
			method: "POST",
			headers: {
				"Idempotency-Key": idempotencyKey,
			},
		});
	},
	onSuccess: (res: any) => {
		queryClient.invalidateQueries({ queryKey: ["workspaces"] });
		const data = res?.data || res;
		const workspaceId = data?.workspace_id;

		if (workspaceId) {
			activeStore.setWorkspace(workspaceId);
			navigateTo(`/workspaces/${workspaceId}`);
		} else {
			navigateTo("/workspaces");
		}
	},
	onError: (error: any) => {
		const errorData = error.response?._data;
		if (errorData?.error?.message) {
			errorMessage.value = errorData.error.message;
		} else {
			errorMessage.value =
				"Failed to redeem invite. It may be expired or invalid.";
		}
	},
});

const handleRedeem = () => {
	errorMessage.value = "";
	if (!code.value) {
		errorMessage.value = "Invalid invite code.";
		return;
	}
	mutate();
};
</script>
