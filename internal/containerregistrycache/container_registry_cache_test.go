// Copyright 2022 Google LLC.
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

package containerregistrycache

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestLatestImage(t *testing.T) {
	proxyTags := NewContainerRegistryCache("gcr.io/cloud-sql-connectors/cloud-sql-proxy")
	defaultImage, err := proxyTags.LatestImage()
	if err != nil {
		t.Errorf("unexpected error loading image %v", err)
	}

	t.Logf("latest: %s", defaultImage)
	if !strings.HasPrefix(defaultImage, "gcr.io/cloud-sql-connectors/cloud-sql-proxy:") {
		t.Errorf("want %s, got %s", "gcr.io/cloud-sql-connectors/cloud-sql-proxy:...", defaultImage)
	}
	if defaultImage == "gcr.io/cloud-sql-connectors/cloud-sql-proxy:latest" {
		t.Errorf("want %s, got %s", "gcr.io/cloud-sql-connectors/cloud-sql-proxy:...", defaultImage)
	}

	// Make sure the second time comes from cache
	fromCache, err := proxyTags.loadTags()
	if err != nil {
		t.Errorf("got error %v, want no error", err)
	}
	if !fromCache {
		t.Errorf("got fromCache=%v, want fromCache=true", fromCache)
	}
	t.Logf("done")
}

func TestRefresh(t *testing.T) {
	const repoName = "example.com/project/imagename"
	tagResponses := []struct {
		desc    string
		expires time.Time
		m       map[string]string
		e       error
		wantTag string
		wantErr bool
	}{
		{
			desc:    "When cache is empty and expires is in the future, request will be made.",
			expires: time.Now().Add(time.Hour),
			m: map[string]string{
				"latest":          "sha256:21dd35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-1": "sha256:21dd35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-0": "sha256:14ca52e9e476093882b9f23a799b2ead32bed1de0b71abf39b3d855083a902f9",
			},
			wantTag: repoName + ":2.0.0-preview-1",
		},
		{
			desc:    "When expires is in the past, request will be made.",
			expires: time.Now().Add(-time.Hour),
			m: map[string]string{
				"latest":          "sha256:333d35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-2": "sha256:333d35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-1": "sha256:21dd35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-0": "sha256:14ca52e9e476093882b9f23a799b2ead32bed1de0b71abf39b3d855083a902f9",
			},
			wantTag: repoName + ":2.0.0-preview-2",
		},
		{
			desc:    "When expires is in the past, request fails, last tag will be returned.",
			expires: time.Now().Add(-time.Hour),
			e:       fmt.Errorf("Error loading tag data"),
			wantTag: repoName + ":2.0.0-preview-2",
		},
		{
			desc:    "When expires is in the past, request succeedes but no 'latest' found, last tag will be returned.",
			expires: time.Now().Add(-time.Hour),
			m: map[string]string{
				"2.0.0-preview-3": "sha256:444d35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-2": "sha256:333d35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-1": "sha256:21dd35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-0": "sha256:14ca52e9e476093882b9f23a799b2ead32bed1de0b71abf39b3d855083a902f9",
			},
			wantTag: repoName + ":2.0.0-preview-2",
		},
		{
			desc:    "When expires is in the past, happy path after errors.",
			expires: time.Now().Add(-time.Hour),
			m: map[string]string{
				"latest":          "sha256:444d35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-3": "sha256:444d35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-2": "sha256:333d35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-1": "sha256:21dd35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
				"2.0.0-preview-0": "sha256:14ca52e9e476093882b9f23a799b2ead32bed1de0b71abf39b3d855083a902f9",
			},
			wantTag: repoName + ":2.0.0-preview-3",
		},
	}
	var tagResponseIndex int

	tc := &ContainerRegistryCache{
		imageURL: repoName,
		loadTagFunc: func(repo string) (map[string]string, error) {
			r := &tagResponses[tagResponseIndex]
			return r.m, r.e
		},
	}
	for tagResponseIndex = 0; tagResponseIndex < len(tagResponses); tagResponseIndex++ {
		r := &tagResponses[tagResponseIndex]

		// set expiration of the current cache
		tc.nextUpdate = r.expires

		// get the latest update
		tag, err := tc.LatestImage()

		// test
		if tag != r.wantTag {
			t.Errorf("got %v, want %v tag image, index: %v", tag, r.wantTag, tagResponseIndex)
		}
		if gotErr := err != nil; gotErr != r.wantErr {
			t.Errorf("got err %v, want err %v error returned, index %v", err, r.wantErr, tagResponseIndex)
		}
	}

}

func TestLatestImageLogic(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
		it      *ContainerRegistryCache
	}{
		{
			name: "happy path",
			want: "example.com/test/image:2.0.0-preview-1",
			it: &ContainerRegistryCache{
				imageURL:   "example.com/test/image",
				nextUpdate: time.Now().Add(time.Hour), // set to the future so it doesn't try to reload
				tagDigests: map[string]string{
					"latest":          "sha256:21dd35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
					"2.0.0-preview-1": "sha256:21dd35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
					"2.0.0-preview-0": "sha256:14ca52e9e476093882b9f23a799b2ead32bed1de0b71abf39b3d855083a902f9",
				},
			},
		},
		{
			name: "happy path next round",
			want: "example.com/test/image:2.0.0-preview-2",
			it: &ContainerRegistryCache{
				imageURL:   "example.com/test/image",
				nextUpdate: time.Now().Add(-time.Hour), // set to the future so it doesn't try to reload
				latest:     "example.com/test/image:2.0.0-preview-1",
				tagDigests: map[string]string{
					"latest":          "sha256:333335a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
					"2.0.0-preview-2": "sha256:333335a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
					"2.0.0-preview-1": "sha256:21dd35a9c02a7eed4219c854ad8b8ab833466658700a2e36bfeb5d74fc3ab677",
					"2.0.0-preview-0": "sha256:14ca52e9e476093882b9f23a799b2ead32bed1de0b71abf39b3d855083a902f9",
				},
			},
		},
		{
			name:    "no latest",
			wantErr: true,
			it: &ContainerRegistryCache{
				imageURL:   "example.com/test/image",
				nextUpdate: time.Now().Add(time.Hour),
				tagDigests: map[string]string{
					"2.0.0-preview-0": "sha256:14ca52e9e476093882b9f23a799b2ead32bed1de0b71abf39b3d855083a902f9",
				},
			},
		},
		{
			name:    "empty",
			wantErr: true,
			it: &ContainerRegistryCache{
				imageURL:   "example.com/test/image",
				nextUpdate: time.Now().Add(time.Hour),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := computeLatestImageURL(test.it.imageURL, test.it.tagDigests)
			if test.want != got {
				t.Errorf("got %v, want %v", got, test.want)
			}
			if test.wantErr && err == nil {
				t.Errorf("got no err, want err")
			}
			if !test.wantErr && err != nil {
				t.Errorf("got error %v, want no err", err)
			}
		})
	}
}
