// Copyright 2023 Google LLC
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

// Copies the static yaml installer output and turns it into a helm chart
// template
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type document struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
}

func main() {
	installYaml := flag.String("installYaml", "", "The install yaml file")
	operatorChartDir := flag.String("operatorChartDir", "", "The operator helm chart directory")
	crdChartDir := flag.String("crdChartDir", "", "The crd helm chart directory")
	flag.Parse()
	fmt.Printf("Converting install yaml %v to helm chart at %v and %v.", *installYaml, *operatorChartDir, *crdChartDir)
	read(*installYaml, *operatorChartDir, *crdChartDir)
}

func read(installYaml, operatorChartDir, crdChardDir string) {
	// read the output.yaml file
	data, err := os.ReadFile(installYaml)
	if err != nil {
		panic(err)
	}

	// Split on '---'
	docs := bytes.Split(data, []byte{'\n', '-', '-', '-', '\n'})
	fmt.Println("Starting docs...")
	fmt.Println()
	for i, docBytes := range docs {

		var doc document
		if err := yaml.Unmarshal(docBytes, &doc); err != nil {
			panic(err)
		}
		fmt.Printf("Doc %d\n", i)
		// print the fields to the console
		fmt.Printf("%d, %v %v\n", i, doc.Kind, doc.Name)

		var filename = fmt.Sprintf("%s-%s.yaml", doc.Kind, strings.Replace(doc.Name, "cloud-sql-proxy-operator-", "", 1))

		var filePath string
		var content []byte
		switch doc.Kind {
		case "Namespace":
			filePath = path.Join(crdChardDir, "templates", filename)
			content = makeCrdChartReplacements(docBytes)
		case "Deployment":
			// ignore the deployment, this is a custom-written chart
		case "CustomResourceDefinition":
			filePath = path.Join(crdChardDir, "templates", filename)
			content = makeCrdChartReplacements(docBytes)
		default:
			filePath = path.Join(operatorChartDir, "templates", filename)
			content = makeChartReplacements(docBytes)
		}

		if filePath == "" {
			continue
		}

		err := os.WriteFile(filePath, content, 0644)
		if err != nil {
			panic(err)
		}
	}

}

func makeChartReplacements(data []byte) []byte {
	content := string(data)

	// Namespace
	content = strings.Replace(content, "cloud-sql-proxy-operator-system", "{{ .Release.Namespace }}", -1)

	// Name
	content = strings.Replace(content, "cloud-sql-proxy-operator", "{{ .Release.Name }}", -1)

	return []byte(content)
}

func makeCrdChartReplacements(data []byte) []byte {
	content := string(data)

	// Namespace
	content = strings.Replace(content, "cloud-sql-proxy-operator-system", "{{ .Values.operatorNamespace }}", -1)

	// Name
	content = strings.Replace(content, "cloud-sql-proxy-operator", "{{ .Values.operatorName }}", -1)

	return []byte(content)
}
