# NavHub v1.0.1

本次更新精简了前端界面，并加入 Windows 后台运行脚本。

## 更新内容

- 浅灰背景、白色卡片与统一线性图标，精简山景、毛玻璃和宣传文案。
- 添加 / 编辑窗口默认展示必填信息，外观选项按需展开。
- 系统设置保存按钮固定在顶部，桌面与手机布局同步优化。
- 新增 `start-background.vbs`、`stop-background.vbs` 和 `navhub-background.ps1`，支持无控制台启动、停止与重复启动检查。
- 重写 README，补充升级、后台运行和备份恢复步骤。

## 下载

| 安装包 | 平台 |
|---|---|
| `navhub-1.0.1-windows-amd64.zip` | Windows x64 |
| `navhub-1.0.1-linux-amd64.zip` | Linux x64 |
| `navhub-1.0.1-linux-arm64.zip` | Linux ARM64 |

Windows 解压后，前台运行双击 `navhub.exe`；后台运行双击 `start-background.vbs`，停止时双击 `stop-background.vbs`。脚本须与程序在同一目录，启动时不自动打开浏览器。默认访问地址为 http://127.0.0.1:3002。

## 从 v1.0.0 升级

先备份并停止旧程序，替换程序和脚本，保留原来的 `config.yaml` 与完整 `data/` 目录。若解压到新目录，启动前先把旧配置和数据复制过去，避免误建空导航。数据库 schema 仍为 1，无需重新创建管理员。

JSON 导出不携带媒体文件；完整迁移应使用系统内的 ZIP 备份或停机复制数据目录。安装包包含 MIT 许可证，不包含用户数据库和配置。

验证范围：Windows 本机、Go / 前端测试及 Chromium / Firefox / WebKit 回归；Linux 两种架构完成交叉编译。Linux 真机、Windows SCM、systemd 和真实 Safari 设备仍待目标环境验收。详细结果见 `docs/ACCEPTANCE.md`。
