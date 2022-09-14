// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"reflect"
	"testing"
)

type data struct {
	name  string
	value string
}

var one = data{
	name:  "one",
	value: "one",
}
var two = data{
	name:  "two",
	value: "two",
}
var twoB = data{
	name:  "two",
	value: "b",
}
var three = data{
	name:  "three",
	value: "three",
}

func isNameEqual(a *data, b *data) bool {
	return a.name == b.name
}

func TestAppendOrReplace(t *testing.T) {

	// replace
	if want, got :=
		[]data{one, twoB},
		appendOrReplace([]data{one, two}, twoB, isNameEqual); !reflect.DeepEqual(want, got) {
		t.Errorf("got %v, want %v", got, want)
	}

	// append
	if want, got :=
		[]data{one, two, three},
		appendOrReplace([]data{one, two}, three, isNameEqual); !reflect.DeepEqual(want, got) {
		t.Errorf("got %v, want %v", got, want)
	}

	// empty
	if want, got :=
		[]data{one},
		appendOrReplace([]data{}, one, isNameEqual); !reflect.DeepEqual(want, got) {
		t.Errorf("got %v, want %v", got, want)
	}

	// nil
	var d []data
	d = nil
	if want, got :=
		[]data{one},
		appendOrReplace(d, one, isNameEqual); !reflect.DeepEqual(want, got) {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestFunctions(t *testing.T) {

	// fnFilter
	if want, got :=
		[]string{"two", "three"},
		fnFilter([]string{"one", "two", "three", "four"}, func(s string) bool { return s[0] == 't' }); !reflect.DeepEqual(want, got) {
		t.Errorf("got %v, want %v", got, want)
	}

	// fnMap
	if want, got :=
		[]int{5, 0, 4, 2, 3},
		fnMap([]string{"five", "zero", "four", "two", "three"}, numToInt); !reflect.DeepEqual(want, got) {
		t.Errorf("got %v, want %v", got, want)
	}

	// fnFlatMap
	if want, got :=
		[]int{5, 4, 0, 4, 4, 4, 2, 3, 3, 5},
		fnFlatMap([]string{"five", "zero", "four", "two", "three"}, numToFlatMap); !reflect.DeepEqual(want, got) {
		t.Errorf("got %v, want %v", got, want)
	}

}

func numToInt(s string) int {
	lookup := []string{"zero", "one", "two", "three", "four", "five"}
	for i := 0; i < len(lookup); i++ {
		if lookup[i] == s {
			return i
		}
	}
	return -1
}
func numToFlatMap(s string) []int {
	lookup := []string{"zero", "one", "two", "three", "four", "five"}
	for i := 0; i < len(lookup); i++ {
		if lookup[i] == s {
			return []int{i, len(s)}
		}
	}
	return []int{}
}
