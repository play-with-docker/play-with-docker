ARG VERSION=docker:stable-dind
FROM ${VERSION}

RUN apk add --no-cache git tmux vim curl bash build-base qemu-img qemu-system-x86_64

ENV GOPATH /root/go
ENV PATH $PATH:$GOPATH

# Use specific moby commit due to vendoring mismatch
ENV MOBY_COMMIT="d9d2a91780b34b92e669bbfa099f613bd9fad6bb"

RUN mkdir /root/go && apk add --no-cache go \
    && go get -u -d github.com/moby/tool/cmd/moby && (cd $GOPATH/src/github.com/moby/tool/cmd/moby && git checkout $MOBY_COMMIT && go install) \
    && go get -u github.com/linuxkit/linuxkit/src/cmd/linuxkit \
    && rm -rf /root/go/pkg && rm -rf /root/go/src && rm -rf /usr/lib/go


# Add bash completion and set bash as default shell
RUN mkdir /etc/bash_completion.d \
    && curl https://raw.githubusercontent.com/docker/cli/master/contrib/completion/bash/docker -o /etc/bash_completion.d/docker \
    && sed -i "s/ash/bash/" /etc/passwd


# Replace modprobe with a no-op to get rid of spurious warnings
# (note: we can't just symlink to /bin/true because it might be busybox)
RUN rm /sbin/modprobe && echo '#!/bin/true' >/sbin/modprobe && chmod +x /sbin/modprobe

# Install a nice vimrc file and prompt (by soulshake)
COPY ["sudo", "/usr/local/bin/"]
COPY [".vimrc", ".profile", ".inputrc", ".gitconfig", "./root/"]
COPY ["motd", "/etc/motd"]
COPY ["daemon.json", "/etc/docker/"]

# Move to our home
WORKDIR /root


# Remove IPv6 alias for localhost and start docker in the background ...
CMD cat /etc/hosts >/etc/hosts.bak && \
    sed 's/^::1.*//' /etc/hosts.bak > /etc/hosts && \
    mount -t securityfs none /sys/kernel/security && \
    dockerd &>/docker.log & \
    while true ; do /bin/bash -l; done
# ... and then put a shell in the foreground, restarting it if it exits
