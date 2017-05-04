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

```bash
apt-get install -y thin-provisioning-tools
pvcreate /dev/xvdb
vgcreate docker /dev/xvdb
lvcreate --wipesignatures y -n thinpool docker -l 95%VG
lvcreate --wipesignatures y -n thinpoolmeta docker -l 1%VG
lvconvert -y --zero n -c 512K --thinpool docker/thinpool --poolmetadata docker/thinpoolmeta
mkdir -p /etc/lvm/profile/
echo '
activation {
    thin_pool_autoextend_threshold=80
    thin_pool_autoextend_percent=20
}
' > /etc/lvm/profile/docker-thinpool.profile
lvchange --metadataprofile docker-thinpool docker/thinpool
lvs -o+seg_monitor
```

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

10. Make sure apparmor file is there and load it
```
#include <tunables/global>


profile docker-dind flags=(attach_disconnected,mediate_deleted) {
  #include <abstractions/base>
  network,
  capability,
  file,
  umount,
  ptrace,
  mount,
  pivot_root,

  # block some other dangerous paths
  deny @{PROC}/sysrq-trigger rwklx,
  deny @{PROC}/mem rwklx,
  deny @{PROC}/kmem rwklx,
  deny @{PROC}/sys/kernel/[^s][^h][^m]* wklx,
  deny @{PROC}/sys/kernel/*/** wklx,
  deny /etc/apparmor.d/cache/docker** rwklx,

  # only allow to mount in graph folder
  deny mount fstype=proc -> /[^g]**,
  deny mount fstype=devtmpfs,

  # only allow to mount in proc folder
  deny mount options=bind /proc/sysrq-trigger -> /[^p]**,



  deny mount /dev/s*,
  deny mount /dev/xv*,


}
```

And then run

```bash
sudo apparmor_parser -W docker-dind.profile
```

11. Pull PWD DinD image
```bash
docker pull franela/dind:overlay2
```

12. Increase arp cache size and inotify handlers

Edit sysctl.conf and add:
```
net.ipv4.neigh.default.gc_thresh3 = 8192
net.ipv4.neigh.default.gc_thresh2 = 8192
net.ipv4.neigh.default.gc_thresh1 = 4096
fs.inotify.max_user_instances = 10000
kernel.sysrq = 0
net.ipv4.tcp_tw_recycle = 1
```

12.a. Change kernel config
```
CONFIG_MAGIC_SYSRQ=n
```

13. Start pwd container
```bash
docker run -d \
        -e DIND_IMAGE=franela/dind:overlay2 \
        -e GOOGLE_RECAPTCHA_DISABLED=true \
        -e APPARMOR_PROFILE=docker-dind \
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
