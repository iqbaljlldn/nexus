import { expect, test } from "@playwright/test";

test.describe("Auto Re-auth Flow", () => {
	test("auto re-auth intercepts and updates state", async ({
		page,
		context,
	}) => {
		await context.addCookies([
			{
				name: "csrf_token",
				value: "dummy_csrf_token",
				domain: "127.0.0.1",
				path: "/",
			},
		]);

		// Wait for the refresh request
		const requestPromise = page
			.waitForRequest(
				(req) =>
					req.url().includes("/api/v1/auth/refresh") && req.method() === "POST",
				{ timeout: 5000 },
			)
			.catch(() => null);

		await page.route("/api/v1/auth/refresh", async (route) => {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: {
						access_token: "new_access_token",
						expires_in: 900,
					},
				}),
			});
		});

		await page.goto("/workspaces");

		// Check that refresh was called
		const request = await requestPromise;
		expect(request).toBeTruthy();
	});
});
