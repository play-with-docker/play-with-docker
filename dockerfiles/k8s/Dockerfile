FROM centos:7

COPY ./systemctl /usr/bin/systemctl
COPY ./kubernetes.repo /etc/yum.repos.d/



RUN yum install -y kubelet kubeadm \
    && mv -f /etc/systemd/system/kubelet.service.d/10-kubeadm.conf /etc/systemd/system/kubelet.service \
    && yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo \
    && yum install -y docker-ce \
    && sed -i -e '4d;5d;8d' /lib/systemd/system/docker.service \
    && yum clean all

RUN curl -Lf -o jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 \
    && chmod +x ./jq && mv jq /usr/bin

COPY ./kube* /etc/systemd/system/
COPY ./wrapkubeadm.sh /usr/local/bin/kubeadm

RUN mkdir -p /root/.kube && ln -s /etc/kubernetes/admin.conf /root/.kube/config \
    && rm -f /etc/machine-id

CMD systemctl start docker && systemctl start kubelet \
    && bash
