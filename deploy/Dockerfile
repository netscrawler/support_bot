# Только runtime-стадия: бинарник уже собран локально через `task docker-run`
# (тем же build.Dockerfile, что и раньше) — здесь Go не компилируется, поэтому
# рассогласование версий Go между Dockerfile/build.Dockerfile/go.mod образ не касается.
FROM debian:12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    wkhtmltopdf \
    ca-certificates \
    fonts-dejavu \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -r bot && useradd -r -g bot -d /sbot bot

WORKDIR /sbot

COPY bin/sbot .

RUN chown -R bot:bot /sbot

USER bot

EXPOSE 8080

ENTRYPOINT ["./sbot"]
