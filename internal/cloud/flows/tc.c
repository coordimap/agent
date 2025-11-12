#include "vmlinux.h"
#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

#define TC_ACT_OK 0
#define ETH_P_IP 0x0800

struct event {
	__u32 saddr;
	__u32 daddr;
	__u16 sport;
	__u16 dport;
	__u8 proto;
    __u64 timestamp;
};

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} rb SEC(".maps");

SEC("tc")
int tc_ingress(struct __sk_buff *skb)
{
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;

    struct ethhdr *eth = data;
    if ((void *)eth + sizeof(*eth) > data_end) {
        return TC_ACT_OK;
    }

    if (eth->h_proto != bpf_htons(ETH_P_IP)) {
        return TC_ACT_OK;
    }

    struct iphdr *ip = data + sizeof(*eth);
    if ((void *)ip + sizeof(*ip) > data_end) {
        return TC_ACT_OK;
    }

    struct event *e;
    e = bpf_ringbuf_reserve(&rb, sizeof(*e), 0);
    if (!e) {
        return TC_ACT_OK;
    }

    e->saddr = ip->saddr;
    e->daddr = ip->daddr;
    e->proto = ip->protocol;
    e->timestamp = bpf_ktime_get_ns();

    switch (ip->protocol) {
        case IPPROTO_TCP:
        {
            struct tcphdr *tcp = (struct tcphdr *)(ip + 1);
            if ((void *)tcp + sizeof(*tcp) > data_end) {
                bpf_ringbuf_discard(e, 0);
                return TC_ACT_OK;
            }
            e->sport = bpf_ntohs(tcp->source);
            e->dport = bpf_ntohs(tcp->dest);
            break;
        }
        case IPPROTO_UDP:
        {
            struct udphdr *udp = (struct udphdr *)(ip + 1);
            if ((void *)udp + sizeof(*udp) > data_end) {
                bpf_ringbuf_discard(e, 0);
                return TC_ACT_OK;
            }
            e->sport = bpf_ntohs(udp->source);
            e->dport = bpf_ntohs(udp->dest);
            break;
        }
        default:
            bpf_ringbuf_discard(e, 0);
            return TC_ACT_OK;
    }

    bpf_ringbuf_submit(e, 0);

    return TC_ACT_OK;
}

char __license[] SEC("license") = "Dual MIT/GPL";
