FROM golang:1.26.5-bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

ENV CGO_ENABLED=0

CMD ["bash"]
