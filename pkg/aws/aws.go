// Copyright 2018-2021 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package aws_gsc

// This file implements an AWS specific means for extracting credentials via session
// objects

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/go-stack/stack"
	"github.com/jjeffery/kv"
)

// GetCredentials is used to extract the AWS credentials using the AWS standard mechanisims for specification of
// things such as env var AWS_PROFILE values or directly using env vars etc.  The intent is that the AWS credentials
// are obtained using the stock AWS client side APIs and then can be used to access minio, as one example.
func GetCredentials() (creds *credentials.Value, err kv.Error) {

	sess, errGo := session.NewSession(&aws.Config{
		Region:      aws.String(""),
		Credentials: credentials.NewSharedCredentials("", ""),
	})
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	values, errGo := sess.Config.Credentials.Get()
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	return &values, nil
}
