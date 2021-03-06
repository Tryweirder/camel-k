/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package install

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Masterminds/semver"

	authorization "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	console "github.com/openshift/api/console/v1"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/defaults"
)

const (
	kamelVersionAnnotation = "camel.apache.org/version"
)

// The following variables may be overridden at build time
var (
	// KamelCLIDownloadName --
	KamelCLIDownloadName = "kamel-cli"
	// KamelCLIDownloadDisplayName is the name as seen in the download page
	KamelCLIDownloadDisplayName = "kamel - Apache Camel K Command Line Interface"
	// KamelCLIDownloadDescription is the description as seen in the download page
	KamelCLIDownloadDescription = "Apache Camel K is a lightweight integration platform, born on Kubernetes, with serverless superpowers.\n\n" +
		"The `kamel` binary can be used to both configure the cluster and run integrations. " +
		"Once you've downloaded the `kamel` binary, log into the cluster using the `oc` client tool and start using the `kamel` CLI.\n\n" +
		"You can run `kamel help` to list the available commands or go to the [Camel K Website](https://camel.apache.org/projects/camel-k/) for more information."

	// KamelCLIDownloadURLTemplate is the download template with 3 missing parameters (version, version, os)
	KamelCLIDownloadURLTemplate = "https://github.com/apache/camel-k/releases/download/v%s/camel-k-client-%s-%s-64bit.tar.gz"
)

// OpenShiftConsoleDownloadLink installs the download link for the OpenShift console
func OpenShiftConsoleDownloadLink(ctx context.Context, c client.Client) error {
	// Check the ConsoleCLIDownload CRD is present, which should be starting OpenShift version 4.2.
	// That check is also enough to exclude Kubernetes clusters.
	ok, err := isAPIResourceInstalled(c, "console.openshift.io/v1", reflect.TypeOf(console.ConsoleCLIDownload{}).Name())
	if err != nil {
		return err
	} else if !ok {
		return nil
	}

	// Check for permission to create the ConsoleCLIDownload resource
	sar := &authorization.SelfSubjectAccessReview{
		Spec: authorization.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorization.ResourceAttributes{
				Group:    "console.openshift.io",
				Resource: "consoleclidownloads",
				Name:     KamelCLIDownloadName,
				Verb:     "create",
			},
		},
	}

	sar, err = c.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		if errors.IsForbidden(err) {
			// Let's just skip the ConsoleCLIDownload resource creation
			return nil
		}
		return err
	} else if !sar.Status.Allowed {
		return nil
	}

	// Check for an existing ConsoleCLIDownload resource
	existing := &console.ConsoleCLIDownload{}
	err = c.Get(ctx, types.NamespacedName{Name: KamelCLIDownloadName}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			existing = nil
		} else {
			return err
		}
	}

	if existing != nil {
		if version, ok := existing.Annotations[kamelVersionAnnotation]; ok {
			current, err := semver.NewVersion(version)
			if err != nil {
				return err
			}
			this, err := semver.NewVersion(defaults.Version)
			if err != nil {
				return err
			}
			if this.LessThan(current) {
				// Keep the most recent version
				return nil
			}
			// Else delete the older version
			err = c.Delete(ctx, existing)
			if err != nil {
				if errors.IsForbidden(err) {
					// Let's just skip the ConsoleCLIDownload resource creation
					return nil
				}
				return err
			}
		}
	}

	// Create the ConsoleCLIDownload for Kamel CLI
	link := console.ConsoleCLIDownload{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				kamelVersionAnnotation: defaults.Version,
			},
			Name: KamelCLIDownloadName,
		},
		Spec: console.ConsoleCLIDownloadSpec{
			DisplayName: KamelCLIDownloadDisplayName,
			Description: KamelCLIDownloadDescription,
			Links: []console.Link{
				{
					Text: "Download the kamel binary for Linux",
					Href: fmt.Sprintf(KamelCLIDownloadURLTemplate, defaults.Version, defaults.Version, "linux"),
				},
				{
					Text: "Download the kamel binary for Mac",
					Href: fmt.Sprintf(KamelCLIDownloadURLTemplate, defaults.Version, defaults.Version, "mac"),
				},
				{
					Text: "Download the kamel binary for Windows",
					Href: fmt.Sprintf(KamelCLIDownloadURLTemplate, defaults.Version, defaults.Version, "windows"),
				},
			},
		},
	}

	err = c.Create(ctx, &link)
	if err != nil {
		return err
	}

	return nil
}

func isAPIResourceInstalled(c client.Client, groupVersion string, kind string) (bool, error) {
	resources, err := c.Discovery().ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	for _, resource := range resources.APIResources {
		if resource.Kind == kind {
			return true, nil
		}
	}

	return false, nil
}
