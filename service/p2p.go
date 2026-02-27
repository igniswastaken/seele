package service

import (
	"fmt"
	"time"

	"github.com/hashicorp/memberlist"
)

type P2P struct {
	list *memberlist.Memberlist
}

func NewP2P(port int, nodeName string) (*P2P, error) {
	config := memberlist.DefaultLANConfig()
	// config := memberlist.DefaultLocalConfig()
	config.Name = nodeName
	config.BindAddr = "127.0.0.1"
	config.BindPort = port
	config.ProtocolVersion = memberlist.ProtocolVersionMax

	events := &P2PEvents{}
	config.Events = events

	list, err := memberlist.Create(config)
	if err != nil {
		return nil, err
	}

	local := list.LocalNode()
	fmt.Printf("P2P: Started gossip on %s:%d (%s)\n", local.Addr, local.Port, local.Name)

	return &P2P{list: list}, nil
}

func (p *P2P) Join(peers []string) error {
	if len(peers) == 0 {
		return nil
	}

	count, err := p.list.Join(peers)
	if err != nil {
		return fmt.Errorf("failed to join cluster: %v", err)
	}

	fmt.Printf("P2P: Joined cluster. Contacted %d nodes.\n", count)
	return nil
}

func (p *P2P) Members() []string {
	var members []string
	for _, node := range p.list.Members() {
		members = append(members, fmt.Sprintf("%s:%d", node.Addr, node.Port))
	}
	return members
}

func (p *P2P) GetNodeName() string {
	return p.list.LocalNode().Name
}

func (p *P2P) Leave(timeout time.Duration) error {
	if p.list != nil {
		if err := p.list.Leave(timeout); err != nil {
			return err
		}
	}
	return nil
}

func (p *P2P) Shutdown() error {
	if p.list != nil {
		if err := p.list.Shutdown(); err != nil {
			return err
		}
	}
	return nil
}

type P2PEvents struct{}

func (e *P2PEvents) NotifyJoin(node *memberlist.Node) {
	fmt.Printf("P2P Event: Member JOINED -> %s (%s)\n", node.Name, node.Addr)
}

func (e *P2PEvents) NotifyLeave(node *memberlist.Node) {
	fmt.Printf("P2P Event: Member LEFT -> %s (%s)\n", node.Name, node.Addr)
}

func (e *P2PEvents) NotifyUpdate(node *memberlist.Node) {
}
