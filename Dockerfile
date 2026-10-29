FROM debian:trixie-slim AS base
CMD ["/yukid"]

FROM base AS multiarch
ARG TARGETARCH
COPY ./dist/docker/${TARGETARCH}/yukid /yukid

# Keep local builds using the binary produced by make.
FROM base AS default
COPY ./yukid /yukid
