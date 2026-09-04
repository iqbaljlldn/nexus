import { useCookie, useNuxtApp } from "#app";
import { useSessionStore } from "../stores/session";

export async function useAuthRefresh(): Promise<boolean> {
	const csrfToken = useCookie("csrf_token").value;
	const { $api } = useNuxtApp();
	const session = useSessionStore();

	try {
		const data = (await $api("/auth/refresh", {
			method: "POST",
			headers: csrfToken ? { "X-CSRF-Token": csrfToken } : undefined,
		})) as { access_token: string };

		if (data && data.access_token) {
			session.setAccessToken(data.access_token);
			return true;
		}
		return false;
	} catch (error) {
		return false;
	}
}
