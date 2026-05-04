FROM oven/bun:1.3.11

WORKDIR /app

COPY index.ts tsconfig.json config.example.yml ./
COPY dashboard/.output/public ./dashboard/.output/public
COPY src/ ./src/

ENV CONFIG_PATH=/data/config.yml

RUN mkdir -p /data

CMD ["bun", "run", "index.ts"]
