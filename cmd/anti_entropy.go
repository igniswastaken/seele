package cmd

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/zerothy/seele/model"
	"github.com/zerothy/seele/service"
)

func (s *Server) StartAntiEntropy(p2p *service.P2P, myPort string) {
	go func() {
		for {
			time.Sleep(5 * time.Second)

			members := p2p.Members()
			if len(members) <= 1 {
				continue
			}

			var peers []string
			for _, member := range members {
				if !strings.Contains(member, p2p.GetNodeName()) && !strings.Contains(member, "proxy") {
					parts := strings.Split(member, ":")
					if len(parts) == 2 {
						var gPort int
						fmt.Sscanf(parts[1], "%d", &gPort)
						httpPort := gPort - 1000
						peers = append(peers, fmt.Sprintf("%s:%d", parts[0], httpPort))
					}
				}
			}

			if len(peers) == 0 {
				continue
			}
			randomPeer := peers[rand.Intn(len(peers))]
			fmt.Printf("Anti-Entropy: Syncing with %s...\n", randomPeer)

			myRoot := s.store.GetMerkleRoot()
			peerRoot, err := getPeerMerkleRoot(randomPeer)
			if err != nil || peerRoot == "" || myRoot == "" || myRoot == peerRoot {
				continue
			}

			s.healFromPeer(randomPeer, myPort)
		}
	}()
}

func (s *Server) healFromPeer(peerAddr string, myPort string) {
	res, err := http.Get(fmt.Sprintf("http:%s/keys", peerAddr))
	if err != nil {
		return
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return
	}

	var keys []string
	if err := json.NewDecoder(res.Body).Decode(&keys); err != nil {
		return
	}

	myKeys := s.store.Keys()
	myKeyMap := make(map[string]bool)

	for _, key := range myKeys {
		myKeyMap[key] = true
	}

	myID := fmt.Sprintf("127.0.0.1:%s", myPort)
	for _, key := range keys {
		if !myKeyMap[key] {
			replicas := s.ring.GetNodes(key, 2)
			shouldIOwnThis := slices.Contains(replicas, myID)
			if shouldIOwnThis {
				fmt.Printf("Anti-Entropy: Key %s belongs to me but is missing! Healing...\n", key)

				getRes, err := http.Get(fmt.Sprintf("http://%s/get?key=%s", peerAddr, key))
				if err == nil && getRes.StatusCode == http.StatusOK {
					var data model.GetResponse
					if err := json.NewDecoder(getRes.Body).Decode(&data); err == nil {
						s.store.Set(key, data.Value)
					}
					getRes.Body.Close()
				}
			}
		}
	}
}

func getPeerMerkleRoot(peerAddr string) (string, error) {
	res, err := http.Get(fmt.Sprintf("http://%s/sync/merkle", peerAddr))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("peer returned status %d", res.StatusCode)
	}

	var data map[string]string
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return "", err
	}

	return data["root_hash"], nil
}
