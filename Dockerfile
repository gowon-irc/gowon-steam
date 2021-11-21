FROM golang:alpine as build-env
COPY . /src
WORKDIR /src
RUN go build -o gowon-steam

FROM alpine:3.14.2
WORKDIR /app
COPY --from=build-env /src/gowon-steam /app/
ENTRYPOINT ["./gowon-steam"]
