import { defineStore } from "pinia";

interface User {
	id: string;
	username: string;
	email: string;
	avatar_url?: string;
}

export const useSessionStore = defineStore("session", {
	state: () => ({
		accessToken: null as string | null,
		user: null as User | null,
	}),
	getters: {
		isAuthenticated: (state) => !!state.accessToken,
	},
	actions: {
		setAccessToken(token: string | null) {
			this.accessToken = token;
		},
		setUser(user: User | null) {
			this.user = user;
		},
		logout() {
			this.accessToken = null;
			this.user = null;
		},
	},
});
