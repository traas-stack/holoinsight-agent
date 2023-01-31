// Package agentmeta 这个文件本来是放在 agent 包下, 但 agent 包下包含一些 manager, 可能会引起循环依赖, 故单独放到 agentmeta 包下
package agentmeta

import (
	"github.com/TRaaSStack/holoinsight-agent/pkg/appconfig"
	"github.com/TRaaSStack/holoinsight-agent/pkg/core"
	"github.com/TRaaSStack/holoinsight-agent/pkg/util"
	"fmt"
	"github.com/google/uuid"
	"io"
	"os"
)

// 读配置文件
// TODO 配置文件在哪个目录? 是否固定? 至少我们要对这个目录有写权限

var agentID = "mock"

func GetAgentId() string {
	return agentID
}

func SetupAgentId() error {
	agentIdPath := "data/agent_id"

	// read from agent_id file
	file, err := os.Open(agentIdPath)
	if err == nil {
		defer file.Close()

		b, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		agentID = string(b)
		return nil
	}

	// maybe permission error
	if !os.IsNotExist(err) {
		return err
	}

	// write to temp file first and then move it to agentIdPath
	tempFile, err := os.CreateTemp("data", "agent_id-*")
	if err != nil {
		return err
	}
	fmt.Printf("cd .."+
		"create temp file %s\n", tempFile.Name())
	defer tempFile.Close()

	agentID, err = generateAgentID()

	fmt.Fprintf(os.Stdout, "gen agentID %s\n", agentID)

	if _, err = tempFile.WriteString(agentID); err != nil {
		fmt.Fprintf(os.Stderr, "write agent id error %s %+v\n", agentIdPath, err)
		return err
	}
	if util.IsLinux() {
		if err := tempFile.Chmod(0644); err != nil {
			fmt.Fprintf(os.Stderr, "chmod error %+v\n", err)
			return err
		}
	}

	// TODO 在 golang 里会引起 invalid cross-device link
	return os.Rename(tempFile.Name(), agentIdPath)
}

func generateAgentID() (string, error) {
	if appconfig.StdAgentConfig.Mode == core.AgentModeCentral {
		// TODO hard code
		centralTenant := os.Getenv("HI_CENTRAL_TENANT")
		if centralTenant != "" {
			return centralTenant + "_" + appconfig.StdAgentConfig.Central.Name + "_" + util.GetHostname(), nil
		}
	}

	uuidInstance, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return uuidInstance.String(), nil
}
