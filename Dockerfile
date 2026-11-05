FROM golang:1.25.3-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# distroless/static: только бинарник, корневые сертификаты и пользователь nonroot, без оболочки.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /server /server
USER nonroot:nonroot
# 8080 — MCP и пробы /healthz, /readyz; 9090 — /metrics (не публикуйте наружу).
EXPOSE 8080 9090
ENTRYPOINT ["/server"]
