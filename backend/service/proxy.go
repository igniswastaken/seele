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
		keys, err := p.fetchAllKeysFromShard(shard)
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

func (p *Proxy) fetchAllKeysFromShard(shard string) ([]string, error) {
	const pageSize = 500
	var allKeys []string
	offset := 0

	for {
		url := fmt.Sprintf("http://%s/keys?limit=%d&offset=%d", shard, pageSize, offset)
		res, err := p.client.Get(url)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("status %d", res.StatusCode)
		}

		var parsed struct {
			Keys []string `json:"keys"`
		}
		if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
			return nil, err
		}

		allKeys = append(allKeys, parsed.Keys...)
		if len(parsed.Keys) < pageSize {
			break
		}
		offset += pageSize
	}
	return allKeys, nil
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
	body, err := json.Marshal(map[string]string{"key": key, "value": value})
	if err != nil {
		return err
	}

	res, err := p.client.Post(url, "application/json", bytes.NewReader(body))
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

func (p *Proxy) HandleKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	seen := make(map[string]struct{})
	var allKeys []string
	for _, shard := range p.shards {
		keys, err := p.fetchAllKeysFromShard(shard)
		if err != nil {
			continue
		}
		for _, k := range keys {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				allKeys = append(allKeys, k)
			}
		}
	}
	if allKeys == nil {
		allKeys = []string{}
	}

	total := len(allKeys)

	offset := 0
	limit := 20
	if v := r.URL.Query().Get("offset"); v != "" {
		fmt.Sscanf(v, "%d", &offset)
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		fmt.Sscanf(v, "%d", &limit)
	}
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 500 {
		limit = 500
	}

	end := offset + limit
	if end > total {
		end = total
	}
	pageKeys := allKeys
	if offset < total {
		pageKeys = allKeys[offset:end]
	} else {
		pageKeys = []string{}
	}

	type keysResponse struct {
		Keys  []string `json:"keys"`
		Total int      `json:"total"`
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keysResponse{Keys: pageKeys, Total: total})
}

func (p *Proxy) HandleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Empty query", http.StatusBadRequest)
		return
	}

	type queryResp struct {
		Success bool        `json:"success"`
		Result  interface{} `json:"result"`
		Error   string      `json:"error,omitempty"`
	}

	var merged []interface{}
	var firstString string
	var lastErr string
	var hardErr string
	anySucess := false

	for _, shard := range p.shards {
		url := fmt.Sprintf("http://%s/query", shard)
		resp, err := p.client.Post(url, "text/plain", bytes.NewReader(body))
		if err != nil {
			lastErr = err.Error()
			continue
		}
		var qr queryResp
		if jsonErr := json.NewDecoder(resp.Body).Decode(&qr); jsonErr != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if !qr.Success {
			lastErr = qr.Error
			if strings.Contains(qr.Error, "already exists") {
				hardErr = qr.Error
			}
			continue
		}
		anySucess = true
		switch v := qr.Result.(type) {
		case []interface{}:
			merged = append(merged, v...)
		case string:
			if firstString == "" {
				firstString = v
			}
		default:
			if firstString == "" {
				marshal, _ := json.Marshal(v)
				firstString = string(marshal)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if hardErr != "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": hardErr})
		return
	}
	if !anySucess {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": lastErr})
		return
	}

	var finalResult interface{}
	if merged != nil {
		finalResult = merged
	} else {
		finalResult = firstString
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": finalResult})
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
		if res != nil {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}
		if err == nil && res.StatusCode == http.StatusOK {
			successCount++
		}
	}

	quorum := (len(shardUrls) / 2) + 1
	if successCount >= quorum && len(shardUrls) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	} else {
		http.Error(w, fmt.Sprintf("Failed to meet Write Quorum: %d/%d succeeded (need %d)", successCount, len(shardUrls), quorum), http.StatusInternalServerError)
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
	mux.HandleFunc("/keys", proxy.HandleKeys)
	mux.HandleFunc("/query", proxy.HandleQuery)
	mux.HandleFunc("/rebalance", proxy.HandleRebalance)
	mux.HandleFunc("/members", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(p2p.Members())
	})
	mux.HandleFunc("/", proxy.HandleRequest)

	fmt.Printf("Proxy started on :%s (Dynamic P2P)\n", port)
	return http.ListenAndServe(":"+port, WithCORS(mux))
}
