FROM golang:1.9

COPY . /go/src/github.com/play-with-docker/play-with-docker

WORKDIR /go/src/github.com/play-with-docker/play-with-docker

RUN ssh-keygen -N "" -t rsa -f /etc/ssh/ssh_host_rsa_key >/dev/null

RUN CGO_ENABLED=0 go build -a -installsuffix nocgo -o /go/bin/play-with-docker .


FROM alpine

RUN apk --update add ca-certificates
RUN mkdir -p /app/pwd

COPY --from=0 /go/bin/play-with-docker /app/play-with-docker
COPY --from=0 /etc/ssh/ssh_host_rsa_key /etc/ssh/ssh_host_rsa_key
COPY ./www /app/www

WORKDIR /app
CMD ["./play-with-docker"]

EXPOSE 3000
