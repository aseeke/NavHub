# NavHub v1.0.0

轻量级、跨平台、自托管的个人导航中心。导航优先、管理简单、界面美观、部署轻量。

- Vue 3 首页：渐变山景、时钟日期、分类卡片、自动文字图标与本地搜索。
- 弹窗和抽屉管理项目、分类与设置，支持图标上传、拖拽排序及跨分类移动。
- 默认私有访问，首次创建管理员；公开模式下访客只读。
- SQLite 持久化、schema 版本管理、并发保存冲突检测。
- JSON 导入导出、包含数据库与上传图片的完整 ZIP 备份。
- Go embed 静态资源，单文件运行，提供滚动日志、Windows 服务支持及 systemd 示例。

## 下载与运行

| 附件 | 平台 | 启动 |
|---|---|---|
| `navhub-1.0.0-windows-amd64.zip` | Windows 10/11 x64 | 解压后双击 `navhub.exe` |
| `navhub-1.0.0-linux-amd64.zip` | Linux x64 | 解压后 `chmod +x navhub && ./navhub` |
| `navhub-1.0.0-linux-arm64.zip` | Linux ARM64 | 解压后 `chmod +x navhub && ./navhub` |

默认端口为 **3002**，首次启动创建配置与数据目录。首次访问创建管理员，无预置密码。

## 验证范围

Go 集成测试、前端单元测试、Chromium / Firefox / WebKit 端到端测试，以及 Windows 发布程序启动、端口占用与重启检查已通过。

Linux 两种架构已交叉编译，Linux 真机、Windows SCM、systemd 和真实 Safari 设备的运行验收尚待完成。

JSON 导出不携带上传图片；完整迁移请使用系统设置中的 ZIP 备份。详细部署、配置与恢复方法见 README。
