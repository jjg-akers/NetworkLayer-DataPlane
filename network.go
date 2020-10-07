package main

import (
	"fmt"
	"log"
	"strconv"
	"sync"
)

//Wrapper class for a queue of packets
// param maxsze - the mix size of the queue storing packets
type NetworkInterface struct {
	mu    sync.Mutex
	Queue []string
	Mtu   int
	maxQueSize
}

func NewNetworkInterface(maxQ int) *NetworkInterface {
	return &NetworkInterface{
		mu:         sync.Mutex{},
		Queue:      []string{},
		Mtu:        1000000,
		maxQueSize: maxQ,
	}
}

//gets a packet from the queue
func (n *NetworkInterface) get() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.Queue) > 1 {
		return n.Queue[0]
		n.Queue = n.Queue[1:]
	}
	return ""
}

//put the packet into the queue
func (n *NetworkInterface) put(pkt string, block bool) {
	n.Queue = append(n.Queue, pkt)
}

//Implements a network layer packet
type NetworkPacket struct {
	DstAddr          int
	DataS            string
	DstAddrStrLength int
}

func NewNetworkPacket(dstAddr int, dataS string) *NetworkPacket {
	return &NetworkPacket{
		DstAddr:          dstAddr,
		DataS:            dataS,
		DstAddrStrLength: 5,
	}
}

//ToBytesS converts packet to a byte string for transmission over links
func (np *NetworkPacket) ToByteS() string {
	byteS := fmt.Sprintf("%0*s%s", np.DstAddrStrLength, strconv.Itoa(np.DstAddr), np.DataS)

	return byteS
	//seqNumS := fmt.Sprintf("%0*s", p.SeqNumSlength, strconv.Itoa(p.SeqNum))
}

//FromByteS extracts a packet object from a byte string
func (np *NetworkPacket) FromByteS(byteS string) (int, string, error) {
	dstAddr, err := strconv.Atoi(byteS[0:np.DstAddrStrLength])
	if err != nil {
		log.Println("Error converting addr to string")
		return 0, "", err
	}

	dataS := byteS[np.DstAddrStrLength:]
	return dstAddr, dataS, nil
}

//Host implements a network host for receiving and transmitting data
type Host struct {
	Addr          int
	InInterfaceL  []*NetworkInterface
	OutInterfaceL []*NetworkInterface
	Stop          bool
}

func NewHost(addr int) *Host {
	return &Host{
		Addr:          addr,
		InInterfaceL:  []*NetworkInterface{NewNetworkInterface(0)},
		OutInterfaceL: []*NetworkInterface{NewNetworkInterface(0)},
		Stop:          false,
	}
}

func (h *Host) str() string {
	return fmt.Sprintf("Host_%d", h.Addr)
}

//UdtSend creates a packet and enqueues for transmission
// dst_addr: destination address for the packet
// data_S: data being transmitted to the network layer
func (h *Host) UdtSend(dstAddr int, dataS string) {
	p := NewNetworkPacket(dstAddr, dataS)

	h.OutInterfaceL[0].put(p.ToByteS(), false) // send packets always enqueued successfully
	fmt.Printf("%s: sending packet \"%s\" on the Out interface with mtu = %d\n", h.str, p.ToByteS(), h.OutInterfaceL[0].Mtu)
}

//UdtReceive receives packest from the network layer
func (h *Host) UdtReceive() {
	pktS := h.InInterfaceL[0].get()
	if pktS != "" {
		fmt.Printf("%s: received packet \"%s\" on the In interface\n", h.str, pktS)
	}
}

//Run startes a routine for the host to keep receiving data
func (h *Host) run() {
	fmt.Println("Starting host receive routine")

	for {
		//receive data arriving in the in interface
		h.UdtReceive()

		if h.Stop {
			fmt.Println("Ending host receive routine")
			return
		}
	}
}

//Router implements a multi-interface router described in class
type Router struct {
	Stop          bool
	Name          string
	InInterfaceL  []*NetworkInterface
	OutInterfaceL []*NetworkInterface
}

func NewRouter(name string, interfaceCount int, maxQueSize int) *Router {
	in := make([]*NetworkInterface, interfaceCount)
	out := make([]*NetworkInterface, interfaceCount)
	for i := 0; i < interfaceCount; i++ {
		in[i] = NewNetworkInterface(maxQueSize)
		out[i] = NewNetworkInterface(maxQueSize)

	}
	return &Router{
		Stop:          false,
		Name:          name,
		InInterfaceL:  in,
		OutInterfaceL: out,
	}
}
