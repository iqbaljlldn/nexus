<template>
  <div class="bg-gray-900 min-h-screen text-white p-8">
    <div class="max-w-3xl mx-auto">
      <h1 class="text-3xl font-bold mb-8">Device & Session Management</h1>

      <!-- Action Buttons -->
      <div class="flex gap-4 mb-8">
        <button
          @click="handleLogoutThisDevice"
          :disabled="isLoggingOut"
          class="bg-gray-700 hover:bg-gray-600 text-white font-medium py-2 px-4 rounded transition-colors"
        >
          {{ isLoggingOut ? 'Logging out...' : 'Logout (This Device)' }}
        </button>

        <button
          @click="handleLogoutAll"
          :disabled="isLoggingOutAll"
          class="bg-red-600 hover:bg-red-700 text-white font-medium py-2 px-4 rounded transition-colors"
        >
          {{ isLoggingOutAll ? 'Logging out all...' : 'Logout from All Devices' }}
        </button>
      </div>

      <!-- Sessions List -->
      <div class="bg-gray-800 rounded-lg shadow-lg overflow-hidden">
        <div class="p-6 border-b border-gray-700">
          <h2 class="text-xl font-semibold">Active Sessions</h2>
          <p class="text-sm text-gray-400 mt-1">Manage the devices that are currently logged into your account.</p>
        </div>

        <div v-if="isLoading" class="p-6 text-center text-gray-400">
          Loading sessions...
        </div>

        <div v-else-if="isError" class="p-6 text-center text-red-500">
          Failed to load sessions.
        </div>

        <ul v-else class="divide-y divide-gray-700">
          <li v-for="session in sessionsData" :key="session.id" class="p-6 flex justify-between items-center">
            <div>
              <p class="font-medium text-lg">{{ session.user_agent }}</p>
              <div class="text-sm text-gray-400 mt-1 space-y-1">
                <p>IP Address: {{ session.ip_address }}</p>
                <p>Started: {{ new Date(session.created_at).toLocaleString() }}</p>
              </div>
            </div>
            
            <button
              @click="handleRevoke(session.id)"
              :disabled="revokingId === session.id"
              class="text-red-400 hover:text-red-300 font-medium px-3 py-1 rounded border border-red-500/30 hover:bg-red-500/10 transition-colors"
            >
              {{ revokingId === session.id ? 'Revoking...' : 'Revoke' }}
            </button>
          </li>
        </ul>

        <div v-if="sessionsData?.length === 0" class="p-6 text-center text-gray-400">
          No active sessions found.
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from "@tanstack/vue-query";
import { ref } from "vue";
import { navigateTo, useNuxtApp } from "#app";
import { useSessionStore } from "../../stores/session";

const { $api } = useNuxtApp();
const sessionStore = useSessionStore();
const queryClient = useQueryClient();

// Fetch active sessions
const {
	data: sessionsData,
	isLoading,
	isError,
} = useQuery({
	queryKey: ["sessions"],
	queryFn: () => $api("/auth/sessions").then((res: any) => res.data),
});

// Logout This Device Mutation
const { mutate: logoutThisDevice, isPending: isLoggingOut } = useMutation({
	mutationFn: () => $api("/auth/logout", { method: "POST" }),
	onSuccess: () => {
		sessionStore.logout();
		navigateTo("/login");
	},
});

// Logout All Devices Mutation
const { mutate: logoutAll, isPending: isLoggingOutAll } = useMutation({
	mutationFn: () => $api("/auth/logout-all", { method: "POST" }),
	onSuccess: () => {
		sessionStore.logout();
		navigateTo("/login");
	},
});

// Revoke Specific Session Mutation
const revokingId = ref<string | null>(null);
const { mutate: revokeSession } = useMutation({
	mutationFn: (sessionId: string) =>
		$api(`/auth/sessions/${sessionId}/revoke`, { method: "POST" }),
	onMutate: (sessionId) => {
		revokingId.value = sessionId;
	},
	onSettled: () => {
		revokingId.value = null;
	},
	onSuccess: () => {
		// Refresh the list of sessions
		queryClient.invalidateQueries({ queryKey: ["sessions"] });
	},
});

const handleLogoutThisDevice = () => {
	if (confirm("Are you sure you want to log out of this device?")) {
		logoutThisDevice();
	}
};

const handleLogoutAll = () => {
	if (
		confirm(
			"Are you sure you want to log out from ALL devices? You will be signed out everywhere.",
		)
	) {
		logoutAll();
	}
};

const handleRevoke = (sessionId: string) => {
	if (confirm("Revoke access for this device?")) {
		revokeSession(sessionId);
	}
};
</script>
