# Lingua Buddy 数据来源与外部服务清单

## 1. 文档目的

本文档统一说明项目中每项资料和能力从哪里获得、运行时是否依赖第三方、数据保存在哪里，以及外部服务失败时如何处理。

核对日期：2026-06-20。

本文档中的“内部/外部”按运行时数据边界划分：

- **内部静态资料**：已经导入并由项目自己的数据库管理，用户请求时不访问第三方。
- **内部业务资料**：用户使用系统后产生，由项目自己的数据库或文件存储管理。
- **外部导入资料**：通过第三方 RSS/API 定时拉取，再保存到内部数据库。
- **外部实时能力**：用户请求发生时调用第三方 AI、ASR 或 TTS 接口。
- **用户输入资料**：由用户输入、录音、上传或作答产生，进入内部业务流程；某些功能会把必要内容发送给外部服务处理。

需要注意：资料的“最初来源”和“运行时来源”可能不同。例如词典数据最初来自外部开源项目，但已经导入本地 MySQL，因此运行时属于内部资料。

## 2. 当前实际接入状态

| 依赖或资料 | 当前状态 | 当前实现情况 |
| --- | --- | --- |
| MySQL `ecdict` 词典 | 已接入 | 当前 Go 命令行程序可以执行精确查词 |
| 用户业务表 | 仅完成设计 | 当前数据库尚未创建，现有 `lingua` 数据库只有 `ecdict` 表 |
| Redis 缓存与限流 | 已选用、尚未接入 | 承载热门查词缓存、联想缓存和分布式限流；MVP 可通过开关暂不启用 |
| 阿里云 OSS 对象存储 | 已选型、尚未接入 | 生产环境保存用户原始音频；开发环境仍使用本地文件目录 |
| 阿里云大模型（通义千问 / DashScope） | 已选型、尚未接入 | `AIProvider` 接口已设计；具体模型、接口地址和区域待确认后登记 |
| ASR 语音识别（第三方语音转文字 API） | 已确定方向、供应商待定、尚未接入 | `ASRProvider` 接口已设计；具体第三方供应商和接口地址待确认 |
| TTS 语音合成（第三方文字转语音 API，P2） | 已确定方向、供应商待定、尚未接入 | TTS 接口待定义；P2 阶段再确定供应商 |
| VOA Learning English RSS | 来源已确定、尚未接入 | 已设计每日同步流程，尚未实现 `cmd/article-sync` |
| QQ 邮箱 SMTP 发信 | 已选型、保留待用 | 基础设施保留；当前注册改为普通账号密码，暂无功能调用，预留未来邮箱验证/找回密码 |

“已选型/已确定方向”表示供应商或基础设施已经定下来，但项目尚未编写对应接入代码。因此，当前项目代码没有在后台偷偷调用任何 AI、翻译、语音、新闻或邮件接口，也没有连接 Redis 或对象存储。当前唯一已经运行的数据连接是 MySQL。

## 3. 总体数据来源矩阵

