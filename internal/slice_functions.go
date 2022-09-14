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

func appendOrReplace[T any](list []T, value T, isEqual func(*T, *T) bool) []T {
	for i := 0; i < len(list); i++ {
		if isEqual(&list[i], &value) {
			list[i] = value
			return list
		}
	}
	list = append(list, value)
	return list
}

func fnFilter[T any](list []T, test func(T) bool) []T {
	result := make([]T, 0, len(list))
	for i := 0; i < len(list); i++ {
		if test(list[i]) {
			result = append(result, list[i])
		}
	}
	return result
}

func fnMap[T any, V any](list []T, apply func(T) V) []V {
	result := make([]V, len(list))
	for i := 0; i < len(list); i++ {
		result[i] = apply(list[i])
	}
	return result
}

func fnFlatMap[T any, V any](list []T, apply func(T) []V) []V {
	result := make([]V, 0, len(list))
	for i := 0; i < len(list); i++ {
		oneResult := apply(list[i])
		for j := 0; j < len(oneResult); j++ {
			result = append(result, oneResult[j])
		}
	}
	return result
}
