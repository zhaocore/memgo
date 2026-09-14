package graph
package graph

import (
	"sort"
	"testing"
)

// stubList 模拟 entity store 返回结果, 用于 Build。
type stubRow struct {
	id      string
	payload map[string]any
}

func newStubLister(rows []stubRow) func(func(entityID string, linkedMemoryIDs []string)) error {
	return func(onEntity func(entityID string, linkedMemoryIDs []string)) error {
		for _, r := range rows {
			raw, _ := r.payload["linked_memory_ids"].([]any)
			var linked []string
			for _, v := range raw {
				if s, ok := v.(string); ok {
					linked = append(linked, s)
				}
			}
			onEntity(r.id, linked)
		}
		return nil
	}
}

// TestBuild verifies double adjacency table construction.
func TestBuild(t *testing.T) {
	idx := NewGraphIndex()
	lister := newStubLister([]stubRow{
		{id: "e1", payload: map[string]any{"linked_memory_ids": []any{"m1", "m2"}}},
		{id: "e2", payload: map[string]any{"linked_memory_ids": []any{"m2", "m3"}}},
	})
	if err := idx.Build(lister); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Forward check
	if got := idx.GetMemoriesForEntity("e1"); !strEq(got, []string{"m1", "m2"}) {
		t.Errorf("e1→memories: got %v", got)
	}
	if got := idx.GetMemoriesForEntity("e2"); !strEq(got, []string{"m2", "m3"}) {
		t.Errorf("e2→memories: got %v", got)
	}

	// Reverse check: m2 linked to both e1 and e2
	entities := idx.GetEntitiesForMemory("m2")
	sort.Strings(entities)
	if !strEq(entities, []string{"e1", "e2"}) {
		t.Errorf("m2→entities: got %v", entities)
	}

	stats := idx.Stats()
	if stats["entities"] != 2 || stats["memories"] != 3 || stats["total_edges"] != 4 {
		t.Errorf("stats: %v", stats)
	}
}

// TestMultiHopSearch tests BFS traversal with decay.
func TestMultiHopSearch(t *testing.T) {
	idx := NewGraphIndex()
	lister := newStubLister([]stubRow{
		{id: "e_alice", payload: map[string]any{"linked_memory_ids": []any{"m_a1", "m_a2"}}},
		{id: "e_acme", payload: map[string]any{"linked_memory_ids": []any{"m_a1", "m_a3"}}},
		{id: "e_sf", payload: map[string]any{"linked_memory_ids": []any{"m_a3", "m_s1"}}},
		{id: "e_isolated", payload: map[string]any{"linked_memory_ids": []any{"m_iso"}}},
	})
	if err := idx.Build(lister); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Hop 0 (direct): e_alice → m_a1, m_a2
	// Hop 1: m_a1 → (e_acme) → m_a3; m_a2 无其他实体
	//   boost = 0.5
	// Hop 2: m_a3 → (e_sf) → m_s1
	//   boost = 0.25
	results := idx.MultiHopSearch([]string{"e_alice"}, 3, 0.5)

	expected := map[string]float64{
		"m_a1":  1.0,
		"m_a2":  1.0,
		"m_a3":  0.5,
		"m_s1":  0.25,
		"m_iso": 0, // not reachable from e_alice
	}
	for mid, wantBoost := range expected {
		if r, ok := results[mid]; !ok {
			if wantBoost > 0 {
				t.Errorf("expected %s with boost %v, not found", mid, wantBoost)
			}
		} else if r.Boost != wantBoost {
			t.Errorf("%s: want boost %v, got %v (hop %d)", mid, wantBoost, r.Boost, r.Hop)
		}
	}
	if _, ok := results["m_iso"]; ok {
		t.Error("m_iso should not be reachable from e_alice")
	}
}