| 功能 | 主要资料来源 | 分类 | 运行时外部调用 | 内部保存位置 |
| --- | --- | --- | --- | --- |
| 注册、登录、英语水平 | 用户输入 | 用户输入 + 内部业务资料 | 无 | `users` |
| 精确查词、联想 | MySQL `ecdict` | 内部静态资料 | 无 | `ecdict` 只读 |
| 音标、释义、词形、考试标签、词频 | MySQL `ecdict` | 内部静态资料 | 无 | `ecdict` 只读 |
| 查词历史 | 用户查询行为 | 内部业务资料 | 无 | `dictionary_query_records` |
| 四六级学习计划和固定队列 | 用户选择的 `ecdict.tag` 词集 | 内部派生资料 | 无 | `word_learning_plans`、`word_learning_plan_items` |
| 生词和渐进学习阶段 | 用户收藏、计划激活和单词训练结果 | 内部业务资料 | 无 | `user_words` |
| 单词笔记 | 用户输入 | 内部业务资料 | 无 | `user_word_notes` |
| 收藏句子 | 用户选择和备注 | 内部业务资料 | 无 | `user_sentences` |
| 复习卡片 | `user_words` + `ecdict` 动态组装 | 内部派生资料 | 无 | 不单独保存卡片 |
| 文本翻译 | 用户文本 + 外部 AI | 外部实时能力 | 是，AI Provider | `history_records` 保存确认后的结果 |
| 翻译解释和备选表达 | 外部 AI | 外部实时能力 | 是，AI Provider | 与翻译结果一起保存或展示 |
| 语法分析 | 用户英文 + 外部 AI | 外部实时能力 | 是，AI Provider | `history_records` |
| 语法纠错和润色 | 用户英文 + 外部 AI | 外部实时能力 | 是，AI Provider | `history_records` |
| AI 例句 | `ecdict` 目标词 + 外部 AI | 混合资料 | 是，AI Provider | 默认不保存；用户收藏后进入 `user_sentences` |
| 录音和音频上传 | 用户设备或文件 | 用户输入 | 浏览器麦克风不是第三方服务 | 文件目录/对象存储 + `audio_files` |
| 语音识别 | 用户音频 + 外部 ASR | 外部实时能力 | 是，ASR Provider | 最终文本进入 `history_records` |
| 语音翻译 | ASR 文本 + 外部 AI | 外部实时能力 | 是，AI Provider | `history_records` |
| TTS 英文朗读 | 内部文本 + 外部 TTS | 外部实时能力 | 是，TTS Provider | 默认不保存生成音频 |
| AI 情景对话 | 用户消息 + 外部 AI | 外部实时能力 | 是，AI Provider | `conversations`、`conversation_messages` |
| 作文批改 | 用户作文 + 外部 AI | 外部实时能力 | 是，AI Provider | `training_answer_records`、`history_records` |
| 渐进式单词训练 | `user_words` + `ecdict` + `worddistractor` | 内部派生资料 | 无 | 提交后写 `training_answer_records` |
| 翻译专项训练 | 外部 AI 生成题目并评价 | 外部实时能力 | 是，AI Provider | 提交后写 `training_answer_records` |
| 作文专项训练 | 外部 AI 生成题目并批改 | 外部实时能力 | 是，AI Provider | 提交后写 `training_answer_records`、`history_records` |
| 单词错误历史 | 已提交单词答案 | 内部业务资料 | 无 | `training_answer_records` |
| 翻译错题 | 用户确认的翻译训练错误 | 内部业务资料 | 评价阶段使用 AI | `user_translation_wrong_questions` |
| 外刊文章 | VOA Learning English RSS | 外部导入资料 | 是，每日定时读取 RSS | `articles` |
| 阅读记录 | 用户阅读行为 | 内部业务资料 | 无 | `user_article_reads` |
| 首页数量和今日任务 | 内部业务表实时查询 | 内部派生资料 | 无 | 不建立独立统计表 |

## 4. 内部资料

### 4.1 基础词典 `ecdict`

运行时来源：MySQL 数据库 `lingua.ecdict`。

当前状态：已经存在并可查询，共 770,611 个唯一词条。

