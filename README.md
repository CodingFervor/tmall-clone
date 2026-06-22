# Tmall Clone | 天猫商城仿制

## English | [中文](#中文)

A Tmall.com (天猫) clone — full-stack mobile e-commerce emphasizing **brand flagship stores (旗舰店)** and **genuine-product guarantee (正品保障)**. Go + Gin backend, Vue 3 + Vant frontend, SQLite storage. Auto-seeds brands/products/reviews on first run.

### Features
- **Home** — search, banner, category grid, brand flagship row, product waterfall with "正品保障" tags
- **Brand Pavilion (品牌馆)** — browse all flagship stores with follower counts
- **Brand Store (旗舰店)** — per-brand page with shop header + product grid
- **Category browsing** — sidebar category tree
- **Product detail** — gallery, brand link, genuine guarantee, **SKU spec selector**, reviews with rating + photo upload
- **Shopping cart** — select/quantity/totals
- **Sandbox payment** — cashier page (Alipay default/WeChat/UnionPay), confirmation callback, order status machine
- **Shipment tracking** — 天猫超市配送 + 菜鸟驿站 trajectory, advance through in_transit→delivered, logistics timeline
- **After-sale refunds** — apply from orders, status tracking, admin approval
- **Coupon marketing** — coupon center with 满666 (满减) + 折扣 types; 4 seeded Tmall coupons
- **FTS5 full-text search** — SQLite FTS5 product search with auto-complete suggestions
- **Image upload** — real multipart upload for product images (admin) + review photos
- **Auth** — register/login with JWT, profile
- **Admin panel** — product CRUD with brand assignment + SKU management + image upload

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
- **商品详情** — 图册、品牌链接、正品保障、**SKU 规格选择**、带评分评价 + 晒图上传
- **购物车** — 勾选/改量/合计
- **沙箱支付** — 收银台（默认支付宝/微信/银联）、支付确认回调、订单状态机
- **物流跟踪** — 天猫超市配送 + 菜鸟驿站轨迹、物流时间线页面
- **售后退货** — 订单页申请退款、状态跟踪、管理员审核
- **优惠券营销** — 领券中心（满减券 + 折扣券）、预置 4 张天猫优惠券
- **FTS5 全文搜索** — 商品搜索 + 输入联想
- **图片上传** — 商品图片 + 评价晒图真实上传
- **登录注册** — JWT 鉴权、个人中心含售后/优惠券入口
- **管理后台** — 商品增删改查（含品牌归属）+ SKU 管理 + 图片上传

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
