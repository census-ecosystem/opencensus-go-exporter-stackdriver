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
	"os"
	"testing"
)

const (
	GCP_ACCOUNT_ID_STR         string = "gcp-project"
	GCP_INSTANCE_ID_STR        string = "instance"
	GCP_ZONE_STR               string = "us-east1"
	GCP_GKE_NAMESPACE_STR      string = "namespace"
	GCP_GKE_POD_ID_STR         string = "pod-id"
	GCP_GKE_CONTAINER_NAME_STR string = "container"
	GCP_GKE_CLUSTER_NAME_STR   string = "cluster"
)

const SAMPLE_AWS_IDENTITY_DOCUMENT = `{
	"devpayProductCodes" : null,
	"marketplaceProductCodes" : [ "1abc2defghijklm3nopqrs4tu" ], 
	"availabilityZone" : "us-west-2b",
	"privateIp" : "10.158.112.84",
	"version" : "2017-09-30",
	"instanceId" : "i-1234567890abcdef0",
	"billingProducts" : null,
	"instanceType" : "t2.micro",
	"accountId" : "123456789012",
	"imageId" : "ami-5fb8c835",
	"pendingTime" : "2016-11-19T16:32:11Z",
	"architecture" : "x86_64",
	"kernelId" : null,
	"ramdiskId" : null,
	"region" : "us-west-2"
	}`

const SAMPLE_NON_AWS_IDENTITY_DOCUMENT = `{
	"foo" : "bar"
	}`

func TestGcpGkeContainerMonitoredResources(t *testing.T) {
	os.Setenv("KUBERNETES_SERVICE_HOST", "127.0.0.1")
	if gcpMetadataConfigMap == nil {
		gcpMetadataConfigMap = make(map[string]string)
	}
	gcpMetadataConfigMap[gcpInstanceIdAttr] = GCP_INSTANCE_ID_STR
	gcpMetadataConfigMap[gcpAccountIdAttr] = GCP_ACCOUNT_ID_STR
	gcpMetadataConfigMap[gcpClusterNameAttr] = GCP_GKE_CLUSTER_NAME_STR
	gcpMetadataConfigMap[gcpContainerNameEnv] = GCP_GKE_CONTAINER_NAME_STR
	gcpMetadataConfigMap[gcpZoneAttr] = GCP_ZONE_STR
	gcpMetadataConfigMap[gcpNamespaceEnv] = GCP_GKE_NAMESPACE_STR
	gcpMetadataConfigMap[gcpPodIdEnv] = GCP_GKE_POD_ID_STR
	mr := GetAutoDetectedDefaultResource()
	if mr == nil {
		t.Fatal("GcpGkeContainerMonitoredResource nil")
	}
	if mr.GetType() != ResourceTypeGkeContainer ||
		mr.GetLabels()[GkeLabelInstanceId] != GCP_INSTANCE_ID_STR ||
		mr.GetLabels()[GkeLabelProjectId] != GCP_ACCOUNT_ID_STR ||
		mr.GetLabels()[GkeLabelClusterName] != GCP_GKE_CLUSTER_NAME_STR ||
		mr.GetLabels()[GkeLabelContainerName] != GCP_GKE_CONTAINER_NAME_STR ||
		mr.GetLabels()[GkeLabelZone] != GCP_ZONE_STR ||
		mr.GetLabels()[GkeLabelNamespaceId] != GCP_GKE_NAMESPACE_STR ||
		mr.GetLabels()[GkeLabelPodId] != GCP_GKE_POD_ID_STR {
		t.Errorf("GcpGkeContainerMonitoredResource Failed: %v", mr)
	}
}

func TestGcpGceInstanceMonitoredResources(t *testing.T) {
	if awsIdentityDoc == nil {
		awsIdentityDoc = new(awsIdentityDocument)
	}

	os.Setenv("KUBERNETES_SERVICE_HOST", "")
	gcpMetadataConfigMap[gcpInstanceIdAttr] = GCP_INSTANCE_ID_STR
	gcpMetadataConfigMap[gcpAccountIdAttr] = GCP_ACCOUNT_ID_STR
	gcpMetadataConfigMap[gcpZoneAttr] = GCP_ZONE_STR
	mr := GetAutoDetectedDefaultResource()
	if mr == nil {
		t.Fatal("GcpGceInstanceMonitoredResource nil")
	}
	if mr.GetType() != ResourceTypeGceInstance ||
		mr.GetLabels()[GceLabelInstanceId] != GCP_INSTANCE_ID_STR ||
		mr.GetLabels()[GceLabelProjectId] != GCP_ACCOUNT_ID_STR ||
		mr.GetLabels()[GceLabelZone] != GCP_ZONE_STR {
		t.Errorf("GcpGceInstanceMonitoredResource Failed: %v", mr)
	}
}

func TestAwsEc2InstanceMonitoredResources(t *testing.T) {
	if gcpMetadataConfigMap == nil {
		gcpMetadataConfigMap = make(map[string]string)
	}
	os.Setenv("KUBERNETES_SERVICE_HOST", "")
	gcpMetadataConfigMap[gcpInstanceIdAttr] = ""
	parseAwsIdentityDocument([]byte(SAMPLE_AWS_IDENTITY_DOCUMENT))
	mr := GetAutoDetectedDefaultResource()
	if mr == nil {
		t.Fatal("AwsEc2InstanceMonitoredResource nil")
	}
	if mr.GetType() != ResourceTypeAwsEc2Instance ||
		mr.GetLabels()[AwsEc2LabelInstanceId] != "i-1234567890abcdef0" ||
		mr.GetLabels()[AwsEc2LabelAwsAccount] != "123456789012" ||
		mr.GetLabels()[AwsEc2LabelRegion] != "us-west-2" {
		t.Errorf("AwsEc2InstanceMonitoredResource Failed: %v", mr)
	}
}

func TestNullMonitoredResources(t *testing.T) {
	os.Setenv("KUBERNETES_SERVICE_HOST", "")
	gcpMetadataConfigMap[gcpInstanceIdAttr] = ""
	awsIdentityDoc = new(awsIdentityDocument)
	parseAwsIdentityDocument([]byte(SAMPLE_NON_AWS_IDENTITY_DOCUMENT))
	mr := GetAutoDetectedDefaultResource()
	if mr != nil {
		t.Errorf("Expected nil MonitoredResource but found %v", mr)
	}
}
