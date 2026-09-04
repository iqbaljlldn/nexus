import { expect, test } from "@playwright/test";

test.describe("Invite Flow", () => {
	test("generates an invite link in settings", async ({ page }) => {
		await page.route("/api/v1/workspaces", async (route) => {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: {
						workspaces: [
							{ id: "ws-111", name: "Engineering Core", owner_id: "user-1" },
						],
					},
				}),
			});
		});

		await page.route("/api/v1/workspaces/ws-111/channels", async (route) => {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: [{ id: "ch-1", name: "general", type: "text" }],
				}),
			});
		});

		await page.route("/api/v1/workspaces/ws-111/invites", async (route) => {
			await route.fulfill({
				status: 201,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: {
						code: "INV123",
						url: "http://127.0.0.1:3002/invite/INV123",
					},
				}),
			});
		});

		await page.goto("/workspaces/ws-111/settings");

		// Click Create Invite button
		await page.click("text=Create Invite");

		// Submit invite creation modal
		await page.click('button:has-text("Generate Link")');

		// Verify returned URL is displayed in input
		await expect(
			page.locator('input[value="http://127.0.0.1:3002/invite/INV123"]'),
		).toBeVisible();
	});

	test("redeems an invite code and redirects to workspace", async ({
		page,
	}) => {
		await page.route("/api/v1/invites/INV123/redeem", async (route) => {
			// Verify Header Idempotency-Key is present
			const headers = route.request().headers();
			expect(headers["idempotency-key"]).toBeTruthy();

			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: {
						workspace_id: "ws-111",
						member_id: "mb-999",
					},
				}),
			});
		});

		await page.route("/api/v1/workspaces", async (route) => {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: {
						workspaces: [
							{ id: "ws-111", name: "Engineering Core", owner_id: "user-1" },
						],
					},
				}),
			});
		});

		await page.route("/api/v1/workspaces/ws-111/channels", async (route) => {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: [{ id: "ch-1", name: "general", type: "text" }],
				}),
			});
		});

		await page.goto("/invite/INV123");

		// Click Accept Invite
		await page.click("text=Accept Invite");

		// Verify redirect to workspace
		await expect(page).toHaveURL(/.*\/workspaces\/ws-111/);
	});
});
