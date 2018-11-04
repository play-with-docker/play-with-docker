FROM centos:7

COPY ./systemctl /usr/bin/systemctl
COPY ./kubernetes.repo /etc/yum.repos.d/



RUN yum install -y kubectl-1.11.3-0 kubeadm-1.11.3-0 kubelet-1.11.3-0 \
    && mv -f /etc/systemd/system/kubelet.service.d/10-kubeadm.conf /etc/systemd/system/kubelet.service \
    && yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo \
    && yum install -y docker-ce git \
    && sed -i -e '4d;5d;8d' /lib/systemd/system/docker.service \
    && yum clean all

RUN curl -Lf -o /usr/bin/jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 \
    && curl -Lf -o /usr/bin/docker-compose https://github.com/docker/compose/releases/download/1.21.0/docker-compose-$(uname -s)-$(uname -m) \
    && chmod +x /usr/bin/jq /usr/bin/docker-compose 


VOLUME ["/var/lib/kubelet"]

COPY ./kube* /etc/systemd/system/
COPY ./wrapkubeadm.sh /usr/local/bin/kubeadm
COPY ./tokens.csv /etc/pki/tokens.csv
COPY ./daemon.json /etc/docker/
COPY ./resolv.conf.override /etc/

COPY motd /etc/motd

RUN echo $'cat /etc/motd \n\
export PS1="[\h \W]$ "' >> /root/.bash_profile

RUN mkdir -p /root/.kube && ln -s /etc/kubernetes/admin.conf /root/.kube/config \
    && rm -f /etc/machine-id

WORKDIR /root

CMD mount --make-shared / && systemctl start docker && systemctl start kubelet \
    && while true; do bash -l; done
