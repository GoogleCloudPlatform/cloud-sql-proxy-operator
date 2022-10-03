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
package v1alpha1_test

import (
	"os"
	"testing"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

func TestUnmarshalAuthProxyWorkloadSample(t *testing.T) {
	wl := &cloudsqlapi.AuthProxyWorkload{}
	data, err := os.ReadFile("../../config/samples/cloudsql_v1alpha1_authproxyworkload_full.yaml")
	if err != nil {
		t.Error(err)
	}
	err = yaml.Unmarshal(data, wl)
	if err != nil {
		t.Errorf("Can't unmarshal, %v", err)
	}
}
