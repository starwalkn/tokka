# -------- СТАДИЯ СБОРКИ --------
FROM node:18 AS builder

WORKDIR /app

COPY /Users/alexander/Documents/Projects/tokka/dashboard/static/package.json /Users/alexander/Documents/Projects/tokka/dashboard/static/tsconfig.json ./

RUN npm install

COPY /Users/alexander/Documents/Projects/tokka/dashboard .

RUN npm run build

FROM node:18-slim

WORKDIR /app

COPY --from=builder /app/dist ./dist
COPY /Users/alexander/Documents/Projects/tokka/dashboard/static/package.json ./

RUN npm install --omit=dev || true

# Запускаем скомпилированный dashboard
CMD ["node", "dist/app.js"]
