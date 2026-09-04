import { expect, test } from "@playwright/test";

test.describe("Login Page", () => {
	test.beforeEach(async ({ page }) => {
		// Navigate to the login page before each test
		await page.goto("/login");
	});

	test("successfully logs in and redirects to workspaces", async ({ page }) => {
		// Mock the backend API response for successful login
		await page.route("/api/v1/auth/login", async (route) => {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: {
						access_token:
							"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyXzEiLCJuYW1lIjoiSm9obiBEb2UiLCJpYXQiOjE1MTYyMzkwMjJ9.mocked_signature",
						expires_in: 900,
					},
				}),
			});
		});

		// Fill out the form
		await page.fill('input[id="identifier"]', "testuser");
		await page.fill('input[id="password"]', "securepassword123");

		// Submit form
		await page.click('button[type="submit"]');

		// Verify redirect to workspaces page
		await expect(page).toHaveURL(/.*\/workspaces/);
	});

	test("displays generic error message for invalid credentials", async ({
		page,
	}) => {
		const backendErrorMessage = "Invalid credentials";

		// Mock the backend API response for invalid credentials
		await page.route("/api/v1/auth/login", async (route) => {
			await route.fulfill({
				status: 401,
				contentType: "application/json",
				body: JSON.stringify({
					success: false,
					error: {
						code: "UNAUTHORIZED",
						message: backendErrorMessage,
					},
				}),
			});
		});

		// Fill out the form
		await page.fill('input[id="identifier"]', "wronguser");
		await page.fill('input[id="password"]', "wrongpass");

		// Submit form
		await page.click('button[type="submit"]');

		// Verify error message is displayed generic
		const errorBanner = page.locator(".bg-red-500\\/10");
		await expect(errorBanner).toBeVisible();
		await expect(errorBanner).toContainText(backendErrorMessage);

		// Should stay on the login page
		await expect(page).toHaveURL(/.*\/login/);
	});
});
