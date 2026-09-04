<template>
  <div class="bg-gray-800 p-8 rounded-lg shadow-lg w-full max-w-md text-white">
    <h2 class="text-2xl font-bold mb-6 text-center">Create an account</h2>

    <form @submit.prevent="handleRegister" novalidate class="space-y-4">
      <!-- Error Message Banner -->
      <div v-if="errorMessage" class="bg-red-500/10 border border-red-500 text-red-500 p-3 rounded text-sm mb-4">
        {{ errorMessage }}
      </div>

      <!-- Email Field -->
      <div>
        <label for="email" class="block text-sm font-medium text-gray-300 mb-1">Email</label>
        <input
          id="email"
          v-model="form.email"
          type="email"
          required
          class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-white focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
          placeholder="Enter your email"
        />
        <p v-if="validationErrors.email" class="text-red-500 text-xs mt-1">{{ validationErrors.email }}</p>
      </div>

      <!-- Username Field -->
      <div>
        <label for="username" class="block text-sm font-medium text-gray-300 mb-1">Username</label>
        <input
          id="username"
          v-model="form.username"
          type="text"
          required
          class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-white focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
          placeholder="Choose a username"
        />
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
          placeholder="Choose a password"
        />
        <p v-if="validationErrors.password" class="text-red-500 text-xs mt-1">{{ validationErrors.password }}</p>
      </div>

      <!-- Submit Button -->
      <button
        type="submit"
        :disabled="isPending"
        class="w-full bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed mt-6"
      >
        <span v-if="isPending">Registering...</span>
        <span v-else>Register</span>
      </button>

      <p class="text-sm text-gray-400 mt-4 text-center">
        Already have an account?
        <NuxtLink to="/login" class="text-blue-400 hover:underline">Log in here</NuxtLink>
      </p>
    </form>
  </div>
</template>

<script setup lang="ts">
import { useMutation } from "@tanstack/vue-query";
import { computed, reactive, ref } from "vue";
import { navigateTo, useNuxtApp } from "#app";

definePageMeta({
	layout: "auth",
});

const { $api } = useNuxtApp();

const form = reactive({
	email: "",
	username: "",
	password: "",
});

const validationErrors = reactive({
	email: "",
	password: "",
});

const errorMessage = ref("");

const validateForm = () => {
	let isValid = true;
	validationErrors.email = "";
	validationErrors.password = "";

	// Simple client-side validation hints
	if (!form.email.includes("@")) {
		validationErrors.email = "Please enter a valid email address.";
		isValid = false;
	}

	if (form.password.length < 8) {
		validationErrors.password = "Password must be at least 8 characters long.";
		isValid = false;
	}

	return isValid;
};

const { mutate, isPending } = useMutation({
	mutationFn: (data: typeof form) =>
		$api("/auth/register", {
			method: "POST",
			body: data,
		}),
	onSuccess: () => {
		// Redirect to login page upon successful registration
		navigateTo("/login");
	},
	onError: (error: any) => {
		// Display the exact backend error message if available
		const errorData = error.response?._data;
		if (errorData?.error?.message) {
			errorMessage.value = errorData.error.message;
		} else {
			errorMessage.value = "An unexpected error occurred. Please try again.";
		}
	},
});

const handleRegister = () => {
	errorMessage.value = "";

	if (validateForm()) {
		mutate(form);
	}
};
</script>
