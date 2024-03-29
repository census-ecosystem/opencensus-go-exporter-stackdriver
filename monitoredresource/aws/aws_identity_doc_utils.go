// Copyright 2020, OpenCensus Authors
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

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

// awsIdentityDocument is used to store parsed AWS Identity Document.
type awsIdentityDocument struct {
	// accountID is the AWS account number for the VM.
	accountID string

	// instanceID is the instance id of the instance.
	instanceID string

	// Region is the AWS region for the VM.
	region string
}

// retrieveAWSIdentityDocument attempts to retrieve AWS Identity Document.
// If the environment is AWS EC2 Instance then a valid document is retrieved.
// Relevant attributes from the document are stored in awsIdentityDoc.
// This is only done once.
func retrieveAWSIdentityDocument() *awsIdentityDocument {
	awsIdentityDoc := awsIdentityDocument{}
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil
	}
	c := imds.NewFromConfig(cfg)
	ec2InstanceIdentifyDocument, err := c.GetInstanceIdentityDocument(ctx, nil)
	if err != nil {
		return nil
	}
	awsIdentityDoc.region = ec2InstanceIdentifyDocument.Region
	awsIdentityDoc.instanceID = ec2InstanceIdentifyDocument.InstanceID
	awsIdentityDoc.accountID = ec2InstanceIdentifyDocument.AccountID

	return &awsIdentityDoc
}
