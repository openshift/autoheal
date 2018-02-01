/*
Copyright (c) 2018 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mapping

// CopyMap copies the contents of one map of strings to another map. If the source map doesn't
// exist, or is empty, then nothing is copied. If there is something to copy, then the destination
// map will be created if needed (that is why the parameter is a pointer) and it will be populated
// with the contents of the source.
//
func CopyMap(from map[string]string, to *map[string]string) {
	if from != nil && len(from) > 0 {
		if *to == nil {
			*to = make(map[string]string, len(from))
		}
		for key, value := range from {
			(*to)[key] = value
		}
	}
}
