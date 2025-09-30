## Kantaloupe 本地调试指南

本指南旨在为 Kantaloupe 开源项目的本地调试提供指导。通过远程连接到集群并进行相应配置，您可以有效地调试 Kantaloupe 组件。

### 1. Apiserver 调试

调试 Apiserver 组件，您需要将开发集群的 kubeconfig 文件配置到本地，并确保 Prometheus 地址正确暴露。

1. **复制 Kubeconfig 文件**：将开发集群的 kubeconfig 文件复制到本地工作环境。请牢记该文件的存储路径。
2. **管理本地 Kubeconfig：**
   - 进入本地终端，导航至 `~/.kube/` 目录。
   - 将现有本地`config`文件重命名为其他名称（例如 `config.bak` 或 `config_local.txt`），以避免与新复制的文件发生命名冲突。
   - 将开发集群的 kubeconfig 文件移动至 `~/.kube/` 目录下，并确保其文件名为`config`。
3. **配置 Prometheus 地址**：确保 Kantaloupe 的 Apiserver 配置中，`prometheus-address` 参数指向您的主机上暴露的 Prometheus 地址。

### 2. Controller Manager 调试

调试 Controller Manager 组件，您需要将开发集群的 kubeconfig 文件配置到本地，并以调试模式启动。

1. **复制 Kubeconfig 文件**：将开发集群的 kubeconfig 文件复制到本地 `~/.kube/config` 路径。
2. **启用调试模式**：使用 `debug-mode=true` 标志（flag）启动本地的 Controller Manager 。这将启用详细日志和调试功能。
3. **禁用集群中的 Controller Manager**：在进行本地调试时，为了避免冲突，请暂时禁用开发集群中原有的 Controller Manager 实例。

## 示例 vscode debug 配置

### 配置调试启动文件

为了方便您在本地调试项目，请按照以下步骤配置 `launch.json` 文件：

1. 在您的集成开发环境（IDE）中，创建或打开项目的 `launch.json` 文件。
2. 在文件类型选择中，请选择 **Go** 语言环境。
3. 将以下内容复制到 `launch.json` 文件中：

```JSON
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "apiserver",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/apiserver/main.go",
            "args": [
                // "--help",
                // "--prometheus-addr=http://10.10.10.128:9090",
            ]
        },
        {
            "name": "controllerManager",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/controller-manager/main.go",
            "args": [
                "--leader-elect=false",
                // "--debug-mode=true"
            ]
        },
    ]
}
```