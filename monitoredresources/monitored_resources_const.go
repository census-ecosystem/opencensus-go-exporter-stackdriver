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

// Resource Type supported for auto detection.
// For definition refer to
// https://cloud.google.com/monitoring/custom-metrics/creating-metrics#which-resource
const (
	ResourceTypeGkeContainer   = "gke_container"
	ResourceTypeGceInstance    = "gce_instance"
	ResourceTypeAwsEc2Instance = "aws_ec2_instance"
)

// Resource labels for resource type aws_ec2_instance
// For definition refer to
// https://cloud.google.com/monitoring/api/resources#tag_aws_ec2_instance
const (
	AWSEC2LabelAwsAccount = "aws_account"
	AWSEC2LabelInstanceID = "instance_id"
	AWSEC2LabelRegion     = "region"
)

// Resource labels for resource type gce_instance
// For definition refer to
// https://cloud.google.com/monitoring/api/resources#tag_gce_instance
const (
	GCELabelProjectID  = "project_id"
	GCELabelInstanceID = "instance_id"
	GCELabelZone       = "zone"
)

// Resource labels for resource type gke_container
// For definition refer to
// https://cloud.google.com/monitoring/api/resources#tag_gke_container
const (
	GKELabelProjectID     = "project_id"
	GKELabelInstanceID    = "instance_id"
	GKELabelClusterName   = "cluster_name"
	GKELabelContainerName = "container_name"
	GKELabelNamespaceID   = "namespace_id"
	GKELabelPodID         = "pod_id"
	GKELabelZone          = "zone"
)

// Attributes retrieved from Metadata Server in case of
// gke_container and gce_instance resource types.
const (
	gcpProjectID           = "project/project-id"
	gcpInstanceID          = "instance/id"
	gcpZone                = "instance/zone"
	gcpClusterName         = "instance/attributes/cluster-name"
	gcpContainerNameEnvVar = "CONTAINER_NAME"
	gcpNamespaceEnvVar     = "NAMESPACE"
	gcpPodIDEnvVar         = "HOSTNAME"
)
