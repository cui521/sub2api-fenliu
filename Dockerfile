FROM golang:1.24-alpine AS build

WORKDIR /src
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/account-auto-dispatch ./cmd/aad

FROM alpine:3.20

RUN adduser -D -H -u 10001 app
WORKDIR /app
COPY --from=build /out/account-auto-dispatch /usr/local/bin/account-auto-dispatch
USER app

ENV AAD_ADDR=0.0.0.0:18080
EXPOSE 18080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:18080/healthz >/dev/null || exit 1

CMD ["/usr/local/bin/account-auto-dispatch"]
