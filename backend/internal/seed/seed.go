package seed

import (
	"database/sql"
	"log"

	"github.com/CodingFervor/tmall-clone/backend/internal/model"
	"github.com/CodingFervor/tmall-clone/backend/internal/repository"
)

// Run populates the database with mock data if empty (idempotent).
func Run(db *sql.DB) {
	userRepo := repository.NewUserRepo(db)
	brandRepo := repository.NewBrandRepo(db)
	catRepo := repository.NewCategoryRepo(db)
	prodRepo := repository.NewProductRepo(db)
	reviewRepo := repository.NewReviewRepo(db)

	if u, _ := userRepo.FindByUsername("admin"); u == nil && !userRepo.Exists("admin") {
		_ = userRepo.Create(&model.User{Username: "admin", Password: "admin123", Nickname: "天猫用户"})
		_ = userRepo.Create(&model.User{Username: "tmail", Password: "123456", Nickname: "天猫达人"})
	}

	// Categories
	if n, _ := catRepo.Count(); n == 0 {
		cats := []model.Category{
			{Name: "天猫超市", Icon: "🛒", SortOrder: 1},
			{Name: "国际美妆", Icon: "💄", SortOrder: 2},
			{Name: "手机数码", Icon: "📱", SortOrder: 3},
			{Name: "服饰鞋包", Icon: "👗", SortOrder: 4},
			{Name: "家用电器", Icon: "🔌", SortOrder: 5},
			{Name: "天猫国际", Icon: "🌍", SortOrder: 6},
			{Name: "母婴亲子", Icon: "🍼", SortOrder: 7},
			{Name: "家居家纺", Icon: "🛏️", SortOrder: 8},
		}
		for _, c := range cats {
			_, _ = db.Exec(`INSERT INTO categories (name, icon, sort_order) VALUES (?,?,?)`, c.Name, c.Icon, c.SortOrder)
		}
	}

	// Brands (flagship stores)
	if n, _ := brandRepo.Count(); n == 0 {
		brands := []model.Brand{
			{Name: "Nike官方旗舰店", Logo: "✓", Description: "耐克官方授权，正品保障", Followers: 89000000, SortOrder: 1},
			{Name: "adidas官方旗舰店", Logo: "▲", Description: "阿迪达斯官方旗舰店", Followers: 76000000, SortOrder: 2},
			{Name: "雅诗兰黛官方旗舰店", Logo: "EL", Description: "雅诗兰黛集团官方授权", Followers: 52000000, SortOrder: 3},
			{Name: "兰蔻官方旗舰店", Logo: "LC", Description: "Lancôme兰蔻官方旗舰", Followers: 48000000, SortOrder: 4},
			{Name: "Apple Store官方旗舰店", Logo: "🍎", Description: "Apple官方授权经销商", Followers: 99000000, SortOrder: 5},
			{Name: "华为官方旗舰店", Logo: "HW", Description: "华为终端官方旗舰店", Followers: 95000000, SortOrder: 6},
			{Name: "SK-II官方旗舰店", Logo: "PITERA", Description: "SK-II官方授权", Followers: 35000000, SortOrder: 7},
			{Name: "小米官方旗舰店", Logo: "MI", Description: "小米官方授权", Followers: 82000000, SortOrder: 8},
		}
		for _, b := range brands {
			_, _ = db.Exec(`INSERT INTO brands (name, logo, description, followers, sort_order) VALUES (?,?,?,?,?)`,
				b.Name, b.Logo, b.Description, b.Followers, b.SortOrder)
		}
	}

	// Products
	var prodCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&prodCount)
	if prodCount == 0 {
		products := []model.ProductInput{
			// 国际美妆 (cat=2) — 旗舰品牌
			{Name: "雅诗兰黛小棕瓶精华50ml 第七代", Subtitle: "雅诗兰黛官方旗舰店 正品保障", Price: 1080, OriginalPrice: 1380, Image: "https://img.alicdn.com/imgextra/est-serum.jpg", Category: "国际美妆", CategoryID: 2, BrandID: 3, BrandName: "雅诗兰黛官方旗舰店", Shop: "雅诗兰黛官方旗舰店", Stock: 200, Sales: 30000, Description: "雅诗兰黛第七代小棕瓶，抗皱紧致，官方正品保障。", Tags: "正品保障,品牌授权"},
			{Name: "兰蔻小黑瓶精华肌底液30ml", Subtitle: "兰蔻官方旗舰店 修护精华", Price: 980, OriginalPrice: 1180, Image: "https://img.alicdn.com/imgextra/lancome.jpg", Category: "国际美妆", CategoryID: 2, BrandID: 4, BrandName: "兰蔻官方旗舰店", Shop: "兰蔻官方旗舰店", Stock: 250, Sales: 28000, Description: "兰蔻小黑瓶精华肌底液，强韧修护，焕亮肌肤。", Tags: "正品保障"},
			{Name: "SK-II神仙水230ml 精华露", Subtitle: "SK-II官方旗舰店 PITERA", Price: 1540, OriginalPrice: 1690, Image: "https://img.alicdn.com/imgextra/sk2.jpg", Category: "国际美妆", CategoryID: 2, BrandID: 7, BrandName: "SK-II官方旗舰店", Shop: "SK-II官方旗舰店", Stock: 150, Sales: 18000, Description: "SK-II神仙水，蕴含90%以上PITERA™精华。", Tags: "正品保障,品牌授权"},
			{Name: "海蓝之谜精华面霜60ml", Subtitle: "天猫国际 正品保税", Price: 2680, OriginalPrice: 2890, Image: "https://img.alicdn.com/imgextra/lamer.jpg", Category: "国际美妆", CategoryID: 2, BrandID: 0, BrandName: "海蓝之谜海外旗舰店", Shop: "LA MER海外旗舰店", Stock: 80, Sales: 8000, Description: "海蓝之谜精华面霜，神奇活性精萃。", Tags: "天猫国际,保税仓"},
			// 手机数码 (cat=3)
			{Name: "Apple iPhone 15 Pro 256GB 蓝色钛金属", Subtitle: "Apple Store官方旗舰店 A17 Pro", Price: 7999, OriginalPrice: 8999, Image: "https://img.alicdn.com/imgextra/iphone15pro.jpg", Category: "手机数码", CategoryID: 3, BrandID: 5, BrandName: "Apple Store官方旗舰店", Shop: "Apple Store官方旗舰店", Stock: 300, Sales: 50000, Description: "iPhone 15 Pro，钛金属设计，A17 Pro芯片。", Tags: "正品保障,官方授权"},
			{Name: "华为 Mate 60 Pro 12+512GB 雅川青", Subtitle: "华为官方旗舰店 卫星通话", Price: 6999, OriginalPrice: 7299, Image: "https://img.alicdn.com/imgextra/mate60pro.jpg", Category: "手机数码", CategoryID: 3, BrandID: 6, BrandName: "华为官方旗舰店", Shop: "华为官方旗舰店", Stock: 200, Sales: 80000, Description: "华为Mate 60 Pro，卫星通话，麒麟芯片回归。", Tags: "正品保障,官方授权"},
			{Name: "小米14 Pro 16+512GB 黑色", Subtitle: "小米官方旗舰店 徕卡光学", Price: 5499, OriginalPrice: 5999, Image: "https://img.alicdn.com/imgextra/mi14pro.jpg", Category: "手机数码", CategoryID: 3, BrandID: 8, BrandName: "小米官方旗舰店", Shop: "小米官方旗舰店", Stock: 180, Sales: 25000, Description: "小米14 Pro，徕卡专业光学，骁龙8 Gen 3。", Tags: "正品保障"},
			// 服饰鞋包 (cat=4)
			{Name: "Nike Air Jordan 1 High OG 黑红", Subtitle: "Nike官方旗舰店 经典复古", Price: 1599, OriginalPrice: 1899, Image: "https://img.alicdn.com/imgextra/aj1.jpg", Category: "服饰鞋包", CategoryID: 4, BrandID: 1, BrandName: "Nike官方旗舰店", Shop: "Nike官方旗舰店", Stock: 100, Sales: 35000, Description: "Air Jordan 1 High OG，经典黑红配色，正品保障。", Tags: "正品保障,限量"},
			{Name: "adidas Ultraboost 22 跑步鞋", Subtitle: "adidas官方旗舰 BOOST科技", Price: 1299, OriginalPrice: 1599, Image: "https://img.alicdn.com/imgextra/ub22.jpg", Category: "服饰鞋包", CategoryID: 4, BrandID: 2, BrandName: "adidas官方旗舰店", Shop: "adidas官方旗舰店", Stock: 150, Sales: 22000, Description: "Ultraboost 22，BOOST中底科技，舒适缓震。", Tags: "正品保障"},
			// 家用电器 (cat=5)
			{Name: "戴森Dyson V12 Detect Slim 无线吸尘器", Subtitle: "天猫国际 海外旗舰", Price: 4690, OriginalPrice: 5290, Image: "https://img.alicdn.com/imgextra/dyson-v12.jpg", Category: "家用电器", CategoryID: 5, BrandID: 0, BrandName: "Dyson海外旗舰店", Shop: "Dyson海外旗舰店", Stock: 90, Sales: 12000, Description: "戴森V12 Detect Slim，激光显尘，强劲吸力。", Tags: "天猫国际,保税仓"},
			{Name: "美的变频空调1.5匹新一级能效", Subtitle: "美的官方旗舰店 节能省电", Price: 2599, OriginalPrice: 3299, Image: "https://img.alicdn.com/imgextra/midea-ac.jpg", Category: "家用电器", CategoryID: 5, BrandID: 0, BrandName: "美的官方旗舰店", Shop: "美的官方旗舰店", Stock: 150, Sales: 20000, Description: "美的变频空调，新一级能效，快速冷暖。"},
			// 天猫超市 (cat=1)
			{Name: "三只松鼠每日坚果750g 混合干果", Subtitle: "天猫超市 次日达", Price: 89, OriginalPrice: 129, Image: "https://img.alicdn.com/imgextra/songshu.jpg", Category: "天猫超市", CategoryID: 1, BrandID: 0, BrandName: "三只松鼠官方旗舰店", Shop: "天猫超市", Stock: 2000, Sales: 100000, Description: "三只松鼠每日坚果，750g混合装，天猫超市次日达。", Tags: "天猫超市"},
			{Name: "百草味猪肉脯靖江特产零食100g*5", Subtitle: "天猫超市 整箱装", Price: 49, OriginalPrice: 69, Image: "https://img.alicdn.com/imgextra/bcw.jpg", Category: "天猫超市", CategoryID: 1, BrandID: 0, BrandName: "百草味官方旗舰店", Shop: "天猫超市", Stock: 3000, Sales: 80000, Tags: "天猫超市"},
			// 天猫国际 (cat=6)
			{Name: "A2白金版婴儿配方奶粉3段 900g", Subtitle: "天猫国际 澳洲直采", Price: 358, OriginalPrice: 418, Image: "https://img.alicdn.com/imgextra/a2milk.jpg", Category: "天猫国际", CategoryID: 6, BrandID: 0, BrandName: "a2海外旗舰店", Shop: "a2海外旗舰店", Stock: 500, Sales: 40000, Description: "A2白金版婴儿配方奶粉，澳洲直采，天猫国际保税。", Tags: "天猫国际,保税仓"},
			// 母婴亲子 (cat=7)
			{Name: "帮宝适纸尿裤L码76片 婴儿尿不湿", Subtitle: "帮宝适官方旗舰店 一级帮", Price: 169, OriginalPrice: 219, Image: "https://img.alicdn.com/imgextra/pampers.jpg", Category: "母婴亲子", CategoryID: 7, BrandID: 0, BrandName: "帮宝适官方旗舰店", Shop: "帮宝适官方旗舰店", Stock: 800, Sales: 60000, Tags: "正品保障"},
		}
		for i := range products {
			if _, err := prodRepo.Create(&products[i]); err != nil {
				log.Printf("seed product %d: %v", i, err)
			}
		}
	}

	// Reviews
	var revCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM reviews`).Scan(&revCount)
	if revCount == 0 {
		reviews := []model.Review{
			{ProductID: 1, UserID: 2, Username: "天猫达人", Rating: 5, Content: "雅诗兰黛小棕瓶正品无疑，官方旗舰店买就是放心，天猫国际物流也快！"},
			{ProductID: 1, UserID: 1, Username: "天猫用户", Rating: 5, Content: "用了一周，皮肤明显细腻了，正品保障值得信赖。"},
			{ProductID: 5, UserID: 2, Username: "天猫达人", Rating: 5, Content: "iPhone 15 Pro 钛金属质感太棒了，Apple官方旗舰店正品！"},
			{ProductID: 8, UserID: 1, Username: "天猫用户", Rating: 5, Content: "AJ1黑红经典，Nike官方旗舰店买的，正品无疑！"},
		}
		for _, rv := range reviews {
			_ = reviewRepo.Create(&rv)
		}
	}

	// Seed SKUs for flagship products (color/spec variants).
	var skuCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM skus`).Scan(&skuCount)
	if skuCount == 0 {
		skus := []model.SKU{
			// 雅诗兰黛小棕瓶 (product 1)
			{ProductID: 1, Spec: `{"规格":"50ml"}`, SpecText: "50ml 经典装", Price: 1080, Stock: 200, SKUCode: "EL-50"},
			{ProductID: 1, Spec: `{"规格":"100ml"}`, SpecText: "100ml 超值装", Price: 1880, Stock: 100, SKUCode: "EL-100"},
			// iPhone 15 Pro (product 5)
			{ProductID: 5, Spec: `{"颜色":"蓝色钛金属","存储":"256GB"}`, SpecText: "蓝色钛金属 256GB", Price: 7999, Stock: 150, SKUCode: "IP15P-BL-256"},
			{ProductID: 5, Spec: `{"颜色":"蓝色钛金属","存储":"512GB"}`, SpecText: "蓝色钛金属 512GB", Price: 8999, Stock: 120, SKUCode: "IP15P-BL-512"},
			{ProductID: 5, Spec: `{"颜色":"原色钛金属","存储":"256GB"}`, SpecText: "原色钛金属 256GB", Price: 7999, Stock: 180, SKUCode: "IP15P-ND-256"},
			// AJ1 (product 8)
			{ProductID: 8, Spec: `{"颜色":"黑红","尺码":"42"}`, SpecText: "黑红 42码", Price: 1599, Stock: 60, SKUCode: "AJ1-BR-42"},
			{ProductID: 8, Spec: `{"颜色":"黑红","尺码":"43"}`, SpecText: "黑红 43码", Price: 1599, Stock: 50, SKUCode: "AJ1-BR-43"},
			{ProductID: 8, Spec: `{"颜色":"白色","尺码":"42"}`, SpecText: "白色 42码", Price: 1599, Stock: 70, SKUCode: "AJ1-WH-42"},
		}
		for _, s := range skus {
			_, _ = db.Exec(`INSERT INTO skus (product_id, spec, spec_text, price, stock, sku_code) VALUES (?,?,?,?,?,?)`,
				s.ProductID, s.Spec, s.SpecText, s.Price, s.Stock, s.SKUCode)
		}
	}

	log.Println("seed: tmall mock data ensured")
}
