package main

import (
	"encoding/json"
	"errors"
	"flag"
	"github.com/acicn/deployer2/pkg/cmds"
	"github.com/acicn/deployer2/pkg/image_tracker"
	"github.com/guoyk93/tempfile"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func exit(err *error) {
	if *err != nil {
		log.Println("错误退出:", (*err).Error())
		os.Exit(1)
	} else {
		log.Println("正常退出")
	}
}

func main() {
	var err error
	defer exit(&err)
	defer tempfile.DeleteAll()

	log.SetOutput(os.Stdout)
	log.SetPrefix("[deployer2] ")

	var (
		optManifest      string
		optImage         string
		optProfile       string
		optWorkloads     UniversalWorkloads
		optCPU           UniversalResource
		optMEM           UniversalResource
		optSkipDeploy    bool
		optIgnoreBuilder bool

		imageNames   ImageNames
		imageTracker = image_tracker.New()
	)

	flag.StringVar(&optManifest, "manifest", "deployer.yml", "指定描述文件")
	flag.StringVar(&optImage, "image", "", "镜像名")
	flag.StringVar(&optProfile, "profile", "", "指定环境名")
	flag.BoolVar(&optSkipDeploy, "skip-deploy", false, "跳过部署流程")
	flag.BoolVar(&optIgnoreBuilder, "ignore-builder", false, "don't use builder image")
	flag.Var(&optWorkloads, "workload", "指定目标工作负载，格式为 \"CLUSTER/NAMESPACE/TYPE/NAME[/CONTAINER]\"")
	flag.Var(&optCPU, "cpu", "指定 CPU 配额，格式为 \"MIN:MAX\"，单位为 m (千分之一核心)")
	flag.Var(&optMEM, "mem", "指定 MEM 配额，格式为 \"MIN:MAX\"，单位为 Mi (兆字节)")
	flag.Parse()

	// 从 $JOB_NAME 获取 image 和 profile 信息
	if optImage == "" || optProfile == "" {
		envJobName := strings.TrimSpace(os.Getenv("CCI_JOB_NAME"))
		if envJobName == "" {
			envJobName = strings.TrimSpace(os.Getenv("JOB_NAME"))
		}
		if jobNameSplits := strings.Split(envJobName, "."); len(jobNameSplits) == 2 {
			if optImage == "" {
				optImage = jobNameSplits[0]
			}
			if optProfile == "" {
				optProfile = jobNameSplits[1]
			}
		} else {
			err = errors.New("缺少 --image 或者 --profile 参数，且无法从 $JOB_NAME 获得有用信息")
			return
		}
	}

	// 从 $BUILD_NUMBER 决定标签
	buildNumber := strings.TrimSpace(os.Getenv("GIT_COMMIT_SHORT"))
	if buildNumber == "" {
		buildNumber = strings.TrimSpace(os.Getenv("CI_BUILD_NUMBER"))
	}
	if buildNumber == "" {
		buildNumber = strings.TrimSpace(os.Getenv("BUILD_NUMBER"))
	}
	if buildNumber != "" {
		imageNames = append(imageNames, optImage+":"+optProfile+"-build-"+buildNumber)
	}
	imageNames = append(imageNames, optImage+":"+optProfile)

	log.Println("------------ deployer2 ------------")

	// 打印 Docker 版本
	_ = cmds.DockerVersion()

	// 加载本地清单文件，即 deployer.yml
	var manifest Manifest
	log.Printf("清单文件: %s", optManifest)
	if err = LoadManifestFile(optManifest, &manifest); err != nil {
		return
	}

	// 加载本地清单文件中对应的 Profile
	log.Printf("使用环境: %s", optProfile)
	var profile Profile
	if profile, err = manifest.Profile(optProfile); err != nil {
		return
	}
	// 如果命令行指定了 --mem 和 --cpu，覆盖 Profile 文件中的设置
	if !optCPU.IsZero() {
		profile.Resource.CPU = &optCPU
	}
	if !optMEM.IsZero() {
		profile.Resource.MEM = &optMEM
	}
	var fileBuild, filePackage string
	if fileBuild, filePackage, err = profile.GenerateFiles(); err != nil {
		return
	}
	log.Printf("写入构建文件: %s", fileBuild)
	log.Printf("写入打包文件: %s", filePackage)

	// 执行构建脚本
	if profile.Builder.Image != "" && !optIgnoreBuilder {
		log.Println("------------ 使用容器构建 ------------")
		cacheGroup := profile.Builder.CacheGroup
		if cacheGroup == "" {
			cacheGroup = "default"
		}
		var home string
		if home, err = os.UserHomeDir(); err != nil {
			return
		}
		if err = cmds.ExecuteInDocker(
			profile.Builder.Image,
			filepath.Join(home, ".deployer2-builder-cache", cacheGroup),
			profile.Builder.Caches,
			fileBuild,
		); err != nil {
			return
		}
	} else {
		log.Println("------------ 构建 ------------")
		if err = cmds.Execute(fileBuild); err != nil {
			return
		}
	}
	log.Println("构建完成")

	// 执行打包脚本，即 docker build
	log.Println("------------ 打包 ------------")
	if err = cmds.DockerBuild(filePackage, imageNames.Primary()); err != nil {
		return
	}
	log.Printf("打包完成: %s", imageNames.Primary())

	// 追踪涉及到的所有临时镜像，用来做事后清理
	imageTracker.Add(imageNames.Primary())
	defer imageTracker.DeleteAll()

	// 遍历所有 --workload 参数，执行推送/部署流程
	for _, workload := range optWorkloads {
		log.Printf("------------ 部署 [%s] ------------", workload.String())

		// 加载集群预置文件
		var preset Preset
		if err = LoadPresetFromHome(workload.Cluster, &preset); err != nil {
			if os.IsNotExist(err) {
				log.Printf("无法找到集群预置文件 %s, 请确认 --workload 参数是否正确", workload.Cluster)
			}
			return
		}

		// 生成 .docker/config.json 和 kubeconfig 文件
		var dcDir, kcFile string
		if dcDir, kcFile, err = preset.GenerateFiles(); err != nil {
			return
		}

		// 打印 kubernetes 集群版本
		_ = cmds.KubectlVersion(kcFile)

		// 使用指定的远程镜像仓库地址
		remoteImageNames := imageNames.Derive(preset.Registry)

		// 推送镜像到远程仓库
		for _, remoteImageName := range remoteImageNames {
			log.Printf("推送镜像: %s", remoteImageName)
			if err = cmds.DockerTag(imageNames.Primary(), remoteImageName); err != nil {
				return
			}
			imageTracker.Add(remoteImageName)
			if err = cmds.DockerPush(remoteImageName, dcDir); err != nil {
				return
			}
		}

		if optSkipDeploy {
			continue
		}

		// 构建工作负载补丁
		patch := CreateUniversalPatch(&preset, &profile, &workload, remoteImageNames.Primary())

		// 执行 kubectl patch 命令，更新工作负载
		var buf []byte
		if buf, err = json.Marshal(patch); err != nil {
			return
		}
		if err = cmds.KubectlPatch(kcFile, workload.Namespace, workload.Name, workload.Type, string(buf)); err != nil {
			return
		}
	}
}
