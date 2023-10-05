// Copyright 2018-2023 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package aws_gsc

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-test/deep"

	"github.com/go-stack/stack"
	"github.com/karlmutch/envflag"
)

func TestMain(m *testing.M) {
	// Only perform this Parsed check inside the test framework. Do not be tempted
	// to do this in the main of any production package
	//
	if !flag.Parsed() {
		envflag.Parse()
	}
	m.Run()
}

func TestAWSExtractCreds(t *testing.T) {

	configs := []string{`[profile default]
output=json
region=us-west-2

[default]
region=us-west-2
output=json
`,
		`[profile test]
output-json
region=us-west-2`,
	}

	creds := []string{`[default]
aws_access_key_id=default_aws_access_key_id
aws_secret_access_key=default_aws_secret_access_key
`, `[test]
aws_access_key_id=test_aws_access_key_id
aws_secret_access_key=test_aws_secret_access_key
`,
	}

	// Write test cred files
	tmpDir, errGo := os.MkdirTemp("", "TestAWSExtractCreds")
	if errGo != nil {
		t.Fatal(errGo.Error(), "stack", stack.Trace().TrimRuntime())
	}
	defer func() {
		if errGo := os.RemoveAll(tmpDir); errGo != nil {
			slog.Warn(errGo.Error(), "stack", stack.Trace().TrimRuntime())
		}
	}()
	if errGo := os.Chmod(tmpDir, 0700); errGo != nil {
		t.Fatal(errGo.Error(), "tmpDir", tmpDir, "stack", stack.Trace().TrimRuntime())
	}

	configFN := filepath.Join(tmpDir, "config")
	credsFN := filepath.Join(tmpDir, "credentials")
	fNames := []string{configFN, credsFN}

	if errGo := os.WriteFile(configFN, []byte(configs[0]), 0600); errGo != nil {
		t.Fatal(errGo.Error(), "fn", configFN, "stack", stack.Trace().TrimRuntime())
	}

	if errGo := os.WriteFile(credsFN, []byte(creds[0]), 0600); errGo != nil {
		t.Fatal(errGo.Error(), "fn", configFN, "stack", stack.Trace().TrimRuntime())
	}

	// Stock with default
	cred, err := AWSExtractCreds(fNames, "default")
	if err != nil {
		t.Fatal(err.Error(), "stack", stack.Trace().TrimRuntime())
	}
	extractCreds, errGo := cred.Creds.Get()
	if errGo != nil {
		t.Fatal(errGo.Error(), "stack", stack.Trace().TrimRuntime())
	}
	if diff := deep.Equal(extractCreds.AccessKeyID, "default_aws_access_key_id"); diff != nil {
		t.Fatal(diff, "stack", stack.Trace().TrimRuntime())
	}
	if diff := deep.Equal(extractCreds.SecretAccessKey, "default_aws_secret_access_key"); diff != nil {
		t.Fatal(diff, "stack", stack.Trace().TrimRuntime())
	}

	// Stock non default
	if errGo := os.WriteFile(configFN, []byte(configs[0]+configs[1]), 0600); errGo != nil {
		t.Fatal(errGo.Error(), "fn", configFN, "stack", stack.Trace().TrimRuntime())
	}

	if errGo := os.WriteFile(credsFN, []byte(creds[0]+creds[1]), 0600); errGo != nil {
		t.Fatal(errGo.Error(), "fn", configFN, "stack", stack.Trace().TrimRuntime())
	}

	cred, err = AWSExtractCreds(fNames, "default")
	if err != nil {
		t.Fatal(err.Error(), "stack", stack.Trace().TrimRuntime())
	}
	extractCreds, errGo = cred.Creds.Get()
	if errGo != nil {
		t.Fatal(errGo.Error(), "stack", stack.Trace().TrimRuntime())
	}
	if diff := deep.Equal(extractCreds.AccessKeyID, "default_aws_access_key_id"); diff != nil {
		t.Fatal(diff, "stack", stack.Trace().TrimRuntime())
	}
	if diff := deep.Equal(extractCreds.SecretAccessKey, "default_aws_secret_access_key"); diff != nil {
		t.Fatal(diff, "stack", stack.Trace().TrimRuntime())
	}

	cred, err = AWSExtractCreds(fNames, "test")
	if err != nil {
		t.Fatal(err.Error(), "stack", stack.Trace().TrimRuntime())
	}
	extractCreds, errGo = cred.Creds.Get()
	if errGo != nil {
		t.Fatal(errGo.Error(), "stack", stack.Trace().TrimRuntime())
	}
	if diff := deep.Equal(extractCreds.AccessKeyID, "test_aws_access_key_id"); diff != nil {
		t.Fatal(diff, "stack", stack.Trace().TrimRuntime())
	}
	if diff := deep.Equal(extractCreds.SecretAccessKey, "test_aws_secret_access_key"); diff != nil {
		t.Fatal(diff, "stack", stack.Trace().TrimRuntime())
	}
}
