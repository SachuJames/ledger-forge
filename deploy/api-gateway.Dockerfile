# ---- build ----
FROM node:24-alpine AS build
WORKDIR /src
COPY api-gateway/package.json api-gateway/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY api-gateway/ ./
RUN npm run build

# ---- run ----
FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY api-gateway/package.json api-gateway/package-lock.json ./
RUN npm ci --omit=dev --no-audit --no-fund
COPY --from=build /src/dist ./dist
EXPOSE 3000
CMD ["node", "dist/main"]
