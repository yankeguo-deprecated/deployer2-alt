package cmds

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gofrs/flock"
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
)

var (
	regexpNonAlphaNumeric = regexp.MustCompile(`[^0-9a-zA-Z]+`)
)

func sanitizePathToPathComponent(path string) string {
	digest := md5.Sum([]byte(path))
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

func ExecuteInDocker(image string, caches []string, script string) (err error) {
	// 计算基础缓存目录
	var home string
	if home, err = os.UserHomeDir(); err != nil {
		return
	}
	cacheBaseDir := filepath.Join(home, ".deployer2-builder-cache", sanitizePathToPathComponent(image))
	if err = os.MkdirAll(cacheBaseDir, 0755); err != nil {
		return
	}
	// 使用 filelock 防止 cache 目录被同时多个容器使用
	fileLock := flock.New(filepath.Join(cacheBaseDir, "lock"))
	if err = fileLock.Lock(); err != nil {
		return
	}
	defer fileLock.Unlock()
	// 将 caches 换算为 mounts
	var mounts []string
	for _, cache := range caches {
		mounts = append(mounts, filepath.Join(cacheBaseDir, sanitizePathToPathComponent(cache))+":"+cache)
	}
	// 准备 Docker 命令
	name := "docker"
	args := []string{"run"}
	// 准备挂载命令
	for _, mount := range mounts {
		args = append(args, "-v", mount)
	}
	// 挂载 /workspace
	var wd string
	if wd, err = os.Getwd(); err != nil {
		return
	}
	args = append(args, "-v", "/workspace:"+wd)
	// 挂载 /deployer2-build-script.sh
	args = append(args, "-v", "/deployer2-build-script.sh:"+script)
	// 准备镜像和 bash -l 命令
	args = append(args, image, "bash", "-l")

	// 准备 stdin
	buf := &bytes.Buffer{}
	buf.WriteString("#!/bin/bash\n")
	buf.WriteString("set -eux\n")
	buf.WriteString("cd /workspace\n")
	buf.WriteString("chmod +x /deployer2-build-script.sh\n")
	buf.WriteString("/deployer2-build-script.sh\n")
	buf.WriteString(fmt.Sprintf("chown -R %d:%d /workspace\n", os.Getuid(), os.Getgid()))

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
