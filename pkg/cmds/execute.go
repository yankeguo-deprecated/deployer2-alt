package cmds

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	KubectlPatchRetries = 3

	InDockerWorkspace = "/workspace"
	InDockerScript    = "/deployer2-in-docker-script.sh"
)

var (
	regexpNonAlphaNumeric = regexp.MustCompile(`[^0-9a-zA-Z]+`)
)

func sanitizePathToPathComponent(path string) string {
	digest := md5.Sum([]byte(path))
	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}
	return regexpNonAlphaNumeric.ReplaceAllString(path, "-") + "-" + hex.EncodeToString(digest[:])
}

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

func ExecuteInDocker(image string, cacheDir string, caches []string, script string) (err error) {
	// 将 caches 换算为 mounts
	var mounts []string
	for _, cache := range caches {
		mounts = append(mounts, filepath.Join(cacheDir, sanitizePathToPathComponent(cache))+":"+cache)
	}
	// 映射 工作目录
	var wd string
	if wd, err = os.Getwd(); err != nil {
		return
	}
	mounts = append(mounts, wd+":"+InDockerWorkspace)
	// 映射主脚本
	mounts = append(mounts, script+":"+InDockerScript)
	// 准备 Docker 命令
	name := "docker"
	args := []string{"run", "-i", "--rm", "--network", "host", "--ipc", "host", "--pid", "host"}
	// 准备挂载命令
	for _, mount := range mounts {
		args = append(args, "-v", mount)
	}
	// 准备镜像和 bash -l 命令
	args = append(args, image, "bash", "-l")

	// 准备 stdin
	buf := &bytes.Buffer{}
	_, _ = fmt.Fprintf(buf, "#!/bin/bash\n")
	_, _ = fmt.Fprintf(buf, "set -eux\n")
	_, _ = fmt.Fprintf(buf, "cd '%s'\n", InDockerWorkspace)
	_, _ = fmt.Fprintf(buf, "chmod +x '%s'\n", InDockerScript)
	_, _ = fmt.Fprintf(buf, "'%s'\n", InDockerScript)
	_, _ = fmt.Fprintf(buf, "chown -R %d:%d '%s'\n", os.Getuid(), os.Getgid(), InDockerWorkspace)

	log.Println("使用镜像: ", image)
	for _, mount := range mounts {
		log.Println("映射路径: ", mount)
	}

	log.Printf("执行: %s %s", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdin = buf
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
