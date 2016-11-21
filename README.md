# play-with-docker

Play With Docker gives you the experience of having a free Alpine Linux Virtual Machine in the cloud
where you can build and run Docker containers and even create clusters with Docker features like Swarm Mode.

Under the hood DIND or Docker-in-Docker is used to give the effect of multiple VMs/PCs.

A live version is available at: http://play-with-docker.com/

## Requirements

Docker 1.13-dev or higher is required to make use of **attachable** overlay networks. You can use docker-machine with the following command:

```
docker-machine create -d virtualbox --virtualbox-boot2docker-url https://github.com/boot2docker/boot2docker/releases/download/v1.13.0-rc1/boot2docker.iso <name>
```

The docker daemon needs to run in swarm mode because PWD uses overlay attachable networks. For that
just run `docker swarm init`.

It's also necessary to manually load the IPVS kernel module because as swarms are created in `dind`, 
the daemon won't load it automatically. Run the following command for that purpose: `sudo lsmod xt_ipvs`

If you want to experiment with a stable version of Docker 1.12 then you can override the requirement for Docker 1.13-dev.

```
DOCKER_VERSION=1.12 ./play-with-docker
```

## Installation

Start the Docker daemon on your machine and run `docker pull docker:1.12.2-rc2-dind`. 

1) Install go 1.7.1 with `brew` on Mac or through a package manager.

2) `go get`

3) `go build`

4) Run the binary produced as `play-with-docker`

5) Point to http://localhost:3000/ and click "New Instance"

Notes:

* There is a hard-coded limit to 5 Docker playgrounds per session. After 1 hour sessions are deleted.
* If you want to override the DIND (Docker-In-Docker) version or image then set the environmental variable i.e.
  `DIND_IMAGE=docker:dind`

## FAQ

### How can I connect to a published port from the outside world?

We're planning to setup a reverse proxy that handles redirection automatically, in the meantime you can use [ngrok](https://ngrok.com) within PWD running `docker run --name supergrok -d jpetazzo/supergrok` then `docker logs --follow supergrok` , it will give you a ngrok URL, now you can go to that URL and add the IP+port that you want to connect to… e.g. if your PWD instance is 10.0.42.3, you can go to http://xxxxxx.ngrok.io/10.0.42.3:8000 (where the xxxxxx is given to you in the supergrok logs).

