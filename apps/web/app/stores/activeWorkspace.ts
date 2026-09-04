import { defineStore } from "pinia";

export const useActiveWorkspaceStore = defineStore("activeWorkspace", {
	state: () => ({
		currentWorkspaceId: null as string | null,
		currentChannelId: null as string | null,
	}),
	actions: {
		setWorkspace(id: string | null) {
			this.currentWorkspaceId = id;
		},
		setChannel(id: string | null) {
			this.currentChannelId = id;
		},
		clear() {
			this.currentWorkspaceId = null;
			this.currentChannelId = null;
		},
	},
});
