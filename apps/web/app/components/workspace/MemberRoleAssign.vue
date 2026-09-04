<template>
  <div class="space-y-6">
    <div>
      <h3 class="text-lg font-bold text-white">Member Role Assignment</h3>
      <p class="text-sm text-gray-400">Assign or remove roles for members in this workspace.</p>
    </div>

    <!-- Member List & Assignment Panel -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <!-- Member Selector -->
      <div class="bg-gray-900 border border-gray-700/60 rounded-xl p-4 space-y-3">
        <h4 class="text-xs font-semibold uppercase tracking-wider text-gray-400">Members</h4>

        <div v-if="isLoadingMembers" class="space-y-2">
          <div v-for="i in 3" :key="i" class="h-10 bg-gray-800 rounded-lg animate-pulse" />
        </div>

        <div v-else-if="members.length === 0" class="text-sm text-gray-500 italic py-2">
          No members found.
        </div>

        <div v-else class="space-y-1">
          <button
            v-for="m in members"
            :key="m.id"
            @click="selectedMemberId = m.id"
            :class="[
              'w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm transition-colors text-left',
              selectedMemberId === m.id
                ? 'bg-indigo-600 text-white font-semibold'
                : 'text-gray-300 hover:bg-gray-800'
            ]"
          >
            <span class="truncate">{{ m.nickname || m.user_id || 'Member' }}</span>
            <svg v-if="selectedMemberId === m.id" class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Role Assignment Panel -->
      <div class="md:col-span-2 bg-gray-900 border border-gray-700/60 rounded-xl p-5 space-y-5">
        <div v-if="!selectedMemberId" class="flex flex-col items-center justify-center py-12 text-gray-500">
          <svg class="w-10 h-10 mb-2 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
          </svg>
          <p class="text-sm">Select a member on the left to edit roles</p>
        </div>

        <div v-else class="space-y-4">
          <h4 class="text-base font-semibold text-white">
            Roles for Member: <span class="text-indigo-400">{{ selectedMemberName }}</span>
          </h4>

          <div v-if="errorMessage" class="bg-red-500/10 border border-red-500/50 text-red-400 p-3 rounded-lg text-sm">
            {{ errorMessage }}
          </div>

          <div v-if="successMessage" class="bg-emerald-500/10 border border-emerald-500/50 text-emerald-400 p-3 rounded-lg text-sm">
            {{ successMessage }}
          </div>

          <div v-if="isLoadingRoles" class="space-y-2">
            <div v-for="i in 2" :key="i" class="h-10 bg-gray-800 rounded-lg animate-pulse" />
          </div>

          <div v-else-if="roles.length === 0" class="text-sm text-gray-500 italic">
            No custom roles created yet. Create a role above first.
          </div>

          <div v-else class="space-y-2">
            <div
              v-for="r in roles"
              :key="r.id"
              class="flex items-center justify-between bg-gray-800/80 border border-gray-700/50 p-3 rounded-lg"
            >
              <div class="flex items-center gap-3">
                <input
                  :id="'role-assign-' + r.id"
                  type="checkbox"
                  :value="r.id"
                  v-model="assignedRoleIds"
                  :disabled="r.is_everyone"
                  class="w-4 h-4 accent-indigo-600 rounded bg-gray-900 border-gray-700 focus:ring-0 cursor-pointer disabled:opacity-50"
                />
                <label :for="'role-assign-' + r.id" class="text-sm font-medium text-white cursor-pointer">
                  {{ r.name }}
                  <span v-if="r.is_everyone" class="text-xs text-gray-500 font-normal"> (Default)</span>
                </label>
              </div>
            </div>
          </div>

          <div class="flex justify-end pt-4 border-t border-gray-700/50">
            <button
              @click="handleSaveRoles"
              :disabled="isSaving"
              class="bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold px-5 py-2 rounded-lg transition-colors disabled:opacity-50 shadow-md"
            >
              <span v-if="isSaving">Updating Roles...</span>
              <span v-else>Save Changes</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useMutation, useQuery } from "@tanstack/vue-query";
import { computed, ref, watch } from "vue";
import { useNuxtApp } from "#app";

const props = defineProps<{
	workspaceId: string;
}>();

const { $api } = useNuxtApp();

const selectedMemberId = ref<string | null>(null);
const assignedRoleIds = ref<string[]>([]);
const errorMessage = ref("");
const successMessage = ref("");

// Fetch Members
const { data: membersResponse, isLoading: isLoadingMembers } = useQuery({
	queryKey: ["members", props.workspaceId],
	queryFn: async () => {
		const res: any = await $api(`/workspaces/${props.workspaceId}/members`, {
			method: "GET",
		});
		return res;
	},
});

const members = computed(() => {
	if (!membersResponse.value) return [];
	const data = membersResponse.value.data || membersResponse.value;
	return Array.isArray(data) ? data : [];
});

const selectedMemberName = computed(() => {
	const found = members.value.find((m: any) => m.id === selectedMemberId.value);
	return found?.nickname || found?.user_id || "Member";
});

// Fetch Roles
const { data: rolesResponse, isLoading: isLoadingRoles } = useQuery({
	queryKey: ["roles", props.workspaceId],
	queryFn: async () => {
		const res: any = await $api(`/workspaces/${props.workspaceId}/roles`, {
			method: "GET",
		});
		return res;
	},
});

const roles = computed(() => {
	if (!rolesResponse.value) return [];
	const data = rolesResponse.value.data || rolesResponse.value;
	return Array.isArray(data) ? data : [];
});

const { mutate: saveRoles, isPending: isSaving } = useMutation({
	mutationFn: (roleIds: string[]) =>
		$api(
			`/workspaces/${props.workspaceId}/members/${selectedMemberId.value}/roles`,
			{
				method: "PATCH",
				body: { role_ids: roleIds },
			},
		),
	onSuccess: () => {
		successMessage.value = "Member roles updated successfully.";
		setTimeout(() => {
			successMessage.value = "";
		}, 3000);
	},
	onError: (error: any) => {
		const errorData = error.response?._data;
		if (errorData?.error?.message) {
			errorMessage.value = errorData.error.message;
		} else {
			errorMessage.value = "Failed to update member roles.";
		}
	},
});

const handleSaveRoles = () => {
	errorMessage.value = "";
	successMessage.value = "";
	if (!selectedMemberId.value) return;
	saveRoles(assignedRoleIds.value);
};
</script>
