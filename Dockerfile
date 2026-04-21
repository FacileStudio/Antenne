FROM oven/bun:1.3.11

WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile --production
RUN bun ./node_modules/@matrix-org/matrix-sdk-crypto-nodejs/download-lib.js
RUN test -f ./node_modules/@matrix-org/matrix-sdk-crypto-nodejs/matrix-sdk-crypto.linux-x64-gnu.node

COPY index.ts tsconfig.json config.example.json ./

ENV CONFIG_PATH=/data/config.json
ENV NOOK_STORAGE_PATH=/data/nook-storage.json

CMD ["bun", "run", "index.ts"]