// TestOnEntityUpsert tests incremental sync.
func TestOnEntityUpsert(t *testing.T) {
	idx := NewGraphIndex()
	lister := newStubLister([]stubRow{
		{id: "e1", payload: map[string]any{"linked_memory_ids": []any{"m1"}}},
	})
	_ = idx.Build(lister)

	// Update: e1 now links to m1 & m2
	idx.OnEntityUpsert("e1", []string{"m1", "m2"}, []string{"m1"})

	if got := idx.GetMemoriesForEntity("e1"); !strEq(got, []string{"m1", "m2"}) {
		t.Errorf("after update: got %v", got)
	}
	if got := idx.GetEntitiesForMemory("m2"); !strEq(got, []string{"e1"}) {
		t.Errorf("m2→entities after update: got %v", got)
	}

	// New entity
	idx.OnEntityUpsert("e2", []string{"m3"}, nil)
	if got := idx.GetMemoriesForEntity("e2"); !strEq(got, []string{"m3"}) {
		t.Errorf("e2→memories: got %v", got)
	}
}

// TestOnMemoryRemove tests deletion cleanup.
func TestOnMemoryRemove(t *testing.T) {
	idx := NewGraphIndex()
	lister := newStubLister([]stubRow{
		{id: "e1", payload: map[string]any{"linked_memory_ids": []any{"m1", "m2"}}},
	})
	_ = idx.Build(lister)

	idx.OnMemoryRemove("m1")

	if got := idx.GetMemoriesForEntity("e1"); !strEq(got, []string{"m2"}) {
		t.Errorf("after remove m1: got %v", got)
	}
	if got := idx.GetEntitiesForMemory("m1"); len(got) != 0 {
		t.Errorf("m1 should have no entities after remove: %v", got)
	}
}

// TestMultiHopMaxHops verifies the maxHops cap.
func TestMultiHopMaxHops(t *testing.T) {
	idx := NewGraphIndex()
	lister := newStubLister([]stubRow{
		{id: "e1", payload: map[string]any{"linked_memory_ids": []any{"m1"}}},
		{id: "e2", payload: map[string]any{"linked_memory_ids": []any{"m1", "m2"}}},
		{id: "e3", payload: map[string]any{"linked_memory_ids": []any{"m2", "m3"}}},
	})
	_ = idx.Build(lister)

	// maxHops=1: only hop 0 (direct) + hop 1
	results := idx.MultiHopSearch([]string{"e1"}, 1, 0.5)
	// Hop 0: e1→m1
	// Hop 1: m1→e2→m2
	// But maxHops=1 means: hop 0 only. Wait - the code has for hop := 1; hop < maxHops; hop++.
	// So maxHops=1 gives only hop 0. maxHops=2 gives hop 0 + hop 1.

	if _, ok := results["m2"]; ok {
		t.Error("m2 should not appear with maxHops=1")
	}
	if _, ok := results["m1"]; !ok {
		t.Error("m1 should appear with maxHops=1")
	}

	// maxHops=3: reach m3
	results3 := idx.MultiHopSearch([]string{"e1"}, 3, 0.5)
	if _, ok := results3["m3"]; !ok {
		t.Error("m3 should appear with maxHops=3")
	}
}

// TestEmptyBuild verifies zero-data Build is safe.
func TestEmptyBuild(t *testing.T) {
	idx := NewGraphIndex()
	if err := idx.Build(func(onEntity func(entityID string, linkedMemoryIDs []string)) error { return nil }); err != nil {
		t.Fatalf("empty build: %v", err)
	}
	if stats := idx.Stats(); stats["entities"] != 0 || stats["memories"] != 0 {
		t.Errorf("empty build stats: %v", stats)
	}
	results := idx.MultiHopSearch([]string{"nonexistent"}, 3, 0.5)
	if len(results) != 0 {
		t.Errorf("empty graph search should return nil: %v", results)
	}
}

// TestMemoryBoosts helper
func TestMemoryBoosts(t *testing.T) {
	results := map[string]MultiHopResult{
		"m1": {MemoryID: "m1", Boost: 1.0, Hop: 0},
		"m2": {MemoryID: "m2", Boost: 0.5, Hop: 1},
	}
	boosts := MemoryBoosts(results)
	if boosts["m1"] != 1.0 || boosts["m2"] != 0.5 {
		t.Errorf("MemoryBoosts: %v", boosts)
	}
}

// strEq check sorted equality.
func strEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sa := make([]string, len(a))
	sb := make([]string, len(b))
	copy(sa, a)
	copy(sb, b)
	sort.Strings(sa)
	sort.Strings(sb)
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}