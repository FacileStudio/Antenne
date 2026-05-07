FROM oven/bun:1 AS client-build

WORKDIR /build/apps/client

COPY apps/client/package.json ./
RUN bun install

COPY apps/client ./

RUN bunx nuxi generate

FROM oven/bun:1

WORKDIR /app

COPY apps/api/package.json ./
RUN bun install --production

COPY apps/api ./
COPY --from=client-build /build/apps/api/dashboard/.output/public ./dashboard/.output/public

ENV NOOK_SETTINGS_PATH=/data/nook.json
VOLUME ["/data"]

CMD ["bun", "run", "index.ts"]
