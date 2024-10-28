# Dockerfile
FROM --platform=$BUILDPLATFORM golang:1.22 as base

RUN adduser \
  --disabled-password \
  --gecos "" \
  --home "/nonexistent" \
  --shell "/sbin/nologin" \
  --no-create-home \
  --uid 65532 \
  script

USER script

WORKDIR /app/

COPY . .

RUN go mod download
RUN go mod verify

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH GOCACHE=/tmp/go-cache go build -buildvcs=false -o ./exporter .

FROM scratch

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base /app/exporter .

# Since we're using scratch, there's no shell or user management. So, the process will run as the root user.
# However, since scratch doesn't contain any OS files, there's minimal risk.
CMD ["/exporter"]
