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
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// NOTE: currently unused
func determineRequestURL(request *http.Request, prefix string) (*url.URL, error) {
	// TODO: determine whether this is (still) required when we're checking the servers (again)
	fullURL := request.URL.String()
	if !strings.HasPrefix(fullURL, prefix) {
		return nil, fmt.Errorf("prefix to cut (%s) from URL (%s) is incorrect", prefix, fullURL)
	}
	url, err := url.ParseRequestURI(strings.TrimPrefix(fullURL, prefix))
	if err != nil {
		return nil, fmt.Errorf("error while cutting off prefix (%s) from URL (%s)", prefix, fullURL)
	}
	fmt.Println(fmt.Sprintf("cut off url: %s", url.String()))
	return url, nil
}
