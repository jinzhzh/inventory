# 小型进销存系统

轻量级进销存 Web 应用，支持 SKU 管理、入库、出库、库存台账和低库存预警。

## 功能模块

- **SKU管理**：新增/编辑/删除商品，设置编码、分类、单位、预警库存、成本价、售价
- **入库**：选择SKU、填写数量和单价，记录采购入库
- **出库**：选择SKU、填写数量和单价，库存不足时拒绝出库
- **库存一览**：实时库存状态，低库存标红预警
- **出入库台账**：所有出入库记录，支持分页
- **数据总览**：SKU总数、总出入库量、低库存预警数、库存总值
- **低库存预警**：库存低于预警线时自动触发

## 技术栈

- Go 1.21 + SQLite（modernc.org/sqlite 纯Go，无CGO）
- 内嵌前端 SPA（Go embed）
- Docker 单镜像部署
- 响应式布局，移动端可用

## 本地运行

```bash
go run .
# 访问 http://localhost:8080
```

## 部署

### Docker 本地运行

```bash
docker build -t inventory .
docker run -p 8080:8080 -v ./data:/data inventory
# 访问 http://localhost:8080
```

### Fly.io 部署（免费档）

1. 安装 [flyctl](https://fly.io/docs/flyctl/install/)
2. 登录：`flyctl auth login`
3. 部署：`flyctl deploy --app team-inventory`
4. 公网 URL：`https://team-inventory.fly.dev`

### GitHub Actions 自动部署

在仓库 Settings → Secrets 添加 `FLY_API_TOKEN`，push 到 main 后自动：
1. 构建 Docker 镜像 → 推送 GHCR（`ghcr.io/jinzhzh/inventory:latest`）
2. 部署到 Fly.io

## GitHub Actions

- `ci.yml`: go vet + build + 静态校验（push/PR 触发，失败阻断 merge）
- `deploy.yml`: 构建 Docker 镜像推送到 GHCR + 部署到 Fly.io