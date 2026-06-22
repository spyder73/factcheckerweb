# Multi-stage Dockerfile for the Alethea web app.
# - `dev` target: vite dev server with HMR (bind-mounted source in compose)
# - `prod` target: static build served by nginx
# Choose at build time via compose `target:`.

FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:20-alpine AS dev
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV HOST=0.0.0.0
EXPOSE 3000
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]

FROM node:20-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM nginx:1.27-alpine AS prod
COPY --from=build /app/dist /usr/share/nginx/html
# SPA: every unknown path serves index.html
RUN printf 'server {\n  listen 80;\n  root /usr/share/nginx/html;\n  index index.html;\n  location / { try_files $uri $uri/ /index.html; }\n  add_header X-Content-Type-Options nosniff;\n  add_header X-Frame-Options DENY;\n  add_header Referrer-Policy no-referrer;\n}\n' > /etc/nginx/conf.d/default.conf
EXPOSE 80
