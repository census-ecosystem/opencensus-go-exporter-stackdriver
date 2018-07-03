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

package stackdriver

import (
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	"os"
)

// GetAutoDetectedDefaultResource auto-detects monitored resources based on
// the environment where the application is running.
// Returns *monitoredrespb.MonitoredResource containing labels relevant to
// detected resource type.
// It supports detection of following resource types
// 1. gke_container:
// 2. gce_instance:
// 3. aws_ec2_instance:
//
// For resource definition go to https://cloud.google.com/monitoring/api/resources

func GetAutoDetectedDefaultResource() *monitoredrespb.MonitoredResource {

	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return createGcpGkeContainerMonitoredResource()
	}
	if getValueFromGcpMetadataConfig(gcpInstanceIdAttr) != "" {
		return createGcpGceInstanceMonitoredResource()
	}
	if isRunningOnAwsEc2() {
		return createAwsEc2InstanceMonitoredResource()
	}
	return nil
}

// createNewMonitoredResources creates a monitored resource of type resourceType.
// Returns *monitoredrespb.MonitoredResource containing empty label map.
func createNewMonitoredResources(resourceType string) *monitoredrespb.MonitoredResource {
	mr := new(monitoredrespb.MonitoredResource)
	if mr == nil {
		return nil
	}
	mr.Labels = make(map[string]string)
	if mr.Labels == nil {
		return nil
	}
	mr.Type = resourceType
	return mr
}

// createAwsEc2InstanceMonitoredResource creates a monitored resource of type
// ResourceTypeAwsEc2Instance.
func createAwsEc2InstanceMonitoredResource() *monitoredrespb.MonitoredResource {
	mr := createNewMonitoredResources(ResourceTypeAwsEc2Instance)
	if mr != nil {
		mr.Labels[AwsEc2LabelAwsAccount] = getValueFromAwsIdentityDocument(awsAccountId)
		mr.Labels[AwsEc2LabelInstanceId] = getValueFromAwsIdentityDocument(awsInstanceId)
		mr.Labels[AwsEc2LabelRegion] = getValueFromAwsIdentityDocument(awsRegion)
	}
	return mr
}

// createGcpGceInstanceMonitoredResource creates a monitored resource of type
// ResourceTypeGceInstance
func createGcpGceInstanceMonitoredResource() *monitoredrespb.MonitoredResource {
	mr := createNewMonitoredResources(ResourceTypeGceInstance)
	if mr != nil {
		mr.Labels[GceLabelProjectId] = getValueFromGcpMetadataConfig(gcpAccountIdAttr)
		mr.Labels[GceLabelInstanceId] = getValueFromGcpMetadataConfig(gcpInstanceIdAttr)
		mr.Labels[GceLabelZone] = getValueFromGcpMetadataConfig(gcpZoneAttr)
	}
	return mr
}

// createGcpGkeContainerMonitoredResource creates a monitored resource of type
// ResourceTypeGkeContainer
func createGcpGkeContainerMonitoredResource() *monitoredrespb.MonitoredResource {
	mr := createNewMonitoredResources(ResourceTypeGkeContainer)
	if mr != nil {
		mr.Labels[GkeLabelProjectId] = getValueFromGcpMetadataConfig(gcpAccountIdAttr)
		mr.Labels[GkeLabelInstanceId] = getValueFromGcpMetadataConfig(gcpInstanceIdAttr)
		mr.Labels[GkeLabelZone] = getValueFromGcpMetadataConfig(gcpZoneAttr)
		mr.Labels[GkeLabelClusterName] = getValueFromGcpMetadataConfig(gcpClusterNameAttr)
		mr.Labels[GkeLabelContainerName] = getValueFromGcpMetadataConfig(gcpContainerNameEnv)
		mr.Labels[GkeLabelNamespaceId] = getValueFromGcpMetadataConfig(gcpNamespaceEnv)
		mr.Labels[GkeLabelPodId] = getValueFromGcpMetadataConfig(gcpPodIdEnv)
	}
	return mr
}
