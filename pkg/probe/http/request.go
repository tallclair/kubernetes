/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package http

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/component-base/version"
	"k8s.io/kubernetes/pkg/probe"
)

func NewProbeRequest(url *url.URL, headers http.Header) (*http.Request, error) {
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	if headers == nil {
		headers = http.Header{}
	}
	if _, ok := headers["User-Agent"]; !ok {
		// explicitly set User-Agent so it's not set to default Go value
		v := version.Get()
		headers.Set("User-Agent", fmt.Sprintf("kube-probe/%s.%s", v.Major, v.Minor))
	}
	if _, ok := headers["Accept"]; !ok {
		// Accept header was not defined. accept all
		headers.Set("Accept", "*/*")
	} else if headers.Get("Accept") == "" {
		// Accept header was overridden but is empty. removing
		headers.Del("Accept")
	}
	req.Header = headers
	req.Host = headers.Get("Host")

	return req, nil
}

func NewRequestForHTTPGetAction(httpGet *v1.HTTPGetAction, container *v1.Container, podIP string) (*http.Request, error) {
	scheme := strings.ToLower(string(httpGet.Scheme))
	host := httpGet.Host
	if host == "" {
		host = podIP
	}
	port, err := probe.ResolveContainerPort(httpGet.Port, container)
	if err != nil {
		return nil, err
	}
	path := httpGet.Path
	url := formatURL(scheme, host, port, path)
	headers := v1HeaderToHttpHeader(httpGet.HTTPHeaders)

	return NewProbeRequest(url, headers)
}

// formatURL formats a URL from args.  For testability.
func formatURL(scheme string, host string, port int, path string) *url.URL {
	u, err := url.Parse(path)
	// Something is busted with the path, but it's too late to reject it. Pass it along as is.
	if err != nil {
		u = &url.URL{
			Path: path,
		}
	}
	u.Scheme = scheme
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u
}

// buildHeaderMap takes a list of HTTPHeader <name, value> string
// pairs and returns a populated string->[]string http.Header map.
func v1HeaderToHttpHeader(headerList []v1.HTTPHeader) http.Header {
	headers := make(http.Header)
	for _, header := range headerList {
		headers[header.Name] = append(headers[header.Name], header.Value)
	}
	return headers
}
