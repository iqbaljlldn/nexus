// IMPORTANT: These bitmask values MUST be manually synchronized with
// apps/api/internal/role/domain/role.go whenever new permission flags are added.

export const PERMISSIONS = {
	SEND_MESSAGES: 1 << 0, // 1
	MANAGE_WORKSPACE: 1 << 1, // 2
	MANAGE_ROLES: 1 << 2, // 4
	MANAGE_CHANNELS: 1 << 3, // 8
	MANAGE_INVITES: 1 << 4, // 16
	MANAGE_MESSAGES: 1 << 5, // 32
	KICK_MEMBERS: 1 << 6, // 64
	BAN_MEMBERS: 1 << 7, // 128
} as const;

export interface PermissionDefinition {
	flag: number;
	name: string;
	description: string;
}

export const PERMISSION_LIST: PermissionDefinition[] = [
	{
		flag: PERMISSIONS.SEND_MESSAGES,
		name: "Send Messages",
		description: "Allows members to send messages in text channels",
	},
	{
		flag: PERMISSIONS.MANAGE_WORKSPACE,
		name: "Manage Workspace",
		description: "Allows changing workspace name and settings",
	},
	{
		flag: PERMISSIONS.MANAGE_ROLES,
		name: "Manage Roles",
		description: "Allows creating, editing, and assigning roles",
	},
	{
		flag: PERMISSIONS.MANAGE_CHANNELS,
		name: "Manage Channels",
		description: "Allows creating, editing, and deleting channels",
	},
	{
		flag: PERMISSIONS.MANAGE_INVITES,
		name: "Manage Invites",
		description: "Allows creating invite links for this workspace",
	},
	{
		flag: PERMISSIONS.MANAGE_MESSAGES,
		name: "Manage Messages",
		description: "Allows deleting or pinning messages from other users",
	},
	{
		flag: PERMISSIONS.KICK_MEMBERS,
		name: "Kick Members",
		description: "Allows removing members from the workspace",
	},
	{
		flag: PERMISSIONS.BAN_MEMBERS,
		name: "Ban Members",
		description: "Allows permanently banning members from the workspace",
	},
];

export function hasPermission(
	userBitmask: number,
	requiredFlag: number,
): boolean {
	return (userBitmask & requiredFlag) === requiredFlag;
}
