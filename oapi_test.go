// Copyright 2020 Herman Slatman
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openapi

import (
	"reflect"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func getDefaultSpecification() *openapi3.T {
	defaultServers := []*openapi3.Server{
		{
			URL:         "http://petstore.swagger.io/v1",
			Description: "some description",
			Variables:   make(map[string]*openapi3.ServerVariable),
		},
	}
	return &openapi3.T{Servers: defaultServers}
}

func check(actual []*openapi3.Server, expected []string) bool {
	servers := []string{}
	for _, s := range actual {
		servers = append(servers, s.URL)
	}
	return reflect.DeepEqual(servers, expected)
}

func Test_addAdditionalServers(t *testing.T) {

	tests := []struct {
		name              string
		additionalServers []string
		expected          []string
	}{
		{name: "nil", additionalServers: nil, expected: []string{"http://petstore.swagger.io/v1"}},
		{name: "empty list", additionalServers: []string{}, expected: []string{"http://petstore.swagger.io/v1"}},
		{name: "list with empty string", additionalServers: []string{""}, expected: []string{"http://petstore.swagger.io/v1"}},
		{name: "some wrong url", additionalServers: []string{"some-wrong-url"}, expected: []string{"http://petstore.swagger.io/v1"}},
		{name: "default /", additionalServers: []string{"/"}, expected: []string{"http://petstore.swagger.io/v1", "/"}},
		{name: "relative /v1/api", additionalServers: []string{"/v1/api"}, expected: []string{"http://petstore.swagger.io/v1", "/v1/api"}},
		{name: "ip", additionalServers: []string{"http://127.0.0.1/"}, expected: []string{"http://petstore.swagger.io/v1", "http://127.0.0.1/"}},
		{name: "some-host", additionalServers: []string{"http://some-host:8080/v1/some-api"}, expected: []string{"http://petstore.swagger.io/v1", "http://some-host:8080/v1/some-api"}},
		{name: "multiple", additionalServers: []string{"https://localhost:9443/api", "http://localhost:9443/api"}, expected: []string{"http://petstore.swagger.io/v1", "https://localhost:9443/api", "http://localhost:9443/api"}},
		{name: "websockets", additionalServers: []string{"ws://some-websocket.example"}, expected: []string{"http://petstore.swagger.io/v1", "ws://some-websocket.example"}},
		{name: "secure websockets", additionalServers: []string{"wss://some-websocket.example"}, expected: []string{"http://petstore.swagger.io/v1", "wss://some-websocket.example"}},
	}

	for _, tt := range tests {
		s := getDefaultSpecification()
		a := addAdditionalServers(s, tt.additionalServers)
		if len(a.Servers) != len(tt.expected) {
			t.Errorf("expected %d server(s)in oapi specification but got %d in test: %s", len(tt.expected), len(a.Servers), tt.name)
		}
		if !check(a.Servers, tt.expected) {
			t.Errorf("expected does not equal actual arary of servers in test: %s", tt.name)
		}
	}
}
