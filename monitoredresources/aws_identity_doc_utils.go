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
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

var awsIdentityDoc *awsIdentityDocument

// awsIdentityDocument is used to store parsed AWS Identity Document.
type awsIdentityDocument struct {
	AccountId  string
	InstanceId string
	Region     string
}

var runningOnAwsEc2 = false

func isRunningOnAwsEc2() bool {
	return runningOnAwsEc2
}

// retrieveAWSIdentityDocument attempts to retrieve AWS Identity Document.
// If the environment is AWS EC2 Instance then a valid document is retrieved.
// Relevant attributes from the document are stored in awsIdentityDoc.
// This is only done once.
func retrieveAWSIdentityDocument() {
	awsIdentityDoc = new(awsIdentityDocument)
	c := ec2metadata.New(session.New())
	ec2InstanceIdentifyDocument, err := c.GetInstanceIdentityDocument()
	if err != nil {
		runningOnAwsEc2 = false
		return
	}
	runningOnAwsEc2 = true
	awsIdentityDoc.Region = ec2InstanceIdentifyDocument.Region
	awsIdentityDoc.InstanceId = ec2InstanceIdentifyDocument.InstanceID
	awsIdentityDoc.AccountId = ec2InstanceIdentifyDocument.AccountID
}

// getAWSIdentityDocument returns AWS Identity Doc.
func getAWSIdentityDocument() *awsIdentityDocument {
	if awsIdentityDoc == nil {
		awsIdentityDoc = new(awsIdentityDocument)
	}
	return awsIdentityDoc
}
