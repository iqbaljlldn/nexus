import { ofetch } from "ofetch";
import { defineNuxtPlugin, navigateTo } from "#app";
import { useAuthRefresh } from "../composables/useAuthRefresh";
import { useSessionStore } from "../stores/session";

export default defineNuxtPlugin(() => {
	const apiClient = ofetch.create({
		baseURL: "http://localhost:8080/api/v1",
		credentials: "include",
		onRequest({ options }) {
			const session = useSessionStore();
			if (session.accessToken) {
				options.headers = options.headers || {};
				if (options.headers instanceof Headers) {
					options.headers.set("Authorization", `Bearer ${session.accessToken}`);
				} else {
					(options.headers as Record<string, string>)["Authorization"] =
						`Bearer ${session.accessToken}`;
				}
			}
		},
		async onResponseError({ response, request, options }) {
			if (response.status === 401) {
				// Prevent infinite loops if the refresh endpoint itself returns 401
				if (request.toString().includes("/auth/refresh")) {
					const session = useSessionStore();
					session.logout();
					navigateTo("/login");
					return;
				}

				const refreshed = await useAuthRefresh();
				if (refreshed) {
					// Retry original request
					return apiClient(request, options);
				}

				// Refresh failed -> logout and redirect
				const session = useSessionStore();
				session.logout();
				navigateTo("/login");
			}
		},
	});

	return {
		provide: {
			api: apiClient,
		},
	};
});
