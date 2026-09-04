import { beforeEach, describe, expect, it, vi } from "vitest";
import { navigateTo } from "#app";
import { useAuthRefresh } from "../app/composables/useAuthRefresh";
import apiClientPlugin from "../app/plugins/api-client";
import { useSessionStore } from "../app/stores/session";

vi.mock("#app", () => ({
	defineNuxtPlugin: (cb: any) => cb,
	navigateTo: vi.fn(),
}));

vi.mock("ofetch", () => ({
	ofetch: {
		create: vi.fn((opts) => {
			const fn = vi.fn();
			Object.assign(fn, opts);
			return fn;
		}),
	},
}));

vi.mock("../app/stores/session", () => ({
	useSessionStore: vi.fn(),
}));

vi.mock("../app/composables/useAuthRefresh", () => ({
	useAuthRefresh: vi.fn(),
}));

describe("api-client plugin interceptors", () => {
	let mockSession: any;
	let pluginOpts: any;

	beforeEach(() => {
		vi.clearAllMocks();
		mockSession = {
			accessToken: "test-token",
			logout: vi.fn(),
		};
		vi.mocked(useSessionStore).mockReturnValue(mockSession as any);

		// pluginOpts is the object passed to ofetch.create
		const pluginResult: any = apiClientPlugin();
		pluginOpts = pluginResult.provide.api;
	});

	it("onRequest should inject Authorization header if token exists", () => {
		const options = { headers: new Headers() };
		pluginOpts.onRequest({ options });

		expect(options.headers.get("Authorization")).toBe("Bearer test-token");
	});

	it("onResponseError should trigger refresh on 401", async () => {
		useAuthRefresh.mockResolvedValueOnce(true);

		const request = "/some/endpoint";
		const options = {};

		// Using vi.fn for the apiClient itself inside the retry
		// Since we return options from our mock ofetch.create, we need to handle the retry call manually
		// In our test, the retry does `return apiClient(request, options)`
		// For simplicity, we just check that useAuthRefresh is called

		await pluginOpts.onResponseError({
			response: { status: 401 },
			request,
			options,
		});

		expect(useAuthRefresh).toHaveBeenCalled();
		expect(mockSession.logout).not.toHaveBeenCalled();
		expect(navigateTo).not.toHaveBeenCalled();
	});

	it("onResponseError should logout and redirect if refresh fails", async () => {
		useAuthRefresh.mockResolvedValueOnce(false);

		await pluginOpts.onResponseError({
			response: { status: 401 },
			request: "/api/v1/protected",
			options: {},
		});

		expect(mockSession.logout).toHaveBeenCalled();
		expect(navigateTo).toHaveBeenCalledWith("/login");
	});

	it("onResponseError should prevent infinite loop on /auth/refresh", async () => {
		await pluginOpts.onResponseError({
			response: { status: 401 },
			request: "/auth/refresh",
			options: {},
		});

		expect(useAuthRefresh).not.toHaveBeenCalled();
		expect(mockSession.logout).toHaveBeenCalled();
		expect(navigateTo).toHaveBeenCalledWith("/login");
	});
});
