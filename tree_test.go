package gosock

import (
	"log"
	"testing"
	"time"
)

type TestRouter struct {
}

func makeHub() *Hub {
	return NewHub(NewPool(1, 1, time.Second))
}

func (tr *TestRouter) Join(c *Channel)          {}
func (tr *TestRouter) RegisterRouter(c *Router) {}

func TestLookup(t *testing.T) {
	tree := NewTree()

	tests := []struct {
		path, testPath string
		param          string
		hasParams      bool
	}{
		{"test", "test", "", false},
		{"test.{param}.one", "test.1.one", "1", true},
		{"test.{param}.two", "test.2.two", "2", true},
		{"test.{param}.three", "test.3.three", "3", true},
		{"other", "other", "", false},
		{"other.test.{param}.one", "other.test.1.one", "1", true},
		{"other.test.{param}.two", "other.test.2.two", "2", true},
		{"other.test.{param}.three", "other.test.3.three", "3", true},
	}

	for _, data := range tests {
		mc := NewRouter(data.testPath, makeHub())
		tree.Add(data.path, mc)
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			match, params := tree.Lookup(tt.testPath)

			if match == nil || match.Channel == nil {
				t.Errorf("Should have found match %s: %s", tt.path, tt.testPath)
			}

			if !tt.hasParams {
				return
			}

			if params == nil {
				t.Errorf("Should have params")
			}

			param, ok := params.Get("param")

			if !ok {
				t.Errorf("Params get should be ok")
			}

			if param != tt.param {
				t.Errorf("Param should equal %s. Got %s", tt.param, param)
			}
		})
	}
}

func TestLookupMultiParam(t *testing.T) {
	tree := NewTree()

	router := NewRouter("test", makeHub())

	tree.Add("test.{paramone}.multi.{paramtwo}.param", router)

	_, params := tree.Lookup("test.1.multi.2.param")

	if params == nil {
		t.Errorf("Params should not be nil")
	}

	paramOne, _ := params.Get("paramone")
	paramTwo, _ := params.Get("paramtwo")

	if paramOne != "1" {
		t.Errorf("Param one should equal 1. Got %s", paramOne)
	}

	if paramTwo != "2" {
		t.Errorf("Param two should equal 1. Got %s", paramTwo)
	}

	log.Printf("%s %s", paramOne, paramTwo)
}

func TestEndingParam(t *testing.T) {
	tree := NewTree()
	router := NewRouter("test", makeHub())

	tree.Add("test.{param}", router)

	_, params := tree.Lookup("test.1")

	if params == nil {
		t.Errorf("Params should not be nil")
	}

	param, ok := params.Get("param")

	if !ok {
		t.Errorf("Param %s should not be nil", "param")
	}

	if param != "1" {
		t.Errorf("Param should equal 1. Got %s", param)
	}
}

func TestLookupNotFound(t *testing.T) {
	tree := NewTree()

	mc := NewRouter("test", makeHub())

	tree.Add("test", mc)
	tree.Add("test_another", mc)

	noMatch, _ := tree.Lookup("nomatch")

	if noMatch != nil {
		t.Errorf("Match should be nil when matching %s", "nomatch")
	}

	someMatch, _ := tree.Lookup("test_nomatch")

	if someMatch != nil {
		t.Errorf("Partial matches should return nil %s", "test_nomatch")
	}
}

func BenchmarkLookupNoParams(b *testing.B) {
	tree := NewTree()
	mc := NewRouter("test", makeHub())

	paths := []string{
		"test",
		"test.one",
		"test.one.one",
		"test.one.one.one",
		"test.one.one.one.one",
		"test.one.two.one.one",
		"test.one.two.two.one",
		"test.one.two.two.two",
		"test.one.one.one.one.one",
		"test.one.two.one.one.one",
		"test.one.two.two.one.one",
		"test.one.two.two.two.one",
		"test.two",
		"test.two.one.one",
		"test.two.one.one.one",
		"test.two.one.two.one",
		"test.two.one.two.two",
		"test.three",
		"test.three.one.one",
		"test.three.one.one.one",
		"test.three.one.two.one",
		"test.three.one.two.two",
	}

	for _, path := range paths {
		tree.Add(path, mc)
	}

	for i := 0; i < b.N; i++ {
		tree.Lookup("test.one.one.one.one")
	}
}

func BenchmarkLookupWithParams(b *testing.B) {
	tree := NewTree()
	router := NewRouter("test", makeHub())

	paths := []string{
		"test",
		"test.test.{param}.one.{param}.one",
		"test.test.{param}.two.{param}.one",
		"test.test.{param}.two.{param}.two",
	}

	for _, path := range paths {
		tree.Add(path, router)
	}

	for i := 0; i < b.N; i++ {
		tree.Lookup("test.test.123.two.456.two")
	}
}
