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
	"log"

	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

const awsIdentityDocumentUri = "http://169.254.169.254/latest/dynamic/instance-identity/document"

var awsIdentityDoc *awsIdentityDocument

// awsIdentityDocument is used to store parsed AWS Identity Document.
type awsIdentityDocument struct {
	AccountId  string
	InstanceId string
	Region     string
}

var runningOnAwsEc2 = false

// isRunningOnAwsEc2 returns true if the application is running on AWS EC2 Instance.
func isRunningOnAwsEc2() bool {
	return runningOnAwsEc2
}

// parseAwsIdentityDocument parse byte encoded json document.
func parseAwsIdentityDocument(bytes []byte) {
	if awsIdentityDoc != nil {
		runningOnAwsEc2 = false
		jsonErr := json.Unmarshal(bytes, awsIdentityDoc)
		if jsonErr != nil {
			log.Printf("Error parsing json AWS Identity Document: %v, %s", jsonErr, string(bytes))
		} else {
			if awsIdentityDoc.InstanceId != "" {
				runningOnAwsEc2 = true
			}
		}
	}
}

// init detects if the application environment is AWS EC2 Instance by
// establishing HTTP connection to AWS instance identity document url.
// If the environment is AWS EC2 Instance then the document returned will be
// a valid JSON document. The document is parsed and stored in awsIdentityDoc.
// This is only done once.
func init() {
	awsIdentityDoc = new(awsIdentityDocument)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}, Timeout: time.Second * 2}

	req, err := http.NewRequest("GET", awsIdentityDocumentUri, nil)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading AWS Identity Document: %v", err)
		return
	}

	parseAwsIdentityDocument(bytes)
}

// getValueFromAwsIdentityDocument returns value for a given key parsed
// from AWS Identity Document.
// Returns "" if the key is not found.
func getValueFromAwsIdentityDocument(key string) string {
	switch key {
	case "accountId":
		return awsIdentityDoc.AccountId
	case "instanceId":
		return awsIdentityDoc.InstanceId
	case "region":
		return awsIdentityDoc.Region
	}
	return ""
}
