# Multi-stage Dockerfile for the Alethea web app.
# - `dev` target: vite dev server with HMR (bind-mounted source in compose).
#                 Re-syncs node_modules from package-lock.json on every start
#                 so adding a dep is "edit package.json, restart" — no rebuild.
# - `prod` target: static build served by nginx.

FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

FROM node:20-alpine AS dev
WORKDIR /app
RUN apk add --no-cache bash
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV HOST=0.0.0.0 \
    CHOKIDAR_USEPOLLING=true \
    npm_config_loglevel=warn
EXPOSE 3000
# Sync node_modules against package-lock.json on boot so adding a dep doesn't
# require a Docker rebuild. The hash of the lockfile is stored at
# /app/node_modules/.lockfile-hash; if it changes, we re-run npm ci.
# `npm ci` against the prefer-offline cache is fast (~5s) after the first run.
CMD ["sh", "-c", "\
  LOCK_HASH=$(sha256sum package-lock.json | cut -d' ' -f1); \
  PREV=$(cat /app/node_modules/.lockfile-hash 2>/dev/null || echo ''); \
  if [ \"$LOCK_HASH\" != \"$PREV\" ]; then \
    echo '[dev] package-lock.json changed — running npm ci'; \
    npm ci --no-audit --no-fund --prefer-offline; \
    echo \"$LOCK_HASH\" > /app/node_modules/.lockfile-hash; \
  fi; \
  exec npm run dev -- --host 0.0.0.0 \
"]

FROM node:20-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM nginx:1.27-alpine AS prod
COPY --from=build /app/dist /usr/share/nginx/html
RUN printf 'server {\n  listen 80;\n  root /usr/share/nginx/html;\n  index index.html;\n  location / { try_files $uri $uri/ /index.html; }\n  add_header X-Content-Type-Options nosniff;\n  add_header X-Frame-Options DENY;\n  add_header Referrer-Policy no-referrer;\n}\n' > /etc/nginx/conf.d/default.conf
EXPOSE 80
