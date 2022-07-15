FROM golang:1.18.3-alpine3.16 AS builder

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git

# Create appuser.
ENV USER=appuser
ENV UID=10001

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR $GOPATH/src/github.com/marioarizaj/payment_gateway

COPY . .

# Build the binary and make it executable.
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/payment_gateway ./cmd/payment_gateway/main.go
RUN chmod +x /go/bin/payment_gateway

FROM scratch
# Import the user and group files from the builder.
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
# Copy our static executable.
COPY --from=builder /go/bin/payment_gateway /go/bin/payment_gateway
# Use an unprivileged user.
USER appuser:appuser

# Run the hello binary.
ENTRYPOINT ["/go/bin/payment_gateway"]