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

// Package containerregistrycache caches the image tags and calculates the
// latest image for a container registry.
package containerregistrycache

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var l = logf.Log.WithName("internal.containerregistrycache")

type loadContainerDigests func(repo string) (map[string]string, error)

// realLoadContainerDigests function actually connects to the imageURL to load tag information
func realLoadContainerDigests(repo string) (map[string]string, error) {
	// list tag names
	tags, err := crane.ListTags(repo, crane.WithAuth(authn.Anonymous))
	if err != nil {
		l.Error(err, "unable to read latest from imageURL %s, falling back to use version latest", "imageURL", repo)
		return nil, err
	}

	// retrieve digests for each tag and put it in the map
	tagDigests := map[string]string{}
	for _, tag := range tags {
		ref := repo + ":" + tag
		d, err := crane.Digest(ref, crane.WithAuth(authn.Anonymous))
		if err != nil {
			l.Error(err, "unable to read latest from, falling back to use version latest", "imageURL", repo)
			continue
		}
		l.Info("Image: ", "ref", d)
		tagDigests[tag] = d
	}
	return tagDigests, nil
}

// computeLatestImageURL finds "latest" tagDigests and computes a stable tag
func computeLatestImageURL(repo string, tagDigests map[string]string) (string, error) {
	d, ok := tagDigests["latest"]
	if !ok {
		// if latest is already set, don't update it
		return "", fmt.Errorf("imageURL %s does not have an image tagged 'latest'", repo)
	}

	for tag, dgst := range tagDigests {
		if dgst == d && tag != "latest" {
			return repo + ":" + tag, nil
		}
	}

	// if there is no stable name, use the digest explicitly instead of returning "latest"
	return repo + "@" + d, nil

}

type ContainerRegistryCache struct {
	imageURL    string
	lock        sync.Mutex
	nextUpdate  time.Time
	tagDigests  map[string]string
	latest      string
	loadTagFunc loadContainerDigests
}

// loadTags actually connects to the image registry, lists
// the tagged images, and then returns the latest tagged image.
func (t *ContainerRegistryCache) loadTags() (bool, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.nextUpdate.After(time.Now()) && len(t.tagDigests) > 0 {
		// no update needed
		return true, nil
	}

	tagDigests, err := t.loadTagFunc(t.imageURL)
	if err != nil {
		return false, err
	}

	// compute latest image
	latest, err := computeLatestImageURL(t.imageURL, tagDigests)
	if err != nil {
		return false, err
	}

	// success, set the next update time
	t.tagDigests = tagDigests
	t.latest = latest
	t.nextUpdate = time.Now().Add(4 * time.Hour)
	return false, nil

}

// LatestImage calculates the latest image based on the tags in the
// well known imageURL.
// K8s best practice is to avoid using the tag "latest" in container specs.
// The "latest" tag creates confusion about exactly which verison is running,
// and when that image gets pulled from the registry. Only when there is an
// error connecting to the registry, it will fall back to the "latest" image version.
func (t *ContainerRegistryCache) LatestImage() (string, error) {
	_, err := t.loadTags()
	if err != nil {
		// only propagate the error if t.latest was never set to a valid value.
		if t.latest == "" {
			return "", err
		}
	}

	return t.latest, nil
}

// NewContainerRegistryCache creates a cache for the image defined in imageURL.
func NewContainerRegistryCache(imageURL string) *ContainerRegistryCache {
	c := &ContainerRegistryCache{
		imageURL:    imageURL,
		loadTagFunc: realLoadContainerDigests,
	}
	// start loading the latest image into cache during startup to make the webhook
	// calls faster
	go c.LatestImage()
	return c
}
