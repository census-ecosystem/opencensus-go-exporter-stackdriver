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

// Resource Type supported for auto detection.
// For definition refer to
// https://cloud.google.com/monitoring/custom-metrics/creating-metrics#which-resource
const (
	ResourceTypeGkeContainer   string = "gke_container"
	ResourceTypeGceInstance    string = "gce_instance"
	ResourceTypeAwsEc2Instance string = "aws_ec2_instance"
)

// Resource labels for resource type aws_ec2_instance
// For definition refer to
// https://cloud.google.com/monitoring/api/resources#tag_aws_ec2_instance
const (
	AwsEc2LabelAwsAccount string = "aws_account"
	AwsEc2LabelInstanceId string = "instance_id"
	AwsEc2LabelRegion     string = "region"
)

// Resource labels for resource type gce_instance
// For definition refer to
// https://cloud.google.com/monitoring/api/resources#tag_gce_instance
const (
	GceLabelProjectId  string = "project_id"
	GceLabelInstanceId string = "instance_id"
	GceLabelZone       string = "zone"
)

// Resource labels for resource type gke_container
// For definition refer to
// https://cloud.google.com/monitoring/api/resources#tag_gke_container
const (
	GkeLabelProjectId     string = "project_id"
	GkeLabelInstanceId    string = "instance_id"
	GkeLabelClusterName   string = "cluster_name"
	GkeLabelContainerName string = "container_name"
	GkeLabelNamespaceId   string = "namespace_id"
	GkeLabelPodId         string = "pod_id"
	GkeLabelZone          string = "zone"
)

// Fields parsed from AWS Identity Document.
const (
	awsAccountId  string = "accountId"
	awsInstanceId string = "instanceId"
	awsRegion     string = "region"
)

// Attributes retrieved from Metadata Server in case of
// gke_container and gce_instance resource types.
const (
	gcpAccountIdAttr    string = "project/project-id"
	gcpInstanceIdAttr   string = "instance/id"
	gcpZoneAttr         string = "instance/zone"
	gcpClusterNameAttr  string = "instance/attributes/cluster-name"
	gcpContainerNameEnv string = "CONTAINER_NAME"
	gcpNamespaceEnv     string = "NAMESPACE"
	gcpPodIdEnv         string = "HOSTNAME"
)
