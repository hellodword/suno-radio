FROM golang:bookworm as builder

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY cmd ./cmd
COPY internal ./internal
RUN go build -x -v -trimpath -ldflags "-s -w" -buildvcs=false -o /usr/local/bin/mp3-to-ogg ./cmd/mp3-to-ogg

FROM linuxserver/ffmpeg:latest

COPY --from=builder /usr/local/bin/mp3-to-ogg /usr/local/bin/mp3-to-ogg

ENTRYPOINT ["/usr/local/bin/mp3-to-ogg"]

HEALTHCHECK --interval=2s --timeout=30s CMD ["/usr/local/bin/mp3-to-ogg", "-healthcheck", "http://127.0.0.1:3001"]
