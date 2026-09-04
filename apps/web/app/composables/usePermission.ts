import { computed } from "vue";
import { hasPermission } from "../constants/permissions";
import { useSessionStore } from "../stores/session";

export function usePermission(
	workspaceOwnerId?: string | null,
	userBitmask?: number | null,
) {
	const session = useSessionStore();

	const isOwner = computed(() => {
		if (!session.user || !workspaceOwnerId) return false;
		return session.user.id === workspaceOwnerId;
	});

	const can = (permissionFlag: number): boolean => {
		// Owner has all permissions
		if (isOwner.value) return true;

		// Fall back to checking bitmask
		if (userBitmask !== undefined && userBitmask !== null) {
			return hasPermission(userBitmask, permissionFlag);
		}

		// Default to true for basic actions if permission data not supplied yet
		return true;
	};

	return {
		isOwner,
		can,
	};
}
