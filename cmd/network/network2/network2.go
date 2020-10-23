package network

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"math/rand"
	"time"
	"strings"

)

type Node interface {
	Run(*sync.WaitGroup)
	GetInInterfaceL() []*NetworkInterface
	GetOutInterfaceL() []*NetworkInterface
	Str() string
}

// Global setting variables
var dstAddrStrLength = 5
var source = rand.NewSource(time.Now().UnixNano())
var r = rand.New(source)

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

type PacketHeader struct{
	DstAddr int
	SrcAddr int
	ID	int
	Length int
	MF int
	FragOffset int
}

func PacketHeaderMF(mf int) Option{
	return func(ph *PacketHeader){
		//set packet header
		if mf >0{
			ph.MF =1
			return
		}
		ph.MF = 0
	}
}

func PacketHeaderFragOffset(offset int) Option{
	return func(ph *PacketHeader){
		ph.FragOffset = offset
	}
}

func PacketHeaderID(id int) Option{
	return func(ph *PacketHeader){
		ph.ID = id
	}
}

type Option func(ph *PacketHeader)

func NewPacketHeader(dstAddr, srcAddr, length int, opts ...Option) *PacketHeader{
	nph :=  &PacketHeader{
		DstAddr: dstAddr,
		SrcAddr: srcAddr,
		Length: length,	//30 bytes in header
	}

	//fmt.Println("newpacket len: ", nph.Length)

	for _, opt := range opts{
		opt(nph)
	}

	if nph.ID == 0{
		nph.ID = r.Intn(99999)
	}

	return nph
}


func (ph *PacketHeader) encodeHeaderToString() string{
	dstAddr  := fmt.Sprintf("%0*s", 5, strconv.Itoa(ph.DstAddr))
	srcAddr := fmt.Sprintf("%0*s", 5, strconv.Itoa(ph.SrcAddr))
	id := fmt.Sprintf("%0*s", 5, strconv.Itoa(ph.ID))
	l :=  fmt.Sprintf("%0*s", 5, strconv.Itoa(ph.Length))

	//fmt.Println("len in encode: ", l)
	mf :=fmt.Sprintf("%0*s", 5, strconv.Itoa(ph.MF))
	fo := fmt.Sprintf("%0*s", 5, strconv.Itoa(ph.FragOffset))

	return dstAddr+srcAddr+id+l+mf+fo
	//byteS := fmt.Sprintf("%0*s%s", np.DstAddrStrLength, strconv.Itoa(np.DstAddr), np.DataS)
}

func parseHeaderFromString(hd string) (*PacketHeader, error){
	if len(hd) != 30 {
		return nil, errors.New("Invalid Header Length")
	}

	// var (
	// 	dstAddr, srcAddr, id, l, mf, fo int
	// )

	dstAddr, err  := strconv.Atoi(hd[:5])
	if err != nil{
		return nil, err
	}
	srcAddr, err := strconv.Atoi(hd[5:10])
	if err != nil{
		return nil, err
	}
	id, err := strconv.Atoi(hd[10:15])
	if err != nil{
		return nil, err
	}
	
	l, err := strconv.Atoi(hd[15:20])
	if err != nil{
		return nil, err
	}

	//fmt.Println("len in parse header: ", l)

	mf, err := strconv.Atoi(hd[20:25])
	if err != nil{
		return nil, err
	}

	fo, err := strconv.Atoi(hd[25:])
	if err != nil{
		return nil, err
	}

	newPH := NewPacketHeader(dstAddr,srcAddr,l, PacketHeaderID(id), PacketHeaderMF(mf), PacketHeaderFragOffset(fo))

	return newPH, nil
}


//Implements a network layer packet
// DstAddr: address of the destination host
// DataS: packet payload
// DstAddrStrLength: packet encoding lengths
type NetworkPacket struct {
	Header *PacketHeader
	DataS            string
}


func NewNetworkPacket(dstAddr, srcAddr int, dataS string) *NetworkPacket {
	return &NetworkPacket{
		Header: NewPacketHeader(dstAddr,srcAddr, len(dataS)),
		DataS:            dataS,
	}
}

func (np *NetworkPacket) Str() string {
	return np.ToByteS()
}

//ToBytesS converts packet to a byte string for transmission over links
func (np *NetworkPacket) ToByteS() string {
	headerS := np.Header.encodeHeaderToString()
	byteS := fmt.Sprintf("%s", np.DataS)

	return headerS + byteS
	//seqNumS := fmt.Sprintf("%0*s", p.SeqNumSlength, strconv.Itoa(p.SeqNum))
}

