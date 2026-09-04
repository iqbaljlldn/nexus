<template>
  <div class="bg-gray-800 p-8 rounded-lg shadow-lg w-full max-w-md text-white">
    <h2 class="text-2xl font-bold mb-6 text-center">Welcome back</h2>

    <form @submit.prevent="handleLogin" novalidate class="space-y-4">
      <!-- Error Message Banner -->
      <div v-if="errorMessage" class="bg-red-500/10 border border-red-500 text-red-500 p-3 rounded text-sm mb-4">
        {{ errorMessage }}
      </div>

      <!-- Identifier Field -->
      <div>
        <label for="identifier" class="block text-sm font-medium text-gray-300 mb-1">Email or Username</label>
        <input
          id="identifier"
          v-model="form.identifier"
          type="text"
          required
          class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-white focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
          placeholder="Enter your email or username"
        />
        <p v-if="validationErrors.identifier" class="text-red-500 text-xs mt-1">{{ validationErrors.identifier }}</p>
      </div>

      <!-- Password Field -->
      <div>
        <label for="password" class="block text-sm font-medium text-gray-300 mb-1">Password</label>
        <input
          id="password"
          v-model="form.password"
          type="password"
          required
          class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-white focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
          placeholder="Enter your password"
        />
        <p v-if="validationErrors.password" class="text-red-500 text-xs mt-1">{{ validationErrors.password }}</p>
      </div>

      <!-- Submit Button -->
      <button
        type="submit"
        :disabled="isPending"
        class="w-full bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed mt-6"
      >
        <span v-if="isPending">Logging in...</span>
        <span v-else>Log In</span>
      </button>

      <p class="text-sm text-gray-400 mt-4 text-center">
        Need an account?
        <NuxtLink to="/register" class="text-blue-400 hover:underline">Register here</NuxtLink>
      </p>
    </form>
  </div>
</template>

<script setup lang="ts">
import { useMutation } from "@tanstack/vue-query";
import { jwtDecode } from "jwt-decode";
import { reactive, ref } from "vue";
import { navigateTo, useNuxtApp } from "#app";
import { useSessionStore } from "../stores/session";

definePageMeta({
	layout: "auth",
});

const { $api } = useNuxtApp();
const session = useSessionStore();

const form = reactive({
	identifier: "",
	password: "",
});

const validationErrors = reactive({
	identifier: "",
	password: "",
});

const errorMessage = ref("");

const validateForm = () => {
	let isValid = true;
	validationErrors.identifier = "";
	validationErrors.password = "";

	if (!form.identifier) {
		validationErrors.identifier = "Email or username is required.";
		isValid = false;
	}

	if (!form.password) {
		validationErrors.password = "Password is required.";
		isValid = false;
	}

	return isValid;
};

const { mutate, isPending } = useMutation({
	mutationFn: (data: typeof form) =>
		$api("/auth/login", {
			method: "POST",
			body: data,
		}),
	onSuccess: (data: any) => {
		// Decode token to extract user ID since API doesn't return user details
		let decodedUser = {
			id: "",
			username: "Unknown User", // Mocked until /users/me is implemented
			email: "unknown@example.com", // Mocked until /users/me is implemented
		};

		try {
			const decoded: any = jwtDecode(data.access_token);
			decodedUser.id = decoded.sub || decoded.id || "";
		} catch (e) {
			console.warn("Failed to decode token", e);
		}

		// Update session store
		session.setAccessToken(data.access_token);
		session.setUser(decodedUser);

		// Redirect to workspaces placeholder page
		navigateTo("/workspaces");
	},
	onError: (error: any) => {
		// Per FR-AUTH-03, use generic errors for login
		const errorData = error.response?._data;
		if (errorData?.error?.message) {
			errorMessage.value = errorData.error.message;
		} else {
			errorMessage.value =
				"Invalid email/username or password. Please try again.";
		}
	},
});

const handleLogin = () => {
	errorMessage.value = "";

	if (validateForm()) {
		mutate(form);
	}
};
</script>
