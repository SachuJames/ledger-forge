# ---- build ----
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY ledger-core/go.mod ledger-core/go.sum ./
RUN go mod download
COPY ledger-core/ ./
RUN CGO_ENABLED=0 go build -o /ledger-core ./cmd/ledger-core

# ---- run ----
FROM alpine:3.21
RUN adduser -D -h /home/ledger ledger
USER ledger
COPY --from=build /ledger-core /usr/local/bin/ledger-core
EXPOSE 8081
ENTRYPOINT ["ledger-core"]
