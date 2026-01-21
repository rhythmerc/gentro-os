FROM rust:slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    pkg-config \
    ca-certificates \
  && rm -rf /var/lib/apt/lists/*

RUN rustup default stable

WORKDIR /workspace/launcher-core

ENV GENTRO_DATA_DIR=/workspace/.gentro/data
ENV GENTRO_LOG_PATH=/workspace/.gentro/logs/gentro-core.log
ENV GENTRO_SOCKET_PATH=/workspace/.gentro/run/launcher.sock

CMD ["cargo", "run"]
