FROM golang:1.7

# Copy the runtime dockerfile into the context as Dockerfile
COPY Dockerfile.run /go/bin/Dockerfile
COPY ./www /go/bin/www

COPY . /go/src/github.com/play-with-docker/play-with-docker

WORKDIR /go/src/github.com/play-with-docker/play-with-docker

RUN go get -v -d ./...

RUN CGO_ENABLED=0 go build -a -installsuffix nocgo -o /go/bin/play-with-docker .

# Set the workdir to be /go/bin which is where the binaries are built
WORKDIR /go/bin

# Export the WORKDIR as a tar stream
CMD tar -cf - .
