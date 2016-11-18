# Prepares the virtual box instance
prepare:
	# Creates the virtual box
	-docker-machine create -d virtualbox --virtualbox-boot2docker-url https://github.com/boot2docker/boot2docker/releases/download/v1.13.0-rc1/boot2docker.iso pwd && true
	# Makes sure the docker daemon has the DinD image pulled
	-docker-machine ssh pwd "docker pull franela/pwd-1.12.3-experimental-dind"
	# Daemon should be swarm
	-docker-machine ssh pwd "docker swarm init --advertise-addr $$(docker-machine ip pwd)"
	# Stops to daemon to do further configurations on the box
	-docker-machine stop pwd
	# Adds the host GOPATH as a shared folder in the box
	-VBoxManage sharedfolder add pwd --name gopathsrc --hostpath ${GOPATH}src --automount
	# Do port forwaring so we can reach the app using localhost:3000
	-VBoxManage modifyvm pwd --natpf1 "localhost,tcp,,3000,,3000"

# Starts the virtual box instance
start:
	# Starts the machine
	-docker-machine start pwd
	# Make sure the folder where we'll mount the shared folder exists
	docker-machine ssh pwd "sudo mkdir -p /go/src"
	# Mount the host's GOPATH shared folder
	docker-machine ssh pwd "sudo mount -t vboxsf gopathsrc /go/src"

# Runs the app
run:
	@eval $$(docker-machine env pwd); \
	docker-compose up

.PHONY: prepare start run
