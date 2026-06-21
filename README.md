# Lingua Buddy · 智能英语学习助手

基于 **Go + Gin + GORM** 后端、**Vue 3 + TypeScript** 前端与**阿里云大模型全家桶**（通义千问 / Paraformer 语音识别 / OSS 对象存储）构建的智能英语学习应用。

项目围绕一个完整的学习闭环设计：

> 输入内容（查词 / 翻译 / 说话 / 写句子） → 理解与纠错 → 收藏沉淀 → 练习 → 间隔复习

不只是「调一次 AI 返回结果」，而是把查询、翻译、语音、语法和 AI 能力连接起来，把用户的生词、错误和练习表现沉淀为可复习、可训练的学习数据。

---

## ✨ 功能总览

### P0 · 核心学习闭环
- **账号体系**：注册（策略模式，可扩展邮箱/OAuth）、登录、JWT 鉴权、英语水平设置
- **智能查词**：精确查词、前缀联想（按词频排序）、拼写纠错建议、变形→原形提示、AI 例句
- **渐进式单词学习**：四级/六级词汇计划（频率分桶打乱 + 固定队列 + 活跃窗口 + 断点续学），单词按 `初识→辨认→默写→已掌握` 四阶段独立晋级/降级，间隔复习
- **生词本 / 单词笔记 / 收藏句子**
- **智能翻译**：英汉互译、语气选择、关键表达解释、译文对比
- **语音学习**：录音/上传 → Paraformer 识别 → 编辑 → 翻译 → 存历史
- **语法工具**：语法结构分析、语法纠错、句子润色
- **统一历史中心**：翻译/语音/语法/纠错/作文历史聚合查看

### P1 · 进阶能力
- **外刊阅读**：VOA Learning English RSS 每日导入、列表/搜索、阅读器划词查词收藏
- **AI 情景对话**：餐厅/旅行/面试等场景多轮对话 + 即时反馈
- **作文批改**：分项评分、问题清单、修改后参考文章、版本分组对比
- **专项训练中心**：翻译训练（AI 出题 + 评价 + 翻译错题本）、作文训练出题

---

## 🏗️ 架构

```text
┌──────────────┐   HTTPS / JSON    ┌──────────────────┐
│  Vue 3 SPA   │ ───────────────▶  │   Go + Gin API   │
│ (Pinia/Router│ ◀───────────────  │  (模块化单体)     │
│  /Axios)     │                   └────────┬─────────┘
└──────────────┘                            │
                          ┌─────────────────┼──────────────────┐
                          ▼                 ▼                  ▼
                    MySQL (业务表)     阿里云 OSS         阿里云 DashScope
                    + ecdict(只读)     (音频存储)      千问 / Paraformer
```

- **模块化单体**：一个 Go API 服务、一个 Vue SPA、一个 MySQL；外部 AI/ASR/OSS 通过可替换的 Provider 接口接入，业务层不直接依赖供应商 SDK。
- **分层**：`handler → service → repository / provider`，业务规则集中在 service 层。
- **可离线**：未配置外部密钥时，AI/ASR 自动回退内置 Mock，便于本地联调。

### 技术栈

| 层 | 选型 |
| --- | --- |
| 后端 | Go 1.25 · Gin · GORM · MySQL Driver |
| 数据库 | MySQL 8（沿用既有 `ecdict` 词典表 + 15 张业务表） |
| 认证 | JWT（HS256）· bcrypt 密码哈希 |
| 大模型 | 阿里云通义千问（DashScope，OpenAI 兼容模式） |
| 语音识别 | 阿里云 Paraformer（录音文件识别，异步） |
| 对象存储 | 阿里云 OSS（语音音频 + 签名 URL） |
| 外刊 | VOA Learning English RSS |
| 前端 | Vue 3 · TypeScript · Vite · Pinia · Vue Router · Axios |

---

## 📁 项目结构

