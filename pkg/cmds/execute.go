package cmds

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	KubectlPatchRetries = 3
)

func Execute(name string, args ...string) (err error) {
	log.Printf("执行: %s %s", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		log.Printf("执行完成: 返回值(%d)", ee.ExitCode())
	}
	return
}

func ExecuteWithRetries(retry int, name string, args ...string) (err error) {
	if retry < 1 {
		retry = 1
	}
	for {
		if err = Execute(name, args...); err == nil {
			return
		}

		retry--
		if retry == 0 {
			return
		}
		time.Sleep(time.Second * 5)
		log.Printf("5s 后重试, 剩余 %d", retry)
	}
}

func DockerVersion() error {
	return Execute("docker", "--version")
}

func DockerBuild(dockerFile, imageName string) error {
	return Execute("docker", "build", "-t", imageName, "-f", dockerFile, ".")
}

func DockerTag(imageName string, imageNameAlt string) error {
	return Execute("docker", "tag", imageName, imageNameAlt)
}

func DockerPush(imageName string, configDir string) error {
	return Execute("docker", "--config", configDir, "push", imageName)
}

func DockerRemoveImage(imageName string) error {
	return Execute("docker", "rmi", imageName)
}

func KubectlVersion(kubeconfig string) error {
	return ExecuteWithRetries(KubectlPatchRetries, "kubectl", "--kubeconfig", kubeconfig,
		"version")
}

func KubectlPatch(kubeconfig, namespace, workload, workloadType, patch string) error {
	return ExecuteWithRetries(KubectlPatchRetries, "kubectl", "--kubeconfig", kubeconfig,
		"--namespace", namespace, "patch", workloadType+"s/"+workload, "-p", patch)
}
