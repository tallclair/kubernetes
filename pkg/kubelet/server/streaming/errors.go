/*
Copyright 2016 The Kubernetes Authors.

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

package streaming

import (
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	ErrorStreamingDisabled = errors.New("streaming methods disabled")
)

func GRPCError(err error) error {
	var code codes.Code
	switch err {
	case ErrorStreamingDisabled:
		code = codes.Unknown
	default:
		code = codes.Unknown
	}
	return grpc.Errorf(code, err.String)
}
