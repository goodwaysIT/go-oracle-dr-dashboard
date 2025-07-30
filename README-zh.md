# Oracle Dashboard 数据库健康监控可视化平台

## 简短描述

Oracle Dashboard 是一款基于 Go 和 Gin 框架开发的多数据库健康监控与可视化 Web 仪表盘，支持多实例状态实时展示、API 查询、静态资源嵌入与高可定制化配置，助力企业高效掌控数据库运行状态。

## 详细描述

本平台为企业级 Oracle 数据库环境提供统一的健康监控与可视化管理能力。支持多套数据库配置，自动采集主库、备库、负载均衡节点等多维度健康信息，通过 Web 仪表盘实时展现数据库状态与告警信息。系统采用 Gin Web 框架，静态前端资源可嵌入编译，支持 RESTful API 数据接口，便于二次开发与集成。平台具备灵活的配置机制、丰富的日志管理和跨平台部署能力。

## 功能列表
- 支持多数据库实例统一监控
- 实时展示主库、备库、负载均衡节点状态
- 提供 RESTful API 数据接口（如 /api/data）
- 静态前端资源嵌入与高性能 Web 服务
- 支持自定义静态资源目录与 BasePath
- 日志记录与多级日志输出
- 配置热加载与参数自定义
- 跨平台编译与部署

## 技术栈与依赖
- 语言：Go 1.18 及以上
- Web 框架：[Gin](https://github.com/gin-gonic/gin)
- 配置解析：gopkg.in/yaml.v3
- 日志记录：标准库 log + 可选 logrotate
- 前端：静态 HTML/CSS/JS（可自定义）
- 运行环境：支持 Windows/Linux

## 使用方法
1. 准备配置文件 `config.yaml`，详见下方示例和字段说明。
2. 将可执行文件与配置文件放置于同一目录。
3. 启动服务：
   ```shell
   ./oracle-dashboard
   ```
   或在 Windows 下：
   ```shell
   oracle-dashboard.exe
   ```
4. 访问 Web 仪表盘：
   浏览器打开 `http://<服务器IP>:8090/`（端口号可在配置文件中修改）
5. 日志输出见 `db-dashboard.log`。

## 配置文件详细介绍（config.yaml）

```yaml
# 服务器配置
server:
  port: "8090"                   # Web 服务监听端口
  static_dir: "./static"         # 静态资源目录（可选）
  refresh_interval: 60            # 数据刷新频率（秒）
  public_base_path: "/"          # 前端 BasePath（部署子路径时修改）

# 日志配置
logging:
  level: "info"                  # 日志级别（info/debug/warn/error）
  filename: "db-dashboard.log"   # 日志文件名
  max_size_mb: 100                # 单日志文件最大容量（MB）
  max_backups: 5                  # 日志文件最大备份数
  max_age_days: 30                # 日志最大保存天数

# 数据库配置
# 可配置多套数据库，字段如下：
databases:
  - name: "小灯SCADA数据库"
    lb_ip: "172.16.4.232"
    prod_ip: "172.16.4.65"
    dr_ip: "172.16.28.239"
    port: 1521
    service_name: "SCADADB"
    username: "system"
    password: "Ora$cle123"
  # ...（可追加多条数据库配置）
```

### 字段说明
- `server.port`：Web 服务监听端口。
- `server.static_dir`：静态资源目录，默认 `./static`，可嵌入编译。
- `server.refresh_interval`：前端自动刷新频率（秒）。
- `server.public_base_path`：前端基础路径，适用于反向代理或子路径部署。
- `logging`：日志相关配置。
- `databases`：数据库列表，支持多实例。

## 示例

1. 启动服务
   ```shell
   ./oracle-dashboard
   ```
2. 访问 Web 页面
   - 浏览器访问 `http://localhost:8090/` 或服务器实际 IP
3. 配置多套数据库时只需在 `databases` 下追加即可。

## 构建与交叉编译

### 本地构建
```shell
go build -o oracle-dashboard main.go config.go logger.go status.go
```

### 交叉编译（示例：Linux 平台）
```shell
GOOS=linux GOARCH=amd64 go build -o oracle-dashboard main.go config.go logger.go status.go
```

### 交叉编译（示例：Windows 平台）
```shell
GOOS=windows GOARCH=amd64 go build -o oracle-dashboard.exe main.go config.go logger.go status.go
```

## 注意事项
- 需确保被监控数据库网络可达，账号权限正确。
- 静态资源可自定义，默认目录为 ./static。
- 部署于子路径下时需设置 public_base_path。
- 建议以后台服务方式运行并结合系统监控。

## License
MIT

---
如需更多帮助，请查阅源码或联系开发者。
