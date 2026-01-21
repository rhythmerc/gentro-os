FROM rust:1.78-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    pkg-config \
    ca-certificates \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace/launcher-core

ENV CARGO_HOME=/cargo
ENV RUSTUP_HOME=/rustup
ENV GENTRO_DATA_DIR=/workspace/.gentro/data
ENV GENTRO_LOG_PATH=/workspace/.gentro/logs/gentro-core.log
ENV GENTRO_SOCKET_PATH=/workspace/.gentro/run/launcher.sock

CMD ["cargo", "run"]
