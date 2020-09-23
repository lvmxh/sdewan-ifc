package main

import (
	"context"
	"flag"
	"fmt"
	"sdewan.akraino.org/sdewan/openwrt"
)

func main() {
	fmt.Println("hello world")
	ctx := context.Background()
	fmt.Printf("ctx: %v", ctx)
	podip := flag.String("ip", "127.0.0.1", "A string for pod ip.")
	flag.Parse()
	fmt.Println("OpenWrt IP:", *podip)
	openwrtClient := openwrt.NewOpenwrtClient(*podip, "root", "")
	fmt.Println("openwrtClient:", openwrtClient)
	service := openwrt.InterfaceClient{OpenwrtClient: openwrtClient}
	fcs, _ := service.GetAvailableInterfaces()
	fmt.Println("AvailableInterfaces:", fcs)
	fmt.Println("Interface 0, name:", fcs.Interfaces[0].Name)
}
