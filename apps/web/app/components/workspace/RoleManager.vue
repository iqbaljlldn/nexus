<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h3 class="text-lg font-bold text-white">Workspace Roles</h3>
        <p class="text-sm text-gray-400">Manage custom roles and permissions for your workspace members.</p>
      </div>
      <button
        @click="showCreateForm = !showCreateForm"
        class="bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold px-4 py-2 rounded-lg transition-colors flex items-center gap-2 shadow-md"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span>{{ showCreateForm ? 'Cancel' : 'Create Role' }}</span>
      </button>
    </div>

    <!-- Create Role Form -->
    <div v-if="showCreateForm" class="bg-gray-900 border border-gray-700/60 rounded-xl p-5 space-y-5">
      <h4 class="text-base font-semibold text-white">Create New Role</h4>

      <div v-if="errorMessage" class="bg-red-500/10 border border-red-500/50 text-red-400 p-3 rounded-lg text-sm">
        {{ errorMessage }}
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label for="role-name" class="block text-xs font-semibold uppercase tracking-wider text-gray-300 mb-1">
            Role Name <span class="text-red-400">*</span>
          </label>
          <input
            id="role-name"
            v-model="form.name"
            type="text"
            required
            placeholder="e.g. Moderator"
            class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3.5 py-2 text-sm text-white focus:outline-none focus:border-indigo-500"
          />
        </div>

        <div>
          <label for="role-pos" class="block text-xs font-semibold uppercase tracking-wider text-gray-300 mb-1">
            Position Order
          </label>
          <input
            id="role-pos"
            v-model.number="form.position"
            type="number"
            min="0"
            class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3.5 py-2 text-sm text-white focus:outline-none focus:border-indigo-500"
          />
        </div>
      </div>

      <!-- Permission Checkboxes -->
      <div>
        <label class="block text-xs font-semibold uppercase tracking-wider text-gray-300 mb-3">
          Permissions
        </label>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div
            v-for="perm in PERMISSION_LIST"
            :key="perm.flag"
            class="flex items-start gap-3 bg-gray-800/60 border border-gray-700/40 p-3 rounded-lg hover:border-gray-600 transition-colors"
          >
            <input
              :id="'perm-' + perm.flag"
              type="checkbox"
              :value="perm.flag"
              v-model="selectedFlags"
              class="mt-0.5 w-4 h-4 accent-indigo-600 rounded bg-gray-900 border-gray-700 focus:ring-0 cursor-pointer"
            />
            <label :for="'perm-' + perm.flag" class="cursor-pointer">
              <span class="block text-sm font-semibold text-gray-200">{{ perm.name }}</span>
              <span class="block text-xs text-gray-400 mt-0.5">{{ perm.description }}</span>
            </label>
          </div>
        </div>
      </div>

      <div class="flex justify-end gap-3 pt-2">
        <button
          type="button"
          @click="showCreateForm = false"
          class="px-4 py-2 text-sm text-gray-400 hover:text-white"
        >
          Cancel
        </button>
        <button
          type="button"
          @click="handleCreateRole"
          :disabled="isPending"
          class="bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold px-5 py-2 rounded-lg transition-colors disabled:opacity-50"
        >
          <span v-if="isPending">Saving...</span>
          <span v-else>Save Role</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useMutation, useQueryClient } from "@tanstack/vue-query";
import { computed, reactive, ref } from "vue";
import { useNuxtApp } from "#app";
import { PERMISSION_LIST } from "~/constants/permissions";

const props = defineProps<{
	workspaceId: string;
}>();

const { $api } = useNuxtApp();
const queryClient = useQueryClient();

const showCreateForm = ref(false);
const selectedFlags = ref<number[]>([]);
const errorMessage = ref("");

const form = reactive({
	name: "",
	position: 1,
});

const calculatedBitmask = computed(() => {
	return selectedFlags.value.reduce((acc, flag) => acc | flag, 0);
});

const { mutate, isPending } = useMutation({
	mutationFn: (body: any) =>
		$api(`/workspaces/${props.workspaceId}/roles`, {
			method: "POST",
			body,
		}),
	onSuccess: () => {
		queryClient.invalidateQueries({
			queryKey: ["roles", props.workspaceId],
		});
		showCreateForm.value = false;
		form.name = "";
		form.position = 1;
		selectedFlags.value = [];
		errorMessage.value = "";
	},
	onError: (error: any) => {
		const errorData = error.response?._data;
		if (errorData?.error?.message) {
			errorMessage.value = errorData.error.message;
		} else {
			errorMessage.value = "Failed to create role. Please try again.";
		}
	},
});

const handleCreateRole = () => {
	errorMessage.value = "";
	if (!form.name.trim()) {
		errorMessage.value = "Role name is required.";
		return;
	}
	mutate({
		name: form.name.trim(),
		permission_bitmask: calculatedBitmask.value,
		position: form.position,
	});
};
</script>
