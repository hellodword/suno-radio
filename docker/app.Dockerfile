FROM node:bullseye as frontend

WORKDIR /usr/src/frontend

COPY frontend .

RUN npm i && npm run build

FROM golang:bookworm as builder

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY cmd ./cmd
COPY internal ./internal
COPY --from=frontend /usr/src/frontend ./frontend

RUN go build -x -v -trimpath -ldflags "-s -w" -buildvcs=false -o /usr/local/bin/suno-radio ./cmd/suno-radio

FROM gcr.io/distroless/base-debian12:nonroot

COPY --from=builder /usr/local/bin/suno-radio /usr/local/bin/suno-radio

ENTRYPOINT ["/usr/local/bin/suno-radio", "-config", "/home/nonroot/suno-radio/server.yml"]

HEALTHCHECK --interval=2s --timeout=30s CMD ["/usr/local/bin/suno-radio", "-healthcheck", "http://127.0.0.1:3000,http://127.0.0.1:7890"]
