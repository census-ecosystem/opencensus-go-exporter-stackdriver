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
	"fmt"
	"os"
	"sync"
)

// A type that represent monitor resource that satisfies monitoredresource.Interface
type Interface interface {
	MonitoredResource() (resType string, labels map[string]string)
}

// A type representing gke_container type monitored resource.
// For definition refer to
// https://cloud.google.com/monitoring/api/resources#tag_gke_container
type GKEContainer struct {

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

func (gke *GKEContainer) MonitoredResource() (resType string, labels map[string]string) {
	labels = map[string]string{
		"project_id":     gke.ProjectID,
		"instance_id":    gke.InstanceID,
		"zone":           gke.Zone,
		"cluster_name":   gke.ClusterName,
		"container_name": gke.ContainerName,
		"namespace_id":   gke.NamespaceID,
		"pod_id":         gke.PodID,
	}
	return "gke_container", labels
}

// A type representing gce_instance type monitored resource.
// For definition refer to
// https://cloud.google.com/monitoring/api/resources#tag_gce_instance
type GCEInstance struct {

	// ProjectID is the identifier of the GCP project associated with this resource, such as "my-project".
	ProjectID string

	// InstanceID is the numeric VM instance identifier assigned by Compute Engine.
	InstanceID string

	// Zone is the Compute Engine zone in which the VM is running.
	Zone string
}

func (gce *GCEInstance) MonitoredResource() (resType string, labels map[string]string) {
	labels = map[string]string{
		"project_id":  gce.ProjectID,
		"instance_id": gce.InstanceID,
		"zone":        gce.Zone,
	}
	return "gce_instance", labels
}

// A type representing aws_ec2_instance type monitored resource.
// For definition refer to
// https://cloud.google.com/monitoring/api/resources#tag_aws_ec2_instance
type AWSEC2Instance struct {

	// AWSAccount is the AWS account number for the VM.
	AWSAccount string

	// InstanceID is the instance id of the instance.
	InstanceID string

	// Region is the AWS region for the VM. The format of this field is "aws:{region}",
	// where supported values for {region} are listed at
	// http://docs.aws.amazon.com/general/latest/gr/rande.html.
	Region string
}

func (aws *AWSEC2Instance) MonitoredResource() (resType string, labels map[string]string) {
	labels = map[string]string{
		"aws_account": aws.AWSAccount,
		"instance_id": aws.InstanceID,
		"region":      aws.Region,
	}
	return "aws_ec2_instance", labels
}

// Autodetect auto detects monitored resources based on
// the environment where the application is running.
// It supports detection of following resource types
// 1. gke_container:
// 2. gce_instance:
// 3. aws_ec2_instance:
//
// Returns MonitoredResInterface which implements getLabels() and getType()
// For resource definition go to https://cloud.google.com/monitoring/api/resources
func Autodetect() Interface {
	return detectOnce()
}

// createAWSEC2InstanceMonitoredResource creates a aws_ec2_instance monitored resource
// awsIdentityDoc contains AWS EC2 specific attributes.
func createAWSEC2InstanceMonitoredResource(awsIdentityDoc *awsIdentityDocument) *AWSEC2Instance {
	aws_instance := AWSEC2Instance{
		AWSAccount: awsIdentityDoc.AccountID,
		InstanceID: awsIdentityDoc.InstanceID,
		Region:     fmt.Sprintf("aws:%s", awsIdentityDoc.Region),
	}
	return &aws_instance
}

// createGCEInstanceMonitoredResource creates a gce_instance monitored resource
// gcpMetadata contains GCP (GKE or GCE) specific attributes.
func createGCEInstanceMonitoredResource(gcpMetadata *GCPMetadata) *GCEInstance {
	gce_instance := GCEInstance{
		ProjectID:  gcpMetadata.ProjectID,
		InstanceID: gcpMetadata.InstanceID,
		Zone:       gcpMetadata.Zone,
	}
	return &gce_instance
}

// createGKEContainerMonitoredResource creates a gke_container monitored resource
// gcpMetadata contains GCP (GKE or GCE) specific attributes.
func createGKEContainerMonitoredResource(gcpMetadata *GCPMetadata) *GKEContainer {
	gke_container := GKEContainer{
		ProjectID:     gcpMetadata.ProjectID,
		InstanceID:    gcpMetadata.InstanceID,
		Zone:          gcpMetadata.Zone,
		ContainerName: gcpMetadata.ContainerName,
		ClusterName:   gcpMetadata.ClusterName,
		NamespaceID:   gcpMetadata.NamespaceID,
		PodID:         gcpMetadata.PodID,
	}
	return &gke_container
}

var once sync.Once

// detectResourceType determines the resource type.
// awsIdentityDoc contains AWS EC2 attributes. nil if it is not AWS EC2 environment
// gcpMetadata contains GCP (GKE or GCE) specific attributes.
func detectResourceType(awsIdentityDoc *awsIdentityDocument, gcpMetadata *GCPMetadata) Interface {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return createGKEContainerMonitoredResource(gcpMetadata)
	} else if gcpMetadata != nil && gcpMetadata.InstanceID != "" {
		return createGCEInstanceMonitoredResource(gcpMetadata)
	} else if awsIdentityDoc != nil {
		return createAWSEC2InstanceMonitoredResource(awsIdentityDoc)
	}
	return nil
}

// detectOnce runs only once to detect the resource type.
// It first attempts to retrieve AWS Identity Doc and GCP metadata.
// It then determines the resource type
func detectOnce() Interface {
	var autoDetected Interface
	once.Do(func() {
		awsIdentityDoc := retrieveAWSIdentityDocument()
		gcpMetadata := retrieveGCPMetadata()
		autoDetected = detectResourceType(awsIdentityDoc, gcpMetadata)
	})
	return autoDetected
}
