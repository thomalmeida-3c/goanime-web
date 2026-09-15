# Build stage: only cmd/server is compiled (no CGO — confirmed the web
# binary doesn't pull in mattn/go-sqlite3, which is only used by the CLI's
# local watch-history feature, not by anything cmd/server imports).
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

# Runtime stage: static binary + CA certs (needed for the HTTPS scraping/
# proxying to Goyabu/SuperFlix/Blogger/MangaLivre/AniList/Mercado Livre).
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/server /server

ENV ADDR=:8080
ENV DATA_DIR=/data
EXPOSE 8080

ENTRYPOINT ["/server"]
