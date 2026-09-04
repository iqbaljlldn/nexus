import { beforeEach, describe, expect, it, vi } from "vitest";
import { useNuxtApp } from "#app";
import { useAuthRefresh } from "../app/composables/useAuthRefresh";
import { useSessionStore } from "../app/stores/session";

// Mock dependencies
vi.mock("#app", () => ({
	useCookie: vi.fn(() => ({ value: "mock-csrf-token" })),
	useNuxtApp: vi.fn(() => ({
		$api: vi.fn(),
	})),
}));

vi.mock("../app/stores/session", () => ({
	useSessionStore: vi.fn(),
}));

describe("useAuthRefresh", () => {
	let mockApi: any;
	let mockSetAccessToken: any;

	beforeEach(() => {
		vi.clearAllMocks();

		mockApi = vi.fn();
		vi.mocked(useNuxtApp).mockReturnValue({ $api: mockApi } as any);

		mockSetAccessToken = vi.fn();
		vi.mocked(useSessionStore).mockReturnValue({
			setAccessToken: mockSetAccessToken,
		} as any);
	});

	it("should refresh token and update store on success", async () => {
		mockApi.mockResolvedValueOnce({ access_token: "new-token" });

		const result = await useAuthRefresh();

		expect(result).toBe(true);
		expect(mockApi).toHaveBeenCalledWith("/auth/refresh", {
			method: "POST",
			headers: { "X-CSRF-Token": "mock-csrf-token" },
		});
		expect(mockSetAccessToken).toHaveBeenCalledWith("new-token");
	});

	it("should return false if request fails", async () => {
		mockApi.mockRejectedValueOnce(new Error("Unauthorized"));

		const result = await useAuthRefresh();

		expect(result).toBe(false);
		expect(mockSetAccessToken).not.toHaveBeenCalled();
	});

	it("should return false if response format is invalid", async () => {
		mockApi.mockResolvedValueOnce({ unexpected_field: "token" });

		const result = await useAuthRefresh();

		expect(result).toBe(false);
		expect(mockSetAccessToken).not.toHaveBeenCalled();
	});
});
