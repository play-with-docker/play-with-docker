ARG VERSION=docker:dind
FROM ${VERSION}

RUN apk add --no-cache py-pip python3-dev libffi-dev openssl-dev git tmux apache2-utils vim build-base gettext-dev curl bash-completion bash util-linux jq openssh openssl tree \
    && ln -s /usr/local/bin/docker /usr/bin/docker

ENV GOPATH /root/go
ENV PATH $PATH:$GOPATH


ENV DOCKER_TLS_CERTDIR=""
ENV DOCKER_CLI_EXPERIMENTAL=enabled

# Install compose
ENV COMPOSE_VERSION=2.18.1
RUN mkdir -p /usr/lib/docker/cli-plugins \
    && curl -LsS https://github.com/docker/compose/releases/download/v$COMPOSE_VERSION/docker-compose-linux-x86_64 -o /usr/lib/docker/cli-plugins/docker-compose \
    && chmod +x /usr/lib/docker/cli-plugins/docker-compose


# Install scout
ENV SCOUT_VERSION=1.0.9
RUN wget -O /tmp/scout.tar.gz https://github.com/docker/scout-cli/releases/download/v1.0.9/docker-scout_1.0.9_linux_amd64.tar.gz \
    && tar -xvf /tmp/scout.tar.gz docker-scout -C /usr/local/bin \
    && chmod +x /usr/local/bin/docker-scout \
    && rm /tmp/scout.tar.gz
    
    

# Add bash completion and set bash as default shell
RUN mkdir /etc/bash_completion.d \
    && curl -sS https://raw.githubusercontent.com/docker/cli/master/contrib/completion/bash/docker -o /etc/bash_completion.d/docker \
    && sed -i "s/ash/bash/" /etc/passwd
 
# Replace modprobe with a no-op to get rid of spurious warnings
# (note: we can't just symlink to /bin/true because it might be busybox)
RUN rm /sbin/modprobe && echo '#!/bin/true' >/sbin/modprobe && chmod +x /sbin/modprobe

# Install a nice vimrc file and prompt (by soulshake)
COPY ["docker-prompt", "sudo", "/usr/local/bin/"]
COPY [".vimrc", ".profile", ".inputrc", ".gitconfig", "./root/"]
COPY ["motd", "/etc/motd"]
COPY ["daemon.json", "/etc/docker/"]


# Move to our home
WORKDIR /root

# Setup certs and ssh keys
RUN mkdir -p /var/run/pwd/certs && mkdir -p /var/run/pwd/uploads \
    && ssh-keygen -N "" -t ed25519 -f  /etc/ssh/ssh_host_ed25519_key >/dev/null \
    && mkdir ~/.ssh && ssh-keygen -N "" -t ed25519 -f ~/.ssh/id_rsa \
    && cat ~/.ssh/id_rsa.pub > ~/.ssh/authorized_keys

# Remove IPv6 alias for localhost and start docker in the background ...
CMD cat /etc/hosts >/etc/hosts.bak && \
    sed 's/^::1.*//' /etc/hosts.bak > /etc/hosts && \
    sed -i "s/\PWD_IP_ADDRESS/$PWD_IP_ADDRESS/" /etc/docker/daemon.json && \
    sed -i "s/\DOCKER_TLSENABLE/$DOCKER_TLSENABLE/" /etc/docker/daemon.json && \
    sed -i "s/\DOCKER_TLSCACERT/$DOCKER_TLSCACERT/" /etc/docker/daemon.json && \
    sed -i "s/\DOCKER_TLSCERT/$DOCKER_TLSCERT/" /etc/docker/daemon.json && \
    sed -i "s/\DOCKER_TLSKEY/$DOCKER_TLSKEY/" /etc/docker/daemon.json && \
    mount -t securityfs none /sys/kernel/security && \
    echo "root:root" | chpasswd &> /dev/null && \
    /usr/sbin/sshd -o PermitRootLogin=yes -o PrintMotd=no 2>/dev/null && \
    dockerd &>/docker.log & \
    while true ; do script -q -c "/bin/bash -l" /dev/null ; done
# ... and then put a shell in the foreground, restarting it if it exits
