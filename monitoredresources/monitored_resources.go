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
	"fmt"
	"os"
	"sync"
)

type MonitoredResInterface interface {
	GetType() string
	GetLabels() map[string]string
}

type GKEContainer struct {
	ProjectID     string
	InstanceID    string
	ClusterName   string
	ContainerName string
	NamespaceID   string
	PodId         string
	Zone          string
}

func (gke *GKEContainer) GetType() string {
	return "gke_container"
}

func (gke *GKEContainer) GetLabels() map[string]string {
	labels := map[string]string{
		GKELabelProjectID:     gke.ProjectID,
		GKELabelInstanceID:    gke.InstanceID,
		GKELabelZone:          gke.Zone,
		GKELabelClusterName:   gke.ClusterName,
		GKELabelContainerName: gke.ContainerName,
		GKELabelNamespaceID:   gke.NamespaceID,
		GKELabelPodID:         gke.PodId,
	}
	return labels
}

type GCEInstance struct {
	ProjectID  string
	InstanceID string
	Zone       string
}

func (gce *GCEInstance) GetType() string {
	return "gce_instance"
}

func (gce *GCEInstance) GetLabels() map[string]string {
	labels := map[string]string{
		GCELabelProjectID:  gce.ProjectID,
		GCELabelInstanceID: gce.InstanceID,
		GCELabelZone:       gce.Zone,
	}
	return labels
}

type AWSEC2Instance struct {
	AWSAccount string
	InstanceID string
	Region     string
}

func (aws *AWSEC2Instance) GetType() string {
	return "aws_ec2_instance"
}

func (aws *AWSEC2Instance) GetLabels() map[string]string {
	labels := map[string]string{
		AWSEC2LabelAwsAccount: aws.AWSAccount,
		AWSEC2LabelInstanceID: aws.InstanceID,
		AWSEC2LabelRegion:     aws.Region,
	}
	return labels
}

var mr MonitoredResInterface

// Autodetect auto detects monitored resources based on
// the environment where the application is running.
// It supports detection of following resource types
// 1. gke_container:
// 2. gce_instance:
// 3. aws_ec2_instance:
//
// Returns MonitoredResInterface which implements getLabels() and getType()
// For resource definition go to https://cloud.google.com/monitoring/api/resources
func Autodetect() MonitoredResInterface {
	detectOnce()
	return mr
}

// createAWSEC2InstanceMonitoredResource creates a aws_ec2_instance monitored resource
func createAWSEC2InstanceMonitoredResource() *AWSEC2Instance {
	awsIdentityDoc := getAWSIdentityDocument()
	aws_instance := AWSEC2Instance{
		AWSAccount: awsIdentityDoc.AccountId,
		InstanceID: awsIdentityDoc.InstanceId,
		Region:     fmt.Sprintf("aws:%s", awsIdentityDoc.Region),
	}
	return &aws_instance
}

// createGCEInstanceMonitoredResource creates a gce_instance monitored resource
func createGCEInstanceMonitoredResource() *GCEInstance {
	gce_instance := GCEInstance{
		ProjectID:  getGCPMetadataConfig(gcpProjectID),
		InstanceID: getGCPMetadataConfig(gcpInstanceID),
		Zone:       getGCPMetadataConfig(gcpZone),
	}
	return &gce_instance
}

// createGKEContainerMonitoredResource creates a gke_container monitored resource
func createGKEContainerMonitoredResource() *GKEContainer {
	gke_container := GKEContainer{

		ProjectID:     getGCPMetadataConfig(gcpProjectID),
		InstanceID:    getGCPMetadataConfig(gcpInstanceID),
		ClusterName:   getGCPMetadataConfig(gcpClusterName),
		ContainerName: getGCPMetadataConfig(gcpContainerNameEnvVar),
		NamespaceID:   getGCPMetadataConfig(gcpNamespaceEnvVar),
		PodId:         getGCPMetadataConfig(gcpPodIDEnvVar),
		Zone:          getGCPMetadataConfig(gcpZone),
	}
	return &gke_container
}

var once sync.Once

// detectResourceType determines the resource type.
func detectResourceType() {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		mr = createGKEContainerMonitoredResource()
	} else if getGCPMetadataConfig(gcpInstanceID) != "" {
		mr = createGCEInstanceMonitoredResource()
	} else if isRunningOnAwsEc2() {
		mr = createAWSEC2InstanceMonitoredResource()
	} else {
		mr = nil
	}
}

// detectOnce runs only once to detect the resource type.
// It first attempts to retrieve AWS Identity Doc and GCP metadata.
// It then determines the resource type
func detectOnce() {
	once.Do(func() {
		retrieveAWSIdentityDocument()
		retrieveGCPMetadata()
		detectResourceType()
	})
}
