package flows

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/rs/zerolog/log"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall -Werror" bpf tc.c -- -I./headers

// event represents the data received from the eBPF program.
// It must match the C struct in tc.c.
type event struct {
	Saddr     uint32
	Daddr     uint32
	Sport     uint16
	Dport     uint16
	Proto     uint8
	Timestamp uint64
}

func MonitorNetworkTraffic(outputChannel chan ConnectionData) error {
	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("failed to remove memlock: %w", err)
	}

	// Load pre-compiled programs and maps into the kernel.
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		return fmt.Errorf("loading objects: %w", err)
	}
	defer objs.Close()

	// Attach the TC program to the eth0 interface.
	ifaceName := "eth0"
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("lookup network iface %q: %w", ifaceName, err)
	}

	// Attach the program.
	l, err := link.AttachXDP(link.XDPOptions{
		Program:   objs.TcIngress,
		Interface: iface.Index,
	})
	if err != nil {
		return fmt.Errorf("could not attach TC program: %w", err)
	}
	defer l.Close()

	log.Info().Msgf("Attached TC program to iface %q (index %d)", iface.Name, iface.Index)

	// Open a ringbuf reader from userspace RING_BUFFER map described in the
	// eBPF C program.
	rd, err := ringbuf.NewReader(objs.Rb)
	if err != nil {
		return fmt.Errorf("opening ringbuf reader: %w", err)
	}
	defer rd.Close()

	// Close the reader when the process exits, so we can exit cleanly.
	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stopper
		rd.Close()
	}()

	log.Info().Msg("Waiting for events...")

	var e event
	for {
		record, err := rd.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				log.Info().Msg("Received signal, exiting...")
				return nil
			}
			log.Warn().Msgf("reading from reader: %s", err)
			continue
		}

		// Parse the ringbuf event entry into a bpfEvent structure.
		if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.BigEndian, &e); err != nil {
			log.Warn().Msgf("parsing ringbuf event: %s", err)
			continue
		}

		outputChannel <- ConnectionData{
			SrcIP:   net.IP{byte(e.Saddr >> 24), byte(e.Saddr >> 16), byte(e.Saddr >> 8), byte(e.Saddr)},
			DstIP:   net.IP{byte(e.Daddr >> 24), byte(e.Daddr >> 16), byte(e.Daddr >> 8), byte(e.Daddr)},
			SrcPort: e.Sport,
			DstPort: e.Dport,
			Proto:   e.Proto,
		}
	}
}
