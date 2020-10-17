package main

import (
	"fmt"
	"log"
)

//Creates a link between two objects by looking up and linking node interfaces
// from_node: network Host from which data will be transfered
// from_intf_num: number of the interface on that node
// to_node: network Host to which data will be transfered
// to_intf_num: number of the interface on that node
// mtu: link maximum transmission unit

// An abstraction of a link between router interfaces
type link struct {
	fromNode    *Host
	fromIntfNum int
	toNode      *Host
	toIntfNum   int
	inIntf      *NetworkInterface
	outIntf     *NetworkInterface
}

func newLink(fromHost *Host, fromIntfNum int, toHost *Host, toIntfNum int, mtu int) *link {
	toReturn := link{
		fromNode:    fromHost,
		fromIntfNum: fromIntfNum,
		toNode:      toHost,
		toIntfNum:   toIntfNum,
		inIntf:      fromHost.OutInterfaceL[fromIntfNum],
		outIntf:     toHost.InInterfaceL[toIntfNum],
	}

	toReturn.inIntf.Mtu = mtu
	toReturn.outIntf.Mtu = mtu

	return &toReturn
}

func (lk *link) str() string {
	return fmt.Sprint("Link %s-%d to %s-%d", lk.fromNode.str, lk.fromIntfNum, lk.toNode.str, lk.toIntfNum)
}

//txPkt transmits a packet from the 'from' interface to the 'to' interface
func (lk *link) txPkt() {
	pktS, err := lk.inIntf.get()
	if err != nil {
		return // no packet to transmit
	}
	if len(pktS) > lk.inIntf.Mtu {
		fmt.Printf("%s: packet '%s' length greater than the From interface MTU (%d)\n", lk.str(), pktS, lk.inIntf.Mtu)
		return // packet too big, return without transmitting
	}

	if len(pktS) > lk.outIntf.Mtu {
		fmt.Printf("%s: packet '%s' length greater than the To interface MTU (%d)\n", lk.str(), pktS, lk.outIntf.Mtu)
		return // packet too big, return without transmitting
	}

	// Transmit packet
	if err := lk.outIntf.put(pktS, false); err != nil {
		fmt.Printf("%s: packet lost\n", lk.str())
		return
	}

	fmt.Printf("%s: transmitting packet '%s'\n", lk.str(), pktS)
}

//LinkL is a list of links in the network
// Stop is for routine termination
type LinkLayer struct {
	LinkL []*link
	Stop  bool
}

//Str returns the name of the network layer
func (ll *LinkLayer) Str() string {
	return "Network"
}

//AddLink add a link to the network
func (ll *LinkLayer) AddLink(lk *link) {
	ll.LinkL = append(ll.LinkL, lk)
}

//Transfer transfers a packet across all links
func (ll *LinkLayer) Transfer() {
	for _, link := range ll.LinkL {
		link.txPkt()
	}
}

//Run starts a routing for the network to keep transmitting data across links
func (ll *LinkLayer) Run() {
	log.Println("LinkLayer 'Run' routine starting")

	for {
		// transfer one packet on all the links
		ll.Transfer()

		// terminate
		if ll.Stop {
			log.Println("LinkLayer 'Run' routine ending")
			return
		}
	}
}
