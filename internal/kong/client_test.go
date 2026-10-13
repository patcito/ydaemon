package kong

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Kong caps an unpaginated vaults query at 100 rows, so the client must keep asking for
// pages until one comes back short.
func TestFetchVaultsForChainWalksEveryPage(t *testing.T) {
	const total = KongVaultsPageSize + 7
	var offsets []int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GraphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		limit := int(req.Variables["limit"].(float64))
		offset := int(req.Variables["offset"].(float64))
		if chainID := int(req.Variables["chainId"].(float64)); chainID != 1 {
			t.Fatalf("chainId = %d, want 1", chainID)
		}
		offsets = append(offsets, offset)

		vaults := []KongVault{}
		for i := offset; i < total && i < offset+limit; i++ {
			vaults = append(vaults, KongVault{Address: fmt.Sprintf("0x%040x", i+1), ChainID: 1})
		}
		data, _ := json.Marshal(VaultsResponse{Vaults: vaults})
		_ = json.NewEncoder(w).Encode(GraphQLResponse{Data: data})
	}))
	defer server.Close()

	client := &Client{httpClient: server.Client(), endpoint: server.URL}
	vaults, err := client.FetchVaultsForChain(context.Background(), 1)
	if err != nil {
		t.Fatalf("FetchVaultsForChain: %v", err)
	}
	if len(vaults) != total {
		t.Fatalf("got %d vaults, want %d", len(vaults), total)
	}
	if len(offsets) != 2 || offsets[0] != 0 || offsets[1] != KongVaultsPageSize {
		t.Fatalf("offsets = %v, want [0 %d]", offsets, KongVaultsPageSize)
	}
}
