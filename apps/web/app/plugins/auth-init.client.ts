import { defineNuxtPlugin, useCookie, useRouter } from "#app";
import { useAuthRefresh } from "../composables/useAuthRefresh";
import { useSessionStore } from "../stores/session";

export default defineNuxtPlugin(async (nuxtApp) => {
	const session = useSessionStore();

	// We only run this logic if there is no active session yet.
	if (!session.isAuthenticated) {
		const csrfToken = useCookie("csrf_token");

		// If the csrf_token cookie exists, there's a high chance a refresh_token cookie also exists
		// (since they are generated together). Attempt an auto re-auth.
		if (csrfToken.value) {
			try {
				const refreshed = await useAuthRefresh();
				if (!refreshed) {
					// If refresh failed (e.g., token expired or invalid), the session remains unauthenticated.
					// Middlewares will handle redirecting the user to /login if they access protected routes.
				}
			} catch (err) {
				console.warn("Auto re-auth failed:", err);
			}
		}
	}
});
