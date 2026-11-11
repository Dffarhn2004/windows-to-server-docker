FROM golang:1.23-alpine AS build
WORKDIR /src
COPY pkg ./pkg
COPY services/api ./services/api
WORKDIR /src/services/api
RUN go mod download && go build -o /out/hcc-api ./cmd/api

FROM alpine:3.20
RUN adduser -D -u 10001 hcc && mkdir -p /data && chown hcc:hcc /data
USER hcc
WORKDIR /data
COPY --from=build /out/hcc-api /usr/local/bin/hcc-api
EXPOSE 8080
ENTRYPOINT ["hcc-api"]
