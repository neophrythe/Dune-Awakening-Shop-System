# syntax=docker/dockerfile:1
# Multi-stage build: compile the React dashboard, embed it, build a static Go
# binary, and ship it on a small image that includes the docker CLI (needed for
# RMQ in-game delivery, which execs rabbitmqctl in the AMP container).

# --- 1. dashboard ---------------------------------------------------------
FROM node:22-alpine AS ui
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# --- 2. binary ------------------------------------------------------------
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui /app/web/dist ./internal/web/dist
ARG VERSION=docker
RUN CGO_ENABLED=0 go build -tags embed \
      -ldflags "-s -w -X main.Version=${VERSION}" \
      -o /out/dune-shop ./cmd/dune-shop

# --- 3. runtime -----------------------------------------------------------
FROM docker:27-cli
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/dune-shop /usr/local/bin/dune-shop
COPY --from=build /src/seed /app/seed
EXPOSE 8090 8091
ENTRYPOINT ["dune-shop"]
CMD ["-config", "/config/config.yaml"]
