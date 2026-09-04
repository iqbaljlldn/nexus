import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it } from "vitest";
import { useActiveWorkspaceStore } from "../../app/stores/activeWorkspace";

describe("activeWorkspace Store", () => {
	beforeEach(() => {
		setActivePinia(createPinia());
	});

	it("initializes with null state", () => {
		const store = useActiveWorkspaceStore();
		expect(store.currentWorkspaceId).toBeNull();
		expect(store.currentChannelId).toBeNull();
	});

	it("sets workspace ID", () => {
		const store = useActiveWorkspaceStore();
		store.setWorkspace("ws-123");
		expect(store.currentWorkspaceId).toBe("ws-123");
	});

	it("sets channel ID", () => {
		const store = useActiveWorkspaceStore();
		store.setChannel("ch-456");
		expect(store.currentChannelId).toBe("ch-456");
	});

	it("clears state", () => {
		const store = useActiveWorkspaceStore();
		store.setWorkspace("ws-123");
		store.setChannel("ch-456");

		store.clear();
		expect(store.currentWorkspaceId).toBeNull();
		expect(store.currentChannelId).toBeNull();
	});
});
