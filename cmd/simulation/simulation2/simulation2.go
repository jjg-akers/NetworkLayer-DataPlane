package main

import (
	"fmt"
	"sync"
	"time"

	link "github.com/jjg-akers/NetworkLayer-DataPlane/cmd/link/link2"
	network "github.com/jjg-akers/NetworkLayer-DataPlane/cmd/network/network2"
)

// PART 3 28:00

//Settings
var (
	hostQueueSize   = 1000
	routerQueueSize = 1000 // 0 means unlimited
	simulationTime  = 2    // give the network sufficient time to transfer all packets before quitting
	wg              = sync.WaitGroup{}
)

func main() {

	//keep track of objects so we can kill their threads
	objectL := []interface{}{}
	// hostL := []*network.Host{}
	// routerL := []*network.Router{}

	// create network nodes
	client := network.NewHost(1, hostQueueSize)
	objectL = append(objectL, client)

	server := network.NewHost(2, hostQueueSize)
	objectL = append(objectL, server)

	routerA := network.NewRouter("A", 1, routerQueueSize)
	objectL = append(objectL, routerA)

	// create a link layer to keep track of links between network nodes
	linkLayer := link.NewLinkLayer()
	objectL = append(objectL, linkLayer)

	// add all the links
	// link paramters ...

	linkLayer.AddLink(link.NewLink(client, 0, routerA, 0, 100))
	linkLayer.AddLink(link.NewLink(routerA, 0, server, 0, 40))

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
	client.UdtSend(2, fmt.Sprintf("0123456789TTTTTTTTTTBBBBB"))

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
