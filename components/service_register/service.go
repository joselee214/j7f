package service_register

import (
	"time"
)

type Node struct {
	NodeId   string                  `json:"node_id"`
	Services map[string]*ServiceInfo `json:"handles"`
	Address  string                  `json:"address"`
	Port     int                     `json:"port"`
	Metadata map[string]string       `json:"metadata"`
}

type ServiceInfo struct {
	Methods  []string          `json:"methods"`
	Version  string            `json:"version"`
	Metadata map[string]string `json:"metadata"`
}

type MethodInfo struct {
	Name           string
	IsClientStream bool
	IsServerStream bool
}

type ServerInfo struct {
	Methods  []MethodInfo
	Metadata interface{}
}

func NewNode(nodeId, address string, port int, metadata map[string]string) *Node {
	return &Node{
		NodeId:   nodeId,
		Services: make(map[string]*ServiceInfo),
		Address:  address,
		Port:     port,
		Metadata: metadata,
	}
}

func (n *Node) SetServices(key string, ss ...*ServiceInfo) {
	for _, s := range ss {
		n.Services[key] = s
	}
}

type Service struct {
	Key   string // unique key, e.g. "/service/foobar/1.2.3.4:8080"
	Value string // returned to subscribers, e.g. "http://1.2.3.4:8080"
	TTL   *TTLOption
}

type TTLOption struct {
	heartbeat time.Duration // e.g. time.Second * 3
	ttl       time.Duration // e.g. time.Second * 10
}

func NewTTLOption(heartbeat, ttl time.Duration) *TTLOption {
	if heartbeat <= minHeartBeatTime {
		heartbeat = minHeartBeatTime
	}
	if ttl <= heartbeat {
		ttl = 3 * heartbeat
	}
	return &TTLOption{
		heartbeat: heartbeat,
		ttl:       ttl,
	}
}
