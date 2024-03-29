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

package workload

import (
	"fmt"
	"hash/fnv"
	"strings"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
)

// ContainerPrefix is the name prefix used on containers added to PodSpecs
// by this operator.
const ContainerPrefix = "csql-"

// ContainerName generates a valid name for a corev1.Container object that
// implements this cloudsql instance. Names must be 63 characters or fewer and
// adhere to the rfc1035/rfc1123 label (DNS_LABEL) format.  r.ObjectMeta.Name
// already is required to be 63 characters or less because it is a name. Because
// we are prepending 'csql-' ContainerPrefix as a marker, the generated name with
// the prefix could be longer than 63 characters.
func ContainerName(r *cloudsqlapi.AuthProxyWorkload) string {
	return SafePrefixedName(ContainerPrefix, r.GetNamespace()+"-"+r.GetName())
}

// VolumeName generates a unique, valid name for a volume based on the AuthProxyWorkload
// name and the Cloud SQL instance name.
func VolumeName(r *cloudsqlapi.AuthProxyWorkload, inst *cloudsqlapi.InstanceSpec, mountType string) string {
	connName := strings.ReplaceAll(strings.ToLower(inst.ConnectionString), ":", "-")
	return SafePrefixedName(ContainerPrefix, r.GetName()+"-"+mountType+"-"+connName)
}

// SafePrefixedName adds a prefix to a name and shortens it while preserving its uniqueness
// so that it fits the 63 character limit imposed by kubernetes.
// Kubernetes names must follow the DNS Label format for all names.
// See https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#rfc-1035-label-names
//
// DO NOT CHANGE THIS ALGORITHM! The names generated by this function are used
// to match containers and volumes to configuration. If the names are generated
// differently between one version of the operator and the next, the operator
// will break. Existing workloads will not be correctly updated.
func SafePrefixedName(prefix, instName string) string {
	const maxNameLen = 63 // maximum character limit for a name

	containerPrefixLen := len(prefix)

	if len(instName)+containerPrefixLen > maxNameLen {
		// string shortener that will still produce a name that is still unique
		// even though it is truncated.
		checksum := mustHash([]byte(instName))
		hashSuffix := fmt.Sprintf("-%x", checksum)
		hashSuffixLen := len(hashSuffix)
		truncateLen := (maxNameLen - hashSuffixLen - containerPrefixLen) / 2
		namePrefix := instName[:truncateLen]
		nameSuffix := instName[len(instName)-truncateLen:]

		return strings.ToLower(strings.Join(
			[]string{prefix, namePrefix, nameSuffix, hashSuffix}, ""))
	}

	return strings.ToLower(prefix + instName)
}

// mustHash simply returns the checksum for a slice of bytes
func mustHash(bytes []byte) uint32 {
	h := fnv.New32a()
	i, err := h.Write(bytes)

	if err != nil || i != len(bytes) {
		panic(fmt.Errorf("unable to calculate mustHash for bytes %v %v", bytes, err))
	}

	return h.Sum32()
}