//FromByteS builds a packet object from a byte string
// Returns error if it cannot convert addres to int
func FromByteS(byteS string) (*NetworkPacket, error) {
	//decode header
	ph, err := parseHeaderFromString(byteS[:30])
	if err != nil{
		return nil, err
	}

	//check packetlength
	if len(byteS) - 30 != ph.Length{
		fmt.Printf("packet length: %d, expeceted len: %d", len(byteS), ph.Length)
		return nil, errors.New("Packet length error")
	}

	// dstAddr, err := strconv.Atoi(byteS[0:dstAddrStrLength])
	// if err != nil {
	// 	log.Println("Error converting addr to string")
	// 	return nil, err
	// }

	dataS := byteS[30:]
	
	return &NetworkPacket{
		Header: ph,
		DataS: dataS,
	}, nil
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

	p := NewNetworkPacket(dstAddr, h.Addr, dataS)

	fmt.Printf("%s: sending packet \"%s\" on the Out interface with mtu = %d\n", h.Str(), p.ToByteS(), h.OutInterfaceL[0].Mtu)

	err := h.OutInterfaceL[0].Put(p.ToByteS(), false) // send packets always enqueued successfully
	if err != nil {
		fmt.Println("err from put in UDTsent: ", err)
	}
}

func reAssemble(packets []*NetworkPacket){
	var sb strings.Builder
	for _, p := range packets{
		sb.WriteString(p.ToByteS()[30:])
	}

	fmt.Println("Reassebled Packet: ", sb.String())
}

func orderer(toOrder chan *NetworkPacket, next chan struct{id int; offset int}){
	toReassemble := make(map[int][]*NetworkPacket)

	// for elem := range queue

	for p := range toOrder{

		// fmt.Println("got packet in orderer. id: ", p.Header.ID)
		// fmt.Println("got packet in orderer. offset: ", p.Header.FragOffset)
		// fmt.Println("got packet in orderer. Mf: ", p.Header.MF)

		
		toReassemble[p.Header.ID] = append(toReassemble[p.Header.ID], p)
		
		if p.Header.MF == 0{
			// fmt.Println("sending to assembler")
			// send to assembler
			reAssemble(toReassemble[p.Header.ID])
			delete(toReassemble,p.Header.ID)
			continue
		}

		// ask for next
		next <- struct{
			id int
			offset int
		}{
			id: p.Header.ID,
			offset: p.Header.Length + p.Header.FragOffset,
		}
	}
}

func storageHandler(toStore, toOrder chan *NetworkPacket, next chan struct{id int; offset int}){

	store := make(map[int]map[int]*NetworkPacket)
	nextMap := make(map[int]int)


	for {
		select{
			// we get something from main routine to store
		case p := <- toStore:
			if p.Header.FragOffset == nextMap[p.Header.ID]{
				// we got the next in line
				toOrder <- p
				nextMap[p.Header.ID] = p.Header.Length + p.Header.FragOffset	
			} else{
				// add to store map
				inner, ok := store[p.Header.ID]
				if !ok{
					inner = make(map[int]*NetworkPacket)
					// {p.Header.FragOffset: p}
					// [p.Header.FragOffset] = p
					store[p.Header.ID]= inner
				}
				inner[p.Header.FragOffset] = p
			}

			// we get next request from orderer routine
		case n := <- next:
			// check if we have the next one
			if v, ok := store[n.id]; ok {
				if v1, ok1 := v[n.offset]; ok1 {
					// send it
					toOrder <- v1
					// update next
					nextMap[n.id] = v1.Header.Length + v1.Header.FragOffset
				} else{
					//store next
					nextMap[n.id] = n.offset
				}
			} else {
				// store next
				nextMap[n.id] = n.offset
			}
		}
	}
}

// The flag "more fragments" is set, which is true for all fragments except the last.
// The field "fragment offset" is nonzero, which is true for all fragments except the first.
func (h *Host) fragHandler(pChan chan *NetworkPacket){
	log.Println("starting fragment handler")
	toStore := make(chan *NetworkPacket, 100)
	toOrder := make(chan *NetworkPacket, 100)
	next := make(chan struct{id int; offset int}, 100)
	//, toOrder chan *NetworkPacket, next chan struct{id; offset int

		go orderer(toOrder, next)
		go storageHandler(toStore, toOrder, next)
	

		for p := range pChan{
			// fmt.Println("got packet in frag handler, offset: ", p.Header.FragOffset)
			if p.Header.FragOffset == 0{
				// first packet
				toOrder <- p
				continue
			}

			toStore <- p
		}


	// toReassemble := make(map[int][]*NetworkPacket)

	// nextFragMap := make(map[int]int)

	// onHandMap := make(map[int][]*NetworkPacket)

}


