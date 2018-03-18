FROM golang:1.9

# Copy the runtime dockerfile into the context as Dockerfile
COPY . /go/src/github.com/play-with-docker/play-with-docker

WORKDIR /go/src/github.com/play-with-docker/play-with-docker


RUN ssh-keygen -N "" -t rsa -f /etc/ssh/ssh_host_rsa_key >/dev/null

WORKDIR /go/src/github.com/play-with-docker/play-with-docker/router/l2

RUN CGO_ENABLED=0 go build -a -installsuffix nocgo -o /go/bin/play-with-docker-l2 .


FROM alpine

RUN apk --update add ca-certificates
RUN mkdir /app

COPY --from=0 /go/bin/play-with-docker-l2 /app/play-with-docker-l2
COPY --from=0 /etc/ssh/ssh_host_rsa_key /etc/ssh/ssh_host_rsa_key

WORKDIR /app
CMD ["./play-with-docker-l2", "-ssh_key_path", "/etc/ssh/ssh_host_rsa_key"]

EXPOSE 22 53 443 8080
