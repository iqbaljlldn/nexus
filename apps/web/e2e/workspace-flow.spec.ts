import { expect, test } from "@playwright/test";

test.describe("Workspace & Channel Flow", () => {
	test("creates a workspace and navigates automatically", async ({ page }) => {
		// Mock initial empty workspace list
		await page.route("/api/v1/workspaces", async (route) => {
			if (route.request().method() === "GET") {
				await route.fulfill({
					status: 200,
					contentType: "application/json",
					body: JSON.stringify({
						success: true,
						data: { workspaces: [] },
					}),
				});
			} else if (route.request().method() === "POST") {
				await route.fulfill({
					status: 201,
					contentType: "application/json",
					body: JSON.stringify({
						success: true,
						data: {
							id: "ws-111",
							name: "Engineering Core",
							owner_id: "user-1",
						},
					}),
				});
			}
		});

		// Mock channels list for newly created workspace
		await page.route("/api/v1/workspaces/ws-111/channels", async (route) => {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: [
						{
							id: "ch-general",
							workspace_id: "ws-111",
							name: "general",
							type: "text",
							position: 1,
						},
					],
				}),
			});
		});

		await page.goto("/workspaces");

		// Click Create Workspace button
		await page.click("text=Create Workspace");

		// Fill in modal
		await page.fill('input[id="ws-name"]', "Engineering Core");

		// Submit
		await page.click('button[type="submit"]');

		// Verify navigation to workspace and channel
		await expect(page).toHaveURL(/.*\/workspaces\/ws-111/);
	});

	test("creates a channel and selects it", async ({ page }) => {
		let channelCreated = false;

		await page.route("/api/v1/workspaces", async (route) => {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: {
						workspaces: [
							{
								id: "ws-111",
								name: "Engineering Core",
								owner_id: "user-1",
							},
						],
					},
				}),
			});
		});

		await page.route("/api/v1/workspaces/ws-111/channels", async (route) => {
			if (route.request().method() === "GET") {
				const channels = [
					{
						id: "ch-general",
						workspace_id: "ws-111",
						name: "general",
						type: "text",
					},
				];
				if (channelCreated) {
					channels.push({
						id: "ch-devs",
						workspace_id: "ws-111",
						name: "devs",
						type: "text",
					});
				}
				await route.fulfill({
					status: 200,
					contentType: "application/json",
					body: JSON.stringify({ success: true, data: channels }),
				});
			} else if (route.request().method() === "POST") {
				channelCreated = true;
				await route.fulfill({
					status: 201,
					contentType: "application/json",
					body: JSON.stringify({
						success: true,
						data: {
							id: "ch-devs",
							workspace_id: "ws-111",
							name: "devs",
							type: "text",
						},
					}),
				});
			}
		});

		await page.goto("/workspaces/ws-111");

		// Click Create Channel (+) icon
		await page.click('button[title="Create Channel"]');

		// Fill channel name
		await page.fill('input[id="ch-name"]', "devs");

		// Submit modal
		await page.click('button[type="submit"]');

		// Verify channel devs created and selected
		await expect(page).toHaveURL(/.*\/workspaces\/ws-111\/channels\/ch-devs/);
	});
});
