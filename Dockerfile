FROM debian:bookworm-slim
COPY ./yukid /yukid
CMD ["/yukid"]
