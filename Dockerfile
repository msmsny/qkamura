FROM golang:1.20 as builder

ENV LANG ja_JP.UTF-8
ENV CGO_ENABLED 0

WORKDIR /app
COPY . .
RUN go install -ldflags '-w -s' -trimpath .

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /go/bin/qkamura /go/bin/
