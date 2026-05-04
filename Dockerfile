# syntax=docker/dockerfile:1
FROM oven/bun:1.3.11 AS builder

WORKDIR /app
COPY package.json bun.lock ./
RUN --network=host bun install
COPY dashboard/ ./dashboard/
RUN bunx nuxi generate dashboard

FROM oven/bun:1.3.11

WORKDIR /app

COPY index.ts tsconfig.json config.example.yml ./
COPY src/ ./src/
COPY --from=builder /app/dashboard/.output/public ./dashboard/.output/public

ENV CONFIG_PATH=/data/config.yml

RUN mkdir -p /data

CMD ["bun", "run", "index.ts"]
