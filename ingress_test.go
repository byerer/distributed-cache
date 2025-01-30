package distributed_cache

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	pb "distributed-cache/gen/v1"
)

// 模拟数据源
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGet(t *testing.T) {
	// 创建一个 loadCounts 统计回源次数
	loadCounts := make(map[string]int, len(db))

	g := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			// 模拟回源
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key]++
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	// 测试基础获取功能
	t.Run("get existing key", func(t *testing.T) {
		if view, err := g.Get("Tom"); err != nil || view.String() != db["Tom"] {
			t.Fatalf("cache hit Tom failed")
		}
		// 再次获取，此时应该命中缓存
		if _, err := g.Get("Tom"); err != nil {
			t.Fatalf("cache hit Tom failed")
		}
		// 验证回源次数
		if loadCounts["Tom"] != 1 {
			t.Fatalf("cache miss count wrong")
		}
	})

	t.Run("get nonexistent key", func(t *testing.T) {
		if _, err := g.Get("unknown"); err == nil {
			t.Fatalf("get nonexistent key should get error")
		}
	})
}

func TestGetGroup(t *testing.T) {
	groupName := "scores"
	NewGroup(groupName, 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			return []byte(key), nil
		}))

	t.Run("group exists", func(t *testing.T) {
		if group := GetGroup(groupName); group == nil {
			t.Fatalf("group %s not exist", groupName)
		}
	})

	t.Run("group not exists", func(t *testing.T) {
		if group := GetGroup("unknown"); group != nil {
			t.Fatalf("expect nil, but got %v", group)
		}
	})
}

// 模拟 PeerPicker
type mockPeerPicker struct {
	peer PeerGetter
}

func (m *mockPeerPicker) PickPeer(key string) (PeerGetter, bool) {
	return m.peer, true
}

// 模拟 PeerGetter
type mockPeerGetter struct {
	mockData map[string][]byte
}

func (m *mockPeerGetter) Get(in *pb.GetRequest, out *pb.GetResponse) error {
	if v, ok := m.mockData[in.Key]; ok {
		out.Value = v
		return nil
	}
	return fmt.Errorf("key %s not found", in.Key)
}

func TestGetFromPeer(t *testing.T) {
	// 创建测试数据
	mockData := map[string][]byte{
		"key1": []byte("value1"),
	}

	// 创建缓存组
	g := NewGroup("test", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			return nil, fmt.Errorf("should not reach local getter")
		}))

	// 注册 peer
	peer := &mockPeerGetter{mockData: mockData}
	g.RegisterPeers(&mockPeerPicker{peer: peer})

	t.Run("get from peer success", func(t *testing.T) {
		if view, err := g.Get("key1"); err != nil {
			t.Fatalf("failed to get from peer: %v", err)
		} else if !reflect.DeepEqual(view.ByteSlice(), mockData["key1"]) {
			t.Fatalf("value mismatch")
		}
	})

	t.Run("get from peer failed", func(t *testing.T) {
		if _, err := g.Get("unknown"); err == nil {
			t.Fatalf("expect error when key not exist")
		}
	})
}

// 测试并发安全性
func TestConcurrentGet(t *testing.T) {
	g := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			return []byte(key), nil
		}))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			view, err := g.Get("key")
			if err != nil {
				t.Error(err)
			}
			if v := view.String(); v != "key" {
				t.Errorf("expected value key, got %s", v)
			}
		}()
	}
	wg.Wait()
}
