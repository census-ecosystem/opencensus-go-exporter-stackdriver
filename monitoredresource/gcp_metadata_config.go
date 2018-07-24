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

package monitoredresource

import (
	"os"
	"strings"

	"cloud.google.com/go/compute/metadata"
)

// A type representing metadata retrieved from GCP.
type GCPMetadata struct {

	// ProjectID is the identifier of the GCP project associated with this resource, such as "my-project".
	ProjectID string

	// InstanceID is the numeric VM instance identifier assigned by Compute Engine.
	InstanceID string

	// ClusterName is the name for the cluster the container is running in.
	ClusterName string

	// ContainerName is the name of the container.
	ContainerName string

	// NamespaceID is the identifier for the cluster namespace the container is running in
	NamespaceID string

	// PodI is the identifier for the pod the container is running in.
	PodID string

	// Zone is the Compute Engine zone in which the VM is running.
	Zone string
}

// retrieveGCPMetadata retrieves value of each Attribute from Metadata Server
// in GKE container and GCE instance environment.
// Some attributes are retrieved from the system environment.
// This is only executed once.
func retrieveGCPMetadata() *GCPMetadata {
	gcpMetadata := GCPMetadata{}

	gcpMetadata.ProjectID, _ = metadata.ProjectID()
	gcpMetadata.InstanceID, _ = metadata.InstanceID()
	gcpMetadata.Zone, _ = metadata.Zone()
	clusterName, _ := metadata.InstanceAttributeValue("cluster-name")
	gcpMetadata.ClusterName = strings.TrimSpace(clusterName)

	// Following attributes are derived from environment variables. They are configured
	// via yaml file. For details refer to:
	// https://cloud.google.com/kubernetes-engine/docs/tutorials/custom-metrics-autoscaling#exporting_metrics_from_the_application
	gcpMetadata.NamespaceID = os.Getenv("NAMESPACE")
	gcpMetadata.ContainerName = os.Getenv("CONTAINER_NAME")
	gcpMetadata.PodID = os.Getenv("HOSTNAME")

	return &gcpMetadata
}
