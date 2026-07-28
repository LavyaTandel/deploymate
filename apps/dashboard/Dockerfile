# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

RUN npm run build

# Production stage
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV production

# Install wget for health checks
RUN apk add --no-cache wget

# Copy the entire standalone output
COPY --from=builder /app/.next/standalone ./

# Copy static assets
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000

HEALTHCHECK --interval=5s --timeout=5s --retries=5 --start-period=30s \
  CMD wget -qO- http://localhost:3000/ || exit 1

# Run the standalone server
CMD ["node", ".next/standalone/server.js"]
