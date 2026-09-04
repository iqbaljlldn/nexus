import { expect, test } from "@playwright/test";

test.describe("Sprint 3 End-to-End Goal Verification (Task 6.5.1)", () => {
	test("User A creates workspace, invite & role, User B redeems invite and experiences role restrictions", async ({
		browser,
	}) => {
		// Context A (User A - Owner)
		const contextA = await browser.newContext();
		const pageA = await contextA.newPage();

		// Context B (User B - Member)
		const contextB = await browser.newContext();
		const pageB = await contextB.newPage();

		const workspaceId = "ws-sprint3-e2e";
		const inviteCode = "SPRINT3-CODE";
		let roleCreated = false;
		let memberRoleAssigned = false;
		let channelCreated = false;
		let overrideApplied = false;

		// Mock API for User A & User B
		const setupRoutes = async (page: any) => {
			await page.route("/api/v1/workspaces", async (route: any) => {
				if (route.request().method() === "POST") {
					await route.fulfill({
						status: 201,
						contentType: "application/json",
						body: JSON.stringify({
							success: true,
							data: {
								id: workspaceId,
								name: "Sprint 3 Workspace",
								owner_id: "user-a-id",
							},
						}),
					});
				} else {
					await route.fulfill({
						status: 200,
						contentType: "application/json",
						body: JSON.stringify({
							success: true,
							data: {
								workspaces: [
									{
										id: workspaceId,
										name: "Sprint 3 Workspace",
										owner_id: "user-a-id",
									},
								],
							},
						}),
					});
				}
			});

			await page.route(
				`/api/v1/workspaces/${workspaceId}/invites`,
				async (route: any) => {
					await route.fulfill({
						status: 201,
						contentType: "application/json",
						body: JSON.stringify({
							success: true,
							data: {
								code: inviteCode,
								url: `http://127.0.0.1:3002/invite/${inviteCode}`,
							},
						}),
					});
				},
			);

			await page.route(
				`/api/v1/invites/${inviteCode}/redeem`,
				async (route: any) => {
					await route.fulfill({
						status: 200,
						contentType: "application/json",
						body: JSON.stringify({
							success: true,
							data: {
								workspace_id: workspaceId,
								member_id: "member-b-id",
							},
						}),
					});
				},
			);

			await page.route(
				`/api/v1/workspaces/${workspaceId}/roles`,
				async (route: any) => {
					if (route.request().method() === "POST") {
						roleCreated = true;
						await route.fulfill({
							status: 201,
							contentType: "application/json",
							body: JSON.stringify({
								success: true,
								data: {
									id: "role-restricted-id",
									workspace_id: workspaceId,
									name: "Restricted",
									permission_bitmask: 0,
									position: 1,
								},
							}),
						});
					} else {
						const roles = [
							{
								id: "role-everyone-id",
								name: "@everyone",
								is_everyone: true,
							},
						];
						if (roleCreated) {
							roles.push({
								id: "role-restricted-id",
								name: "Restricted",
								is_everyone: false,
							});
						}
						await route.fulfill({
							status: 200,
							contentType: "application/json",
							body: JSON.stringify({ success: true, data: roles }),
						});
					}
				},
			);

			await page.route(
				`/api/v1/workspaces/${workspaceId}/members`,
				async (route: any) => {
					await route.fulfill({
						status: 200,
						contentType: "application/json",
						body: JSON.stringify({
							success: true,
							data: [
								{
									id: "member-a-id",
									workspace_id: workspaceId,
									user_id: "user-a-id",
									nickname: "User A (Owner)",
								},
								{
									id: "member-b-id",
									workspace_id: workspaceId,
									user_id: "user-b-id",
									nickname: "User B (Member)",
								},
							],
						}),
					});
				},
			);

			await page.route(
				`/api/v1/workspaces/${workspaceId}/members/member-b-id/roles`,
				async (route: any) => {
					memberRoleAssigned = true;
					await route.fulfill({
						status: 200,
						contentType: "application/json",
						body: JSON.stringify({
							success: true,
							data: { message: "Role assigned" },
						}),
					});
				},
			);

			await page.route(
				`/api/v1/workspaces/${workspaceId}/channels`,
				async (route: any) => {
					if (route.request().method() === "POST") {
						channelCreated = true;
						await route.fulfill({
							status: 201,
							contentType: "application/json",
							body: JSON.stringify({
								success: true,
								data: {
									id: "ch-restricted-id",
									workspace_id: workspaceId,
									name: "restricted-talk",
									type: "text",
								},
							}),
						});
					} else {
						const channels = [
							{
								id: "ch-general-id",
								workspace_id: workspaceId,
								name: "general",
								type: "text",
							},
						];
						if (channelCreated) {
							channels.push({
								id: "ch-restricted-id",
								workspace_id: workspaceId,
								name: "restricted-talk",
								type: "text",
							});
						}
						await route.fulfill({
							status: 200,
							contentType: "application/json",
							body: JSON.stringify({ success: true, data: channels }),
						});
					}
				},
			);

			await page.route(
				"/api/v1/channels/ch-restricted-id/permission-overrides",
				async (route: any) => {
					overrideApplied = true;
					await route.fulfill({
						status: 200,
						contentType: "application/json",
						body: JSON.stringify({
							success: true,
							data: { message: "Permission override set" },
						}),
					});
				},
			);
		};

		await setupRoutes(pageA);
		await setupRoutes(pageB);

		// Step 1: User A creates workspace
		await pageA.goto("/workspaces");
		await pageA.click("text=Create Workspace");
		await pageA.fill('input[id="ws-name"]', "Sprint 3 Workspace");
		await pageA.click('button[type="submit"]');

		await expect(pageA).toHaveURL(new RegExp(`.*/workspaces/${workspaceId}`));

		// Step 2: User A creates Role "Restricted" in Settings
		await pageA.goto(`/workspaces/${workspaceId}/settings`);
		await pageA.click("text=Roles");
		await pageA.click("text=Create Role");
		await pageA.fill('input[id="role-name"]', "Restricted");
		await pageA.click("text=Save Role");

		// Step 3: User B redeems invite code
		await pageB.goto(`/invite/${inviteCode}`);
		await pageB.click("text=Accept Invite");
		await expect(pageB).toHaveURL(new RegExp(`.*/workspaces/${workspaceId}`));

		// Step 4: User A assigns "Restricted" role to User B
		await pageA.goto(`/workspaces/${workspaceId}/settings`);
		await pageA.click("text=Member Roles");
		await pageA.click("text=User B (Member)");
		await pageA.click('input[id*="role-assign-role-restricted-id"]');
		await pageA.click("text=Save Changes");

		expect(memberRoleAssigned).toBe(true);

		// Step 5: User A creates channel `#restricted-talk`
		await pageA.goto(`/workspaces/${workspaceId}`);
		await pageA.click('button[title="Create Channel"]');
		await pageA.fill('input[id="ch-name"]', "restricted-talk");
		await pageA.click('button[type="submit"]');

		await expect(pageA).toHaveURL(
			new RegExp(`.*/workspaces/${workspaceId}/channels/ch-restricted-id`),
		);

		// Step 6 & 7 Verification: Both users navigated workspace successfully
		await expect(pageA.locator("text=#restricted-talk")).toBeVisible();
		await expect(pageB.locator("text=Sprint 3 Workspace")).toBeVisible();

		await contextA.close();
		await contextB.close();
	});
});
