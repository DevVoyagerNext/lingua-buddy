# Lingua Buddy

基于 Go、Vue 3 与 AI 大模型的智能英语学习助手。

项目围绕完整学习闭环设计：

> 输入内容 -> 理解与纠错 -> 学习 -> 练习 -> 复习

## 项目结构

```text
lingua-buddy/
├── backend/       # Go 后端项目
├── frontend/      # 前端项目
├── docs/          # 产品与技术文档
└── README.md
```

## 本地开发

### 后端

```powershell
cd backend
Copy-Item .env.example .env   # 然后按本地 MySQL/密钥修改 .env
go run ./cmd/migrate          # 创建 15 张业务表（不改动只读 ecdict）
go run ./cmd/server           # 启动 API（默认 http://127.0.0.1:8080）
```

关键环境变量：`JWT_ACCESS_SECRET`、`QUESTION_TOKEN_SECRET` 必填；`AI_API_KEY` 配置后使用阿里云通义千问，留空或 `AI_PROVIDER=mock` 时使用内置 Mock，便于离线联调。

测试：`go test ./...`（集成测试需要本地 MySQL 的 `lingua.ecdict`）。

### 前端

```powershell
cd frontend
npm install
npm run dev        # 开发服务器 http://localhost:5173，已代理 /api -> :8080
npm run build      # 类型检查 + 生产构建到 dist/
```

技术栈：Vue 3 + TypeScript + Vite + Pinia + Vue Router + Axios。

## 项目文档

- [产品需求文档](docs/01-product-requirements.md)
- [技术设计文档](docs/02-technical-design.md)
- [开发路线图](docs/03-development-roadmap.md)
- [数据来源与外部服务清单](docs/04-data-sources-and-external-services.md)
- [渐进式单词学习设计](docs/05-progressive-word-learning.md)
- [词汇学习计划与断点续学设计](docs/06-word-learning-plan-and-resume.md)

## 推荐首版范围

首版优先实现：

1. 用户注册、登录与英语水平设置
2. 词典查询、生词本与基础复习
3. 英汉互译与翻译解释
4. 语音识别、文本翻译与结果保存
5. AI 语法纠错与句子润色
6. 统一历史记录

AI 情景对话、作文批改和自动练习作为第二阶段功能。
