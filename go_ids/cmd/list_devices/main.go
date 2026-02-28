package main

import (
	"fmt"
	"log"

	"github.com/google/gopacket/pcap"
)

func main() {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Available Network Devices:")
	for _, device := range devices {
		fmt.Printf("\nName: %s\n", device.Name)
		fmt.Printf("Description: %s\n", device.Description)
		for _, address := range device.Addresses {
			fmt.Printf("- IP: %s\n", address.IP)
		}
	}
}
