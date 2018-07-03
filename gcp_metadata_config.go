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
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const gcpMetadataUrl = "http://metadata/computeMetadata/v1/"

// gcpMetadataConfigMap holds mapping of Attribute Name and Attribute Value
// in GKE container and GCE instance environment
var gcpMetadataConfigMap map[string]string

// init retrieves value of each Attribute from Metadata Server (see gcpMetadataUrl)
// in GKE container and GCE instance environment.
// Some attributes are retrieved from the system environment.
// This is only executed once.
func init() {
	gcpMetadataConfigMap = make(map[string]string)

	gcpMetadataConfigMap[gcpInstanceIdAttr] = getAttribute(gcpInstanceIdAttr)
	gcpMetadataConfigMap[gcpAccountIdAttr] = getAttribute(gcpAccountIdAttr)
	gcpMetadataConfigMap[gcpClusterNameAttr] = getAttribute(gcpClusterNameAttr)
	gcpMetadataConfigMap[gcpZoneAttr] = getZone()
	gcpMetadataConfigMap[gcpContainerNameEnv] = os.Getenv(gcpContainerNameEnv)
	gcpMetadataConfigMap[gcpNamespaceEnv] = os.Getenv(gcpNamespaceEnv)
	gcpMetadataConfigMap[gcpPodIdEnv] = os.Getenv(gcpPodIdEnv)

}

// getZone retrieves the zone attribute from Metadata Server.
// It converts into simple zone name from fully qualified name.
// For example: 'projects/{project-id}/zones/us-central1-c' is converted to 'us-central1-c'
func getZone() string {

	zoneStr := getAttribute(gcpZoneAttr)
	if strings.Contains(zoneStr, "/") {
		subStrs := strings.Split(zoneStr, "/")
		return subStrs[len(subStrs)-1]
	}

	return zoneStr
}

// getAttribute retrieves value of attribute attributeName from Metadata Server.
// Returns "" if the application is not running in GKE container or GCE Instance
// environment.
func getAttribute(attributeName string) string {
	client := &http.Client{Timeout: time.Second * 2}
	req, err := http.NewRequest("GET", gcpMetadataUrl+attributeName, nil)
	if err != nil {
		return ""
	}
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		// do nothing
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading GCP Metadata attribute %s: %v", attributeName, err)
		return ""
	}
	return string(bytes)
}

// getValueFromGcpMetadataConfig returns value of attribute attributeName from
// map stored locally.
func getValueFromGcpMetadataConfig(attributeName string) string {
	return gcpMetadataConfigMap[attributeName]
}
