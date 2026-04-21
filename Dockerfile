FROM oven/bun:1.3.11-alpine

WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile --production

COPY index.ts tsconfig.json config.example.json ./

ENV CONFIG_PATH=/data/config.json
ENV NOOK_STORAGE_PATH=/data/nook-storage.json

CMD ["bun", "run", "index.ts"]
