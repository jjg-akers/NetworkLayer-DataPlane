package main

import (
	"fmt"
	"sync"
	"time"

	link "github.com/jjg-akers/NetworkLayer-DataPlane/cmd/link/link3"
	network "github.com/jjg-akers/NetworkLayer-DataPlane/cmd/network/network3"
)

// PART 3 28:00

//Settings
var (
	hostQueueSize   = 1000
	routerQueueSize = 1000 // 0 means unlimited
	simulationTime  = 4    // give the network sufficient time to transfer all packets before quitting
	wg              = sync.WaitGroup{}
)

func main() {

	//keep track of objects so we can kill their threads
	objectL := []interface{}{}
	// hostL := []*network.Host{}
	// routerL := []*network.Router{}

	// create network nodes
	host1 := network.NewHost(1, hostQueueSize)
	objectL = append(objectL, host1)

	host2 := network.NewHost(2, hostQueueSize)
	objectL = append(objectL, host2)

	host3 := network.NewHost(3, hostQueueSize)
	objectL = append(objectL, host3)

	host4 := network.NewHost(4, hostQueueSize)
	objectL = append(objectL, host4)

	// routerA should have two in, two out interfaces
	routerA := network.NewRouter("A", 2, routerQueueSize)
	objectL = append(objectL, routerA)

	// RouterB and C should have only 1 in/ out interface
	routerB := network.NewRouter("B", 1, routerQueueSize)
	objectL = append(objectL, routerB)

	routerC := network.NewRouter("C", 1, routerQueueSize)
	objectL = append(objectL, routerC)

	// routerD should have two in/ out interfaces
	routerD := network.NewRouter("D", 2, routerQueueSize)
	objectL = append(objectL, routerD)

	// create a link layer to keep track of links between network nodes
	linkLayer := link.NewLinkLayer()
	objectL = append(objectL, linkLayer)

	// add all the links
	// link paramters ...

	// Add 'in' links to RouterA
	//	host1 -> interface0
	linkLayer.AddLink(link.NewLink(host1, 0, routerA, 0, 100))
	//	host2 -> interface1
	linkLayer.AddLink(link.NewLink(host2, 0, routerA, 1, 100))

	// Router A -> RouterB
	//	RouterA out 0 -> RouterB in 0
	linkLayer.AddLink(link.NewLink(routerA, 0, routerB, 0, 40))

	//Router A -> RouterC
	//	RouterA out 1 -> RouterC in 0
	linkLayer.AddLink(link.NewLink(routerA, 1, routerC, 0, 40))

	//Router B -> RouterD
	//	RouterB out 0 -> RouterD in 0
	linkLayer.AddLink(link.NewLink(routerB, 0, routerD, 0, 40))

	//RouterC -> RouterD
	//	RouterC out 0 -> RouterD in 1
	linkLayer.AddLink(link.NewLink(routerC, 0, routerD, 1, 40))

	//RouterD -> Host3
	//	RouterD out 0 -> Host3 in 0
	linkLayer.AddLink(link.NewLink(routerD, 0, host3, 0, 40))

	//RouterD -> Host4
	//	RouterD out 1 -> Host4 in 0
	linkLayer.AddLink(link.NewLink(routerD, 1, host4, 0, 40))

	//RouterB -> RouterD
	// start all the objects
	for _, obj := range objectL {
		switch v := obj.(type) {
		case *network.Host:
			v.Run(&wg)
		case *network.Router:
			v.Run(&wg)
		case *link.LinkLayer:
			v.Run(&wg)

		default:
			fmt.Printf("type: %T, value: %v\n", v, v)
			fmt.Println("default")
		}
	}

	// create some events
	// i := 0
	// for i < 3 {
	// 	client.UdtSend(2, fmt.Sprintf("Sample data %d", i))
	// 	i++
	// }
	host2.UdtSend(2, fmt.Sprintf("0123456789TTTTTTTTTTBBBBB"))

	// give the network sufficient time to transfer all packets before quitting
	time.Sleep(time.Duration(simulationTime) * time.Second)

	// join all thread
	for _, obj := range objectL {
		switch v := obj.(type) {
		case *network.Host:
			v.Stop <- true
		case *network.Router:
			v.Stop <- true
		case *link.LinkLayer:
			v.Stop <- true

		default:
			fmt.Printf("type: %T, value: %v\n", v, v)
			fmt.Println("default")
		}
	}

	// send the stop signal and wait
	wg.Wait()

	fmt.Println("Program Exiting")
	// need to wait here for routines
}
