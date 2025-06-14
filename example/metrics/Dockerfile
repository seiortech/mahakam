FROM golang:1.24.3-alpine AS build

WORKDIR /app

COPY go.mod ./

RUN go mod download && go mod verify

COPY . .

RUN go build -o /server example/metrics/cmd/main.go

FROM alpine

WORKDIR /app
RUN mkdir -p /app/logs

COPY --from=build /server /app/server

ENTRYPOINT ["/app/server"]