FROM golang:1.18-alpine3.14 as builder

RUN apk add --no-cache build-base \
    git gcc musl-dev

WORKDIR /dist
COPY . .

# Install the package
RUN go mod download \
    && go build -tags musl -o go_aibiz_server .


FROM alpine:3.14
# This container exposes port 8000 to the outside world
RUN apk add --no-cache gcc

ENV PORT 8000
WORKDIR /app
COPY --from=builder /dist/go_aibiz_server .
COPY --from=builder /dist/go.mod .
COPY --from=builder /dist/go.sum .
COPY --from=builder /dist/.env .
RUN mkdir /data/ \
    && mkdir /hana/ \
    && mkdir /dbhandle/
COPY --from=builder /dist/data /app/data/
# COPY --from=builder /dist/hana /app/hana/attachment.png
COPY --from=builder /dist/dbhandle /app/dbhandle/

# Run the executable
CMD ["./go_aibiz_server"]