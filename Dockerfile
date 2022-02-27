FROM golang:1.17 AS build

RUN mkdir /app
WORKDIR /app
COPY ./go.* /app/
RUN go mod download

COPY . /app/
RUN go build -o /app/crates-io-proxy

FROM ubuntu:latest

COPY --from=build /app/crates-io-proxy /bin/crates-io-proxy

ENV PORT=8000 HOST=0.0.0.0 UPSTREAM=https://crates.io/
EXPOSE 8000

ENTRYPOINT [ "/bin/crates-io-proxy" ]
