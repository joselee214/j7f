package util

import (
	"context"
	"fmt"
	"google.golang.org/grpc/metadata"
	"net"
)

func GetMetaData(ctx context.Context, meta string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if metaVal, ok := md[meta]; ok {
			return metaVal[0]
		}
	}

	return "mock-metadata-" + meta
}


func GetLocalIps() []string {
	ips := make([]string,0)
	netInterfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("net.Interfaces failed, err:", err.Error())
	}
	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()

			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						ips = append(ips,ipnet.IP.String())
					}
				}
			}
		}
	}
	//fmt.Println(ips)
	return ips
}