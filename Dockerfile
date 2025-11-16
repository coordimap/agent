FROM golang:1.23-alpine AS build-env

# Add dependencies for building and for eBPF code generation
# llvm is a dependency for clang
# libbpf-dev provides the C headers for libbpf
RUN apk add --no-cache git clang llvm bpftool libbpf-dev

# Add source code
ADD . /src
WORKDIR /src

# Configure git for private repositories
ARG GIT_TOKEN
RUN git config --global url."https://${GIT_TOKEN}@dev.azure.com/bloopi/bloopi/_git/shared_models.git".insteadOf "https://dev.azure.com/bloopi/bloopi/_git/shared_models.git"

RUN echo ${GIT_TOKEN}

# Generate eBPF Go files. This requires kernel headers (BTF).
# First, generate vmlinux.h from the running kernel's BTF info.
# Note: This requires the build environment to have access to /sys/kernel/btf/vmlinux
RUN mkdir -p internal/cloud/flows/headers && \
    bpftool btf dump file /sys/kernel/btf/vmlinux format c > internal/cloud/flows/headers/vmlinux.h

# Now, run go generate which uses the header file created above.
# The `generate.go` file will also clean up the .c file afterwards.
RUN go generate ./...

# Build the final Go binary
RUN CGO_ENABLED=1 go build -a -o cli/agent/agent cli/agent/main.go

# --- Final Stage ---
FROM alpine:latest

COPY --from=build-env /src/cli/agent/agent /agent

CMD [ "/agent" ]
