# Cleaning up everything PWD creates

```bash
docker rm -fv $(docker ps -aq --filter status=exited --filter status=created)
docker network rm $(docker network ls | grep "-" | cut -d ' ' -f1)
```

# Installing PWD on AWS

1. Install overlay and ipvs modules
```bash
sudo modprobe overlay
sudo modprobe xt_ipvs
```

2. Install docker daemon
```bash
curl -sSL https://get.docker.com/ | sh
```

3. Stop docker daemon
```bash
sudo service docker stop
```

4. Install devicemapper with directlvm
Follow instructions on [https://docs.docker.com/engine/userguide/storagedriver/device-mapper-driver/#configure-direct-lvm-mode-for-production]

5. Remove files
```bash
sudo rm -fr /var/lib/docker
```

6. Create `daemon.json` in `/etc/docker`
```json
{
    "storage-driver": "devicemapper",
    "storage-opts": [
        "dm.thinpooldev=/dev/mapper/docker-thinpool",
        "dm.use_deferred_removal=true",
        "dm.use_deferred_deletion=true"
    ],
    "dns": [
        "172.18.0.1",
        "8.8.8.8",
        "10.0.0.2"
    ]
}
```

6a. Remove `search` in `/etc/resolv.conf`

7. Start docker daemon
```bash
sudo service docker start
```

8. Apply iptable rule
```bash
sudo iptables -t nat -A PREROUTING -p tcp -m multiport --dports 1024:2376,2378:7945,7947:65535 -j REDIRECT --to-ports 80
```

9. Start docker swarm
```bash
docker swarm init
```

10. Pull PWD DinD image
```bash
docker pull franela/dind:overlay2
```

11. Increase arp cache size and inotify handlers
```
net.ipv4.neigh.default.gc_thresh3 = 8192
net.ipv4.neigh.default.gc_thresh2 = 8192
net.ipv4.neigh.default.gc_thresh1 = 4096
fs.inotify.max_user_instances = 10000
```

12. Start pwd container
```bash
docker run -d \
        -e DIND_IMAGE=franela/dind:overlay2 \
        -e GOOGLE_RECAPTCHA_DISABLED=true \
        -e MAX_PROCESSES=10000 \
        -e EXPIRY=4h \
        --name pwd \
        --dns 8.8.8.8 \
        -p 80:3000 \
        -p 443:3001 \
        -p 53:53/udp \
        -p 53:53/tcp \
        -v /var/run/docker.sock:/var/run/docker.sock -v sessions:/app/pwd/ \
        --restart always \
        franela/play-with-docker:latest
```