//UdtReceive receives packest from the network layer
func (h *Host) UdtReceive(fragChan chan *NetworkPacket) {
	pktS, err := h.InInterfaceL[0].Get()
	if err == nil {
		p, err := FromByteS(pktS)
		if err != nil {
			fmt.Println("error converting packet in udtRecieve: ", err)
		}

		if p.Header.FragOffset != 0 || p.Header.MF != 0{
			// log.Println("got a fragment")
			// its a fragment
			fragChan <- p
			return
		}

		fmt.Printf("%s: received packet \"%s\" on the In interface\n", h.Str(), pktS)

		//convert to to packet
		
		// pass to reassemby routine
	}

	// reassembly
	// 1. determine if packet is fragment
	// 2. add fragments to a map based on packet id
	// 3. add to staging
	// 4. call reassemble if all packets are there


}

//Run startes a routine for the host to keep receiving data
func (h *Host) Run(wg *sync.WaitGroup) {
	fmt.Println("Starting host receive routine")
	wg.Add(1)
	fragChan := make(chan *NetworkPacket, 100)
	
	go h.fragHandler(fragChan)

	go func(fChan chan *NetworkPacket, wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			select {
			case <-h.Stop:
				// log.Println("Host got close signal")
				log.Println("Ending host receive routine")
				return
			default:
				//receive data arriving in the in interface
				h.UdtReceive(fragChan)
			}
		}
	}(fragChan, wg)
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

func (rt *Router) fragment(maxLength int, p *NetworkPacket) []*NetworkPacket{
	// fmt.Printf("in fragment, maxlen: %d\n", maxLength)
	toReturn := []*NetworkPacket{} 
	currentOffSet := 0
	data := p.DataS

	for len(data) > 0{
		// fmt.Println("len : ", len(data))
		var newData string
		// check length
		if len(data) <= maxLength-30{ // has to be enough room for header
			// this will be the last fragment
			newData = data
			fragPacketHeader := NewPacketHeader(p.Header.DstAddr,p.Header.SrcAddr,len(newData), PacketHeaderID(p.Header.ID), PacketHeaderMF(0), PacketHeaderFragOffset(currentOffSet))
			packet := &NetworkPacket{
				Header: fragPacketHeader,
				DataS: newData,
			}

			toReturn = append(toReturn, packet)
			break		
		}

		// else we need to pull out the next chunk of data
		newData = data[:(maxLength-30)]

		// update the original data
		data = data[(maxLength-30):]

		// build new packet
		fragPacketHeader := NewPacketHeader(p.Header.DstAddr,p.Header.SrcAddr,len(newData), PacketHeaderID(p.Header.ID), PacketHeaderMF(1), PacketHeaderFragOffset(currentOffSet))
		
		//updata the offset
		currentOffSet += len(newData)

		// add new packet to the return slice
		toReturn = append(toReturn, &NetworkPacket{
			Header: fragPacketHeader,
			DataS: newData,
		})

		//time.Sleep(time.Second)
	}

	//fmt.Printf("returning packet slice. len: %d\n", len(toReturn))
	return toReturn
}

// PacketHeaderID(id), PacketHeaderMF(mf), PacketHeaderFragOffset(fo))

//look through the content of incoming interfaces and forward to appropriate outgoing interfaces
func (rt *Router) forward() {
	for i, v := range rt.InInterfaceL {
		//pktS := ""

		// get packet from interface i
		if pktS, err := v.Get(); err == nil {
			//fmt.Println("in routher forward, packet from Get(): ", pktS)
			// if packet exists make a forwarding decision
			p, err := FromByteS(pktS)
			if err != nil {
				log.Println("Could not get packet: ", err)
				continue
			}

			// HERE you will need to implement a lookup into the
			// forwarding table to find the appropriate outgoing interface
			// for now we assume the outgoing interface is also i
			fmt.Printf("%s: forwarding packet %s from interface %d to %d with mtu %d\n", rt.Str(), p.Str(), i, i, rt.OutInterfaceL[i].Mtu)

			//check out going mtu
			if rt.OutInterfaceL[i].Mtu < p.Header.Length + 30{
				// call fragment
				log.Printf("Packet size: %d too big for Mtu: %d. Fragmenting packet...", p.Header.Length +30, rt.OutInterfaceL[i].Mtu)
				packetFrags := rt.fragment(rt.OutInterfaceL[i].Mtu, p)

				for _, frag := range packetFrags{
					if err = rt.OutInterfaceL[i].Put(frag.ToByteS(), false); err != nil {
						//log.Printf("Could not put packet %s in router %s, into outInterface %d. Error: %s", p.str, rt.forward, i, err)
						log.Printf("%s: packet '%s' lost on interface %d\n", rt.Str(),p.Str(), i)
					}
				}
				return
			}


			if err = rt.OutInterfaceL[i].Put(p.ToByteS(), false); err != nil {
				//log.Printf("Could not put packet %s in router %s, into outInterface %d. Error: %s", p.str, rt.forward, i, err)
				log.Printf("%s: packet '%s' lost on interface %d\n", rt.Str(), p.Str(), i)
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
				// log.Println("router got close signal")
				log.Printf("%s: Ending\n", rt.Str())
				return
			default:
				rt.forward()
			}
		}
	}(wg)
}
