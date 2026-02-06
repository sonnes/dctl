package compose

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolveOrder_NoDeps(t *testing.T) {
	services := map[string]Service{
		"a": {Image: "alpine"},
		"b": {Image: "alpine"},
		"c": {Image: "alpine"},
	}

	order, err := ResolveOrder(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("got %v, want %v", order, want)
	}
}

func TestResolveOrder_LinearChain(t *testing.T) {
	services := map[string]Service{
		"a": {
			Image: "alpine",
			DependsOn: map[string]DependsOnCondition{
				"b": {Condition: "service_started"},
			},
		},
		"b": {
			Image: "alpine",
			DependsOn: map[string]DependsOnCondition{
				"c": {Condition: "service_started"},
			},
		},
		"c": {Image: "alpine"},
	}

	order, err := ResolveOrder(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"c", "b", "a"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("got %v, want %v", order, want)
	}
}

func TestResolveOrder_Diamond(t *testing.T) {
	services := map[string]Service{
		"a": {
			Image: "alpine",
			DependsOn: map[string]DependsOnCondition{
				"b": {Condition: "service_started"},
				"c": {Condition: "service_started"},
			},
		},
		"b": {
			Image: "alpine",
			DependsOn: map[string]DependsOnCondition{
				"d": {Condition: "service_started"},
			},
		},
		"c": {
			Image: "alpine",
			DependsOn: map[string]DependsOnCondition{
				"d": {Condition: "service_started"},
			},
		},
		"d": {Image: "alpine"},
	}

	order, err := ResolveOrder(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"d", "b", "c", "a"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("got %v, want %v", order, want)
	}
}

func TestResolveOrder_CycleDetection(t *testing.T) {
	services := map[string]Service{
		"a": {
			Image: "alpine",
			DependsOn: map[string]DependsOnCondition{
				"b": {Condition: "service_started"},
			},
		},
		"b": {
			Image: "alpine",
			DependsOn: map[string]DependsOnCondition{
				"a": {Condition: "service_started"},
			},
		},
	}

	_, err := ResolveOrder(services)
	if err == nil {
		t.Fatal("expected error for cycle, got nil")
	}
	if !strings.Contains(err.Error(), "dependency cycle detected") {
		t.Errorf("expected cycle error, got: %v", err)
	}
}

func TestResolveOrder_Deterministic(t *testing.T) {
	services := map[string]Service{
		"api": {
			Image: "alpine",
			DependsOn: map[string]DependsOnCondition{
				"db":    {Condition: "service_started"},
				"cache": {Condition: "service_started"},
			},
		},
		"web": {
			Image: "alpine",
			DependsOn: map[string]DependsOnCondition{
				"api": {Condition: "service_started"},
			},
		},
		"db":    {Image: "postgres"},
		"cache": {Image: "redis"},
		"worker": {
			Image: "alpine",
			DependsOn: map[string]DependsOnCondition{
				"db":    {Condition: "service_started"},
				"cache": {Condition: "service_started"},
			},
		},
	}

	first, err := ResolveOrder(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := 0; i < 10; i++ {
		order, err := ResolveOrder(services)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if !reflect.DeepEqual(order, first) {
			t.Fatalf("iteration %d: got %v, want %v", i, order, first)
		}
	}
}
