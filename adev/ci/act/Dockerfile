# Runner image for local `act` runs.
# The default act image (node:16-buster-slim) has no CA bundle, no git and no C
# compiler, and buster is end-of-life, so its apt mirrors no longer answer.
# This builds on a supported Debian and adds only what the Go job needs.
FROM node:20-bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        git \
        gcc \
        libc6-dev \
        make \
    && rm -rf /var/lib/apt/lists/*
