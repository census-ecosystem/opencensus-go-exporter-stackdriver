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

// A type representing metadata retrieved from GKE and GCE environment.
type gcpMetadata struct {

	// projectID is the identifier of the GCP project associated with this resource, such as "my-project".
	projectID string

	// instanceID is the numeric VM instance identifier assigned by Compute Engine.
	instanceID string

	// clusterName is the name for the cluster the container is running in.
	clusterName string

	// containerName is the name of the container.
	containerName string

	// namespaceID is the identifier for the cluster namespace the container is running in
	namespaceID string

	// podID is the identifier for the pod the container is running in.
	podID string

	// zone is the Compute Engine zone in which the VM is running.
	zone string
}

// retrieveGCPMetadata retrieves value of each Attribute from Metadata Server
// in GKE container and GCE instance environment.
// Some attributes are retrieved from the system environment.
// This is only executed once.
func retrieveGCPMetadata() *gcpMetadata {
	gcpMetadata := gcpMetadata{}

	gcpMetadata.projectID, _ = metadata.ProjectID()
	gcpMetadata.instanceID, _ = metadata.InstanceID()
	gcpMetadata.zone, _ = metadata.Zone()
	clusterName, _ := metadata.InstanceAttributeValue("cluster-name")
	gcpMetadata.clusterName = strings.TrimSpace(clusterName)

	// Following attributes are derived from environment variables. They are configured
	// via yaml file. For details refer to:
	// https://cloud.google.com/kubernetes-engine/docs/tutorials/custom-metrics-autoscaling#exporting_metrics_from_the_application
	gcpMetadata.namespaceID = os.Getenv("NAMESPACE")
	gcpMetadata.containerName = os.Getenv("CONTAINER_NAME")
	gcpMetadata.podID = os.Getenv("HOSTNAME")

	return &gcpMetadata
}
