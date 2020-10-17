package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
)

// Global setting variables
var dstAddrStrLength = 5

//Wrapper class for a queue of packets
// param maxsize - the max size of the queue storing packets
type NetworkInterface struct {
	mu         sync.Mutex
	Queue      []string
	Mtu        int
	maxQueSize int
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
// returns an error if the 'queue' is empty
func (n *NetworkInterface) get() (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.Queue) > 1 {
		toReturn := n.Queue[0]
		n.Queue = n.Queue[1:]
		return toReturn, nil
	}
	return "", errors.New("Empty")
}

//put the packet into the queue
// put returns an error if the queue is full
func (n *NetworkInterface) put(pkt string, block bool) error {
	// if block is true, block until there is room in the queue
	// if false, throw queue full error
	if block == true {
		for {
			// obtain lock
			n.mu.Lock()
			if len(n.Queue) < n.maxQueSize {
				// add to queue
				n.Queue = append(n.Queue, pkt)
				n.mu.Unlock()
				return nil
			}
			// unlock until next loop
			n.mu.Unlock()
			continue
		}
	}

	// if block != true
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.Queue) < n.maxQueSize {
		n.Queue = append(n.Queue, pkt)
		return nil
	}

	return errors.New("Queue Full")
}

//Implements a network layer packet
// DstAddr: address of the destination host
// DataS: packet payload
// DstAddrStrLength: packet encoding lengths
type NetworkPacket struct {
	DstAddr          int
	DataS            string
	DstAddrStrLength int
}

func NewNetworkPacket(dstAddr int, dataS string) *NetworkPacket {
	return &NetworkPacket{
		DstAddr: dstAddr,
		DataS:   dataS,
	}
}

func (np *NetworkPacket) str() string {
	return np.ToByteS()
}

//ToBytesS converts packet to a byte string for transmission over links
func (np *NetworkPacket) ToByteS() string {
	byteS := fmt.Sprintf("%0*s%s", np.DstAddrStrLength, strconv.Itoa(np.DstAddr), np.DataS)

	return byteS
	//seqNumS := fmt.Sprintf("%0*s", p.SeqNumSlength, strconv.Itoa(p.SeqNum))
}

//FromByteS builds a packet object from a byte string
// Returns error if it cannot convert addres to int
func FromByteS(byteS string) (*NetworkPacket, error) {
	dstAddr, err := strconv.Atoi(byteS[0:dstAddrStrLength])
	if err != nil {
		log.Println("Error converting addr to string")
		return nil, err
	}

	dataS := byteS[dstAddrStrLength:]
	return NewNetworkPacket(dstAddr, dataS), nil
}

//Host implements a network host for receiving and transmitting data
// Addr: address of this node represented as an integer
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

// Called when printing the objects
func (h *Host) str() string {
	return fmt.Sprintf("Host_%d", h.Addr)
}

//UdtSend creates a packet and enqueues for transmission
// dst_addr: destination address for the packet
// data_S: data being transmitted to the network layer
func (h *Host) UdtSend(dstAddr int, dataS string) {
	p := NewNetworkPacket(dstAddr, dataS)

	fmt.Printf("%s: sending packet \"%s\" on the Out interface with mtu = %d\n", h.str, p.ToByteS(), h.OutInterfaceL[0].Mtu)

	h.OutInterfaceL[0].put(p.ToByteS(), false) // send packets always enqueued successfully
}

//UdtReceive receives packest from the network layer
func (h *Host) UdtReceive() {
	pktS, err := h.InInterfaceL[0].get()
	if err == nil {
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
// Name: friendly router nam for debugging
type Router struct {
	Stop          bool
	Name          string
	InInterfaceL  []*NetworkInterface
	OutInterfaceL []*NetworkInterface
}

//NewRouter returns a new router with given specs
// interfaceCount: the number of input and output interfaces
// maxQueSize: max queue legth (passed to interfacess)
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

//Called when printing the object
func (rt *Router) str() string {
	return fmt.Sprintf("Router_%s\n", rt.Name)
}

//lok through the content of incoming interfaces and forward to appropriate outgoing interfaces
func (rt *Router) forward() {
	for i, v := range rt.InInterfaceL {
		//pktS := ""

		// TRYE
		// get packet from interface i
		if pktS, err := v.get(); err != nil {
			// if packet exists make a forwarding decision
			p, err := FromByteS(pktS)
			if err != nil {
				log.Println("Could not get packet")
				continue
			}

			// HERE you will need to implement a lookup into the
			// forwarding table to find the appropriate outgoing interface
			// for now we assume the outgoing interface is also i
			fmt.Printf("%s: forwarding packet %s from interface %d to %d with mtu %d\n", rt.str, p.str, i, i, rt.OutInterfaceL[i].Mtu)

			if err = rt.OutInterfaceL[i].put(p.ToByteS(), false); err != nil {
				//log.Printf("Could not put packet %s in router %s, into outInterface %d. Error: %s", p.str, rt.forward, i, err)
				log.Printf("%s: packet '%s' lost on interface %d\n", rt.str, i)
			}
		}
	}
}

func (rt *Router) Run() {
	fmt.Printf("%s: starting\n", rt.str)
	for {
		rt.forward()
		if rt.Stop {
			fmt.Printf("%s: Ending\n", rt.str)
		}
	}
}
