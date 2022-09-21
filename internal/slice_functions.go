// Copyright 2022 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

// Very simple functional programming concepts filter, map, flatMap implemented
// on slices. It's not worth importing a 3rd party functional library for these
// functions.

// appendOrReplace checks if the value is in the list using the matches function
// and then either replaces the first matching value, or if no match was found
// appends value to the list.
func appendOrReplace[T any](list []T, value T, matches func(*T, *T) bool) []T {
	for i := 0; i < len(list); i++ {
		if matches(&list[i], &value) {
			list[i] = value
			return list
		}
	}
	return append(list, value)
}

// fnFilter returns a slice containing all values of list where test function
// returns true.
func fnFilter[T any](list []T, test func(T) bool) []T {
	result := make([]T, 0, len(list))
	for i := 0; i < len(list); i++ {
		if test(list[i]) {
			result = append(result, list[i])
		}
	}
	return result
}

// fnMap returns a slice containing the result of apply(v) for each
// value in list.
func fnMap[T any, V any](list []T, apply func(T) V) []V {
	result := make([]V, len(list))
	for i := 0; i < len(list); i++ {
		result[i] = apply(list[i])
	}
	return result
}

// fnFlatMap returns a slice containing concatenated results of apply(v) for each
// value in list.
func fnFlatMap[T any, V any](list []T, apply func(T) []V) []V {
	result := make([]V, 0, len(list))
	for i := 0; i < len(list); i++ {
		result = append(result, apply(list[i])...)
	}
	return result
}
