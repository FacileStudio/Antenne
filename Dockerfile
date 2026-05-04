FROM oven/bun:1.3.11 AS builder

WORKDIR /app/dashboard
COPY dashboard/package.json dashboard/bun.lock* ./
RUN bun install
COPY dashboard/ ./
RUN bunx nuxi generate

FROM oven/bun:1.3.11

WORKDIR /app

COPY index.ts tsconfig.json config.example.yml ./
COPY src/ ./src/
COPY --from=builder /app/dashboard/.output/public ./dashboard/.output/public

ENV CONFIG_PATH=/data/config.yml

RUN mkdir -p /data

CMD ["bun", "run", "index.ts"]
