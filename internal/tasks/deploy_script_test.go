package tasks

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Subilan/go-aliyunmc/internal/config"
	"github.com/Subilan/go-aliyunmc/internal/remote_util"
	"github.com/Subilan/go-aliyunmc/internal/testutil"
)

// 部署脚本必须保持可渲染且语法正确；同时每个步骤都要有名称，失败点才能被清晰记录。
func TestDeployScriptTemplatesCompile(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	root, err := testutil.ProjectRoot()
	if err != nil {
		t.Fatalf("locate project root failed: %v", err)
	}

	vars := deployTaskVars{
		TasksDeployVars: config.TasksDeployVars{
			SSHAccessPublicKey: "ssh-rsa AAAA access",
			JavaVersion:        21,
			Packages:           []string{"curl", "zip"},
			ArchiveOSSPath:     "oss://bucket/archive",
		},
		SSHGomcPublicKey: "ssh-ed25519 AAAA gomc",
		Username:         "mc",
		Password:         "secret",
		RegionId:         "cn-test",
		AccessKeyId:      "ak",
		AccessKeySecret:  "sk",
		DataDiskSize:     100,
	}

	seen := make(map[string]bool, len(deploySteps))
	for i, step := range deploySteps {
		if step.name == "" {
			t.Fatalf("deploy step %d has no name", i+1)
		}
		if seen[step.name] {
			t.Fatalf("duplicate deploy step name: %s", step.name)
		}
		seen[step.name] = true

		if step.stepType != deployStepScript {
			continue
		}

		templatePath := filepath.Join(root, scriptBasePath, step.script)
		script, err := remote_util.RenderScriptTemplate(templatePath, vars)
		if err != nil {
			t.Fatalf("render %s failed: %v", step.script, err)
		}

		cmd := exec.Command("bash", "-n")
		cmd.Stdin = strings.NewReader(script)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("bash syntax check failed for %s: %v\n%s", step.script, err, out)
		}
	}
}
