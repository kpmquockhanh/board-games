# Native replacement for coleifer/sqlite-web, which is published for
# linux/amd64 only and therefore runs emulated on Apple Silicon.
FROM python:3.12-slim

RUN pip install --no-cache-dir sqlite-web

ENV SQLITE_DATABASE=/data/ping.db
EXPOSE 8080

# -x keeps it from trying to open a browser inside the container.
CMD ["sh", "-c", "exec sqlite_web -H 0.0.0.0 -p 8080 -x \"$SQLITE_DATABASE\""]
