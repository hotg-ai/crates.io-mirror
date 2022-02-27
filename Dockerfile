FROM golang:1.17 AS build

RUN mkdir /app
WORKDIR /app
COPY ./go.* /app/
RUN go mod download

COPY . /app/
RUN go build -o /app/crates-io-mirror

FROM ubuntu:latest

COPY --from=build /app/crates-io-mirror /bin/crates-io-mirror

ENV PORT=8000 HOST=0.0.0.0 UPSTREAM=https://crates.io/ S3_BUCKET=
EXPOSE 8000

ENTRYPOINT [ "/bin/crates-io-mirror" ]
