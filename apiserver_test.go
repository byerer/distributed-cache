package distributed_cache

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	pb "distributed-cache/gen/v1"
	"google.golang.org/protobuf/proto"
)

func TestHTTPPool_ServeHTTP(t *testing.T) {
	// 创建一个新的缓存组用于测试
	NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			if key == "Tom" {
				return []byte("630"), nil
			}
			return nil, fmt.Errorf("key not found")
		}))

	// 创建一个测试服务器
	pool := NewHTTPPool("http://example.com")
	server := httptest.NewServer(pool)
	defer server.Close()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid request",
			path:       "/cache/scores/Tom",
			wantStatus: http.StatusOK,
			wantBody:   "630",
		},
		{
			name:       "invalid path",
			path:       "/invalid/path",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "no such group",
			path:       "/cache/invalid-group/Tom",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "no such key",
			path:       "/cache/scores/unknown",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(server.URL + tt.path)
			if err != nil {
				t.Fatal("unexpected error:", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("got status %v, want %v", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestHTTPPool_Set(t *testing.T) {
	pool := NewHTTPPool("http://localhost:8001")
	peers := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}

	// 测试设置节点
	pool.Set(peers...)

	// 验证是否正确设置了 httpGetters
	if len(pool.httpGetters) != len(peers) {
		t.Errorf("wrong number of peers: got %v, want %v",
			len(pool.httpGetters), len(peers))
	}

	// 验证每个节点的 baseURL 是否正确
	for _, peer := range peers {
		if getter, ok := pool.httpGetters[peer]; !ok {
			t.Errorf("missing getter for peer %s", peer)
		} else {
			expectedURL := peer + defaultBasePath
			if getter.baseURL != expectedURL {
				t.Errorf("wrong base URL: got %v, want %v",
					getter.baseURL, expectedURL)
			}
		}
	}
}

func TestHTTPGetter_Get(t *testing.T) {
	// 创建一个测试服务器
	handler := func(w http.ResponseWriter, r *http.Request) {
		// 模拟成功响应
		if r.URL.Path == "/cache/scores/Tom" {
			response := &pb.GetResponse{
				Value: []byte("630"),
			}
			bytes, _ := proto.Marshal(response)
			w.Write(bytes)
			return
		}
		// 模拟失败响应
		http.Error(w, "not found", http.StatusNotFound)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	getter := &httpGetter{baseURL: server.URL + "/cache/"}

	tests := []struct {
		name    string
		req     *pb.GetRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &pb.GetRequest{
				Group: "scores",
				Key:   "Tom",
			},
			wantErr: false,
		},
		{
			name: "invalid request",
			req: &pb.GetRequest{
				Group: "scores",
				Key:   "unknown",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response pb.GetResponse
			err := getter.Get(tt.req, &response)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// 测试并发访问
func TestHTTPPool_ConcurrentAccess(t *testing.T) {
	pool := NewHTTPPool("http://localhost:8001")
	peers := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}
	pool.Set(peers...)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i)
			_, _ = pool.PickPeer(key)
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 成功完成
	case <-time.After(time.Second * 5):
		t.Error("timeout waiting for concurrent access")
	}
}
