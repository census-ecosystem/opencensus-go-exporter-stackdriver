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
	"testing"
)

const (
	GCP_PROJECT_ID_STR         = "gcp-project"
	GCP_INSTANCE_ID_STR        = "instance"
	GCP_ZONE_STR               = "us-east1"
	GCP_GKE_NAMESPACE_STR      = "namespace"
	GCP_GKE_POD_ID_STR         = "pod-id"
	GCP_GKE_CONTAINER_NAME_STR = "container"
	GCP_GKE_CLUSTER_NAME_STR   = "cluster"
)

func TestGKEContainerMonitoredResources(t *testing.T) {
	os.Setenv("KUBERNETES_SERVICE_HOST", "127.0.0.1")
	gcpMetadata := gcpMetadata{
		instanceID:    GCP_INSTANCE_ID_STR,
		projectID:     GCP_PROJECT_ID_STR,
		zone:          GCP_ZONE_STR,
		clusterName:   GCP_GKE_CLUSTER_NAME_STR,
		containerName: GCP_GKE_CONTAINER_NAME_STR,
		namespaceID:   GCP_GKE_NAMESPACE_STR,
		podID:         GCP_GKE_POD_ID_STR,
	}
	autoDetected := detectResourceType(nil, &gcpMetadata)

	if autoDetected == nil {
		t.Fatal("GKEContainerMonitoredResource nil")
	}
	resType, labels := autoDetected.MonitoredResource()
	if resType != "gke_container" ||
		labels["instance_id"] != GCP_INSTANCE_ID_STR ||
		labels["project_id"] != GCP_PROJECT_ID_STR ||
		labels["cluster_name"] != GCP_GKE_CLUSTER_NAME_STR ||
		labels["container_name"] != GCP_GKE_CONTAINER_NAME_STR ||
		labels["zone"] != GCP_ZONE_STR ||
		labels["namespace_id"] != GCP_GKE_NAMESPACE_STR ||
		labels["pod_id"] != GCP_GKE_POD_ID_STR {
		t.Errorf("GKEContainerMonitoredResource Failed: %v", autoDetected)
	}
}

func TestGCEInstanceMonitoredResources(t *testing.T) {
	os.Setenv("KUBERNETES_SERVICE_HOST", "")
	gcpMetadata := gcpMetadata{
		instanceID: GCP_INSTANCE_ID_STR,
		projectID:  GCP_PROJECT_ID_STR,
		zone:       GCP_ZONE_STR,
	}
	autoDetected := detectResourceType(nil, &gcpMetadata)

	if autoDetected == nil {
		t.Fatal("GCEInstanceMonitoredResource nil")
	}
	resType, labels := autoDetected.MonitoredResource()
	if resType != "gce_instance" ||
		labels["instance_id"] != GCP_INSTANCE_ID_STR ||
		labels["project_id"] != GCP_PROJECT_ID_STR ||
		labels["zone"] != GCP_ZONE_STR {
		t.Errorf("GCEInstanceMonitoredResource Failed: %v", autoDetected)
	}
}

func TestAWSEC2InstanceMonitoredResources(t *testing.T) {
	os.Setenv("KUBERNETES_SERVICE_HOST", "")
	gcpMetadata := gcpMetadata{}

	awsIdentityDoc := &awsIdentityDocument{
		"123456789012",
		"i-1234567890abcdef0",
		"us-west-2",
	}
	autoDetected := detectResourceType(awsIdentityDoc, &gcpMetadata)

	if autoDetected == nil {
		t.Fatal("AWSEC2InstanceMonitoredResource nil")
	}
	resType, labels := autoDetected.MonitoredResource()
	if resType != "aws_ec2_instance" ||
		labels["instance_id"] != "i-1234567890abcdef0" ||
		labels["aws_account"] != "123456789012" ||
		labels["region"] != "aws:us-west-2" {
		t.Errorf("AWSEC2InstanceMonitoredResource Failed: %v", autoDetected)
	}
}

func TestNullMonitoredResources(t *testing.T) {
	os.Setenv("KUBERNETES_SERVICE_HOST", "")
	mr := Autodetect()
	if mr != nil {
		t.Errorf("Expected nil MonitoredResource but found %v", mr)
	}
}