原始资料来源：现有表结构、字段和数据规模与开源项目 [ECDICT](https://github.com/skywind3000/ECDICT) 一致。ECDICT 使用 MIT License，并包含英汉释义、音标、词形、考试标签和词频信息。项目发布前仍需确认当前导入文件的具体来源和版本，并在仓库中保留许可证与署名。

运行方式：

```text
用户查词
  -> Lingua Buddy 后端
  -> MySQL ecdict
  -> 返回结构化词条
```

查词时不会调用在线词典、搜索引擎或 AI。即使 AI 服务不可用，基础查词仍然可用。

主要字段：

| 字段 | 资料内容 | 来源处理 |
| --- | --- | --- |
| `word` | 单词或词组 | 直接读取 |
| `phonetic` | 音标 | 有值时展示 |
| `definition` | 英文释义 | 将字面量换行解析为数组 |
| `translation` | 中文释义 | 将字面量换行解析为数组 |
| `collins` | 柯林斯星级 | 1-5 星 |
| `oxford` | 牛津 3000 标记 | 值为 1 时展示 |
| `tag` | 考试标签 | 按空格拆分 |
| `bnc`、`frq` | 语料库词频排名 | 大于 0 时展示 |
| `exchange` | 词形变化 | 解析为过去式、复数等结构 |

考试标签包括：

| 原始标签 | 页面名称 |
| --- | --- |
| `zk` | 中考 |
| `gk` | 高考 |
| `cet4` | 大学英语四级 |
| `cet6` | 大学英语六级 |
| `ky` | 考研 |
| `toefl` | 托福 |
| `ielts` | 雅思 |
| `gre` | GRE |

截至 2026-06-20 的实测数据：CET-4 标签词条 3,849 个，CET-6 标签词条 5,407 个，同时带两个标签的词条 3,451 个。

### 4.2 内部业务数据库

计划与 `ecdict` 共用 MySQL `lingua` 数据库，但业务表由项目自己的 GORM 模型和迁移命令创建。

| 表 | 内部资料 |
| --- | --- |
| `users` | 账号、密码哈希、注册方式和英语水平 |
| `word_learning_plans` | 四级、六级等计划配置和固定随机种子 |
| `word_learning_plan_items` | 完整计划队列、激活和首次掌握状态 |
| `user_words` | 已收藏或已激活单词的当前学习阶段 |
| `user_word_notes` | 用户单词笔记 |
| `user_sentences` | 用户收藏的英文句子、翻译和备注 |
| `dictionary_query_records` | 查词次数和最近查询时间 |
| `articles` | 从授权外部来源导入后的文章或元数据 |
| `user_article_reads` | 阅读进度 |
| `history_records` | 翻译、语音、语法、纠错和作文结果 |
| `conversations` | AI 对话会话信息 |
| `conversation_messages` | 用户和 AI 的对话消息 |
| `audio_files` | 音频文件路径和元数据 |
| `training_answer_records` | 每一次已经提交的专项训练答案 |
| `user_translation_wrong_questions` | 当前仍需重练的翻译错题 |

这些表目前仍是设计，尚未在当前数据库中创建。

### 4.3 内部文件资料

开发环境中，用户上传的原始音频保存在本地上传目录；MySQL 的 `audio_files` 只保存路径、原文件名、MIME 类型、大小和所属用户。

生产环境可以改为对象存储，但对象存储供应商尚未确定。无论使用本地目录还是对象存储，它都只是文件保存设施，不负责生成识别文本。

### 4.4 内部动态生成资料

以下内容由内部规则根据数据库实时组装，不需要外部接口：

- 四六级固定队列：从 `ecdict.tag` 创建计划快照并保存 `queue_position`。
- 今日到期单词：在当前 active 计划中查询已激活且 `user_words.next_review_at <= 当前时间` 的词。
- 新词激活：按每日新增和活跃窗口限制，从计划队列创建或复用 `user_words`。
- 渐进训练题：根据 `user_words.learning_stage` 自动选择英文选中文、中文选英文或中文默写。
- 正确答案：从 `ecdict` 获取。
- 选择题干扰项：由内部 `worddistractor` 模块从 `ecdict` 动态评分和筛选。
- 页面掌握标签：由 `learning_stage` 派生，不单独存库。
- 首页已激活词、计划等待词、到期词和翻译错题数：直接查询业务表。

这些内容不使用机器学习推荐算法，也不提前建立卡片表、任务表、推荐结果表或干扰项表。

### 4.5 内部缓存与限流（Redis）

Redis 是项目自管的内部基础设施，不是学习内容来源，也不保存业务真相。用途：

- 热门词典查询缓存（TTL 1-24 小时）。
- 搜索联想缓存（TTL 5-30 分钟）。
- 登录、AI、ASR、上传等接口的分布式限流。
- 可选缓存 `worddistractor` 对热门目标词的候选结果。

原则：

- MVP 可以不启用 Redis，通过 `REDIS_ENABLED` 开关控制；不可用时退化为直连数据库或单机限流，不影响核心数据正确性。
- 缓存内容随时可由 MySQL 和 `ecdict` 重新计算，绝不把 Redis 当作唯一数据源。
- 不在 Redis 中保存密码、JWT 明文或完整音频。

## 5. 外部资料和接口

### 5.1 VOA Learning English 外刊 RSS

用途：每日外刊文章导入。

官方 RSS 目录：[https://learningenglish.voanews.com/rssfeeds](https://learningenglish.voanews.com/rssfeeds)

内容使用说明：[https://learningenglish.voanews.com/p/6861.html](https://learningenglish.voanews.com/p/6861.html)

当前状态：数据源已确定，但同步程序尚未实现。

调用方式：

- 独立命令 `cmd/article-sync` 每天北京时间 07:00 执行。
- 使用 HTTP GET 读取配置的一个或多个 RSS Feed URL。
- Feed URL 不写死在业务代码中，通过 `ARTICLE_FEED_URLS` 配置。
- 使用原文 URL 作为 `articles.source_url` 唯一键，重复同步时更新而不重复插入。
- 保存标题、摘要、栏目、发布时间、来源链接和署名。
- 只有确认允许再利用的 VOA 自制内容才保存正文。

版权边界：VOA Learning English 说明其自制文本、音频、图片和视频属于公共领域并允许教育或商业使用，但应注明来源；文章中来自 AP、Reuters、AFP 等第三方的内容不属于公共领域，不得复制发布。遇到来源不清晰的文章，只保存元数据、摘要和原文链接。

失败处理：

- RSS 读取失败只记录任务错误，不影响 API 服务。
- 已经导入的文章继续可读。
- 下一次定时任务重新同步。
- 不允许为绕过限制而抓取受版权保护的全文。

### 5.2 AI 大模型 Provider

用途：文本翻译、翻译解释、语法分析、纠错、润色、AI 例句、情景对话、作文批改、翻译训练和作文训练。

当前状态：**已选型阿里云通义千问（DashScope），但尚未接入，当前没有实际外部 AI 调用。** 具体模型、接口地址和部署区域确认后按第 10 节要求补充登记。

已登记信息：

| 项目 | 内容 |
| --- | --- |
| 供应商 | 阿里云百炼 / DashScope（通义千问 Qwen 系列） |
| 接口形态 | 服务端 HTTP API，支持 OpenAI 兼容模式和结构化输出 |
| 候选模型 | 简单翻译/纠错用较小模型（如 `qwen-turbo`/`qwen-plus`），作文批改用更强模型（如 `qwen-max`）；最终型号待确认 |
| 接口地址与区域 | 确认后填入 `AI_API_BASE`，并记录部署区域 |
| 发送数据 | 见下文“各功能发送给外部 AI 的最小资料”；不发送密码、JWT、邮箱 |
| 数据保留与训练 | 选型时确认能否关闭训练用途，记录日志保留期 |
| 失败降级 | 见第 9 节，已保存答案保留并允许重试 |

项目只先定义内部抽象：

```go
type AIProvider interface {
    Translate(ctx context.Context, input TranslationInput) (TranslationOutput, error)
    AnalyzeGrammar(ctx context.Context, input GrammarAnalysisInput) (GrammarAnalysisOutput, error)
    Correct(ctx context.Context, input CorrectionInput) (CorrectionOutput, error)
    GenerateExamples(ctx context.Context, input ExampleInput) ([]Example, error)
    Chat(ctx context.Context, input ChatInput) (ChatOutput, error)
    ReviewEssay(ctx context.Context, input EssayInput) (EssayReviewOutput, error)
    GenerateTranslationExercise(ctx context.Context, input TranslationExerciseInput) (TranslationExercise, error)
    EvaluateTranslation(ctx context.Context, input TranslationEvaluationInput) (TranslationEvaluation, error)
    GenerateEssayTopic(ctx context.Context, input EssayTopicInput) (EssayTopic, error)
}
```

各功能发送给外部 AI 的最小资料：

| 功能 | 发送内容 | 不发送内容 | 保存结果 |
| --- | --- | --- | --- |
| 翻译 | 原文、语言方向、语气、英语水平 | 密码、JWT、邮箱 | `history_records` |
| 语法分析 | 用户英文、英语水平 | 用户账号凭据 | `history_records` |
| 纠错/润色 | 用户英文、目标风格、英语水平 | 无关历史 | `history_records` |
| AI 例句 | 目标单词、主题、难度 | 用户完整学习历史 | 默认不保存 |
| 情景对话 | 当前会话必要上下文、场景、难度 | 其他会话内容 | `conversation_messages` |
| 作文批改 | 作文正文、题目要求、考试类型 | 密码、JWT | `training_answer_records`、`history_records` |
| 翻译训练 | 方向、难度；评价时发送题目和用户译文 | 其他用户数据 | `training_answer_records` |
| 作文出题 | 作文类型、难度 | 用户隐私资料 | 题目未提交时不保存 |

AI 结果必须经过后端 JSON 结构校验，不能把模型返回的任意文本直接当作可信业务数据。

供应商选型前必须确认：

- 是否支持服务端 API 和结构化输出。
- 输入数据是否用于供应商模型训练，能否关闭该用途。
- 数据保留期限和数据处理地区。
- 请求速率、价格、超时和内容长度限制。
- 是否支持开发环境 Mock Provider。

外部 AI 失败时：基础词典、生词本、单词复习和单词训练继续工作；AI 页面返回可重试错误，不写空的成功历史。专项训练已经提交的答案必须先保存在内部数据库，再调用 AI。

### 5.3 ASR 语音识别 Provider

用途：把用户上传或录制的中文/英文音频转换成文字。

当前状态：已确定使用**第三方语音转文字 API**，具体供应商和接口地址待最终确认，当前没有实际 ASR 调用。供应商可以是阿里云智能语音交互，也可以是其他第三方；`ASRProvider` 接口与本项目其余部分解耦，更换供应商只改配置和适配层。

发送给外部 ASR 的资料：音频的**公网可访问 URL**、用户选择的语言或自动检测参数。不得发送密码、JWT 或不相关的用户资料。

**音频必须先存 OSS（开发环境也是）**：阿里云 DashScope Paraformer 录音文件识别是**异步**接口，且只接受可访问的音频 URL，不能直接读本地磁盘。因此语音这条线即使在开发环境也不用本地 `./uploads`，而是上传 OSS 取得私有读 + 短期签名 URL 再识别。对外仍是一个**同步**接口：后端在处理函数内部完成“上传 OSS → 提交 Paraformer → 轮询结果 → 返回文本”，前端只感知一次同步请求（详见[技术设计](02-technical-design.md)第 9 节）。

内部保存：

- 原始音频保存在 OSS，对象键与元数据保存在 `audio_files`。
- 识别结果由 `/speech/transcribe` 同步返回，但**不直接入库**；用户确认/编辑后调用 `/speech/results` 保存。
- 用户确认后的最终识别文本（及语音翻译结果）保存在 `history_records` 的同一条 `record_type=speech` 记录。

ASR 失败或轮询超时返回可重试状态，保留已上传音频；不得写入伪造的识别文本。Paraformer 的文件大小、时长、格式、语言支持与数据保留政策以官方文档为准。

### 5.4 TTS 语音合成 Provider

用途：朗读单词、例句、翻译和纠错结果。

当前状态：P2 功能，已确定使用**第三方文字转语音 API**，具体供应商待 P2 阶段确认，当前未实现。

发送资料：需要朗读的英文文本、英式/美式发音、语速。默认只把返回音频流交给前端播放，不长期保存生成音频。

TTS 不负责词典音标或释义。`ecdict.audio` 当前没有有效数据，因此以后要实现发音，需要外部 TTS 或另一套明确授权的音频词库。

### 5.5 对象存储

用途：生产环境保存用户原始音频。

当前状态：已选型**阿里云 OSS** 保存用户原始音频。**注意：因为 ASR（Paraformer）需要公网可访问的音频 URL，语音功能在开发环境也使用 OSS，而非本地目录**；其他非音频的本地文件可继续用 `./uploads`。Endpoint、Bucket、区域和访问密钥通过环境变量配置（见第 7 节）。

对象存储属于外部基础设施，不是学习内容来源。数据库只保存对象键或路径，访问必须通过后端鉴权或短期签名 URL（OSS 私有读 + STS/签名 URL），不能把永久公开 URL 直接暴露给其他用户。

### 5.6 QQ 邮箱 SMTP（基础设施保留，当前未启用）

用途：作为邮件发信通道**保留**。当前普通账号密码注册不使用邮件；保留给未来的邮箱验证注册、找回密码等场景。

当前状态：已选型 **QQ 邮箱 SMTP**，尚未接入，**当前没有任何功能调用发信**。

已登记信息：

| 项目 | 内容 |
| --- | --- |
| 供应商 | QQ 邮箱 SMTP |
| 接口形态 | SMTP 发信，`smtp.qq.com`，SSL 端口 465（备用 587 STARTTLS） |
| 认证方式 | QQ 邮箱**授权码**（在邮箱设置开启 SMTP 后获取），不是登录密码 |
| 配置项 | `MAIL_*` 环境变量（见第 7 节），功能上线前可留空 |
| 未来用途 | 邮箱验证注册、找回密码等 |

注册方式采用策略模式，首版只实现普通账号密码方式；未来新增邮箱验证、手机号 OTP、微信/GitHub OAuth 等方式时再启用本通道，且不改动核心注册服务，详见[技术设计](02-technical-design.md)第 10.1 节。

QQ 邮箱 SMTP 是发信通道，不是学习内容来源；它不读取也不返回任何词典或学习数据。

## 6. 各功能完整数据流

### 6.1 查词

```text
用户输入单词
  -> 后端校验和标准化
  -> 内部 MySQL ecdict
  -> 返回音标、释义、词形、考试标签和词频
  -> 登录用户的查询行为写入 dictionary_query_records
```

外部接口：无。

### 6.2 生词收藏和复习

```text
ecdict 词条
  + 用户收藏/渐进训练答案
  -> user_words
  -> 到期时动态查询
  -> 再从 ecdict 读取释义
  -> 组装复习卡片
```

外部接口：无。

### 6.3 文本翻译

```text
用户原文
  -> Lingua Buddy 后端
  -> 外部 AI Provider
  -> 后端校验结构化翻译结果
  -> 返回前端
  -> 成功结果写入 history_records
```

翻译不是从 `ecdict` 拼接得出。`ecdict` 只用于单词级词典查询，句子和段落翻译由外部 AI 完成。

### 6.4 语音识别和翻译

```text
用户录音/上传
  -> 上传 OSS（取得签名 URL）+ audio_files
  -> 外部 ASR Provider（Paraformer，提交+轮询，同步返回文本）
  -> 用户检查或修改识别文本
  -> 外部 AI Provider 翻译（复用 /translations）
  -> /speech/results 保存为一条 record_type=speech 的 history_records
```

这里会依次调用两个不同的外部能力：ASR 负责音频转文字，AI 负责文字翻译。音频必须经 OSS 才能交给 Paraformer；语音 + 翻译只保存一条 `speech` 历史，不重复生成 `translation` 历史。

### 6.5 语法工具

```text
用户英文
  -> 外部 AI Provider
  -> 后端结构校验
  -> 分析/纠错/润色结果
  -> history_records
```

外部 AI 只给学习建议，不修改 `ecdict`，也不自动创建错题。

### 6.6 单词专项训练

```text
word_learning_plans 确定当前计划
  -> word_learning_plan_items 提供固定队列和断点
  -> 激活或选择到期的 user_words
  -> learning_stage 决定题型
  -> ecdict 提供正确答案
  -> worddistractor 从 ecdict 生成三个干扰项
  -> 用户提交答案
  -> StagePolicy 判分并计算晋级/降级
  -> training_answer_records
  -> 更新 user_words
  -> 答错历史保留在 training_answer_records
  -> user_words 降级并安排下次复习
```

外部接口：无。`worddistractor` 是后端内部模块，不调用 AI；生成结果不单独存表，用户提交后只保存选项快照。阶段规则见[渐进式单词学习设计](05-progressive-word-learning.md)，固定队列、活跃窗口和断点续学见[词汇学习计划与断点续学设计](06-word-learning-plan-and-resume.md)。

### 6.7 翻译和作文专项训练

```text
外部 AI 生成题目（未提交不入库）
  -> 用户提交答案
  -> training_answer_records 先保存 pending
  -> 外部 AI 评价/批改
  -> 更新 completed 或 failed
  -> 必要时写 history_records / user_translation_wrong_questions
```

### 6.8 外刊阅读

```text
VOA Learning English RSS
  -> 每日 article-sync
  -> 版权和来源检查
  -> articles
  -> 用户阅读
  -> user_article_reads
```

文章进入 `articles` 后，普通阅读请求只读取内部数据库，不需要每次访问 VOA。

## 7. 外部依赖配置

以下是计划中的配置项。真实供应商确定后必须补充实际文档链接、API 版本和区域信息。

```dotenv
# 内部数据库
DB_ADDR=127.0.0.1:3306
DB_NAME=lingua
DB_USER=
DB_PASSWORD=

# Redis 缓存与限流（MVP 可暂不启用）
REDIS_ENABLED=false
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0

# 邮件：QQ 邮箱 SMTP（基础设施保留，当前注册未使用，预留未来邮箱功能）；MAIL_PASSWORD 填授权码
MAIL_PROVIDER=qq_smtp
MAIL_SMTP_HOST=smtp.qq.com
MAIL_SMTP_PORT=465
MAIL_USERNAME=
MAIL_PASSWORD=
MAIL_FROM=
MAIL_CODE_TTL_MINUTES=10
MAIL_RESEND_INTERVAL_SECONDS=60

# 外刊 RSS
ARTICLE_SOURCE=voa_learning_english
ARTICLE_FEED_URLS=
ARTICLE_SYNC_CRON=0 0 7 * * *
ARTICLE_SYNC_TIMEZONE=Asia/Shanghai

# AI：阿里云通义千问（DashScope），模型与地址确认后填写
AI_PROVIDER=aliyun_dashscope
AI_API_BASE=
AI_API_KEY=
AI_MODEL=

# ASR：第三方语音转文字，供应商确认后填写
ASR_PROVIDER=
ASR_API_BASE=
ASR_API_KEY=
ASR_MODEL=

# TTS：第三方文字转语音（P2），供应商确认后填写
TTS_PROVIDER=
TTS_API_BASE=
TTS_API_KEY=
TTS_MODEL=

# 文件存储：开发用本地目录，生产用阿里云 OSS
UPLOAD_STORAGE=local
UPLOAD_DIR=./uploads
OBJECT_STORAGE_PROVIDER=aliyun_oss
OBJECT_STORAGE_ENDPOINT=
OBJECT_STORAGE_REGION=
OBJECT_STORAGE_BUCKET=
OBJECT_STORAGE_ACCESS_KEY=
OBJECT_STORAGE_SECRET_KEY=
```

所有密钥只允许保存在后端环境变量或密钥管理系统中，不得提交 Git，不得返回前端。

## 8. 隐私与安全边界

- 密码只保存安全哈希，绝不发送给 AI、ASR、TTS、VOA、对象存储或邮件接口。
- QQ 邮箱授权码等邮件凭据只保存在后端环境变量；未来启用邮件功能时，验证码只存哈希、不写日志。
- JWT、数据库账号和 API Key 不得进入提示词、日志和业务历史。
- 调用 AI 时只发送当前功能所需文本，不发送用户完整档案或无关历史。
- 调用 ASR 时只发送当前音频和必要语言参数。
- 用户删除语音历史时，同时删除内部音频文件或对象存储对象。
- 外部供应商返回内容先通过后端校验，再写入业务数据库。
- 所有内部业务记录必须按当前 `user_id` 隔离。
- 外刊导入必须保存 `source_name`、`source_url` 和 `attribution`，不能抹掉来源。

## 9. 外部服务故障降级

| 故障 | 仍可使用 | 暂不可使用 | 数据处理 |
| --- | --- | --- | --- |
| AI 不可用 | 查词、生词、复习、单词训练、已有文章 | 翻译、语法、对话、作文和 AI 训练 | 已保存答案保留，允许重试 |
| ASR 不可用 | 文本输入和文本翻译 | 音频自动识别 | 音频可保留等待重试 |
| TTS 不可用 | 所有文本学习功能 | 在线朗读 | 不影响已有数据 |
| VOA RSS 不可用 | 已导入文章 | 当天新文章同步 | 下次任务重试 |
| QQ 邮箱 SMTP 不可用 | 全部功能（当前注册不依赖邮件） | 无（未来邮箱功能上线后才有影响） | 不影响现有数据 |
| 对象存储不可用 | 非音频功能 | 新音频上传 | 不创建伪成功历史 |
| Redis 不可用 | 全部核心功能（退化为直连数据库或单机限流） | 分布式限流与缓存加速 | 缓存可重建，不影响业务数据 |
| MySQL 不可用 | 无法保证核心业务 | 所有依赖数据库的功能 | 返回服务不可用，不调用外部 AI 生成孤立结果 |

## 10. 选型和变更要求

每次确定或更换外部供应商时，必须更新本文档并记录：

1. 供应商名称和官方接口文档。
2. API Base URL、API 版本和部署地区。
3. 使用的模型或服务名称。
4. 发送哪些用户数据。
5. 数据保留、训练使用和删除政策。
6. 限流、价格、超时和最大输入限制。
7. 失败重试和降级行为。
8. 是否需要更新用户隐私说明。

未在本文档登记的第三方接口不得直接加入生产代码。
