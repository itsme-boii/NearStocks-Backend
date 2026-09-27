FROM --platform=linux/amd64 public.ecr.aws/docker/library/golang:1.24.0-alpine AS build

WORKDIR /app

COPY . .
RUN rm -rf ./build

RUN go mod download
RUN go build -o ./build/main ./services/indexer-server/cmd/main.go

FROM --platform=linux/amd64 public.ecr.aws/l8t0j4y9/alpine:latest

WORKDIR /app
COPY --from=build ./app/build/main .
EXPOSE 8080

CMD ["./main"]