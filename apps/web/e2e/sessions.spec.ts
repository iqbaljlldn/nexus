import { expect, test } from "@playwright/test";

test.describe("Device & Session Management", () => {
	test.beforeEach(async ({ page }) => {
		// Navigate to sessions page
		await page.goto("/settings/sessions");
	});

	test("displays list of active sessions", async ({ page }) => {
		// Mock the GET sessions endpoint
		await page.route("/api/v1/auth/sessions", async (route) => {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: [
						{
							id: "session_1",
							user_agent: "Mozilla Firefox",
							ip_address: "192.168.1.1",
							created_at: "2026-09-04T12:00:00Z",
						},
						{
							id: "session_2",
							user_agent: "Chrome Mobile",
							ip_address: "10.0.0.5",
							created_at: "2026-09-03T15:30:00Z",
						},
					],
				}),
			});
		});

		// Reload to ensure mock applies
		await page.reload();

		// Verify sessions are rendered
		await expect(page.locator("text=Mozilla Firefox")).toBeVisible();
		await expect(page.locator("text=192.168.1.1")).toBeVisible();
		await expect(page.locator("text=Chrome Mobile")).toBeVisible();
	});

	test("logout from this device redirects to login", async ({ page }) => {
		// Setup initial sessions data to satisfy page load
		await page.route("/api/v1/auth/sessions", async (route) => {
			await route.fulfill({
				status: 200,
				body: JSON.stringify({ success: true, data: [] }),
			});
		});

		// Mock logout endpoint
		await page.route("/api/v1/auth/logout", async (route) => {
			await route.fulfill({ status: 204 });
		});

		// Accept any native confirm dialogues
		page.on("dialog", (dialog) => dialog.accept());

		// Click logout this device
		await page.click('button:has-text("Logout (This Device)")');

		// Verify redirect
		await expect(page).toHaveURL(/.*\/login/);
	});

	test("logout from all devices redirects to login", async ({ page }) => {
		// Setup initial sessions data
		await page.route("/api/v1/auth/sessions", async (route) => {
			await route.fulfill({
				status: 200,
				body: JSON.stringify({ success: true, data: [] }),
			});
		});

		// Mock logout-all endpoint
		await page.route("/api/v1/auth/logout-all", async (route) => {
			await route.fulfill({ status: 204 });
		});

		// Accept native confirm dialogues
		page.on("dialog", (dialog) => dialog.accept());

		// Click logout all devices
		await page.click('button:has-text("Logout from All Devices")');

		// Verify redirect
		await expect(page).toHaveURL(/.*\/login/);
	});
});
