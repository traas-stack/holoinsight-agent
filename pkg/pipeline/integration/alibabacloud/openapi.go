/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

import (
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	ims20190815 "github.com/alibabacloud-go/ims-20190815/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

func isValidKeyAndSecret(endpoint, ak, sk string) (bool, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(ak),
		AccessKeySecret: tea.String(sk),
	}
	// 访问的域名
	// 这里可以写任意的域名 它总是查出所有ecs的指标
	config.Endpoint = tea.String(endpoint)
	client, err := ims20190815.NewClient(config)
	if err != nil {
		return false, err
	}
	_, err = client.GetAccountSummary()
	if err != nil {
		return false, err
	}
	return true, nil
}
