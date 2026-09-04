<template>
  <div class="flex-1 flex flex-col h-full bg-gray-800 text-white overflow-hidden">
    <!-- Header -->
    <header class="h-14 border-b border-gray-700/60 px-6 flex items-center justify-between shrink-0 bg-gray-800/80 backdrop-blur-xs shadow-xs">
      <div class="flex items-center gap-3">
        <NuxtLink
          :to="`/workspaces/${workspaceId}`"
          class="p-1.5 rounded-lg text-gray-400 hover:text-white hover:bg-gray-700/50 transition-colors"
          title="Back to Workspace"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
        </NuxtLink>
        <h2 class="text-lg font-bold">Workspace Settings</h2>
      </div>
    </header>

    <!-- Main Settings Body -->
    <div class="flex-1 overflow-y-auto p-6 max-w-4xl w-full mx-auto space-y-6">
      <!-- Tabs Navigation -->
      <div class="flex items-center gap-2 border-b border-gray-700/60 pb-3">
        <button
          @click="activeTab = 'general'"
          :class="[
            'px-4 py-2 rounded-lg text-sm font-semibold transition-colors',
            activeTab === 'general'
              ? 'bg-indigo-600 text-white shadow-md'
              : 'text-gray-400 hover:text-white hover:bg-gray-700/40'
          ]"
        >
          Overview
        </button>
        <button
          @click="activeTab = 'roles'"
          :class="[
            'px-4 py-2 rounded-lg text-sm font-semibold transition-colors',
            activeTab === 'roles'
              ? 'bg-indigo-600 text-white shadow-md'
              : 'text-gray-400 hover:text-white hover:bg-gray-700/40'
          ]"
        >
          Roles
        </button>
        <button
          @click="activeTab = 'members'"
          :class="[
            'px-4 py-2 rounded-lg text-sm font-semibold transition-colors',
            activeTab === 'members'
              ? 'bg-indigo-600 text-white shadow-md'
              : 'text-gray-400 hover:text-white hover:bg-gray-700/40'
          ]"
        >
          Member Roles
        </button>
      </div>

      <!-- Tab Content: General Overview -->
      <div v-if="activeTab === 'general'" class="space-y-6">
        <div class="bg-gray-900 border border-gray-700/60 rounded-xl p-6 space-y-4">
          <h3 class="text-lg font-bold text-white">Workspace Overview</h3>
          <p class="text-sm text-gray-400">View and manage basic details for your workspace.</p>

          <div class="pt-4 border-t border-gray-700/40 flex items-center justify-between">
            <div>
              <h4 class="text-sm font-semibold text-white">Invite People</h4>
              <p class="text-xs text-gray-400">Generate an invite link to invite members to this workspace.</p>
            </div>
            <button
              @click="isInviteModalOpen = true"
              class="bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold px-4 py-2 rounded-lg transition-colors flex items-center gap-2 shadow-md"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" />
              </svg>
              <span>Create Invite</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Tab Content: Roles -->
      <div v-else-if="activeTab === 'roles'">
        <RoleManager v-if="workspaceId" :workspace-id="workspaceId" />
      </div>

      <!-- Tab Content: Member Role Assignment -->
      <div v-else-if="activeTab === 'members'">
        <MemberRoleAssign v-if="workspaceId" :workspace-id="workspaceId" />
      </div>
    </div>

    <!-- Invite Modal -->
    <InviteModal
      v-if="workspaceId"
      :is-open="isInviteModalOpen"
      :workspace-id="workspaceId"
      @close="isInviteModalOpen = false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "#app";
import InviteModal from "~/components/workspace/InviteModal.vue";
import MemberRoleAssign from "~/components/workspace/MemberRoleAssign.vue";
import RoleManager from "~/components/workspace/RoleManager.vue";

const route = useRoute();
const activeTab = ref<"general" | "roles" | "members">("general");
const isInviteModalOpen = ref(false);

const workspaceId = computed(() => (route.params.id as string) || "");
</script>
