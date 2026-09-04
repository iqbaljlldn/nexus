import { expect, test } from "@playwright/test";

test.describe("Register Page", () => {
	test.beforeEach(async ({ page }) => {
		// Navigate to the register page before each test
		await page.goto("/register");
	});

	test("successfully registers and redirects to login", async ({ page }) => {
		// Mock the backend API response for successful registration
		await page.route("/api/v1/auth/register", async (route) => {
			await route.fulfill({
				status: 201,
				contentType: "application/json",
				body: JSON.stringify({
					success: true,
					data: {
						user: {
							id: "user_1",
							username: "testuser",
							email: "test@example.com",
						},
					},
				}),
			});
		});

		// Fill out the form
		await page.fill('input[id="email"]', "test@example.com");
		await page.fill('input[id="username"]', "testuser");
		await page.fill('input[id="password"]', "securepassword123");

		// Submit form
		await page.click('button[type="submit"]');

		// Verify redirect to login page
		await expect(page).toHaveURL(/.*\/login/);
	});

	test("displays exact backend error for duplicate email", async ({ page }) => {
		const backendErrorMessage =
			"Email is already registered. Please login or use a different email.";

		// Mock the backend API response for duplicate email error
		await page.route("/api/v1/auth/register", async (route) => {
			await route.fulfill({
				status: 409,
				contentType: "application/json",
				body: JSON.stringify({
					success: false,
					error: {
						code: "CONFLICT",
						message: backendErrorMessage,
					},
				}),
			});
		});

		// Fill out the form
		await page.fill('input[id="email"]', "existing@example.com");
		await page.fill('input[id="username"]', "existinguser");
		await page.fill('input[id="password"]', "securepassword123");

		// Submit form
		await page.click('button[type="submit"]');

		// Verify error message is displayed exactly as returned from backend
		const errorBanner = page.locator(".bg-red-500\\/10");
		await expect(errorBanner).toBeVisible();
		await expect(errorBanner).toContainText(backendErrorMessage);

		// Should stay on the register page
		await expect(page).toHaveURL(/.*\/register/);
	});

	test("displays client-side validation hints", async ({ page }) => {
		// Fill out the form with invalid data
		await page.fill('input[id="email"]', "invalid-email");
		await page.fill('input[id="username"]', "user");
		await page.fill('input[id="password"]', "short");

		// Note: since the email input has type="email", HTML5 validation might kick in first,
		// but Vue's `@submit.prevent` usually overrides default submission behavior in Nuxt/Vue,
		// allowing our manual validation logic to run. If HTML5 blocks it, Playwright might not trigger the click properly.
		// For this test, we verify that our custom error hint is shown when we try to submit.

		// Playwright force click in case HTML5 validation overlay pops up
		await page.click('button[type="submit"]', { force: true });

		// Verify client-side error hints are displayed
		await expect(
			page.locator("text=Please enter a valid email address."),
		).toBeVisible();
		await expect(
			page.locator("text=Password must be at least 8 characters long."),
		).toBeVisible();
	});
});
