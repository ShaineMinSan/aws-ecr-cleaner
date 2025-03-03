# AWS ECR Cleaner
- AWS ECR Cleaner 是一个用于管理和清理 AWS Elastic Container Registry (ECR) 中未使用或过期镜像的工具。

```
aws-ecr-cleaner/
├── IMG_LIST
│   ├── MGMT_IMG_LIST.txt     # 管理环境的镜像列表文件
│   ├── PRD_IMG_LIST.txt        # 生产环境的镜像列表文件
│   └── PRE_IMG_LIST.txt        # 预发布/测试环境的镜像列表文件
├── README.md                 # 项目说明文档
├── cmd
│   └── main.go               # 程序入口
├── go.mod                    # Go 模块管理文件
├── go.sum                    # Go 模块依赖校验文件
├── .env                      # 环境变量配置文件
├── internal
│   ├── cleaner
│   │   └── cleaner.go      # 清理流程逻辑：扫描、过滤、删除
│   ├── config
│   │   └── config.go       # 环境变量及配置加载
│   ├── ecr
│   │   └── ecr.go          # AWS ECR 操作封装，包括仓库、镜像扫描与删除
│   ├── k8s
│   │   └── k8s.go          # 从 Kubernetes 集群中拉取正在使用的镜像列表
│   ├── logger
│   │   └── logger.go       # 日志初始化，根据配置决定是否保留终端输出
│   └── util
│       └── regex.go        # 正则匹配工具函数
└── logs                      # 程序运行日志文件目录
    ├── ecr_cleaner_app_YYYYMMDD_HHMMSS.log
    └── ...                 # 其它日志文件
```

###### 该项目能够：
- 扫描 ECR 仓库，基于正则表达式匹配需要处理的仓库。
- 根据配置规则过滤出候选删除镜像（支持未打标签镜像和打标签镜像）。
- 集成 Kubernetes，从集群中提取正在使用的镜像列表，以防止误删。
- 支持干运行（dry-run）、仅列出候选镜像（list-only）以及自动确认等模式。
- 删除空仓库（使用 Force 参数）以保持 ECR 整洁。

###### 特性
##### 仓库与镜像扫描
- 根据配置中的正则表达式扫描指定 AWS ECR 仓库，并获取仓库内的所有镜像详情。

##### 候选镜像过滤
- 根据 HOLD_TAG_REGEX 保留特定镜像，对未打标签的镜像也加入删除候选列表，同时保护最新的正在使用镜像。

##### Kubernetes 集成
- 自动拉取 Kubernetes 集群中各类工作负载（Pods、Deployments、StatefulSets、Jobs、DaemonSets、CronJobs 等）的镜像，并生成 in-use 列表，避免删除正在使用的镜像。

##### 仓库清理
- 删除候选镜像后，如果仓库内已无镜像，则自动删除空仓库（使用 Force 参数）。

##### 灵活配置
- 通过 .env 文件配置日志、调试、干运行、自动确认、目标仓库匹配规则、保护镜像数量等参数。

#### 环境要求
- Go 1.16 及以上版本
- AWS 账户及 ECR 权限（确保具有 Describe、DeleteRepository、BatchDeleteImage 等权限）
- 有效的 Kubernetes 集群（若使用 k8s 镜像提取功能）
- Docker（如果需要构建 Docker 镜像）

`
# 安装与运行
## 1. 克隆项目
`
git clone <your-repo-url>
cd aws-ecr-cleaner
`

## 2. 配置环境变量
#### 在项目根目录下创建 .env 文件，并参考下面示例内容进行配置：

`
##### 日志目录：存放日志文件的目录
- LOGDIR=./logs

##### 调试模式：启用调试日志
- DEBUG=true

##### 干运行模式：模拟删除操作（不会实际删除）
- DRYRUN=false

##### 仅列出候选镜像，不执行删除操作
- LIST_ONLY=false

##### 保护最新 in-use 镜像数量
- PROTECT_LATEST=3

##### 是否保护 Kubernetes 中正在使用的镜像
- PROTECT_INUSE_BY_K8S=true

##### 目标仓库正则表达式（匹配需要处理的 ECR 仓库）
- TARGET_REPO_REGEX=^my-repo.*

##### 排除仓库正则表达式（匹配的仓库不会被处理）
- EXCLUDE_REPO_REGEX=^my-repo-exclude.*

##### 保留标签正则表达式（匹配到的镜像标签不会删除）
- HOLD_TAG_REGEX=^stable$

##### AWS 区域（例如 us-east-1）
- AWS_REGION=us-east-1

##### 环境标识（对应不同的 IMG_LIST 文件，取值：prd, pre, mgmt）
- ENV=prd

