package main

// An abstraction of a link between router interfaces

//Creates a link between two objects by looking up and linking node interfaces
// from_node: node from which data will be transfered
// from_intf_num: number of the interface on that node
// to_node: node to which data will be transfered
// to_intf_num: number of the interface on that node
// mtu: link maximum transmission unit
type link struct{
	fromNode string
	fromIntfNum int
	toNode string
	toIntfNum int
	mtu int
}

//Transmit a packet from the 'from' to the 'to' interface
func (l *link) txPkt(){
	pktS := l.in
}