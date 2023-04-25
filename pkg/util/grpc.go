/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package util

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"google.golang.org/grpc/credentials"
)

func NewClientTLSFromBase64(certBase64, serverNameOverride string) (credentials.TransportCredentials, error) {
	b, err := base64.StdEncoding.DecodeString(certBase64)
	if err != nil {
		return nil, err
	}
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(b) {
		return nil, fmt.Errorf("credentials: failed to append certificates")
	}
	return credentials.NewTLS(&tls.Config{ServerName: serverNameOverride, RootCAs: cp}), nil
}
