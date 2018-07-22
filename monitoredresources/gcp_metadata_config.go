// Copyright 2018, OpenCensus Authors
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

package monitoredresources

import (
	"cloud.google.com/go/compute/metadata"
	"os"
	"strings"
)

// gcpMetadataConfigMap holds mapping of Attribute Name and Attribute Value
// in GKE container and GCE instance environment
var gcpMetadataConfigMap map[string]string

// retrieveGCPMetadata retrieves value of each Attribute from Metadata Server
// in GKE container and GCE instance environment.
// Some attributes are retrieved from the system environment.
// This is only executed once.
func retrieveGCPMetadata() {
	gcpMetadataConfigMap = make(map[string]string)

	gcpMetadataConfigMap[gcpInstanceID], _ = metadata.InstanceID()
	gcpMetadataConfigMap[gcpProjectID], _ = metadata.ProjectID()
	clusterName, _ := metadata.InstanceAttributeValue(gcpClusterName)
	gcpMetadataConfigMap[gcpClusterName] = strings.TrimSpace(clusterName)
	gcpMetadataConfigMap[gcpZone], _ = metadata.Zone()

	// Following attributes are derived from environment variables. They are configured
	// via yaml file. For details refer to:
	// https://cloud.google.com/kubernetes-engine/docs/tutorials/custom-metrics-autoscaling#exporting_metrics_from_the_application
	gcpMetadataConfigMap[gcpContainerNameEnvVar] = os.Getenv(gcpContainerNameEnvVar)
	gcpMetadataConfigMap[gcpNamespaceEnvVar] = os.Getenv(gcpNamespaceEnvVar)
	gcpMetadataConfigMap[gcpPodIDEnvVar] = os.Getenv(gcpPodIDEnvVar)
}

// getGCPMetadataConfig returns value of attribute attributeName from
// map stored locally.
func getGCPMetadataConfig(attributeName string) string {
	return gcpMetadataConfigMap[attributeName]
}
