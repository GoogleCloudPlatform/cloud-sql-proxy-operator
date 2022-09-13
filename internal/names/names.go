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

// Package names contains functions that help format safe names for
// kubernetes resources, following the rfc1035/rfc1123 label (DNS_LABEL) format.
package names

import (
	"fmt"
	"hash/fnv"
	"strings"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
)

const maxNameLen = 63
const hashLen = 8
const dividerLen = 2

// ContainerPrefix is the name prefix used on containers added to PodSpecs
// by this operator.
const ContainerPrefix = "csql-"

// ContainerName generates a valid name for a corev1.Container object that
// implements this cloudsql instance. Names must be 63 characters or less and
// adhere to the rfc1035/rfc1123 label (DNS_LABEL) format.  r.ObjectMeta.Name
// already is required to be 63 characters or less because it is a name. Because
// we are prepending 'csql-' ContainerPrefix as a marker, the generated name with
// the prefix could be longer than 63 characters.
func ContainerName(r *cloudsqlapi.AuthProxyWorkload) string {
	return SafePrefixedName(ContainerPrefix, r.GetName())
}

func VolumeName(r *cloudsqlapi.AuthProxyWorkload, inst *cloudsqlapi.InstanceSpec, mountType string) string {
	connName := strings.ReplaceAll(strings.ToLower(inst.ConnectionString), ":", "-")
	return SafePrefixedName(ContainerPrefix, r.GetName()+"-"+mountType+"-"+connName)
}

// SafePrefixedName adds a prefix to a name and shortens it while preserving its uniqueness
// so that it fits the 63 character limit imposed by kubernetes.
// Kubernetes names must follow the DNS Label format for all names.
// See https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#rfc-1035-label-names
func SafePrefixedName(prefix string, name string) string {
	containerPrefixLen := len(prefix)
	truncateLen := (maxNameLen - hashLen - dividerLen - containerPrefixLen) / 2

	instName := name
	if len(instName)+containerPrefixLen > maxNameLen {
		// string shortener that will still produce a name that is still unique
		// even though it is truncated.
		namePrefix := instName[:truncateLen]
		nameSuffix := instName[len(instName)-truncateLen:]
		checksum := hash(instName)
		return strings.ToLower(fmt.Sprintf("%s%s-%s-%x", prefix, namePrefix, nameSuffix, checksum))
	} else {
		return strings.ToLower(fmt.Sprintf("%s%s", prefix, instName))
	}

}

// hash simply returns the checksum for a string
func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
