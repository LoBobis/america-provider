FROM golang:1.24-bullseye

ENV DEBIAN_FRONTEND=noninteractive
ENV CGO_ENABLED=0
ENV GO111MODULE=on
ENV PATH=$PATH:/usr/local/go/bin

# -----------------------------------------------------
# 1. System tools
# -----------------------------------------------------
RUN apt-get update && apt-get install -y --no-install-recommends \
    apt-transport-https ca-certificates curl bash git make jq unzip \
    build-essential sudo gnupg lsb-release \
    && rm -rf /var/lib/apt/lists/*

# -----------------------------------------------------
# 2. Add user for VSCode
# -----------------------------------------------------
RUN useradd -m vscode && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
WORKDIR /workspace

# -----------------------------------------------------
# 3. Install Kubernetes tools (kubectl, kind, kustomize)
# -----------------------------------------------------
ENV KUBECTL_VERSION=v1.31.1
ENV KIND_VERSION=v0.23.0
ENV KUSTOMIZE_VERSION=v5.4.1

RUN curl -fsSLo /usr/local/bin/kubectl https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && chmod +x /usr/local/bin/kubectl
RUN curl -fsSLo /usr/local/bin/kind https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64 && chmod +x /usr/local/bin/kind 

# -----------------------------------------------------
# 4. Install Crossplane CLI
# -----------------------------------------------------
ENV CROSSPLANE_CLI_VERSION=v1.16.0
RUN curl -sL https://releases.crossplane.io/stable/${CROSSPLANE_CLI_VERSION}/bin/linux_amd64/crossplane \
    -o /usr/local/bin/crossplane && chmod +x /usr/local/bin/crossplane

# -----------------------------------------------------
# 5. Install controller-gen (for CRD generation)
# -----------------------------------------------------
RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.4

# -----------------------------------------------------
# 6. Install other Go tools used in provider builds
# -----------------------------------------------------
#RUN go install github.com/crossplane/crossplane-runtime/cmd/angryjet@latest
#RUN go install github.com/google/go-licenses@latest

# -----------------------------------------------------
# 7. Copy dependencies (optional for offline)
# -----------------------------------------------------
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Optional: vendor dependencies for offline mode
# RUN go mod vendor

USER vscode
