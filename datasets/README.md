# 可复现演示数据

此目录提供本地 SQLite 演示数据导入工具。默认只读取 UCI 零售交易文件前 10,000 行，并导入唐诗源文件中的全部记录；数据库文件默认写入 `datasets/blog-demo.sqlite`，已被 Git 忽略。原始交易 CSV 不保存到仓库。

## 来源与许可

### A：零售订单与合成社区内容

- 来源：[UCI Online Retail](https://archive.ics.uci.edu/dataset/352/online+retail)，Daqing Chen，DOI [10.24432/C5BW33](https://doi.org/10.24432/C5BW33)。
- 数据集许可：CC BY 4.0；再分发或改编数据时须注明出处。
- 包含发票、商品、数量、日期、英镑单价、匿名客户编号和国家。它不是 GitHub 用户数据，也不含真实博客/帖子。
- 导入器不会保留原始 CustomerID、国家或发票号作为用户信息；使用本次导入范围内生成的 `retail-member-NNNNNN` ID，并以 `retail-order-NNNNNNN` 代替外部发票号。
- 生成的社区文章明确标记 `content_origin=synthetic`，文字说明其不是消费者真实发布或评价。不得将其作为真实用户言论呈现。
- 缺失 CustomerID 的行被跳过；以 `C` 开头的发票记为取消；金额以整数便士存储。

### B：唐诗作者、文章与分类

- 来源：[chinese-poetry/chinese-poetry](https://github.com/chinese-poetry/chinese-poetry)，文件 `全唐诗/唐诗三百首.json`，固定上游 revision `d651e494e0ecb49555f9f473aa7c97ed0e4b53b7`。
- 上游仓库采用 MIT 许可，使用时须保留版权和许可声明。见上游 [LICENSE](https://github.com/chinese-poetry/chinese-poetry/blob/master/LICENSE)。
- 文件名为“唐诗三百首”，但该 revision 文件实际上包含 **366 首诗、86 位作者、239 个源标签（包括合集标记）**；导入后得到 238 个分类。导入时保留源诗 ID 和标签；作者映射为演示会员/作者档案，诗映射为文章，标签按诗歌体裁、主题或教学分类存储。
- 作者资料不虚构传记信息；作者简介留空。

## SQLite 结构

`schema.sql` 将演示语义规范化为：

- `members`、`author_profiles`
- `articles`、`categories`、`article_categories`
- `products`、`orders`、`order_items`

该数据集数据库用于本地演示和导入测试，不是生产迁移。生产 MySQL schema/API 尚需单独实施与审查。

## 导入和测试

```sh
go test ./...
python3 datasets/import_datasets.py --db datasets/blog-demo.sqlite --max-rows 10000
python3 -m unittest discover -s datasets -p 'test_*.py' -v
```

默认不覆盖已有数据库；确认需要重建时加 `--replace`。也可通过 `--retail-csv` 和 `--tang-json` 使用本地文件。

Python 导入器测试使用内存 SQLite 和合成小样本，不访问网络、不使用真实顾客标识。Go DAO/service/API 集成测试也使用独立的内存 SQLite 数据库，不连接 MySQL。脚本依赖 Python 标准库。
