FROM golang:latest AS build
WORKDIR /app
COPY . .

ENV GOPROXY=proxy.golang.org

RUN apt-get update && apt-get install -y \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

RUN cp .env.example .env
RUN go mod download
RUN go build -o /app/mattermost_bot ./cmd/main.go

FROM debian:stable-slim AS run
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /app/mattermost_bot /app/mattermost_bot
COPY --from=build /app/.env /app/.env

CMD ["/app/mattermost_bot"]