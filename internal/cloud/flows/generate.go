//go:build ignore

package flows

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall -Werror" bpf ./bpf_probe/tc.c -- -I./headers
