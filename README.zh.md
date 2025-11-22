# **ComposePack**

[**English**](README.md) | **简体中文**

> 🧩 **重生之我用Helm配置Docker Compose**  
> 把 Helm 式的配置和打包体验带到 Docker Compose 上。

<p align="center">
  <img src="docs/images/banner.svg" width="1000" alt="ComposePack banner" />
</p>

<p align="center">
  <a href="https://github.com/composepack/composepack/actions/workflows/ci.yml">
    <img src="https://img.shields.io/github/actions/workflow/status/composepack/composepack/ci.yml?label=CI" alt="CI Status">
  </a>
  <a href="https://github.com/composepack/composepack/releases">
    <img src="https://img.shields.io/github/v/release/composepack/composepack?display_name=tag&sort=semver" alt="Latest Release">
  </a>
  <a href="https://github.com/composepack/composepack/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/composepack/composepack" alt="License">
  </a>
  <a href="https://github.com/composepack/composepack/stargazers">
    <img src="https://img.shields.io/github/stars/composepack/composepack?style=social" alt="GitHub stars">
  </a>
</p>

Docker Compose好用，但它**缺了一些关键能力**：

> 没有模板、没有结构化配置、也没有干净的覆写机制。

结果就是：

- 为了不同的环境，不得不在不同的 YAML 和 .env 文件之间来回复制粘贴  
- 手动修改部署文件，切换 profile  
- 为了统一配置来源，额外挂脚本处理环境变量  
- 客户改了改 `.env`：诶我的某些功能怎么挂了？
  
**ComposePack 解决的就是这些问题。** ✨  

它为 Docker Compose 引入了 **模板引擎**、**可覆写的配置系统** 和 **Helm Chart式打包工作流**，  
同时保持与原生 Compose CLI **100% 兼容**。

<p align="center">
  <b>⚓ Helm 风格工作流 • 🎛️ 动态模板 • 📦 可安装的 Chart</b><br>
  <b>→ 用在 Docker Compose 上 ←</b>
</p>

使用 ComposePack，你可以：

- 📝 用 **Go Template** 写 Compose 模板  
- ⚙️ 用 `values.yaml` 管理系统默认值 + 用户覆盖值  
- 📦 像 Helm 一样打包成 **可安装 Chart**  
- 🔐 为每次发布生成独立、可复现的 **运行目录**  
- 🧩 在运行时自动合并出一个 `docker-compose.yaml`  
- 🚀 继续用熟悉的命令：`install`、`up`、`down`、`logs`、`ps` …

底层仍然是大家熟悉的 **`docker compose`**。

```bash
composepack install ./charts/myapp --name prod -f values-prod.yaml --auto-start
```

---

## ⚖️ ComposePack vs. Docker Compose

| 能力                                 | Docker Compose | **ComposePack** |
| ------------------------------------ | :------------: | :-------------: |
| Compose 模板化                       |       ❌        |      **✅**      |
| 结构化配置（系统值 vs 用户值）       | ❌（扁平 .env） |      **✅**      |
| 可安装包（Chart 打包与分发）         |       ❌        |      **✅**      |
| 每个发布拥有独立、可复现的运行环境   |       ❌        |      **✅**      |
| 完全兼容原生 `docker compose` 运行时 |       ✅        |      **✅**      |

---

## 📚 目录

