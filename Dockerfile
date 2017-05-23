FROM golang:1.7

# Copy the runtime dockerfile into the context as Dockerfile
COPY Dockerfile.run /go/bin/Dockerfile
COPY ./www /go/bin/www

COPY . /go/src/github.com/play-with-docker/play-with-docker

WORKDIR /go/src/github.com/play-with-docker/play-with-docker

RUN go get -v -d ./...

RUN CGO_ENABLED=0 go build -a -installsuffix nocgo -o /go/bin/play-with-docker .

FROM alpine

RUN apk --update add ca-certificates
RUN mkdir -p /app/pwd

COPY --from=0 /go/bin/play-with-docker /app/play-with-docker
COPY ./www /app/www

WORKDIR /app
CMD ["./play-with-docker"]

EXPOSE 3000
