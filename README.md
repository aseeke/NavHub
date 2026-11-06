# NavHub

轻量级、跨平台、自托管的个人导航中心。

将不同服务器、域名和云平台上的常用系统整理到一个首页。NavHub 使用 **Vue 3 + Go + SQLite**，前端资源嵌入程序，运行时无需额外安装数据库、Node.js 或 Go。

[下载安装包](https://github.com/aseeke/NavHub/releases/latest) · [源码仓库](https://github.com/aseeke/NavHub) · [更新记录](CHANGELOG.md) · [MIT 许可证](LICENSE)

![NavHub 首页](docs/screenshots/home-desktop.png)

> 截图使用示例数据。首次启动为空导航，默认私有访问，没有预设管理员或密码。

## 功能

- **清晰的首页**：浅色背景、五列导航卡片、分类、日期和时钟，手机自动调整布局。
- **简单的管理**：通过弹窗和抽屉新增、编辑项目，管理分类及页面设置。
- **自动文字图标**：`GitHub → GI`、`Oracle Cloud → OC`、`数据管理 → 数据`，无需第三方图标服务。
- **拖拽排序**：支持分类排序、分类内排序和跨分类移动，保存后刷新不丢失。
- **本地搜索**：搜索名称、网址和描述；`/` 打开搜索，`Esc` 关闭。
- **访问控制**：私有模式须登录；公开模式允许浏览，管理员才能修改。
- **外观设置**：自定义背景、卡片颜色、图标、圆角、间距及打开方式。
- **数据备份**：JSON 导入导出，以及包含数据库与上传图片的完整 ZIP 备份。
- **单文件部署**：Windows x64、Linux x64、Linux ARM64；Windows 提供后台运行脚本。

保持个人导航的定位，不包含服务器监控、Docker 管理、多用户权限或插件市场。

## 下载

在 [GitHub Releases](https://github.com/aseeke/NavHub/releases/latest) 下载对应平台的安装包。

| 安装包 | 适用平台 | 包内程序 |
|---|---|---|
| `navhub-1.0.1-windows-amd64.zip` | Windows 10/11 x64 | `navhub.exe` |
| `navhub-1.0.1-linux-amd64.zip` | Linux x64 | `navhub` |
| `navhub-1.0.1-linux-arm64.zip` | Linux ARM64 | `navhub` |

`SHA256SUMS.txt` 用于核对下载文件。安装包包含程序、示例配置、文档与许可证，不包含用户配置或数据库。

## Windows 启动

解压到可写目录，例如 `D:\NavHub`。解压后的 Windows 安装目录包含：

```text
NavHub/
  navhub.exe
  start-background.vbs
  stop-background.vbs
  navhub-background.ps1
  config.example.yaml
  README.md
  LICENSE
  docs/
  deploy/
```

### 前台运行

双击 `navhub.exe`。程序显示版本、服务地址、数据目录和运行日志，并在交互桌面自动打开浏览器。

首次启动自动创建 `config.yaml` 和 `data/`。默认访问地址为 [http://127.0.0.1:3002](http://127.0.0.1:3002)，按页面提示创建管理员即可。密码需为 10–72 字节，中文字符通常占用多个字节。

控制台中的“数据库初始化完成”“运行中”“管理操作完成”是正常日志。退出时在控制台按 `Ctrl+C`。

### 后台运行，不显示控制台

1. 如果当前已打开 `navhub.exe`，先在原控制台按 `Ctrl+C` 退出。
2. 双击 **`start-background.vbs`**。
3. 手动打开 [http://127.0.0.1:3002](http://127.0.0.1:3002)。使用自定义端口时以配置为准。
4. 停止时双击 **`stop-background.vbs`**。

三个脚本必须与 `navhub.exe` 放在同一目录。后台启动不打开浏览器，仍使用该目录的 `config.yaml` 和既有数据，不会重新初始化管理员。重复点击启动脚本不会创建第二个同目录进程；停止脚本只匹配本目录的程序，不影响其他安装目录，也不删除数据。

后台脚本不是 Windows 服务，不设置开机启动。注销或重启 Windows 后需要重新运行；需要持续托管时参考下方的 Windows 服务方式。

如果系统禁用了 Windows Script Host，可以在安装目录执行：

```powershell
# 启动器退出后，NavHub 继续在后台运行
powershell -NoProfile -ExecutionPolicy Bypass -File .\navhub-background.ps1 -Action Start

# 停止
powershell -NoProfile -ExecutionPolicy Bypass -File .\navhub-background.ps1 -Action Stop
```

### Windows 服务（可选）

程序支持 Windows Service Control Manager。先在固定目录放好程序、配置和数据目录，再使用管理员终端注册：

```powershell
sc.exe create NavHub binPath= '"D:\NavHub\navhub.exe" --config "D:\NavHub\config.yaml" --no-browser' start= auto
sc.exe start NavHub
sc.exe stop NavHub

# 卸载服务，不删除数据
sc.exe delete NavHub
```

服务名固定为 `NavHub`，不会自动打开浏览器。服务身份需要读取配置、写入数据目录的权限；不要与双击启动的实例同时占用同一端口。SCM 的实际注册及关机通知仍需目标机器验收。

## Linux 启动

下载与 CPU 架构对应的安装包，解压后进入目录：

```bash
chmod +x navhub
./navhub

# 可选参数
./navhub --config ./config.yaml
./navhub --port 3002
./navhub --version
./navhub --help
```

Linux 不自动打开浏览器。程序使用纯 Go SQLite 驱动，发布时设置 `CGO_ENABLED=0`，无需安装系统 SQLite 动态库。

### systemd（可选）

以 `/opt/navhub` 为例，先把程序与 `config.yaml` 放好，再创建专用运行用户和数据目录：

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin navhub
sudo mkdir -p /opt/navhub/data
sudo chown navhub:navhub /opt/navhub/data
sudo chmod +x /opt/navhub/navhub
sudo cp deploy/navhub.service /etc/systemd/system/navhub.service
sudo systemctl daemon-reload
sudo systemctl enable --now navhub
sudo systemctl status navhub
journalctl -u navhub -f
```

[服务文件](deploy/navhub.service) 默认只允许写入 `/opt/navhub/data`。请预先创建配置文件；更换数据目录时同步修改 `ReadWritePaths`。systemd 安装与 Linux 真机运行仍需目标环境验收。

## 配置

默认读取当前工作目录的 `config.yaml`，不存在时自动创建；后台脚本固定使用程序同目录的配置。`data_dir` 的相对路径以配置文件所在目录为基准。

```yaml
host: 0.0.0.0
port: 3002
data_dir: data
open_browser: true
secure_cookie: false
log_file: true
```

| 配置项 | 说明 |
|---|---|
| `host` | `0.0.0.0` 允许其他设备通过本机 IP 访问；仅本机使用改为 `127.0.0.1` |
| `port` | HTTP 端口，命令行 `--port` 优先；修改后需重启 |
| `data_dir` | 数据库、图片、备份与日志目录 |
| `open_browser` | 交互桌面是否自动打开浏览器；`--no-browser` 可覆盖 |
| `secure_cookie` | HTTPS 反向代理部署设为 `true`；HTTP 直连保持 `false` |
| `log_file` | 是否写入滚动文件日志 |

HTTPS 反向代理应保留原始 Host，服务端据此检查写入请求来源。公网访问建议配置 HTTPS。端口占用时先退出旧实例，或通过 `--port` 更换端口，不需要删除数据库。

## 使用导航

1. 首次创建管理员，再添加分类。
2. 右下角操作栏 → **添加项目**，填写名称、HTTP / HTTPS 网址及分类。
3. 默认生成文字图标；展开“外观与打开方式”可设置上传图标、卡片颜色和跳转方式。
4. 进入 **管理模式**，点击卡片编辑，拖动卡片排序或移动分类，完成后退出管理模式。
5. 在 **分类管理** 中重命名、排序、隐藏分类。非空分类须先迁移或单独删除项目才能删除。
6. 在 **系统设置** 中调整外观、访问权限和备份。保存按钮始终显示在抽屉顶部。

搜索在浏览器内完成，不会随每次输入请求后端。搜索时停用拖拽，清除搜索后再排序。多个窗口同时编辑时，旧版本保存会被拒绝，重新加载后再操作可避免覆盖他人的更新。

## 数据、日志与备份

```text
data/
  navhub.db       SQLite 数据库
  uploads/       上传图标和背景
  backups/       手动生成的完整 ZIP 备份
  navhub.log     文件日志
```

运行时可能出现 `navhub.db-wal` 和 `navhub.db-shm`，属于正常文件。日志默认每份约 5MB，保留 3 份历史文件；记录启动、退出及操作异常，不记录密码、Cookie 或明文会话令牌。关闭 `log_file` 后不会生成文件日志。

### JSON 导入导出

系统设置 → **导出 JSON / 导入 JSON**。导出包含分类、项目、页面和卡片设置，不包含管理员或会话。导入需要确认，会替换当前导航和设置，失败不会修改现有数据。

**JSON 仅保存上传图片的路径，不携带图片文件。** 完整迁移请选择 ZIP 备份，或额外复制原来的 `uploads/`。

### 完整 ZIP 备份

系统设置 → **完整 ZIP 备份**。程序创建 SQLite 一致性快照，并打包数据库和上传图片，保存至 `data/backups/` 后提供下载。不包含配置、日志或旧备份；请定期清理不需要的备份文件。

完整备份包含管理员密码哈希和会话摘要，请按私密文件保管。恢复时：

1. 停止程序，并另存当前 `data/` 作为回退。
2. 将 ZIP 解压至新的空数据目录，使 `navhub.db` 和 `uploads/` 直接位于目录内。
3. 将 `config.yaml` 的 `data_dir` 指向恢复目录，再启动程序，使用原管理员账户登录。

也可停机后复制整个 `data/`。不要只复制运行中的数据库主文件，也不要将旧的 WAL / SHM 混入恢复目录。替换图片不会自动清理旧媒体，以免破坏已有引用。

## 从 v1.0.0 升级

1. 先完成备份，再停止旧程序或服务。
2. 替换 `navhub.exe` / `navhub`，以及 Windows 后台脚本。
3. **保留原来的 `config.yaml` 和整个 `data/`。** 如果将新版解压到新目录，启动前先复制这两项，或在新配置中指向原数据目录。
4. 启动新版，检查项目、图片和管理员登录。

v1.0.1 的数据库 schema 仍为 1，不需要重新创建管理员。程序使用 `PRAGMA user_version` 管理结构版本；迁移失败或数据库版本过高会停止启动，不会清空数据库。已有自定义外观设置也不会被重置。

## 开发与构建

开发环境需要 Go 1.26+、Node.js 22.12+ 和 npm。运行安装包不需要这些依赖。

```bash
# 终端一：后端，默认 3002 端口
go mod download
go run . --no-browser

# 终端二：前端，通常为 5173 端口
cd web
npm ci
npm run dev
```

Vite 将 `/api` 和 `/uploads` 代理至 `127.0.0.1:3002`。前端生产构建输出至 `static/`；修改前端后需重建前端和 Go 程序，旧的 exe 不会自动更新。

```powershell
# Windows：测试、构建前端、编译三平台、打包和生成校验值
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/build.ps1
```

```bash
# Linux：生成三平台程序及 tar.gz 包
sh scripts/build.sh
```

产物位于 `release/`，Windows 构建脚本生成 ZIP，Linux 脚本生成 tar.gz。Windows 安装包自动附带三个后台运行脚本，所有安装包均使用白名单打包，排除运行时数据。

```bash
go test -v ./...
go vet ./...
cd web
npm test
npx playwright install chromium firefox webkit
npm run test:e2e
```

Windows 后台脚本还可在项目根目录单独验证：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-background.ps1
```

自动化测试使用临时目录，不修改真实导航。详细结果见 [验收记录](docs/ACCEPTANCE.md)。Linux 两种架构已交叉编译；真实 Linux、Windows SCM、systemd、Safari 设备和 1GB VPS 压测的验证范围以验收记录为准。

## 技术与许可

后端使用 Go 标准 HTTP 服务、[modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)、bcrypt 和 YAML；前端使用 [Vue](https://vuejs.org/)、[Vite](https://vite.dev/) 和 SortableJS。运行时不加载外部 CDN 脚本或字体，不引入大型图标库，不进行后台轮询，时钟每分钟在浏览器更新一次。

首页布局参考 Sun-Panel 的使用方式，未复制其代码、品牌或素材。NavHub 使用 [MIT License](LICENSE)；使用、修改或分发时请保留版权和许可声明，第三方依赖遵循各自许可证。
