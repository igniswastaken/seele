package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Proxy struct {
	shards []string
	ring   *HashRing
	client *http.Client
}

func NewProxy(shards []string) *Proxy {
	ring := NewHashRing(50)

	for _, shard := range shards {
		ring.AddNode(shard)
	}

	return &Proxy{
		shards: shards,
		ring:   ring,
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 1000,
				IdleConnTimeout:     90 * time.Second,
			},
			Timeout: 10 * time.Second,
		},
	}
}

func (p *Proxy) GetShard(key string) string {
	return p.ring.GetNode(key)
}

func (p *Proxy) Rebalance() {
	for _, shard := range p.shards {
		keys, err := p.fetchKeysFromShard(shard)
		if err != nil {
			fmt.Printf("Failed to fetch keys from %s: %v\n", shard, err)
			continue
		}

		for _, key := range keys {
			correctOwner := p.ring.GetNode(key)
			if correctOwner != shard {
				val, err := p.fetchValue(shard, key)
				if err != nil {
					fmt.Printf("Failed to get value for %s: %v\n", key, err)
					continue
				}

				if err := p.setValue(correctOwner, key, val); err != nil {
					fmt.Printf("Failed to move %s to %s: %v\n", key, correctOwner, err)
					continue
				}

				if err := p.deleteKey(shard, key); err != nil {
					fmt.Printf("Failed to delete %s from %s: %v\n", key, shard, err)
				}

				fmt.Printf("Moved %s from %s to %s\n", key, shard, correctOwner)
			}
		}

		if err := p.compactShard(shard); err != nil {
			fmt.Printf("Failed to compact %s: %v\n", shard, err)
		}
	}
}

func (p *Proxy) fetchKeysFromShard(shard string) ([]string, error) {
	url := fmt.Sprintf("http://%s/keys", shard)
	res, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", res.StatusCode)
	}

	var keys []string
	if err := json.NewDecoder(res.Body).Decode(&keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (p *Proxy) fetchValue(shard, key string) (string, error) {
	url := fmt.Sprintf("http://%s/get?key=%s", shard, key)
	res, err := p.client.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", res.StatusCode)
	}

	var parsed struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", err
	}
	return parsed.Value, nil
}

func (p *Proxy) setValue(shard, key, value string) error {
	url := fmt.Sprintf("http://%s/set", shard)
	payload := fmt.Sprintf(`{"key":"%s", "value":"%s"}`, key, value)

	res, err := p.client.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", res.StatusCode)
	}
	return nil
}

func (p *Proxy) deleteKey(shard, key string) error {
	url := fmt.Sprintf("http://%s/delete?key=%s", shard, key)
	req, _ := http.NewRequest(http.MethodDelete, url, nil)

	res, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", res.StatusCode)
	}
	return nil
}

func (p *Proxy) compactShard(shard string) error {
	url := fmt.Sprintf("http://%s/compact", shard)
	res, err := p.client.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", res.StatusCode)
	}
	return nil
}

func (p *Proxy) HandleRequest(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Shard key required", http.StatusBadRequest)
		return
	}

	shardUrls := p.ring.GetNodes(key, 2)
	if len(shardUrls) == 0 {
		http.Error(w, "No active shards available", http.StatusServiceUnavailable)
		return
	}

	if r.Method == http.MethodGet {
		p.HandleRead(w, r, shardUrls)
	} else {
		p.HandleWrite(w, r, shardUrls)
	}
}

func (p *Proxy) HandleRead(w http.ResponseWriter, r *http.Request, shardUrls []string) {
	for _, shardUrl := range shardUrls {
		targetUrl := fmt.Sprintf("http://%s%s", shardUrl, r.URL.RequestURI())

		req, _ := http.NewRequest(r.Method, targetUrl, nil)
		for k, v := range r.Header {
			req.Header[k] = v
		}
		res, err := p.client.Do(req)
		if err == nil && res.StatusCode == http.StatusOK {
			defer res.Body.Close()
			for k, v := range res.Header {
				w.Header()[k] = v
			}
			w.WriteHeader(res.StatusCode)
			io.Copy(w, res.Body)
			return
		}
		if res != nil {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}
	}
	http.Error(w, "Failed to read from any replica", http.StatusBadGateway)
}

func (p *Proxy) HandleWrite(w http.ResponseWriter, r *http.Request, shardUrls []string) {
	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	successCount := 0
	for _, shardUrl := range shardUrls {
		targetUrl := fmt.Sprintf("http://%s%s", shardUrl, r.URL.RequestURI())

		req, _ := http.NewRequest(r.Method, targetUrl, bytes.NewReader(bodyBytes))
		for k, v := range r.Header {
			req.Header[k] = v
		}
		res, err := p.client.Do(req)
		if err == nil && res.StatusCode == http.StatusOK {
			successCount++
		} else {
			if res != nil {
				res.Body.Close()
			}
		}

		if res != nil {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}
	}

	if successCount == len(shardUrls) && len(shardUrls) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	} else {
		http.Error(w, fmt.Sprintf("Failed to meet Write Quorum: %d/%d succeeded", successCount, len(shardUrls)), http.StatusInternalServerError)
	}
}

func (p *Proxy) HandleRebalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	go p.Rebalance()

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Rebalancing started...")
}

func StartProxy(port string, joinAddr string) error {
	proxy := NewProxy([]string{})

	var p2pPort int
	fmt.Sscanf(port, "%d", &p2pPort)
	p2pPort += 1000

	p2p, err := NewP2P(p2pPort, "proxy")
	if err != nil {
		return fmt.Errorf("Failed to start proxy p2p: %v", err)
	}

	if joinAddr != "" {
		if err := p2p.Join([]string{joinAddr}); err != nil {
			return fmt.Errorf("Failed to join cluster: %v", err)
		}
	}

	go func() {
		for {
			time.Sleep(1 * time.Second)
			members := p2p.Members()

			var activeShards []string
			for _, member := range members {
				parts := strings.Split(member, ":")
				if len(parts) != 2 {
					continue
				}

				ip := parts[0]
				var gPort int
				fmt.Sscanf(parts[1], "%d", &gPort)

				if gPort == p2pPort {
					continue
				}

				httpPort := gPort - 1000
				nodeAddr := fmt.Sprintf("%s:%d", ip, httpPort)
				activeShards = append(activeShards, nodeAddr)
			}

			if len(activeShards) > 0 {
				proxy.ring.SetNodes(activeShards)
				proxy.shards = activeShards
			} else {
				fmt.Printf("Proxy Debug: 0 active shards! P2P Members: %v\n", members)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", proxy.HandleRequest)
	mux.HandleFunc("/rebalance", proxy.HandleRebalance)
	mux.HandleFunc("/members", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(p2p.Members())
	})

	fmt.Printf("Proxy started on :%s (Dynamic P2P)\n", port)
	return http.ListenAndServe(":"+port, mux)
}
