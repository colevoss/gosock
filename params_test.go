package gosock

import (
	"fmt"
	"testing"
)

func TestAdd(t *testing.T) {
	params := Params{}

	params.Add("test", "value")

	if len(params) != 1 {
		t.Errorf("Params should have length of 1. Has: %d", len(params))
	}
}

func TestGet(t *testing.T) {
	params := Params{}

	tests := []struct {
		k, v string
		want string
	}{
		{"test1", "value1", "value1"},
		{"test2", "value2", "value2"},
	}

	for _, pData := range tests {
		params.Add(pData.k, pData.v)
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("%s:%s", tt.k, tt.v)
		t.Run(testName, func(t *testing.T) {
			param, ok := params.Get(tt.k)

			if !ok {
				t.Errorf("Ok should be true when getting an existing param")
			}

			if param != tt.v {
				t.Errorf("got %s, wanted %s", param, tt.want)
			}
		})
	}
}

func TestBadGet(t *testing.T) {
	params := Params{}

	badParam := "notOk"
	param, ok := params.Get(badParam)

	if ok {
		t.Errorf("Ok should not be true when param does not exist: %s", badParam)
	}

	if param != "" {
		t.Errorf("Get should return empty string if param does not exist. Got %s", param)
	}
}

func TestReset(t *testing.T) {
	params := Params{}

	params.Add("test", "value")

	params.Reset()

	if len(params) != 0 {
		t.Errorf("Params should have a length of zero. Got %d", len(params))
	}
}

func BenchmarkGet(b *testing.B) {
	params := Params{}

	for i := 0; i < 100; i++ {
		params.Add("test"+string(rune(i)), "bench")
	}

	for i := 0; i < b.N; i++ {
		params.Get("test20")
	}
}