```text
lingua-buddy/
├── backend/
│   ├── cmd/
│   │   ├── server/        # API 服务入口
│   │   ├── migrate/       # 业务表迁移命令
│   │   └── article-sync/  # VOA 外刊导入命令
│   ├── internal/
│   │   ├── app/           # 组合根：装配依赖与路由
│   │   ├── config/        # 环境变量配置
│   │   ├── httpx/         # 统一响应 / 错误码 / 分页
│   │   ├── middleware/    # 鉴权 / CORS
│   │   ├── models/        # GORM 模型（15 张业务表 + ecdict）
│   │   ├── platform/      # 数据库连接
│   │   ├── auth/ user/    # 注册登录、个人资料
│   │   ├── lexicon/       # 词典领域模型 / canonicalGloss / 查询
│   │   ├── dictionary/    # 查词、联想、纠错建议、AI 例句、查词历史
│   │   ├── worddistractor/# 选择题干扰项生成
│   │   ├── wordlearning/  # 渐进学习引擎：StagePolicy/令牌/计划/取题/答题事务
│   │   ├── sentence/ wordnote/   # 收藏句子、单词笔记
│   │   ├── translation/ grammar/ # 翻译、语法工具
│   │   ├── speech/ asr/ storage/ # 语音识别、Paraformer、OSS
│   │   ├── ai/            # 大模型 Provider 抽象 + DashScope + Mock
│   │   ├── article/       # 外刊列表/阅读 + RSS 同步
│   │   ├── conversation/ essay/ training/ trainrec/  # 对话、作文、训练
│   │   └── history/       # 统一历史
│   ├── .env.example
│   └── go.mod
├── frontend/
│   └── src/
│       ├── api/           # Axios 客户端（token 拦截、错误码解包）
│       ├── stores/        # Pinia（auth）
│       ├── router/        # 路由 + 守卫
│       ├── layouts/       # 主布局（侧栏导航）
│       └── pages/         # 19 个页面
├── docs/                  # 产品与技术设计文档
└── README.md
```

---

## 🚀 本地开发

### 前置条件
- Go 1.25+
- Node.js 20+（建议 22/24）
- MySQL 8，且 `lingua` 库中已有只读词典表 `ecdict`（约 77 万词条）

### 1. 后端

```powershell
cd backend
Copy-Item .env.example .env   # 然后按本地 MySQL / 密钥修改 .env

go run ./cmd/migrate          # 创建 15 张业务表（不改动只读 ecdict）
go run ./cmd/server           # 启动 API，默认 http://127.0.0.1:8080
go run ./cmd/article-sync     # 可选：导入 VOA 外刊（建议由系统计划任务每日触发）
```

### 2. 前端

```powershell
cd frontend
npm install
npm run dev        # 开发服务器 http://localhost:5173，已代理 /api -> :8080
npm run build      # vue-tsc 类型检查 + 生产构建到 dist/
```

打开 `http://localhost:5173`，注册账号 → 设置英语水平 → 即可开始使用。

---

## 🔑 环境变量

`backend/.env`（参考 `.env.example`）关键项：

| 变量 | 说明 |
| --- | --- |
| `JWT_ACCESS_SECRET` | **必填**，登录令牌签名密钥 |
| `QUESTION_TOKEN_SECRET` | **必填**，单词题目令牌 HMAC 签名密钥 |
| `DB_USER` / `DB_PASSWORD` / `DB_ADDR` / `DB_NAME` | MySQL 连接 |
| `AI_API_KEY` / `AI_API_BASE` / `AI_MODEL` | 通义千问；留空或 `AI_PROVIDER=mock` 回退 Mock |
| `ASR_API_KEY` / `ASR_API_BASE` / `ASR_MODEL` | Paraformer；留空或 `ASR_PROVIDER=mock` 回退 Mock |
| `OBJECT_STORAGE_*` | 阿里云 OSS（Endpoint/Bucket/Region/AccessKey/Secret） |
| `ARTICLE_FEED_URLS` | VOA RSS Feed 地址（逗号分隔） |
| `REDIS_*` | 缓存与限流（MVP 可不启用） |
| `MAIL_*` | QQ 邮箱 SMTP（基础设施保留，当前未启用） |

