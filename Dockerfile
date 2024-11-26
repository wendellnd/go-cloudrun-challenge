FROM golang:1.22 as build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o cloudrun

FROM alpine:3.12
WORKDIR /app
COPY --from=build /app/cloudrun .
COPY .env /app/.env
RUN apk add --no-cache ca-certificates
CMD ["./cloudrun"]
