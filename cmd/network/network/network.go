package network

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
)

type Node interface {
	Run(*sync.WaitGroup)
	GetInInterfaceL() []*NetworkInterface
	GetOutInterfaceL() []*NetworkInterface
	Str() string
}

// Global setting variables
var dstAddrStrLength = 5

//Wrapper class for a queue of packets
// param maxsize - the max size of the queue storing packets
type NetworkInterface struct {
	mu         sync.Mutex
	Queue      []string
	Mtu        int
	MaxQueSize int
}

func NewNetworkInterface(maxQ int) *NetworkInterface {
	return &NetworkInterface{
		mu:         sync.Mutex{},
		Queue:      []string{},
		Mtu:        1000000,
		MaxQueSize: maxQ,
	}
}

//gets a packet from the queue
// returns an error if the 'queue' is empty
func (n *NetworkInterface) Get() (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	//fmt.Println("qu len: ", len(n.Queue))
	if len(n.Queue) > 0 {
		toReturn := n.Queue[0]
		n.Queue = n.Queue[1:]
		return toReturn, nil
	}
	return "", errors.New("Empty")
}

//put the packet into the queue
// put returns an error if the queue is full
func (n *NetworkInterface) Put(pkt string, block bool) error {
	// if block is true, block until there is room in the queue
	// if false, throw queue full error
	if block == true {
		for {
			// obtain lock
			n.mu.Lock()
			if len(n.Queue) < n.MaxQueSize {
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
	if len(n.Queue) < n.MaxQueSize {
		//fmt.Println("putting pakcet in queue")
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
		DstAddr:          dstAddr,
		DataS:            dataS,
		DstAddrStrLength: dstAddrStrLength,
	}
}

func (np *NetworkPacket) Str() string {
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
	Stop          chan interface{}
}

func (h *Host) GetInInterfaceL() []*NetworkInterface {
	return h.InInterfaceL
}

func (h *Host) GetOutInterfaceL() []*NetworkInterface {
	return h.OutInterfaceL
}

func NewHost(addr int, maxQSize int) *Host {
	return &Host{
		Addr:          addr,
		InInterfaceL:  []*NetworkInterface{NewNetworkInterface(maxQSize)},
		OutInterfaceL: []*NetworkInterface{NewNetworkInterface(maxQSize)},
		Stop:          make(chan interface{}, 1),
	}
}

// Called when printing the objects
func (h *Host) Str() string {
	return fmt.Sprintf("Host_%d", h.Addr)
}

//UdtSend creates a packet and enqueues for transmission
// dst_addr: destination address for the packet
// data_S: data being transmitted to the network layer
func (h *Host) UdtSend(dstAddr int, dataS string) {
	p := NewNetworkPacket(dstAddr, dataS)

	fmt.Printf("%s: sending packet \"%s\" on the Out interface with mtu = %d\n", h.Str(), p.ToByteS(), h.OutInterfaceL[0].Mtu)

	err := h.OutInterfaceL[0].Put(p.ToByteS(), false) // send packets always enqueued successfully
	if err != nil {
		fmt.Println("err from put in UDTsent: ", err)
	}
}

//UdtReceive receives packest from the network layer
func (h *Host) UdtReceive() {
	pktS, err := h.InInterfaceL[0].Get()
	if err == nil {
		fmt.Printf("%s: received packet \"%s\" on the In interface\n", h.Str(), pktS)
	}
}

//Run startes a routine for the host to keep receiving data
func (h *Host) Run(wg *sync.WaitGroup) {
	fmt.Println("Starting host receive routine")
	wg.Add(1)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			select {
			case <-h.Stop:
				log.Println("Host got close signal")
				fmt.Println("Ending host receive routine")
				return
			default:
				//receive data arriving in the in interface
				h.UdtReceive()
			}
		}
	}(wg)
}

//Router implements a multi-interface router described in class
// Name: friendly router nam for debugging
type Router struct {
	Stop          chan interface{}
	Name          string
	InInterfaceL  []*NetworkInterface
	OutInterfaceL []*NetworkInterface
}

//NewRouter returns a new router with given specs
// interfaceCount: the number of input and output interfaces
// maxQueSize: max queue legth (passed to interfacess)

// rounter needs to implement packet segmentation is packet is too big for interface
func NewRouter(name string, interfaceCount int, maxQueSize int) *Router {
	in := make([]*NetworkInterface, interfaceCount)
	out := make([]*NetworkInterface, interfaceCount)
	for i := 0; i < interfaceCount; i++ {
		in[i] = NewNetworkInterface(maxQueSize)
		out[i] = NewNetworkInterface(maxQueSize)

	}
	return &Router{
		Stop:          make(chan interface{}, 1),
		Name:          name,
		InInterfaceL:  in,
		OutInterfaceL: out,
	}
}

func (rt *Router) GetInInterfaceL() []*NetworkInterface {
	return rt.InInterfaceL
}

func (rt *Router) GetOutInterfaceL() []*NetworkInterface {
	return rt.OutInterfaceL
}

//Called when printing the object
func (rt *Router) Str() string {
	return fmt.Sprintf("Router_%s", rt.Name)
}

//lok through the content of incoming interfaces and forward to appropriate outgoing interfaces
func (rt *Router) forward() {
	for i, v := range rt.InInterfaceL {
		//pktS := ""

		// TRYE
		// get packet from interface i
		if pktS, err := v.Get(); err == nil {
			//fmt.Println("in routher forward, packet from Get(): ", pktS)
			// if packet exists make a forwarding decision
			p, err := FromByteS(pktS)
			if err != nil {
				log.Println("Could not get packet")
				continue
			}

			// HERE you will need to implement a lookup into the
			// forwarding table to find the appropriate outgoing interface
			// for now we assume the outgoing interface is also i
			fmt.Printf("%s: forwarding packet %s from interface %d to %d with mtu %d\n", rt.Str(), p.Str(), i, i, rt.OutInterfaceL[i].Mtu)

			if err = rt.OutInterfaceL[i].Put(p.ToByteS(), false); err != nil {
				//log.Printf("Could not put packet %s in router %s, into outInterface %d. Error: %s", p.str, rt.forward, i, err)
				log.Printf("%s: packet '%s' lost on interface %d\n", rt.Str(), i)
			}
		}
		//log.Println("no packet to forard in router")
	}
}

func (rt *Router) Run(wg *sync.WaitGroup) {
	fmt.Printf("%s: starting\n", rt.Str())

	wg.Add(1)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			select {
			case <-rt.Stop:
				log.Println("router got close signal")
				fmt.Printf("%s: Ending\n", rt.Str())
				return
			default:
				rt.forward()
			}
		}
	}(wg)
}
