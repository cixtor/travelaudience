package main

import (
	"reflect"
	"testing"
)

func TestValidEndpoints(t *testing.T) {
	input := []string{
		"foo@bar.com",
		"example.com",
		"https://google.com/",
		"ftp://example.com/foo/bar",
		"http://foobar.com/p/r/i",
		"https://cixtor.com/numbers",
		"dolor",
	}
	expected := []string{
		"https://google.com/",
		"http://foobar.com/p/r/i",
		"https://cixtor.com/numbers",
	}
	result := validApiEndpoints(input)

	if !reflect.DeepEqual(result, expected) {
		t.Fatal("URL list validation was incorrect")
	}
}

func TestNumberCollector(t *testing.T) {
	input := []string{
		"http://127.0.0.1:8090/primes",
		"http://127.0.0.1:8090/fibo",
		"http://127.0.0.1:8090/odd",
		"http://127.0.0.1:8090/rand",
		"http://127.0.0.1:8090/none",
		"http://127.0.0.1:8090",
	}

	result := collectAllNumbers(input)

	if reflect.TypeOf(result).String() != "[]int" {
		t.Fatal("Data returned by the APIs is not []int")
	}
}

func TestNumberCollectorMax(t *testing.T) {
	input := []string{
		"http://127.0.0.1:8090/primes",
		"http://127.0.0.1:8090/fibo",
		"http://127.0.0.1:8090/odd",
		"http://127.0.0.1:8090/rand",
		"http://127.0.0.1:8090/none",
		"http://127.0.0.1:8090",
		"http://127.0.0.1:8090/primes",
		"http://127.0.0.1:8090/fibo",
		"http://127.0.0.1:8090/odd",
		"http://127.0.0.1:8090/rand",
		"http://127.0.0.1:8090/none",
		"http://127.0.0.1:8090",
		"http://127.0.0.1:8090/primes",
		"http://127.0.0.1:8090/fibo",
		"http://127.0.0.1:8090/odd",
		"http://127.0.0.1:8090/rand",
		"http://127.0.0.1:8090/none",
		"http://127.0.0.1:8090",
	}

	result := collectAllNumbers(input)

	if reflect.TypeOf(result).String() != "[]int" {
		t.Fatal("Data returned by the APIs is not []int")
	}
}

func TestUniqueNumbers(t *testing.T) {
	input := []int{1, 1, 1, 2, 3, 3, 4, 5, 6, 6, 6, 6, 6, 7, 8, 9, 9}
	expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	result := simpleUniqueNumbers(input)

	if !reflect.DeepEqual(result, expected) {
		t.Fatal("Numbers in the list are not unique")
	}
}
