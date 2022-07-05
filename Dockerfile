FROM debian:bookworm-20220622
ARG KUBECTL_VERSION=v1.24.0 \
    KREW_VERSION=v0.4.3
RUN apt update -y && \
  apt install -y curl git && \
  curl -LO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" && \
  curl -LO "https://github.com/kubernetes-sigs/krew/releases/download/${KREW_VERSION}/krew-linux_amd64.tar.gz" && \
  install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
  tar -C /usr/local/bin -xvf krew-linux_amd64.tar.gz && \
  rm krew-linux_amd64.tar.gz && \
  useradd -rm -d /home/promdump -s /bin/bash promdump
USER promdump
WORKDIR /home/promdump
RUN  /usr/local/bin/krew-linux_amd64 install krew
ENV PATH="/home/promdump/.krew/bin:${PATH}"
RUN kubectl krew update && \
  kubectl krew install promdump
CMD ["kubectl", "promdump", "-h"]