##### 自动确认删除操作（true 则自动删除，不需要交互确认）
- AUTO_CONFIRM=false

##### 交互模式（true 则保留终端输出用于交互确认）
- INTERACTIVE_MODE=true

`

## 3. 构建与运行

直接运行：
在项目根目录下执行：

`
go run cmd/main.go
`

根据配置（AUTO_CONFIRM 和 INTERACTIVE_MODE），程序会在终端显示候选镜像列表和交互提示。

编译后运行：
`
go build -o aws-ecr-cleaner cmd/main.go 
./aws-ecr-cleaner
`

## 项目目录结构说明

aws-ecr-cleaner/
项目根目录，包含整个 AWS ECR 清理器的所有代码、配置文件和日志等。

IMG_LIST
存放 Kubernetes 拉取下来的镜像列表文件。
MGMT_IMG_LIST.txt：管理环境的镜像列表文件。
PRD_IMG_LIST.txt：生产环境的镜像列表文件。
PRE_IMG_LIST.txt：预发布/测试环境的镜像列表文件。
README.md

项目的说明文件，通常包含项目概述、使用方法、配置说明、部署指南等。
cmd/

存放程序入口文件。
main.go：项目的主入口，负责加载配置、初始化日志，然后启动清理流程。
go.mod 和 go.sum

Go 模块管理文件，用于记录项目依赖以及版本信息。
internal/

存放项目内部实现的各个模块，主要用于业务逻辑封装，外部不直接引用。

internal/cleaner/

cleaner.go：负责整个清理流程的协调工作，包括扫描 ECR 仓库、过滤待删除镜像、执行删除操作以及在仓库为空时删除仓库。
internal/config/

config.go：读取环境变量和 .env 文件中的配置信息，生成统一的配置结构体供项目其他模块使用。
internal/ecr/

ecr.go：封装与 AWS ECR 相关的操作，如获取仓库列表、获取镜像详情、过滤候选镜像（包括未打标签的镜像）、删除镜像以及调用 STS 获取账户 ID。
此模块还提供了辅助函数（如 MultiRegexMatch）用于仓库名称和镜像标签的正则匹配。
internal/k8s/

k8s.go：封装与 Kubernetes 集群交互的逻辑，负责拉取各类工作负载（Pods、Deployments、StatefulSets、Jobs、DaemonSets、CronJobs 等）的镜像，并将结果写入对应的 IMG_LIST 文件，同时支持从文件加载 in-use 镜像映射。
internal/logger/

logger.go：负责日志系统的初始化，根据配置决定是否将标准输出重定向到日志文件，从而实现交互模式下保留终端输出。
通过该模块可以把运行日志写入 logs 目录下的文件。
internal/util/

regex.go：封装常用的正则匹配工具函数，如 MultiRegexMatch（支持 "OR" 和 "&&" 逻辑）、HoldTagMatch（用于判断镜像标签是否需要保留）以及 TrimRegistry（去除仓库 URI 中的注册中心前缀）。
logs/

存放程序运行期间生成的日志文件。
每个日志文件的文件名通常包含时间戳（如 ecr_cleaner_app_20250227_124304.log），便于追踪运行记录和调试问题。




```
ENV=pre
DEBUG=true
DRYRUN=false
LIST_ONLY=false
PROTECT_LATEST=3
PROTECT_INUSE_BY_K8S=true
EXCLUDE_REPO_REGEX=saas/logstash
TARGET_REPO_REGEX=^(f?saas)
HOLD_TAG_REGEX=.*2\.(84|85|86|87|88|89|90).*$
AWS_REGION=ap-northeast-1
LOGDIR=./logs
AUTO_CONFIRM=true
INTERACTIVE_MODE=false
```

###### 多容器获取镜像地址
- reference the IMG_LIST/list_img.sh

#### 除掉非业务项目开头的镜像地址
```
EXCLUDE_REPO_REGEX=^(f?saas)OR^(devops)
TARGET_REPO_REGEX=.*
HOLD_TAG_REGEX=releaseOR2\.(84|85|86|87|88|89|90)
```

```
EXCLUDE_REPO_REGEX=searcherORboom
TARGET_REPO_REGEX=^(middleware)OR^(devops)
HOLD_TAG_REGEX=v\.*OR(2025|2024)
```

#### 业务项目开头的镜像地址不包含镜像里面带有release or 版本号(2.84 ~ 2.90)
```
EXCLUDE_REPO_REGEX=
TARGET_REPO_REGEX=^(f?saas)
HOLD_TAG_REGEX=releaseOR2\.(84|85|86|87|88|89|90) OR HOLD_TAG_REGEX=.*2\.(84|85|86|87|88|89|90).*$
```
