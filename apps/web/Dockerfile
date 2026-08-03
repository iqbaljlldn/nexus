FROM node:24.11.1-alpine AS builder

# Set working directory
WORKDIR /app

# Enable pnpm
RUN corepack enable pnpm

# Copy package.json to install dependencies
COPY package.json ./

# Install dependencies
RUN pnpm install

# Copy all files
COPY . .

# Build the Nuxt application
RUN pnpm build

# Runner stage
FROM node:24.11.1-alpine AS runner

WORKDIR /app

# Copy the standalone output from the builder stage
COPY --from=builder /app/.output ./.output

EXPOSE 3000

# Set required environment variables
ENV PORT=3000
ENV HOST=0.0.0.0
ENV NODE_ENV=production

# Start the standalone server
CMD ["node", ".output/server/index.mjs"]
