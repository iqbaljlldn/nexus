<template>
  <div class="h-screen w-full flex bg-gray-900 text-white overflow-hidden font-sans">
    <!-- Server Sidebar (Leftmost 72px) -->
    <ServerSidebar />

    <!-- Channel Sidebar (Middle 240px) -->
    <ChannelSidebar v-if="showChannelSidebar" />

    <!-- Main Content Area -->
    <main class="flex-1 flex flex-col h-full overflow-hidden bg-gray-800">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "#app";
import ChannelSidebar from "../components/channel/ChannelSidebar.vue";
import ServerSidebar from "../components/workspace/ServerSidebar.vue";

const route = useRoute();

// Show channel sidebar when navigating inside /workspaces
const showChannelSidebar = computed(() => {
	return route.path.startsWith("/workspaces");
});
</script>
