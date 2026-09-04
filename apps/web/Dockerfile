FROM node:24.11.1-alpine AS builder

# Set working directory
WORKDIR /app

# Enable pnpm
RUN corepack enable pnpm

# Copy package.json to install dependencies
COPY package.json ./

# Install dependencies (increase timeout for large binaries like biome)
RUN pnpm config set fetch-timeout 600000 && pnpm config set fetch-retries 5 && pnpm install --ignore-scripts

# Copy all files
COPY . .

# Run postinstall scripts now that files are copied
RUN pnpm run postinstall

# Build the Nuxt application
RUN pnpm build

# Runner stage
FROM node:24.11.1-alpine AS runner

WORKDIR /app

# Copy the standalone output from the builder stage
COPY --from=builder /app/.output ./.output

EXPOSE 3001

# Set required environment variables
ENV PORT=3001
ENV HOST=0.0.0.0
ENV NODE_ENV=production

# Start the standalone server
CMD ["node", ".output/server/index.mjs"]
