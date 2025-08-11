package config

import "testing"

func TestConfigStoreDeepCopy(t *testing.T) {
	original := []Model{
		{
			ID:           "m1",
			Capabilities: map[string]float64{"a": 1},
			Tags:         []string{"tag1"},
		},
	}
	store := NewConfigStore()
	store.SetModels(original)

	// Mutate the original slice after storing to ensure the store isn't affected.
	original[0].Capabilities["a"] = 0
	original[0].Tags[0] = "modified"

	got := store.GetModels()
	if got[0].Capabilities["a"] != 1 {
		t.Fatalf("capability modified via original slice: %v", got[0].Capabilities["a"])
	}
	if got[0].Tags[0] != "tag1" {
		t.Fatalf("tag modified via original slice: %v", got[0].Tags[0])
	}

	// Mutate the returned slice to ensure the store still isn't affected.
	got[0].Capabilities["a"] = 0.5
	got[0].Tags[0] = "changed"

	again := store.GetModels()
	if again[0].Capabilities["a"] != 1 {
		t.Fatalf("capability modified via returned slice: %v", again[0].Capabilities["a"])
	}
	if again[0].Tags[0] != "tag1" {
		t.Fatalf("tag modified via returned slice: %v", again[0].Tags[0])
	}
}
