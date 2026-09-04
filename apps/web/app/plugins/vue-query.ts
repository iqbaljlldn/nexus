import type { VueQueryPluginOptions } from "@tanstack/vue-query";
import { QueryClient, VueQueryPlugin } from "@tanstack/vue-query";
import { defineNuxtPlugin } from "#app";

export default defineNuxtPlugin((nuxtApp) => {
	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				staleTime: 5000,
			},
		},
	});

	const options: VueQueryPluginOptions = {
		queryClient,
	};

	nuxtApp.vueApp.use(VueQueryPlugin, options);
});
