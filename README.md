# Tmall Clone | 天猫商城仿制

## English | [中文](#中文)

A Tmall.com (天猫) clone — full-stack mobile e-commerce emphasizing **brand flagship stores (旗舰店)** and **genuine-product guarantee (正品保障)**. Go + Gin backend, Vue 3 + Vant frontend, SQLite storage. Auto-seeds brands/products/reviews on first run.

### Features
- **Home** — search, banner, category grid, brand flagship row, product waterfall with "正品保障" tags
- **Brand Pavilion (品牌馆)** — browse all flagship stores with follower counts
- **Brand Store (旗舰店)** — per-brand page with shop header + product grid
- **Category browsing** — sidebar category tree
- **Product detail** — gallery, brand link, genuine guarantee, reviews with rating
- **Shopping cart** — select/quantity/totals
- **Orders** — place order, pay, order history
- **Auth** — register/login with JWT, profile
- **Search** — keyword + history + hot terms
- **Admin panel** — product CRUD with brand assignment

### Tech Stack
- **Backend**: Go 1.22 + Gin + SQLite (`modernc.org/sqlite`, pure-Go, CGO-free)
- **Frontend**: Vue 3 + Vite + Vant 4 + Vue Router + Axios
- **Deploy**: Docker Compose (backend + nginx frontend) + SQLite volume

### Quick Start

#### Docker Compose
```bash
docker-compose up -d --build
# Frontend: http://localhost  ·  API: http://localhost:8080
```

#### Run separately (dev)
```bash
cd backend && go run ./cmd/server      # :8080, auto-seeds
cd frontend && npm install && npm run dev   # :5173
```

### Demo Account
`admin` / `admin123`

### API Endpoints
| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/auth/login · /register | Auth |
| GET | /api/v1/auth/profile | Profile (auth) |
| GET | /api/v1/categories | Categories |
| GET | /api/v1/brands | Brand list |
| GET | /api/v1/brands/:id | Brand flagship store + products |
| GET | /api/v1/products | List (?page=&category_id=&brand_id=&q=) |
| GET | /api/v1/products/:id | Detail + reviews |
| GET/POST/PUT/DELETE | /api/v1/cart | Cart (auth) |
| GET/POST | /api/v1/orders | Orders (auth); /orders/:id/pay |
| POST | /api/v1/reviews | Add review (auth) |
| POST/PUT/DELETE | /api/v1/admin/products | Product CRUD |

### License
MIT — see [LICENSE](LICENSE).

---

<a id="中文"></a>
# 天猫商城仿制

天猫商城（Tmall.com）仿制 —— 全栈移动电商，强调**品牌旗舰店**与**正品保障**。Go + Gin 后端 + Vue 3 + Vant 前端 + SQLite，首次启动自动填充品牌/商品/评价。

### 功能特性
- **首页** — 搜索、Banner、分类宫格、品牌旗舰入口、带"正品保障"标签的商品瀑布流
- **品牌馆** — 浏览全部旗舰店，含粉丝数
- **旗舰店** — 品牌专属页，含店铺头图 + 商品列表
- **分类** — 侧边分类树 + 商品列表
- **商品详情** — 图册、品牌链接、正品保障、带评分评价
- **购物车** — 勾选/改量/合计
- **订单** — 下单、付款、订单列表
- **登录注册** — JWT 鉴权、个人中心
- **搜索** — 关键词 + 历史 + 热门词
- **管理后台** — 商品增删改查（含品牌归属）

### 技术栈
- **后端**：Go 1.22 + Gin + SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO-free）
- **前端**：Vue 3 + Vite + Vant 4 + Vue Router + Axios
- **部署**：Docker Compose（后端 + nginx 前端）+ SQLite 数据卷

### 快速开始
```bash
# Docker 一键
docker-compose up -d --build
# 分别运行
cd backend && go run ./cmd/server
cd frontend && npm install && npm run dev
```

### 演示账号
`admin` / `admin123`

API 端点详见上方英文版表格。

### 开源许可
MIT — 详见 [LICENSE](LICENSE)。