- [⚡ 60 秒快速上手](#-60-秒快速上手)
- [📦 安装](#-安装)
- [🧠 整体工作原理](#-整体工作原理)
- [🚀 使用方式](#-使用方式)
  - [🛠️ Chart 制作者（Shippers）](#️-chart-制作者shippers)
  - [🧑‍💻 Chart 使用者（Consumers）](#-chart-使用者consumers)
- [🧩 模板基础](#-模板基础)
- [📂 Chart 结构与文件类型](#-chart-结构与文件类型)
- [🏗️ 运行目录结构](#️-运行目录结构)
- [📏 运行规则与注意事项](#-运行规则与注意事项)
- [📝 常见问题](#-常见问题)
- [🤝 参与贡献](#-参与贡献)

---

## ⚡ 60 秒快速上手

```bash
# 1. 初始化一个 Chart
composepack init charts/demo --name demo --version 0.1.0

# 2. 带自定义 values 安装成一个 release
composepack install charts/demo --name myapp -f values-prod.yaml --auto-start

# 3. 查看日志
composepack logs myapp --follow
```

就这么简单：
在 Docker Compose 之上，拥有了模板化配置 + 可复现的运行环境。

---

## 📦 安装

> ComposePack 是一个单独的二进制CLI客户端，仅依赖系统已安装的 Docker / Docker Compose。

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/composepack/composepack/main/scripts/install.sh | bash
```

- 默认安装到 `/usr/local/bin/composepack`，无权限时会落到 `~/.local/bin/composepack`
- 可通过 `COMPOSEPACK_INSTALL_DIR` 自定义安装目录

卸载：

```bash
./scripts/uninstall.sh
```

### Windows（PowerShell）

```powershell
./scripts/install.ps1 -Version v1.0.0 -InstallDir "$env:ProgramFiles\ComposePack"
```

卸载：

```powershell
./scripts/uninstall.ps1
```

### 从源码构建

```bash
git clone https://github.com/composepack/composepack.git
cd composepack
make build
```

如需使用 `go generate`（Wire 等），可执行：

```bash
make generate
```

---

## 🧠 整体工作原理

<p align="center">
  <img src="docs/images/flow.svg" width="1000" alt="ComposePack flow" />
</p>

- 定义一个 **Chart**（Compose 模板 + 运行期文件）
- 用户传入配置（`values.yaml`、`-f`、`--set`、环境变量）
- ComposePack 将它们渲染为一个 **独立的 release 目录**
- 随后在这个目录里运行Docker Compose

换句话说：
ComposePack 负责「整合生成一个干净的运行目录」，
而实际起容器这件事，仍然交给 `docker compose`。

---

## 🚀 使用方式

ComposePack 主要有两类使用者：

- **Chart 制作者（Shippers）**：编写、打包、分发 Chart
- **Chart 使用者（Consumers）**：安装并运行这些 Chart

下面分别介绍两种角色的工作流。

---

### 🛠️ Chart 制作者（Shippers）

> 适合打包自己应用、对外发版的团队。

#### 1️⃣ 创建一个新的 Chart（脚手架）

```bash
composepack init charts/example --name example --version 0.1.0
```

生成的目录类似：

```text
charts/example/
  Chart.yaml
  values.yaml
  templates/
    compose/00-app.tpl.yaml
    files/config/message.txt.tpl
    helpers/_helpers.tpl
  files/
    config/
```

#### 2️⃣ 本地渲染 / 仅预览

```bash
composepack template dev --chart charts/example
```

只渲染模板，不创建或修改 release 目录。

#### 3️⃣ 安装 Chart 进行本地调试

```bash
composepack install charts/example --name dev --auto-start
```

会生成 `.cpack-releases/dev/` 并在其中执行 `docker compose up`。

#### 4️⃣ 打包用于分发

```bash
composepack package charts/example --destination dist/
```

生成：

```text
dist/example-0.1.0.cpack.tgz
```

也可以自定义输出文件名：

```bash
composepack package charts/example --output dist/example.cpack.tgz
```

你可以把 `.cpack.tgz`：

- 放到 HTTP(S) 服务器
- 当作构建产物流转
- 收进内部软件仓库

---

### 🧑‍💻 Chart 使用者（Consumers）

> 适合内部开发者或客户侧运维人员。

#### 1️⃣ 从包或目录安装 Chart

```bash
composepack install example.cpack.tgz --name myapp -f custom-values.yaml --auto-start
```

`install` 支持：

- 本地 `.cpack.tgz` 文件
- 本地 Chart 目录
- 指向 `.cpack.tgz` 的 HTTP / HTTPS URL

#### 2️⃣ 管理你的部署

```bash
composepack up myapp
composepack down myapp --volumes
composepack logs myapp --follow
composepack ps myapp
composepack template myapp
```

该 release 的所有运行文件位于：

```text
.cpack-releases/myapp/
  docker-compose.yaml
  files/
  release.json
```

需要时，你也可以手动使用原生命令：

```bash
cd .cpack-releases/myapp
docker compose up
```

如果希望在其他工作目录下直接操作，也可以使用 `--runtime-dir` 指向某个 release 目录：

```bash
composepack up myapp --runtime-dir /opt/releases/myapp
composepack logs myapp --runtime-dir /opt/releases/myapp --follow
```

---

## 🧩 模板基础

ComposePack 使用 **Go Template**（与很多 Helm Chart 类似），
如果你团队已经在用 Helm，几乎没有学习成本。

例子：

```yaml
# templates/compose/00-app.tpl.yaml
services:
  app:
    image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
    environment:
      DB_HOST: "{{ .Values.db.host }}"
      DB_PASSWORD: "{{ env "DB_PASSWORD" }}"
```

可用的主要对象：

- `.Values` —— 系统默认值与用户覆盖值合并后的结果
- `.Env` —— 环境变量
- `.Release` —— release 名称、版本等信息
- 常用模板函数（如 `default`、`include`、`quote`、`toJson` 等）

---

## 📂 Chart 结构与文件类型

这一部分会说明：**什么文件放在哪里，以及 ComposePack 会如何处理它们。**

### 典型 Chart 结构

```text
myapp/
  Chart.yaml
  values.yaml
  templates/
    compose/
      00-app.tpl.yaml
      10-worker.tpl.yaml
    files/
      config/app.env.tpl
    helpers/
      _helpers.tpl
  files/
    config/
    scripts/
```

### 关键文件与目录

#### `Chart.yaml`

- **必需**
- Chart 元数据，用于标识 Chart 与生成 `release.json`：

  - `name`：名称（必需）
  - `version`：版本（必需）
  - `description`：描述
  - `maintainers`：维护者列表

#### `values.yaml`

- **必需**
- Chart 的 **系统默认配置**。
- 用户可以通过自定义 `values-*.yaml` 或 `--set` 做覆盖。
- 推荐理解为：「产品默认值」vs「用户按环境定制的值」。

---

#### `templates/compose/*.tpl.yaml`

- **必需目录**
- 每个文件都是一个 **Docker Compose 片段模板**。
- 文件名必须以 **`.tpl.yaml`** 结尾。
- ComposePack 会：

  1. 基于 `.Values` / `.Env` 渲染这些片段
  2. 把所有渲染结果合并成单个 `docker-compose.yaml`

示例：

```text
templates/compose/
  00-core.tpl.yaml
  10-db.tpl.yaml
  20-api.tpl.yaml
```

---

#### `templates/files/*.tpl`

- 可选。
- 用于生成运行期需要的各种文件：

  - 配置文件
  - shell 脚本
  - 其他需要挂载到容器里的资产
- 文件名必须以 **`.tpl`** 结尾。
- ComposePack 会：

  - 渲染它们
  - 去掉 `.tpl` 后缀
  - 写入到 release 目录的 `files/` 下

示例：

```text
templates/files/
  config/app.env.tpl       -> files/config/app.env
  scripts/init.sh.tpl      -> files/scripts/init.sh
```

---

#### `templates/helpers/*.tpl`

- 可选。
- 存放可复用的模板片段和 helper 函数。
- 可以在其他模板中通过 `{{ include "helper.name" . }}` 引用。

示例：

```text
templates/helpers/_helpers.tpl
```

```yaml
{{- define "myapp.fullname" -}}
{{ printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end -}}
```

---

#### `files/`

- 可选。
- 不需要模板渲染的 **静态资源**。
- 这些文件会原样复制到 release 目录的 `files/` 中，适合：

  - 静态配置
  - 证书
  - 种子数据
  - 不需要按环境变化的脚本

示例：

```text
files/
  config/defaults.json
  scripts/migrate.sh
```

生成的运行目录中：

```text
.cpack-releases/<name>/
  files/config/defaults.json
  files/scripts/migrate.sh
```

---

## 🏗️ 运行目录结构

每个 release 都有自己独立的一套运行文件：

```text
.cpack-releases/<release>/
  docker-compose.yaml   # 合并后的 Compose 文件
  files/                # 模板渲染 + 静态文件
    config/...
    scripts/...
  release.json          # Chart / values / 环境等元数据
```

---

## 📏 运行规则与注意事项

这些固定设计有助于让 Chart 行为更一致、更易排查问题。

### 1️⃣ 所有挂载文件都来自 `./files/`

在 release 目录中，所有非 Compose 资产都放在 `files/` 里。
因此，**Compose 模板中的本地挂载路径需要写成 `./files/...`**。

例子：

```yaml
# templates/compose/*.tpl.yaml
services:
  app:
    volumes:
      - ./files/config/app.env:/app/app.env:ro
      - ./files/scripts/init.sh:/docker-entrypoint.d/init.sh:ro
```

如果你引用了 `./files/` 之外的路径，容器会找不到对应文件。

---

### 2️⃣ 模板文件的后缀规则

ComposePack 通过后缀来决定如何处理文件：

- Compose 模板文件：**必须** 以 `.tpl.yaml` 结尾

  - 如：`10-api.tpl.yaml`
- 其他需要渲染的文件：**必须** 以 `.tpl` 结尾

  - 如：`app.env.tpl`、`init.sh.tpl`
- 不需要渲染的静态文件：直接放到 `files/`，**不要** 带 `.tpl`

后缀不正确时，文件可能：

- 被当作普通文件直接复制（未渲染）
- 或根本不会被视为 Compose 片段

---

### 3️⃣ 运行命令绑定到 release 目录

ComposePack 总是从对应的 release 目录中执行 Docker Compose：

```text
.cpack-releases/<release>/
  docker-compose.yaml
  files/
```

当你运行：

```bash
composepack up myapp
```

等价于：

```bash
cd .cpack-releases/myapp
docker compose -f docker-compose.yaml up
```

必须在有 `.cpack-releases` 的父目录下执行 ComposePack 或者指明 `--runtime-dir`，否则 ComposePack 无法找到正确的文件。
如果你想手动用 `docker compose` 排查问题，请先 `cd` 到对应的 release 目录。

---

## 📝 常见问题

### ComposePack 会取代 **Docker Compose** 吗？

不会。ComposePack 是对 Docker Compose 的 **封装和增强**，不是替代。

- ComposePack 负责：模板渲染、配置管理、Chart 打包、release 目录管理
- Docker Compose 负责：真正起容器、跑服务

你随时可以进入 `.cpack-releases/<name>/` 手动运行 `docker compose`。

---

### 既然有 docker-compose + `.env`，为什么还需要它？

`.env` 在小项目里很好用，但存在一些限制：

- 配置是扁平的，不利于表达结构化信息
- 无法满足同配置不同格式需求，必须额外挂脚本处理环境变量
- 系统默认值 vs 用户覆盖值 很难干净地区分
- 升级产品版本时，`.env` 很难平滑对齐新版本
- 无法优雅地打包成可重复安装的「产品配置」

ComposePack 提供：

- `values.yaml` 作为系统默认配置入口
- 用户 `values-*.yaml` / `--set` 叠加在上面
- 清晰区分「你交付的东西」和「用户自己改的东西」
- 每个环境一个独立的 release 目录，便于备份、迁移、回滚

---

### 为什么不用 Helm？

你说的对, 但是Helm只能在 **Kubernetes** 上使用。

适合用 Helm 的场景：

- 你已经在部署 K8s 集群
- 团队有完整的 Kubernetes 运维体系

适合用 **ComposePack** 的场景：

- 想要 Helm 一样的模板和 Chart 体验
- 想继续使用 **纯 Docker Compose**
- 不想引入完整的 Kubernetes 复杂度
- 你被公司要求同时维护Docker Compose和Helm的解决方案，发现原生Compose体验太差，希望可以接近Helm的体验 （并非本人亲身经历）

可以简单地理解为：
**ComposePack = 为 Compose 带来 Helm 级别的发布体验。**

---

### 还能直接用 docker-compose 吗？

可以。

ComposePack 只是在 `.cpack-releases/<release>/` 下生成：

```text
docker-compose.yaml
files/
release.json
```

你可以随时进入该目录：

```bash
cd .cpack-releases/<release>
docker compose up
docker compose ps
docker compose logs
```

ComposePack 做的是「标准化这个目录怎么生成」，而不是限制你怎么用 Compose。

---

### 目前适合用在生产环境吗？

当前项目还处于**早期阶段**，API 可能会有迭代。

适合的场景包括：

- 对新工具有兴趣的早期 adopters
- 内部环境 / 工具链项目
- 能够阅读 Go 代码、愿意一起打磨工具的团队

如果你已经在生产环境中使用：

- 欢迎提 Issue 反馈问题
- 欢迎提交 PR 修 bug / 补文档 🙏

---

## 🤝 参与贡献

欢迎提交 Issue 与 PR。

### 开发流程

```bash
make fmt
make test
make build
make generate
```

会触发构建流程，打包多平台二进制并上传到 GitHub Releases。

> 如果这个项目对你有帮助，欢迎点个 ⭐ Star，
> 也欢迎在团队里分享，让更多还在维护巨大 docker-compose.yaml 的同学看到~。
