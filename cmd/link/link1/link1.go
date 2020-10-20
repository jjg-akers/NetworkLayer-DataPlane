package link

import (
	"fmt"
	"log"
	"sync"

	network "github.com/jjg-akers/NetworkLayer-DataPlane/cmd/network/network1"
)

//Creates a link between two objects by looking up and linking node interfaces
// from_node: network Host from which data will be transfered
// from_intf_num: number of the interface on that node
// to_node: network Host to which data will be transfered
// to_intf_num: number of the interface on that node
// mtu: link maximum transmission unit

//Link is An abstraction of a link between router interfaces
type Link struct {
	FromNode    network.Node
	FromIntfNum int
	ToNode      network.Node
	ToIntfNum   int
	InIntf      *network.NetworkInterface
	OutIntf     *network.NetworkInterface
}

//NewLink return a new link
func NewLink(fromHost network.Node, fromIntfNum int, toHost network.Node, toIntfNum int, mtu int) *Link {
	toReturn := Link{
		FromNode:    fromHost,
		FromIntfNum: fromIntfNum,
		ToNode:      toHost,
		ToIntfNum:   toIntfNum,
		InIntf:      fromHost.GetOutInterfaceL()[fromIntfNum],
		OutIntf:     toHost.GetInInterfaceL()[toIntfNum],
	}

	toReturn.InIntf.Mtu = mtu
	toReturn.OutIntf.Mtu = mtu

	return &toReturn
}

func (lk *Link) str() string {
	return fmt.Sprintf("Link %s-%d to %s-%d", lk.FromNode.Str(), lk.FromIntfNum, lk.ToNode.Str(), lk.ToIntfNum)
}

//txPkt transmits a packet from the 'from' interface to the 'to' interface
func (lk *Link) TxPkt() {
	//fmt.Println("tranmit packet called")
	pktS, err := lk.InIntf.Get()
	if err != nil {
		//log.Println("no packet to transmit")
		//time.Sleep(time.Second)

		return // no packet to transmit
	}
	if len(pktS) > lk.InIntf.Mtu {
		fmt.Printf("%s: packet '%s' length greater than the From interface MTU (%d)\n", lk.str(), pktS, lk.InIntf.Mtu)
		return // packet too big, return without transmitting
	}

	if len(pktS) > lk.OutIntf.Mtu {
		fmt.Printf("%s: packet '%s' length greater than the To interface MTU (%d)\n", lk.str(), pktS, lk.OutIntf.Mtu)
		return // packet too big, return without transmitting
	}

	// Transmit packet
	if err := lk.OutIntf.Put(pktS, false); err != nil {
		fmt.Printf("%s: packet lost\n", lk.str())
		return
	}

	fmt.Printf("%s: transmitting packet '%s'\n", lk.str(), pktS)
	//time.Sleep(time.Second)
}

//LinkLayer is a list of links in the network
// Stop is for routine termination
type LinkLayer struct {
	LinkL []*Link
	Stop  chan interface{}
}

func NewLinkLayer() *LinkLayer {
	return &LinkLayer{
		LinkL: []*Link{},
		Stop:  make(chan interface{}, 1),
	}
}

//Str returns the name of the network layer
func (ll *LinkLayer) Str() string {
	return "Network"
}

//AddLink add a link to the network
func (ll *LinkLayer) AddLink(lk *Link) {
	ll.LinkL = append(ll.LinkL, lk)
}

//Transfer transfers a packet across all links
func (ll *LinkLayer) Transfer() {
	for _, link := range ll.LinkL {
		link.TxPkt()
	}
}

//Run starts a routing for the network to keep transmitting data across links
func (ll *LinkLayer) Run(wg *sync.WaitGroup) {
	log.Println("LinkLayer 'Run' routine starting")
	wg.Add(1)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			select {
			case <-ll.Stop:
				log.Println("linklayer got close signal")
				log.Println("LinkLayer 'Run' routine ending")
				return
			default:
				// transfer one packet on all the links
				ll.Transfer()
			}
		}
	}(wg)
}