> **真实 vs Mock**：配置了对应密钥即调用真实阿里云接口；未配置则回退内置 Mock，便于离线联调。
> **语音识别依赖 OSS**：Paraformer 录音文件识别是异步接口、需公网可访问的音频 URL，因此音频先上传 OSS 取签名 URL 再识别（开发环境亦然）。
> 密钥只保存在后端环境变量，**不得提交到 Git**（`.env` 已忽略）。

---

## 🔌 API 概览

统一前缀 `/api/v1`，统一响应 `{ "code", "message", "data" }`，分页 `{ items, page, page_size, total }`。

| 领域 | 主要接口 |
| --- | --- |
| 认证/用户 | `POST /auth/register` · `POST /auth/login` · `GET/PATCH /users/me` |
| 词典 | `GET /dictionary/entries/{word}` · `GET /dictionary/suggestions` · `POST /dictionary/examples` · `GET/DELETE /dictionary/history` |
| 生词/计划/学习 | `POST/GET/DELETE /vocabulary` · `POST/GET /word-learning/plans` · `GET /word-learning/next` · `POST /word-learning/answer` · `GET /word-learning/due` |
| 句子/笔记 | `POST/GET/PATCH/DELETE /sentences` · `/word-notes` |
| 翻译/语法 | `POST /translations` · `POST /translations/compare` · `POST /grammar/analysis` · `POST /corrections` |
| 语音 | `POST /speech/transcribe` · `POST /speech/results` |
| 外刊 | `GET /articles` · `GET /articles/{id}` · `POST /articles/{id}/read` · `GET/DELETE /articles/history` |
| 对话 | `POST/GET /conversations` · `POST/GET /conversations/{id}/messages` · `POST /conversations/{id}/finish` |
| 作文/训练 | `POST /essays/review` · `POST /training/translations/next` · `POST /training/translations/evaluate` · `POST /training/answers/{id}/confirm-wrong` · `/translation-wrong-questions` |
| 历史 | `GET /history` · `DELETE /history/{id}` |

错误码节选：`VALIDATION_ERROR` · `UNAUTHORIZED` · `CONFLICT` · `NO_ACTIVE_PLAN` · `NO_DUE_WORDS` · `QUESTION_TOKEN_EXPIRED` · `AI_TIMEOUT` · `ASR_FAILED`。

---

## 🧪 测试

```powershell
cd backend
go test ./...        # 含纯逻辑单测（StagePolicy）与真实 MySQL 集成测试（学习闭环）
go vet ./...
```

> 集成测试需要本地可连接的 MySQL `lingua.ecdict`；连不上时会自动跳过。

前端：`npm run build`（含 `vue-tsc` 类型检查）。

---

## 📦 部署建议

演示/生产环境使用 Nginx + Docker Compose：

```text
Nginx
├── /        -> Vue 静态文件（frontend/dist）
└── /api/    -> Go API

Go API ── MySQL / OSS / 阿里云 DashScope
```

- 生产数据库使用权限受限的专用账号（`ecdict` 只读 + 业务表读写），禁用 `root`。
- 外刊同步由系统 Cron / 容器定时任务调用 `cmd/article-sync`（每天北京时间 07:00），不要在每个 API 实例内各自启动定时器。
- 多实例部署时启用 Redis 做分布式限流。

---

## 📚 设计文档

- [产品需求文档](docs/01-product-requirements.md)
- [技术设计文档](docs/02-technical-design.md)
- [开发路线图](docs/03-development-roadmap.md)
- [数据来源与外部服务清单](docs/04-data-sources-and-external-services.md)
- [渐进式单词学习设计](docs/05-progressive-word-learning.md)
- [词汇学习计划与断点续学设计](docs/06-word-learning-plan-and-resume.md)

---

## 📄 许可与署名

- 词典数据来自开源项目 [ECDICT](https://github.com/skywind3000/ECDICT)（MIT License），发布时保留其许可证与署名。
- 外刊内容来自 VOA Learning English，仅导入标题/摘要/链接并保留来源署名。
