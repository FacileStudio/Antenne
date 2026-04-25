FROM oven/bun:1.3.11

WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY index.ts tsconfig.json config.example.yml ./
COPY dashboard/ ./dashboard/
COPY src/ ./src/

RUN bun run dashboard:build

ENV CONFIG_PATH=/data/config.yml

RUN mkdir -p /data

CMD ["bun", "run", "index.ts"]
