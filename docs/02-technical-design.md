# Lingua Buddy 技术设计

## 1. 技术选型

| 层级 | 推荐方案 |
| --- | --- |
| 前端 | Vue 3 + TypeScript + Vite |
| UI | Element Plus 或 Naive UI，二选一 |
| 状态管理 | Pinia |
| 路由 | Vue Router |
| HTTP | Axios |
| 后端 | Go + Gin |
| ORM | GORM + MySQL Driver |
| 数据库 | 沿用现有 MySQL，并在其中新建业务表 |
| 缓存 | Redis（缓存与分布式限流），MVP 可通过开关暂不启用 |
| 认证 | 单 JWT Token |
| 文件存储 | 开发环境本地目录，生产环境阿里云 OSS |
| AI | 可替换的大模型 Provider，选用阿里云通义千问（DashScope） |
| 语音 | 可替换的 ASR/TTS Provider，使用第三方语音转文字/文字转语音 API |
| API 文档 | OpenAPI 3 / Swagger |
| 部署 | Docker Compose + Nginx |

项目统一使用 GORM 访问数据库。现有命令行查词原型需要重构为 GORM 实现，后续业务代码不直接编写完整 SQL 语句，也不混用多套数据访问方式。

### 1.1 数据来源和外部依赖状态

各功能的资料来源、内部表、外部接口、数据发送范围、版权边界和故障降级，以[数据来源与外部服务清单](04-data-sources-and-external-services.md)为准。

当前实现状态：

- 已接入：MySQL `lingua.ecdict`。
- 已确定但未接入：VOA Learning English RSS。
- 已选型但未接入：阿里云通义千问（AI）、阿里云 OSS（生产对象存储）、Redis（缓存与限流）、QQ 邮箱 SMTP（预留，当前注册未使用）。
- 已确定方向、供应商待定：第三方 ASR（语音转文字）与 TTS（文字转语音）。
- 当前数据库实际只有 `ecdict` 表，15 张业务表均尚未创建。

任何新外部接口在进入代码前，必须先在数据来源清单登记供应商、官方文档、发送数据、保留政策和降级方式。

## 2. 架构原则

采用模块化单体：

- 一个 Go API 服务
- 一个 Vue 单页应用
- 一个主数据库
- 可选 Redis
- 外部 AI、ASR 和 TTS 服务

这比微服务更适合课程项目，部署和调试成本较低，同时通过模块边界保留未来拆分能力。

```text
Vue 3 Web
    |
    | HTTPS / JSON
    v
Go + Gin API
    |
    +-- Auth
    +-- Dictionary
    +-- Vocabulary / Review
    +-- Translation
    +-- Speech
    +-- Correction
    +-- Conversation / Essay / Practice
    |
    +------ Database
    +------ File Storage
    +------ AI Provider
    +------ ASR Provider
    +------ TTS Provider
```

## 3. 推荐目录结构

```text
lingua-buddy/
├── backend/
│   ├── cmd/server/
│   │   └── main.go
│   ├── internal/
│   │   ├── auth/
│   │   ├── user/
│   │   ├── dictionary/
│   │   ├── worddistractor/
│   │   ├── vocabulary/
│   │   ├── review/
│   │   ├── translation/
│   │   ├── speech/
│   │   ├── correction/
│   │   ├── conversation/
│   │   ├── essay/
│   │   ├── practice/
│   │   ├── ai/
│   │   ├── platform/
│   │   │   └── database/
│   │   └── shared/
│   ├── api/openapi/
│   ├── tests/
│   ├── .env.example
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── api/
│   │   ├── components/
│   │   ├── layouts/
│   │   ├── pages/
│   │   ├── router/
│   │   ├── stores/
│   │   ├── types/
│   │   └── utils/
│   ├── package.json
│   └── vite.config.ts
├── docs/
├── deploy/
│   ├── nginx/
│   └── docker-compose.yml
└── README.md
```

每个后端业务模块建议包含：

```text
dictionary/
├── handler.go
├── service.go
├── repository.go
├── model.go
├── dto.go
└── errors.go
```

调用方向为 `handler -> service -> repository/provider`，业务规则集中在 service 层。

## 4. 词典接入策略

当前已经确认基础词典表为 `ecdict`，项目通过 GORM 模型映射这张已有数据表。

### 4.1 数据库实测结果

检查时间：2026-06-16。

| 项目 | 实测结果 |
| --- | --- |
| MySQL 版本 | 8.0.45 |
| 数据库 | `lingua` |
| 表名 | `ecdict` |
| 存储引擎 | InnoDB |
| 字符集与排序规则 | `utf8mb4` / `utf8mb4_0900_ai_ci` |
| 精确行数 | 770,611 |
| 主键 | `id`，`bigint unsigned`，自增 |
| 唯一索引 | `uk_ecdict_word(word)` |
| 普通索引 | `idx_ecdict_bnc(bnc)`、`idx_ecdict_frq(frq)` |
| 重复或空单词 | 0 |

当前表与 [ECDICT 官方字段格式](https://github.com/skywind3000/ECDICT#数据格式)一致，并额外使用自增 `id` 作为主键。

完整字段映射：

| 字段 | MySQL 类型 | 可空 | 索引 | 业务含义 |
| --- | --- | --- | --- | --- |
| `id` | `bigint unsigned` | 否 | 主键 | 词条 ID |
| `word` | `varchar(255)` | 否 | 唯一 | 单词或词组 |
| `phonetic` | `varchar(255)` | 是 | - | 音标，以英式音标为主 |
| `definition` | `mediumtext` | 是 | - | 英文释义，多条释义使用转义换行符分隔 |
| `translation` | `mediumtext` | 是 | - | 中文释义，多条释义使用转义换行符分隔 |
| `pos` | `varchar(255)` | 是 | - | 语料库词性占比分布，不是单一词性 |
| `collins` | `int` | 是 | - | 柯林斯星级，取值 1-5 |
| `oxford` | `int` | 是 | - | 是否属于牛津 3000 核心词，`1` 表示是 |
| `tag` | `varchar(255)` | 是 | - | 考试标签，以空格分隔 |
| `bnc` | `int` | 是 | 普通 | BNC 传统语料库词频排名 |
| `frq` | `int` | 是 | 普通 | 当代语料库词频排名 |
| `exchange` | `text` | 是 | - | 词形变化编码，以 `/` 分隔 |
| `detail` | `mediumtext` | 是 | - | JSON 扩展信息，当前没有有效数据 |
| `audio` | `text` | 是 | - | 发音音频 URL，当前没有数据 |

字段实际填充情况：

| 字段 | 有效记录数 | 使用建议 |
| --- | ---: | --- |
| `translation` | 768,739 | 中文释义的主要来源 |
| `phonetic` | 218,065 | 有值时展示 |
| `definition` | 160,884 | 有值时展示英文释义 |
| `exchange` | 96,290 | 解析后展示词形变化 |
| `bnc` | 45,443 | 大于 0 时展示排名 |
| `frq` | 42,231 | 大于 0 时展示排名 |
| `tag` | 14,942 | 解析为考试标签数组 |
| `collins` | 13,633 | 解析为 1-5 星 |
| `oxford` | 3,461 | 值为 1 时展示核心词标记 |
| `pos` | 0 | 当前版本不用于页面展示 |
| `detail` | 0 | 当前版本不用于页面展示 |
| `audio` | 0 | 当前版本不用于页面展示 |

`pos` 的官方语义是类似 `n:46/v:54` 的词性使用比例。由于当前数据库该字段全部为空，首版不能依赖它提供词性。`definition` 和 `translation` 中已经包含 `n.`、`v.` 等释义前缀，可按行解析展示。

释义覆盖情况：

| 类型 | 词条数 |
| --- | ---: |
| 同时有中英文释义 | 159,012 |
| 仅有中文释义 | 609,727 |
| 仅有英文释义 | 1,872 |
| 中英文释义同时为空 | 0 |

API 必须允许单侧释义为空，不能因为缺少英文或中文释义而把有效词条判定为查询失败。

### 4.2 GORM 模型

GORM 持久化模型建议贴合真实表字段：

```go
type ECDICTEntry struct {
    ID          uint64  `gorm:"column:id;primaryKey;autoIncrement"`
    Word        string  `gorm:"column:word"`
    Phonetic    *string `gorm:"column:phonetic"`
    Definition  *string `gorm:"column:definition"`
    Translation *string `gorm:"column:translation"`
    POS         *string `gorm:"column:pos"`
    Collins     *int    `gorm:"column:collins"`
    Oxford      *int    `gorm:"column:oxford"`
    Tag         *string `gorm:"column:tag"`
    BNC         *int    `gorm:"column:bnc"`
    FRQ         *int    `gorm:"column:frq"`
    Exchange    *string `gorm:"column:exchange"`
    Detail      *string `gorm:"column:detail"`
    Audio       *string `gorm:"column:audio"`
}

func (ECDICTEntry) TableName() string {
    return "ecdict"
}
```

业务层使用独立领域模型，避免把 GORM 标签和数据库字段泄漏到 API：

```go
type DictionaryEntry struct {
    ID              uint64
    Word            string
    Phonetic        string
    Definitions     []string
    Translations    []string
    POSDistribution map[string]int
    CollinsStars    *int
    OxfordCore      bool
    Tags            []string
    BNCRank         *int
    FrequencyRank   *int
    WordForms       []WordForm
}
```

实现 `DictionaryRepository` 接口封装 GORM 查询：

```go
type DictionaryRepository interface {
    FindExact(ctx context.Context, word string) (*DictionaryEntry, error)
    Suggest(ctx context.Context, prefix string, limit int) ([]WordSuggestion, error)
    // SuggestSimilar 在精确查词未命中时返回相近拼写建议（DICT-01）。
    SuggestSimilar(ctx context.Context, word string, limit int) ([]WordSuggestion, error)
}
```

#### 相近拼写建议（DICT-01 “查不到时给建议”）

前缀联想（`Suggest`）解决不了拼写错误：用户把 `accomplish` 打成 `accomplsh` 时，前缀 `accomplsh%` 命中为空。因此 `FindExact` 返回 `NOT_FOUND` 时，service 再调用 `SuggestSimilar` 给出纠错候选，仅使用现有 MySQL：

1. 取输入词的前 3–4 个字符作为前缀，用 `word LIKE 'pre%'`（命中 `word` 索引）拉取一批候选；为容忍首字母打错，可对“去掉/替换首字符”各再取一次前缀。
2. 在 Go 内存中计算候选与输入词的 Levenshtein 距离，保留距离 ≤2 的词（可复用 `worddistractor` 的编辑距离实现）。
3. 按“编辑距离升序、`frq` 升序”排序取前 `limit` 条返回。

候选规模受前缀约束（不是全表 77 万行扫描），满足查词的 P95 目标；首版不需要额外的拼写纠错库或基础设施。

精确查询使用 GORM 链式 API：

```go
func (r *dictionaryRepository) FindExact(
    ctx context.Context,
    word string,
) (*DictionaryEntry, error) {
    normalized := strings.TrimSpace(word)

    var record ECDICTEntry
    err := r.db.WithContext(ctx).
        Where(map[string]any{"word": normalized}).
        Take(&record).Error
    if err != nil {
        return nil, err
    }

    return toDictionaryEntry(record), nil
}
```

前缀联想同样通过 GORM 构建查询，不使用 `Raw` 或 `Exec`：

```go
func (r *dictionaryRepository) Suggest(
    ctx context.Context,
    prefix string,
    limit int,
) ([]WordSuggestion, error) {
    normalized := strings.ToLower(strings.TrimSpace(prefix))

    var records []ECDICTEntry
    err := r.db.WithContext(ctx).
        Select("word", "frq").
        Where("word LIKE ?", normalized+"%").
        // DICT-02 要求“按常用程度排序”：frq 越小越常用，NULL（无词频）排最后；
        // frq 有普通索引，词频相同再按字母序保证结果稳定。
        Order("frq IS NULL ASC").
        Order("frq ASC").
        Order("word ASC").
        Limit(limit).
        Find(&records).Error
    if err != nil {
        return nil, err
    }

    return toWordSuggestions(records), nil
}
```

接入规则：

- 不直接修改现有词典数据。
- `id` 是当前表的稳定主键，`word` 已有唯一索引。
- `word` 使用大小写不敏感排序规则，查询时只需去除首尾空格。
- 联想查询只做前缀匹配，使数据库能够利用 `word` 索引。
- 生词本只保存标准化后的单词，展示时按 `word` 从只读词典表重新查询。
- `definition` 和 `translation` 中的字面量 `\n` 解析为释义数组。
- `tag` 按空格解析；`zk`、`gk`、`cet4`、`cet6`、`ky`、`toefl`、`ielts`、`gre` 映射为用户可读标签。
- `bnc` 和 `frq` 为排名，数值越小代表频率越高；空值或 `0` 不展示。
- `exchange` 的词形编码在 repository 内解析，不把原始编码直接暴露给前端。
- 例句、搭配和近反义词应作为独立增强数据，不伪装成现有词典字段。
- Repository 使用 GORM 链式 API，不使用 `db.Raw`、`db.Exec` 或手写完整 SQL。

### 4.3 词形变化解析

`exchange` 使用 `/` 分隔项目，类型代码如下：

| 代码 | 含义 |
| --- | --- |
| `p` | 过去式 |
| `d` | 过去分词 |
| `i` | 现在分词 |
| `3` | 第三人称单数 |
| `r` | 形容词比较级 |
| `t` | 形容词最高级 |
| `s` | 名词复数 |
| `0` | 该词条对应的原形 Lemma |
| `1` | 当前词条相对于 Lemma 的变化类型 |

例如 `go` 的数据是 `i:going/p:went/d:gone/3:goes`，API 应返回结构化词形列表。

### 4.4 规范中文释义 `canonicalGloss`

`translation` 是按字面量 `\n` 分隔的多行多义串（实测 768,739 条均如此），既要展示又要用于单词训练判分。后端提供一个确定性函数 `canonicalGloss(translation) -> string`，供词典、单词训练和 `worddistractor` 共用，避免“出题时截断展示、判分时整段重读”导致选择题永远判错：

1. 按字面量 `\n` 拆分、丢弃空行，取第一条非空释义。
2. 去掉开头词性前缀（`n.`、`v.`、`vt.`、`vi.`、`adj.`、`adv.`、`prep.`、`conj.`、`pron.`、`num.`、`art.`、`int.` 等）。
3. 按 `；;，,` 截取前 1–2 个义项片段并归一化空白与标点，输出稳定短 gloss。

用途见[渐进式单词学习设计](05-progressive-word-learning.md)第 5.4 节：`word_to_meaning_choice` 的正确选项展示与判分、`meaning_to_word_choice`/`meaning_to_word_spelling` 的中文题干、干扰项“释义不同”去重三处统一调用本函数；其版本号并入 `generator_version`。`Translations []string`（完整多义）仍用于词典详情页展示，不受影响。

## 5. GORM 使用约定

### 5.1 数据库初始化

```go
func OpenDatabase(cfg Config) (*gorm.DB, error) {
    db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
        TranslateError: true,
    })
    if err != nil {
        return nil, err
    }

    return db, nil
}
```

应用启动时只创建一个 `*gorm.DB`，通过依赖注入传给各个 Repository。请求上下文通过 `db.WithContext(ctx)` 传递。

### 5.2 模型约定

- 已有词典表使用显式 `TableName()` 和 `column` 标签映射。
- 新业务表只保存业务真正需要的字段；需要更新时间时再增加 `UpdatedAt`。
- 小项目使用硬删除，不引入软删除字段。
- 索引、唯一约束和字段长度通过 GORM 标签声明。
- API DTO 与 GORM Model 分离，禁止直接把持久化模型作为接口响应。
- 关联数据按需使用 `Preload`，列表接口避免无条件加载全部关联。

基础业务模型示例：

```go
type UserWord struct {
    ID                  uint64     `gorm:"primaryKey"`
    UserID              uint64     `gorm:"not null;uniqueIndex:uk_user_word"`
    ECDICTEntryID       *uint64    `gorm:"index"`
    Word                string     `gorm:"size:255;not null;uniqueIndex:uk_user_word"`
    LearningStage       string     `gorm:"size:24;not null;index"`
    StageCorrectStreak  int        `gorm:"not null;default:0"`
    NextReviewAt        time.Time  `gorm:"not null;index"`
    LastTrainedAt       *time.Time
    StageChangedAt      time.Time  `gorm:"not null"`
    FirstMasteredAt     *time.Time
    TotalCorrectCount   int        `gorm:"not null;default:0"`
    TotalWrongCount     int        `gorm:"not null;default:0"`
    LastAnswerCorrect   *bool
    CreatedAt           time.Time
    UpdatedAt           time.Time
}
```

### 5.3 表结构管理

- `ecdict` 是已有只读词典表，不执行 `AutoMigrate`。
- 用户、生词、历史记录等业务表统一由 GORM 模型管理。
- 开发环境可在启动命令或独立迁移命令中执行 `AutoMigrate`。
- 正式环境通过独立的 `cmd/migrate` 命令执行 GORM 迁移，避免每次 API 启动都自动修改表结构。
- 上线前检查 GORM 生成的字段、索引和约束是否符合预期。
- 破坏性结构调整需要编写专门迁移逻辑，不能依赖 `AutoMigrate` 删除字段。

## 6. 数据模型

以下是当前版本的数据模型。除已有只读词典表 `ecdict` 外，新增 15 张业务表。词汇计划保存固定队列和断点，只有激活的单词才进入 `user_words`；只有用户实际提交的答案才形成答题记录。

| 表 | 用途 |
| --- | --- |
| `users` | 用户账号、注册方式和英语水平 |
| `word_learning_plans` | 四级、六级等词汇学习计划 |
| `word_learning_plan_items` | 计划的固定单词队列、激活和首次掌握状态 |
| `user_words` | 已收藏或已激活单词的当前学习阶段 |
| `user_word_notes` | 用户为单词添加的笔记 |
| `user_sentences` | 用户收藏的句子 |
| `articles` | 可阅读的外刊文章 |
| `user_article_reads` | 用户的文章阅读记录 |
| `dictionary_query_records` | 用户查词历史 |
| `history_records` | 翻译、语音、语法分析、纠错和作文的简单历史 |
| `conversations` | AI 对话会话 |
| `conversation_messages` | 用户与 AI 的对话历史 |
| `audio_files` | 原始音频文件信息 |
| `training_answer_records` | 用户已经提交的专项训练答案与评价结果 |
| `user_translation_wrong_questions` | 当前仍需重练的翻译训练错题 |

### 6.1 用户

#### `users`

| 字段 | 说明 |
| --- | --- |
| `id` | 用户 ID |
| `username` | 唯一用户名 |
| `email` | 邮箱，可空；填写时唯一。当前注册不验证邮箱 |
| `password_hash` | 密码哈希；纯第三方 OAuth 注册方式（未来）下可空 |
| `registration_method` | 注册方式：当前 `username_password`，未来扩展 `email_verified`、`phone_otp`、`wechat_oauth` 等 |
| `english_level` | 初级、中级、高级、CET-4、CET-6 |
| `created_at` | 注册时间 |
| `updated_at` | 更新时间 |

首版只实现普通账号密码注册（`registration_method=username_password`）：用户名和密码必填，邮箱可选且不验证，不做注册限制。`registration_method` 记录账号是用哪种策略创建的，便于未来扩展邮箱验证、手机号或第三方登录。未来实现邮箱验证或找回密码时，再补充 `email_verified_at` 字段和 `email_verification_codes` 表（验证码哈希 + 待完成数据 + 过期/尝试/限频）。

`english_level` 在注册时**不收集**，由用户在 onboarding 页设置。用户若跳过 onboarding 直接使用 AI 功能（翻译、纠错、例句等），后端把空 `english_level` 视为默认档 **中级**，保证 AI 提示词可用；用户之后通过 `PATCH /users/me` 设置后即按实际等级调整难度。AI 调用前的等级取值统一经过这个默认兜底，不允许把 null 直接拼进提示词。

### 6.2 单词、句子收藏与复习

#### `word_learning_plans`

保存用户选择的词汇学习目标。MVP 同一用户最多一个 active 计划。

| 字段 | 说明 |
| --- | --- |
| `id` | 计划 ID |
| `user_id` | 用户 ID |
| `name` | 计划名称，例如四级词汇 |
| `source_type` | ecdict_tag/manual |
| `source_value` | cet4/cet6 等 |
| `ordering_mode` | frequency_shuffled |
| `shuffle_seed` | 固定打乱种子 |
| `source_snapshot_count` | 创建时匹配的词数 |
| `daily_new_word_limit` | 每日新增词上限，默认10 |
| `active_word_limit` | 首次学习中的词上限，默认20 |
| `status` | active/paused/archived；完成里程碑由 `completed_at` 表示 |
| `started_at` | 开始时间 |
| `completed_at` | 全部词首次掌握时间，可空 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

创建四级计划时，从 `ecdict.tag` 查询独立 `cet4` 标签，先排除含空格或连字符的词组（首版只学单词），再按词频分桶后使用 `shuffle_seed` 桶内打乱。`source_snapshot_count` 为过滤后入队的单词数，少于或等于实测的 3,849。

#### `word_learning_plan_items`

保存计划的固定队列。创建计划时批量建立全部计划项，但尚未学习的计划项不会创建 `user_words`。

| 字段 | 说明 |
| --- | --- |
| `id` | 计划词条 ID |
| `plan_id` | 所属计划 |
| `ecdict_entry_id` | 词典 ID |
| `word` | 单词快照 |
| `queue_position` | 固定队列位置 |
| `user_word_id` | 激活后关联用户单词，未激活为空 |
| `activated_at` | 进入活跃窗口时间，可空 |
| `first_mastered_at` | 首次达到 mastered 时间，可空 |
| `skipped_at` | 永久跳过时间，可空 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

唯一索引：`plan_id + ecdict_entry_id`、`plan_id + queue_position`。

状态由时间字段派生：未激活、首次学习中、至少掌握过一次或已跳过，不再重复保存状态字符串。完整流程见[词汇学习计划与断点续学设计](06-word-learning-plan-and-resume.md)。

#### `user_words`

只保存用户已经收藏或已经被计划激活的单词，不保存整个四级词表副本。

| 字段 | 说明 |
| --- | --- |
| `id` | 用户单词 ID |
| `user_id` | 用户 ID |
| `ecdict_entry_id` | 对应词典 ID，可空 |
| `word` | 标准化后的单词快照 |
| `learning_stage` | recognition/discrimination/spelling/mastered |
| `stage_correct_streak` | 当前阶段连续正确次数 |
| `next_review_at` | 下次应训练或复习时间 |
| `last_trained_at` | 最近一次提交时间，可空 |
| `stage_changed_at` | 最近阶段变化时间 |
| `first_mastered_at` | 第一次达到 mastered 时间，可空 |
| `total_correct_count` | 累计正确次数 |
| `total_wrong_count` | 累计错误次数 |
| `last_answer_correct` | 最近答案是否正确，可空 |
| `created_at` | 收藏或首次激活时间 |
| `updated_at` | 更新时间 |

唯一约束为 `user_id + word`。一个单词即使属于多个计划，也共用一条全局学习进度。

旧设计中的 `mastery_level` 和 `correct_streak` 删除，使用单一的 `learning_stage` 和 `stage_correct_streak`。完整错误历史来自 `training_answer_records`；`total_wrong_count` 只是便于列表排序的聚合计数。

阶段和默认题型：

| `learning_stage` | 页面含义 | 默认题型 | 晋级条件 |
| --- | --- | --- | --- |
| `recognition` | 初识/不认识 | 英文选中文 | 连续正确2次 |
| `discrimination` | 辨认/模糊 | 中文选英文 | 连续正确2次 |
| `spelling` | 默写/认识 | 中文默写英文 | 连续正确3次 |
| `mastered` | 已掌握 | 到期中文默写 | 正确则保持 |

阶段更新由纯规则组件 `StagePolicy` 计算。详细规则见[渐进式单词学习设计](05-progressive-word-learning.md)。

#### `user_word_notes`

| 字段 | 说明 |
| --- | --- |
| `id` | 笔记 ID |
| `user_id` | 用户 ID |
| `word` | 笔记对应的英文单词 |
| `content` | 笔记内容 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

同一个用户可以为同一个单词添加多条笔记。笔记不强制关联 `user_words`，取消收藏单词时不会删除笔记。查询时使用 `user_id + 标准化后的 word`。

#### `user_sentences`

| 字段 | 说明 |
| --- | --- |
| `id` | 收藏 ID |
| `user_id` | 用户 ID |
| `sentence` | 收藏的英文句子 |
| `sentence_hash` | 标准化句子的 SHA-256，用于去重唯一索引 |
| `translation` | 中文翻译，可空 |
| `note` | 用户备注，可空 |
| `created_at` | 收藏时间 |
| `updated_at` | 更新时间 |

唯一索引：`uk_user_sentence(user_id, sentence_hash)`。

句子可以来自翻译、语法纠错、AI 对话、作文批改或 AI 例句。句子收藏不设置学习阶段和复习时间，也不保存来源表和来源 ID。

去重不能只靠应用层 `SELECT` 后插入（并发下两个请求可能都查不到再都插入，产生重复）。英文句子是长文本，`utf8mb4` 直接对全文做唯一索引会超出索引长度上限（约 768 字符）。因此增加一列 `sentence_hash`（对标准化后的句子取 SHA-256），建唯一索引 `uk_user_sentence(user_id, sentence_hash)`，由数据库保证“同一用户不重复收藏相同句子”，插入冲突即返回 `CONFLICT`。这与翻译错题用 `question_key` 哈希去重是同一套路。

#### 活跃窗口与断点续学

学习页不直接从全部 `ecdict` 随机取词。后端先读取 active 计划，再处理计划项：

1. 优先查询已经激活且 `user_words.next_review_at <= now` 的计划项。
2. 没有到期词时，统计今日激活数和首次学习中的活跃数。
3. 在 `daily_new_word_limit`、`active_word_limit` 范围内锁定下一批未激活计划项。
4. 对计划项关联的单词执行 `user_words` upsert，并回填 `user_word_id`、`activated_at`。
5. 每次只选择其中一个词生成一道题。

用户退出时不需要保存 session：计划队列在 `word_learning_plan_items`，当前阶段和下次时间在 `user_words`，已提交历史在 `training_answer_records`。未提交题目没有数据库记录，下一次打开按原阶段重新生成。

选择题从 `ecdict` 读取正确答案并调用 `worddistractor.Service`；数据库中没有复习卡片表、页面会话表、题库表或干扰项表。

### 6.3 外刊阅读

#### `articles`

| 字段 | 说明 |
| --- | --- |
| `id` | 文章 ID |
| `title` | 英文标题 |
| `summary` | 简短摘要，可空 |
| `content` | 英文正文，可空；只保存授权允许的正文 |
| `difficulty` | beginner/intermediate/advanced |
| `source_name` | 来源名称 |
| `source_url` | 原文链接，唯一索引 |
| `attribution` | 来源署名或版权说明 |
| `published_at` | 原文发布时间，可空 |
| `created_at` | 导入时间 |
| `updated_at` | 最近同步时间 |

文章通过初始化数据或定时导入命令加入，不在用户请求时临时抓取网页。只导入公开领域、获得授权或明确允许使用的内容；没有全文授权时只保存标题、简短摘要和原文链接。

首个数据源使用 VOA Learning English 官方 RSS。新增独立命令 `cmd/article-sync`，由操作系统 Cron、Windows 任务计划或容器定时任务每天北京时间 07:00 调用。不要在每个 API 实例内部各自启动定时器，避免多实例重复导入。

同步过程：

1. 读取 RSS 中的新文章。
2. 解析标题、摘要、发布时间、原文 URL 和栏目。
3. 根据栏目或文本难度映射 `difficulty`。
4. 以 `source_url` 执行幂等新增或更新。
5. 只有确认允许再利用的 VOA 自制内容才保存正文，并保留来源署名。
6. AP、Reuters、AFP 等第三方内容或授权不明确的文章只保存元数据和链接。

#### `user_article_reads`

| 字段 | 说明 |
| --- | --- |
| `id` | 阅读记录 ID |
| `user_id` | 用户 ID |
| `article_id` | 文章 ID |
| `is_finished` | 是否读完 |
| `last_read_at` | 最近阅读时间 |

唯一约束为 `user_id + article_id`。用户打开文章时新增或更新阅读记录，读完后将 `is_finished` 设置为 `true`。

文章阅读页不额外保存划词数据。点击单词时调用词典接口；收藏单词、添加单词笔记和收藏句子分别复用已有接口。

### 6.4 查词历史

#### `dictionary_query_records`

| 字段 | 说明 |
| --- | --- |
| `id` | 记录 ID |
| `user_id` | 用户 ID |
| `word` | 查询的单词 |
| `query_count` | 查询次数 |
| `last_queried_at` | 最近查询时间 |

唯一约束为 `user_id + 标准化后的 word`。重复查询同一个单词时不新增记录，只增加 `query_count` 并更新 `last_queried_at`。

### 6.5 统一历史

#### `history_records`

翻译、语音识别、语法分析、语法纠错和作文批改共用一张简单历史表：

| 字段 | 说明 |
| --- | --- |
| `id` | 历史 ID |
| `user_id` | 用户 ID |
| `record_type` | translation/speech/grammar_analysis/correction/essay |
| `input_text` | 用户输入或识别文本 |
| `result_text` | 最终译文、纠错文本或批改结果 |
| `audio_file_id` | 关联原始音频，可空 |
| `created_at` | 创建时间 |

失败请求、AI 模型信息、token、耗时和临时处理状态不进入业务数据库。

### 6.6 AI 对话历史

#### `conversations`

| 字段 | 说明 |
| --- | --- |
| `id` | 会话 ID |
| `user_id` | 用户 ID |
| `title` | 会话标题 |
| `scene` | 对话场景 |
| `difficulty` | 对话难度 |
| `status` | active/finished |
| `created_at` | 创建时间 |
| `updated_at` | 最近对话时间 |

会话列表按 `updated_at` 倒序展示。`title` 可以使用场景名称，也可以由第一条消息截取生成。

#### `conversation_messages`

| 字段 | 说明 |
| --- | --- |
| `id` | 消息 ID |
| `conversation_id` | 所属会话 ID |
| `role` | user/assistant |
| `content` | 消息内容 |
| `feedback` | AI 对用户回复的反馈，可空 |
| `created_at` | 发送时间 |

同一会话的消息按 `created_at` 排序。删除 `conversations` 记录时级联删除对应消息。AI 反馈可以保存用于回看，但不进入错题本。

### 6.7 原始音频

#### `audio_files`

音频文件本体保存在本地目录或对象存储，数据库只保存文件信息：

| 字段 | 说明 |
| --- | --- |
| `id` | 文件 ID |
| `user_id` | 用户 ID |
| `file_path` | 文件存储路径 |
| `original_name` | 原始文件名 |
| `mime_type` | 文件类型 |
| `file_size` | 文件大小 |
| `created_at` | 上传时间 |

用户删除语音历史（`DELETE /history/{id}` 且 `record_type=speech`）时，在同一操作内级联删除其关联的 `audio_files` 记录与 **OSS 对象**；任一步失败需保证不留下“历史已删但音频残留”或反之的孤儿数据（先删历史与 `audio_files` 行，OSS 对象删除失败则记录待清理日志，不阻断用户操作）。

### 6.8 专项训练答题记录

#### `training_answer_records`

这张表保存用户每一次已经提交的专项训练答案。它是追加式历史表，不保存只展示、跳过或关闭页面时未提交的题目。

| 字段 | 说明 |
| --- | --- |
| `id` | 答题记录 ID |
| `user_id` | 用户 ID |
| `submission_id` | 前端生成的 UUID，用于提交幂等 |
| `training_type` | word/translation/essay |
| `question_type` | 具体题型 |
| `question_key` | 稳定题目标识，用于关联同一道逻辑题 |
| `answer_source` | word_learning/translation_training/essay_training/translation_wrong_retry |
| `user_word_id` | 单词学习时关联 `user_words`，其他训练为空 |
| `word_learning_plan_id` | 本次单词答案来自哪个计划，可空 |
| `word_learning_plan_item_id` | 本次对应的计划项，可空 |
| `question_text` | 用户提交时的题目快照 |
| `options` | 选择题选项 JSON，非选择题为空 |
| `user_answer` | 用户实际提交的答案或作文原文 |
| `reference_answer` | 标准答案或 AI 参考答案，可空 |
| `is_correct` | 单词题为 true/false；开放题可空 |
| `used_hint` | 是否使用提示 |
| `learning_stage_before` | 单词作答前阶段，其他训练为空 |
| `learning_stage_after` | 单词作答后阶段，其他训练为空 |
| `generator_version` | 题目与干扰项算法版本，例如 word-v1 |
| `evaluation_status` | pending/completed/failed |
| `evaluation_result` | AI 评价或结构化批改结果 JSON，可空 |
| `history_record_id` | 对应统一历史记录 ID，可空 |
| `submitted_at` | 用户提交时间 |
| `evaluated_at` | 完成判分或 AI 评价时间，可空 |
| `created_at` | 创建时间 |
| `updated_at` | 评价状态更新时间 |

索引与约束：

- 唯一索引 `uk_training_submission(user_id, submission_id)`，防止网络重试产生重复记录。
- 普通索引 `idx_training_user_time(user_id, submitted_at)`，用于查询个人答题历史。
- 普通索引 `idx_training_question(user_id, training_type, question_key)`，用于查看同一道题的多次提交。
- `user_answer` 创建后不得覆盖；AI 重试只能更新评价状态、参考答案、评价结果和评价时间。
- 单词训练必须保存用户单词、计划、计划项、作答前后阶段以及生成器版本，选项快照保存在 `options`。
- 同一道题再次提交使用新的 `submission_id`，新增一条记录，保留每次真实作答。

建议 GORM 模型：

```go
type TrainingAnswerRecord struct {
    ID               uint64
    UserID           uint64    `gorm:"not null;uniqueIndex:uk_training_submission;index:idx_training_user_time,priority:1;index:idx_training_question,priority:1"`
    SubmissionID     string    `gorm:"size:36;not null;uniqueIndex:uk_training_submission"`
    TrainingType     string    `gorm:"size:20;not null;index:idx_training_question,priority:2"`
    QuestionType     string    `gorm:"size:40;not null"`
    QuestionKey      string    `gorm:"size:255;not null;index:idx_training_question,priority:3"`
    AnswerSource           string  `gorm:"size:32;not null"`
    UserWordID             *uint64 `gorm:"index"`
    WordLearningPlanID     *uint64 `gorm:"index"`
    WordLearningPlanItemID *uint64 `gorm:"index"`
    QuestionText     string    `gorm:"type:text;not null"`
    Options          []byte    `gorm:"type:json"`
    UserAnswer       string    `gorm:"type:mediumtext;not null"`
    ReferenceAnswer  *string   `gorm:"type:mediumtext"`
    IsCorrect          *bool
    UsedHint           bool      `gorm:"not null;default:false"`
    LearningStageBefore *string   `gorm:"size:24"`
    LearningStageAfter  *string   `gorm:"size:24"`
    GeneratorVersion   *string   `gorm:"size:40"`
    EvaluationStatus   string    `gorm:"size:20;not null"`
    EvaluationResult []byte    `gorm:"type:json"`
    HistoryRecordID  *uint64
    SubmittedAt      time.Time `gorm:"not null;index:idx_training_user_time,priority:2"`
    EvaluatedAt      *time.Time
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

#### 独立 `worddistractor` 模块

目录：

```text
internal/worddistractor/
├── service.go
├── repository.go
├── scorer.go
├── normalizer.go
├── model.go
└── errors.go
```

接口：

```go
type Service interface {
    FindMeaningDistractors(ctx context.Context, target DictionaryEntry, count int) ([]MeaningDistractor, GenerationMeta, error)
    FindWordDistractors(ctx context.Context, target DictionaryEntry, count int) ([]WordDistractor, GenerationMeta, error)
}
```

模块边界：

- 输入是目标词和 `ecdict` 领域模型，不读取用户学习状态。
- 输出只包含三个干扰项和生成策略版本，不包含正确答案。
- 不决定题型、不签发令牌、不判分、不写业务数据库。
- Repository 只读查询 `ecdict`；禁止调用 AI 补齐干扰项。

候选拉取必须走索引、避免全表扫描：`ecdict` 上只有 `word`（唯一）、`bnc`、`frq` 三个索引，**`tag` 没有索引**，且 `tag LIKE '%cet4%'` 本身也用不上索引。实时出题在 P95 < 500ms 预算内，绝不能为“同考试标签候选”对 77 万行做 `tag LIKE`。因此：

- 候选用**有索引的 `frq`/`bnc` 词频邻域**拉取（目标词频排名上下取一段区间，`WHERE frq BETWEEN ? AND ?`），一次取回几十到上百个候选。
- 考试标签重合、词性等只作为**内存打分因子**（在已取回的候选上判断 `tag` 是否含目标标签），不写进 SQL 的 `WHERE`。
- 目标词无 `frq`（多数非常用词）时退化为按 `word` 前缀/字母邻域取候选，再内存打分。
- 热门目标词候选可进程内缓存（见下）；缓存未命中也只查索引邻域，不全表扫描。

中文释义干扰项评分优先考虑：有效且不同的释义、相同词性、考试标签重合、词频区间接近。英文单词干扰项优先考虑：Levenshtein 编辑距离、长度、前后缀、考试标签、词频和词性。完整权重和降级规则见[渐进式单词学习设计](05-progressive-word-learning.md)。

候选选择使用 strict -> balanced -> fallback 三层降级。所有层都必须排除目标词、目标词形、重复选项和相同核心释义。无法获得三个安全候选时返回 `ErrInsufficientDistractors`。

首版不建立干扰项表。热门词候选可以使用进程内缓存；以后可增加 Redis，但缓存不属于业务真相。用户提交后的选项快照由 `training_answer_records.options` 保存。

#### 题目令牌与“跳过不保存”

为了避免先把题目写入数据库，获取下一题接口返回一个短期有效的签名 `question_token`：

- 单词题令牌包含用户 ID、`word_learning_plan_id`、`word_learning_plan_item_id`、`user_word_id`、`learning_stage`、题型、选项快照、`generator_version`、`question_key`、签发时间和随机 nonce；散收生词没有计划时，前两个计划字段为空；标准答案不放入客户端可读取字段。
- 翻译或作文题令牌包含用户 ID、语言方向或作文类型、难度、题目正文、`question_key`、签发时间和随机 nonce。
- 令牌使用后端密钥进行 HMAC 或 JWT 签名，客户端不能修改题目来源和标准答案依据。
- 令牌建议 24 小时过期；过期题目不能提交，需要重新获取。
- 用户获取下一题、刷新或关闭页面时，后端不接收答案提交请求，因此数据库没有任何新增记录。
- 当前版本不提供“记录跳过”接口；点击下一题只是丢弃前端当前的 `question_token`。

#### 单词训练提交事务

`POST /word-learning/answer` 请求至少包含：

```json
{
  "submission_id": "uuid",
  "question_token": "signed-token",
  "answer": "accomplish",
  "used_hint": false
}
```

处理流程：

1. 校验登录用户、`submission_id`、答案非空和长度限制。
2. 验证 `question_token` 的签名、用户和过期时间。
3. 使用行锁查询 `user_words`（计划词再加锁对应 `word_learning_plan_items`），确认令牌中的 `user_word_id`、当前阶段与数据库一致；散收生词没有计划项，跳过计划项校验。
4. 从 `ecdict` 重新读取正确答案，不能信任客户端提交的标准答案。
5. 调用纯规则 `StagePolicy` 计算阶段、连续正确次数和 `next_review_at`。
6. 插入 `training_answer_records`，保存用户单词、计划、计划项和阶段前后值。
7. 更新 `user_words` 的阶段、下次时间、累计正确/错误次数和最近答案结果。
8. 第一次进入 mastered 时更新所有关联该 `user_word_id` 的计划项 `first_mastered_at`。
9. 如果计划全部非跳过词条都已首次掌握，写入 `completed_at`；计划可继续保持 active 进行维护复习。
10. 提交事务后返回判分结果、阶段变化和下一次复习时间。

如果唯一索引发现相同 `submission_id` 已存在，接口直接返回原答题结果，不重复更新单词阶段、累计次数或计划进度。已有 AI 记录为 `pending` 时返回处理中状态；为 `failed` 时引导调用重试评价接口。

#### 翻译和作文提交

AI 调用不能放在长数据库事务中，采用“先保存、后评价”流程：

1. 校验请求和题目令牌。
2. 短事务插入答题记录，写入用户原始答案，`evaluation_status=pending`，并记录 `submitted_at`。
3. 事务提交后调用 AI Provider。
4. 成功时短事务更新同一记录的参考答案、评价 JSON、`evaluation_status=completed` 和 `evaluated_at`。
5. 作文批改成功时，在同一短事务中写入 `history_records`，并把其 ID 回填到答题记录。
6. AI 超时或输出无效时，把同一记录更新为 `failed`；用户答案保持不变。
7. 重新评价通过答题记录 ID 发起，只更新原记录，不新增第二条答案。

**`pending` 卡死与超时回收**：第 3、4 步之间如果进程崩溃或更新失败，记录会永久停在 `pending`，前端会一直显示“评价中”。为避免这种死状态：

- `pending` 设软超时阈值 `evaluation_pending_timeout`（默认 60 秒，以 `submitted_at`/`updated_at` 为基准）。
- 查询答题记录时，对 `evaluation_status=pending` 且已超过阈值的记录，按“可重试的卡住状态”返回（错误码 `EVALUATION_PENDING_TIMEOUT`），前端展示重试入口，而不是无限转圈。
- `POST /training/answers/{id}/retry-evaluation` 同时接受 `failed` 和“超时的 pending”两种状态；它只重新调用 AI 并更新评价相关字段，绝不改写已保存的 `user_answer`。
- 重试遵循幂等：以答题记录 ID 为准，成功后置为 `completed`，再次失败置为 `failed`，不新增答案行。

翻译训练的 AI 评价只提供建议。用户通过 `POST /training/answers/{id}/confirm-wrong` 确认明显错误后，后端才新增或更新翻译错题。

### 6.9 错误记录与翻译错题

#### 单词错误记录

单词不建立当前错题表：

- 每次错误答案保存在 `training_answer_records`，不会被后续正确答案覆盖。
- `user_words.learning_stage`、`next_review_at` 和 `last_answer_correct` 表示当前补学状态。
- 当前薄弱词查询 `learning_stage IN ('recognition','discrimination') OR last_answer_correct=false`。
- 历史错词查询 `training_answer_records WHERE training_type='word' AND is_correct=false`。

这避免了 `user_words` 与单词错题表同时维护“是否仍需补学”而产生冲突。

#### `user_translation_wrong_questions`

翻译训练没有对应的持久单词进度，因此继续使用独立的待重练表：

| 字段 | 说明 |
| --- | --- |
| `id` | 翻译错题 ID |
| `user_id` | 用户 ID |
| `direction` | zh_to_en/en_to_zh |
| `question_key` | 语言方向和原文的稳定哈希 |
| `question_text` | 待翻译原文 |
| `reference_answer` | 最近参考答案 |
| `user_answer` | 最近错误译文 |
| `wrong_count` | 累计确认错误次数 |
| `last_answer_record_id` | 最近答题记录 ID |
| `created_at` | 首次加入时间 |
| `updated_at` | 最近更新时间 |

唯一约束为 `user_id + question_key`。用户确认翻译错误时新增或更新；重练后由用户确认已经解决时删除。删除翻译错题不删除答题历史。

#### 单词题生成

`GET /word-learning/next` 不查询题库、不调用 AI，也不使用机器学习推荐算法。它服务两类到期单词：来自 active 计划激活的词，以及用户从查词页散收（`VOCAB-01`，没有任何计划项）的词。两类词都在 `user_words` 中，凭 `next_review_at` 到期即可被选中，因此**没有计划也能复习散收生词**：

1. 收集当前用户全部到期的活跃单词：`user_words.next_review_at <= now`，且未被任何计划标记为 `skipped`。这一步不要求存在 active 计划，散收生词与计划词一起参与候选。
2. 如果有到期词，按“逾期更久 -> recognition -> discrimination -> spelling -> mastered -> 最近训练更早”排序，选出一道，跳到第 5 步。
3. 如果没有到期词，且用户存在 active 计划：在事务中对该计划行加锁（`SELECT ... FOR UPDATE`），重新统计当日新增数和活跃数，在 `daily_new_word_limit`、`active_word_limit` 名额内从最小 `queue_position` 激活 `activated_at IS NULL AND skipped_at IS NULL` 的新计划项并 upsert `user_words`，再从新激活的词中选一道。加锁保证并发的 `next` 请求串行结算名额，不会超额激活。
4. 如果没有到期词、也没有可新增的计划词：
   - 用户有词但都未到期（散收或计划词未到期）：返回 `NO_DUE_WORDS` 和下一次到期时间。
   - 用户完全没有可学的词（既无 active 计划，也无任何 `user_words`）：返回 `NO_ACTIVE_PLAN`，引导创建计划或先收藏生词。
5. 根据选中单词的 `learning_stage` 自动确定题型。
6. 从 `ecdict` 获取正确答案；选择题调用 `worddistractor` 获取三个干扰项。
7. QuestionBuilder 打乱选项并签发 `question_token`；计划词的令牌带计划与计划项 ID，散收生词的这两个字段为空。
8. 返回一道题，不创建答题记录；只有答案提交接口会落库。

第 3 步的激活是一个会写库的动作（改计划项、建 `user_words`、消耗当日额度），必须放在加锁事务里完成。即使前端双击、预取或重复请求 `next`，加锁后每次都重新结算名额，激活总数不会突破 `daily_new_word_limit` 和 `active_word_limit`。

#### 翻译与作文题生成

- `POST /training/translations/next` 根据语言方向、英语水平和难度调用 AI 生成一句文本，只返回题目和签名令牌。
- `POST /training/essays/topic` 根据作文类型和难度生成题目，或为用户自定义题目签发令牌。
- 生成结果本身不入库。只有用户提交译文或作文时才创建答题记录。

### 6.10 不入库的数据

- 只展示但未提交的专项训练题目
- 用户点击下一题而放弃的题目
- 前端丢弃的 `question_token`
- 复习流水和动态生成的复习卡片
- 推荐结果或练习批次
- AI 调用 token、耗时和供应商信息

## 7. API 设计

统一前缀：`/api/v1`

统一响应建议：

```json
{
  "code": "OK",
  "message": "success",
  "data": {}
}
```

分页响应：

```json
{
  "items": [],
  "page": 1,
  "page_size": 20,
  "total": 0
}
```

### 7.1 认证与用户

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/auth/register` | 注册：body 含 `method`、用户名、密码、可选邮箱；创建用户并签发 JWT |
| POST | `/auth/login` | 登录 |
| GET | `/users/me` | 当前用户 |
| PATCH | `/users/me` | 更新邮箱或英语水平 |

注册接口按 `method` 分发到对应 `RegistrationStrategy`（见第 10.1 节）。首版仅支持 `method=username_password`（普通账号密码、不验证邮箱、不限制注册），未知方式返回 `REGISTRATION_METHOD_UNSUPPORTED`。

### 7.2 词典、收藏和复习

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/dictionary/entries/{word}` | 精确查词 |
| GET | `/dictionary/suggestions?q=` | 搜索联想 |
| GET | `/dictionary/history` | 查词历史 |
| DELETE | `/dictionary/history/{id}` | 删除查词历史 |
| POST | `/vocabulary` | 收藏生词 |
| GET | `/vocabulary` | 生词列表 |
| DELETE | `/vocabulary/{id}` | 移出生词本 |
| POST | `/word-notes` | 添加单词笔记 |
| GET | `/word-notes?word=` | 查询某个单词的笔记 |
| PATCH | `/word-notes/{id}` | 修改单词笔记 |
| DELETE | `/word-notes/{id}` | 删除单词笔记 |
| POST | `/sentences` | 收藏句子 |
| GET | `/sentences` | 收藏句子列表 |
| PATCH | `/sentences/{id}` | 修改翻译或备注 |
| DELETE | `/sentences/{id}` | 取消收藏句子 |
| POST | `/word-learning/plans` | 创建四级、六级或自定义词汇计划 |
| GET | `/word-learning/plans` | 查询个人词汇计划 |
| GET | `/word-learning/plans/{id}` | 查询计划队列和进度数量 |
| POST | `/word-learning/plans/{id}/activate` | 激活或切换计划 |
| POST | `/word-learning/plans/{id}/pause` | 暂停计划 |
| GET | `/word-learning/due` | 查询当前计划到期概览 |
| GET | `/word-learning/next` | 获取下一道到期题，必要时激活新词 |
| POST | `/word-learning/answer` | 提交答案并更新阶段、计数和计划进度 |
| POST | `/word-learning/plan-items/{id}/skip` | 跳过该计划词，写入 `skipped_at`，释放活跃槽位 |
| GET | `/word-learning/words` | 查询计划词条及当前阶段 |
| GET | `/word-learning/wrong-answers` | 查询历史单词错误答案 |

### 7.3 外刊阅读

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/articles?difficulty=&keyword=` | 查询外刊文章列表 |
| GET | `/articles/{id}` | 查询文章正文 |
| POST | `/articles/{id}/read` | 记录打开文章或标记读完 |
| GET | `/articles/history` | 查询个人阅读历史 |
| DELETE | `/articles/history/{id}` | 删除阅读记录 |

文章详情页选中单词后调用 `/dictionary/entries/{word}`；收藏单词、笔记和句子继续调用现有接口。

### 7.4 翻译与语音

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/translations` | 创建翻译 |
| POST | `/translations/compare` | 用户译文对比（TRANS-04，P1）：输入原文与用户译文，返回参考译文与准确性/语法/自然度反馈 |
| POST | `/speech/transcribe` | 上传音频，同步返回识别文本与音频文件 ID（不写历史） |
| POST | `/speech/results` | 用户确认（可编辑）识别文本后保存：写入语音类型统一历史并关联音频 |

`/speech/transcribe` 只负责识别并落 `audio_files`，不写 `history_records`；是否保存、保存什么由用户在前端确认后调用 `/speech/results` 决定。语音翻译复用 `/translations` 得到译文，再由 `/speech/results` 统一保存最终文本与译文（字段映射见第 9 节）。

### 7.5 语法工具与历史

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/grammar/analysis` | 分析英文句子结构 |
| POST | `/corrections` | 创建纠错或润色 |
| GET | `/history` | 查询统一历史 |
| DELETE | `/history/{id}` | 删除历史 |

### 7.6 第二阶段

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/conversations` | 创建对话会话 |
| GET | `/conversations` | 查询对话会话列表 |
| POST | `/conversations/{id}/messages` | 发送消息 |
| GET | `/conversations/{id}/messages` | 查询会话消息 |
| POST | `/conversations/{id}/finish` | 结束会话 |
| DELETE | `/conversations/{id}` | 删除会话及消息 |
| POST | `/essays/review` | 提交作文并返回批改 |
| POST | `/training/translations/next` | AI 生成一句翻译训练文本，不入库 |
| POST | `/training/translations/evaluate` | 先保存译文，再调用 AI 评价 |
| POST | `/training/essays/topic` | AI 生成作文题目，不入库 |
| POST | `/training/essays/review` | 先保存作文，再调用 AI 批改 |
| GET | `/training/answers` | 分页查询当前用户答题记录 |
| GET | `/training/answers/{id}` | 查询答题详情和评价结果 |
| POST | `/training/answers/{id}/retry-evaluation` | 重试 `failed` 或超时卡住的 `pending` AI 评价，只更新评价、不新增答案 |
| POST | `/training/answers/{id}/confirm-wrong` | 确认翻译答案错误并加入翻译错题 |
| GET | `/translation-wrong-questions` | 查询当前翻译错题 |
| POST | `/translation-wrong-questions/{id}/answer` | 提交翻译错题重练并保存答题记录 |
| DELETE | `/translation-wrong-questions/{id}` | 确认解决并移除翻译错题 |

**作文版本历史（ESSAY-03 “同一作文不同版本分数变化”）**：独立作文批改 `/essays/review` 与作文训练 `/training/essays/review` 都通过 `training_answer_records` 落库（`training_type=essay`），并写一条 `record_type=essay` 的 `history_records`。为支持版本对比，同一篇作文的多次提交使用**稳定的** `question_key = hash(user_id + 标准化标题 + 目标考试)`：每次“修改后再次提交”都新增一条答题记录但共享同一 `question_key`，`GET /training/answers?training_type=essay&question_key=…` 即可按 `submitted_at` 取出同一作文的历次评分。若用户改了标题则视为新作文（新 `question_key`）。这样无需给 `history_records` 增加分组字段。

训练模块不提供“跳过题目”写入接口。前端点击下一题时只请求新的题目并丢弃旧令牌，后端不产生数据库记录。

## 8. AI 接入设计

### 8.1 Provider 接口

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

不同业务可以共用底层模型客户端，但必须使用独立提示词和输入输出类型。

语法分析响应建议：

```json
{
  "sentence_type": "complex",
  "main_clause": {
    "subject": "The book",
    "predicate": "is",
    "complement": "interesting"
  },
  "clauses": [
    {
      "type": "relative_clause",
      "text": "that you recommended"
    }
  ],
  "tense": "simple_present",
  "voice": "active",
  "grammar_points": [
    {
      "name": "relative clause",
      "explanation_zh": "that 引导定语从句，修饰 book。"
    }
  ]
}
```

服务端必须验证返回结构。语法分析与语法纠错使用不同 DTO，不能把分析结果当作纠错结果处理。

### 8.2 结构化输出

翻译示例：

```json
{
  "translated_text": "I really like playing basketball.",
  "key_expressions": [
    {
      "expression": "like doing something",
      "explanation_zh": "表示喜欢做某事"
    }
  ],
  "alternatives": []
}
```

纠错示例：

```json
{
  "corrected_text": "I really like playing basketball.",
  "issues": [
    {
      "type": "word_choice",
      "original": "very like",
      "replacement": "really like",
      "explanation_zh": "very 通常不直接修饰动词 like。"
    },
    {
      "type": "grammar",
      "original": "like play",
      "replacement": "like playing",
      "explanation_zh": "这里使用 like doing something 结构。"
    }
  ]
}
```

服务端处理顺序：

1. 校验用户输入。
2. 构建版本化提示词。
3. 调用 Provider，并设置超时。
4. 解析结构化输出。
5. 使用代码再次验证字段和枚举值。
6. 返回客户端；用户需要历史时只保存输入文本和最终结果。

不得把模型返回的任意 JSON 不经校验直接写入数据库。

### 8.3 成本控制

- 对输入字符数和对话上下文长度设上限。
- 对每个用户和 IP 限流。
- 使用较小模型完成简单翻译和纠错。
- 只有作文批改等复杂任务使用能力更强的模型。
- 缓存不涉及隐私的重复公共请求，例如同一单词同等级的 AI 例句。
- 调试时可将耗时和错误码写入普通应用日志，但不建立数据库日志表。

## 9. ASR 与文件处理

```go
type ASRProvider interface {
    Transcribe(ctx context.Context, audio AudioInput) (Transcript, error)
}
```

**关键约束：识别需要公网可访问的音频 URL，因此音频统一存 OSS（开发环境也是）。** 当前 ASR 选用阿里云 DashScope Paraformer 录音文件识别，它是**异步**接口（提交任务 → 轮询结果），且只接受一个公网可访问的音频 URL，不能直接读本地磁盘文件。所以即使开发环境，音频也先上传到 OSS 取得签名 URL 再识别，不能用本地 `./uploads` 目录喂给 Paraformer。

对外仍是**一个同步接口**：`/speech/transcribe` 在处理函数内部完成“上传 OSS → 提交 Paraformer → 轮询直到完成或超时 → 返回文本”，前端只看到一次同步请求。

`POST /speech/transcribe` 流程：

1. 验证登录状态、文件类型、真实 MIME、大小（≤20MB）和时长（≤5 分钟）。
2. 将音频上传到 OSS，得到对象键与可访问 URL（私有读 + 短期签名 URL）。
3. 在 `audio_files` 保存对象键、原文件名、MIME、大小和所属用户，得到 `audio_file_id`。
4. 用音频 URL 同步调用 Paraformer：提交任务后在处理函数内**轮询**任务状态，直到成功、失败或达到超时上限（按 PRD 11.1，5 分钟音频目标 60s 内返回；轮询整体超时建议略大于该目标）。
5. 成功则返回识别文本、检测语言和 `audio_file_id`；**此步不写 `history_records`**。
6. 失败或超时返回 `ASR_FAILED`（可重试），已上传音频与 `audio_files` 记录保留，供重试，不写伪造文本。

> 同步轮询会占用一个较长的 HTTP 请求（最坏接近识别目标时长）。需为该路由单独设置更长的服务端写超时与网关超时，并对该接口限流，避免长请求堆积。这是“保持同步接口”的已知代价；若未来并发压力大，可改造为“提交 + 轮询”两段式接口而不改变前端其余流程。

`POST /speech/results` 保存流程（用户确认/编辑识别文本后）：

1. 校验 `audio_file_id` 属于当前用户。
2. 写入一条 `history_records`：`record_type=speech`、`input_text=` 最终识别文本、`result_text=` 语音翻译结果（无翻译时留空）、`audio_file_id=` 关联音频。
3. 语音 + 翻译只保存**一条** `speech` 历史（不再额外生成 `translation` 历史），避免同一次操作重复入库。

用户删除该 `speech` 历史时，连带删除关联的 `audio_files` 记录与 OSS 对象（见错误处理与第 6.7 节）。

## 10. 认证与权限

- JWT Token 有效期建议 1-7 天。
- 退出登录时前端删除本地 Token；小项目不实现刷新令牌和服务端撤销列表。
- 所有用户资源查询同时包含资源 ID 和当前 `user_id` 条件。
- 当前版本只有普通用户，不设计应用角色字段和角色权限中间件。
- 本地检查可以使用数据库高权限账号，生产环境禁止使用 MySQL `root`。
- 生产数据库账号只授予 `ecdict` 的读取权限，以及业务表所需的读写权限。

### 10.1 注册方式策略模式

注册方式未来可能扩展（邮箱验证、手机号 OTP、第三方 OAuth），因此用**策略模式**隔离每种方式的差异，核心注册服务只按 `method` 分发，不内嵌具体方式逻辑。新增一种方式只实现接口并注册，不改调用方。

```go
type RegistrationMethod string

const (
    RegistrationMethodUsernamePassword RegistrationMethod = "username_password"
    // 预留：RegistrationMethodEmailVerified、RegistrationMethodPhoneOTP、RegistrationMethodWechatOAuth ...
)

type RegistrationStrategy interface {
    Method() RegistrationMethod
    // 校验输入并创建用户。
    // username_password：直接创建账号（不验证邮箱、不限制注册）。
    // 未来 email_verified / OAuth：在创建前各自增加验证码或授权步骤，可附加独立的预备接口。
    Register(ctx context.Context, req RegistrationRequest) (*User, error)
}

// 按 method 分发；新增方式只需在启动时注册一个实现
type RegistrationRegistry struct {
    strategies map[RegistrationMethod]RegistrationStrategy
}

func (r *RegistrationRegistry) Get(m RegistrationMethod) (RegistrationStrategy, error) {
    s, ok := r.strategies[m]
    if !ok {
        return nil, ErrRegistrationMethodUnsupported // -> REGISTRATION_METHOD_UNSUPPORTED
    }
    return s, nil
}
```

首版只实现 `UsernamePasswordStrategy`：用户名 + 密码直接创建账号，邮箱可选且不验证，不做注册数量或频率限制。`AuthService` 持有 `RegistrationRegistry`，收到 `method` 后取对应策略；未知 `method` 返回 `REGISTRATION_METHOD_UNSUPPORTED`。登录沿用用户名或邮箱 + 密码。

### 10.2 账号密码注册流程

```text
POST /auth/register
  -> 校验 method、用户名和密码（前后端同时校验）
  -> 用户名已被占用（或邮箱填写且重复）返回 CONFLICT
  -> 创建 users(registration_method=username_password)，密码以 Argon2id/bcrypt 哈希存储
  -> 签发 JWT 并返回，等价于自动登录
```

要点：

- 普通注册不发送邮件、不验证邮箱、不限制注册次数。
- 密码只存哈希，明文不落库、不写日志。
- 邮箱为可选信息；填写时保持唯一，但不做验证。

### 10.3 EmailProvider 与 QQ 邮箱（保留，当前未启用）

QQ 邮箱 SMTP 作为基础设施**保留**，但当前普通账号密码注册不使用它，暂时没有任何功能调用邮件发送。保留的 `EmailProvider` 抽象和配置预留给未来的邮箱验证注册、找回密码等场景：

```go
type EmailProvider interface {
    SendVerificationCode(ctx context.Context, to, code, purpose string) error
}
```

未来实现时由 `QQMailProvider` 通过 QQ 邮箱 SMTP（`smtp.qq.com`，SSL 465）发送，认证使用 QQ 邮箱**授权码**而非登录密码；供应商、地址和凭据已在 `MAIL_*` 环境变量预留（见第 14 节）。在该功能上线前，这些配置可留空。

## 11. 错误处理

建议错误码：

| 错误码 | 含义 |
| --- | --- |
| `VALIDATION_ERROR` | 参数校验失败 |
| `UNAUTHORIZED` | 未登录或令牌失效 |
| `FORBIDDEN` | 无访问权限 |
| `NOT_FOUND` | 资源不存在 |
| `CONFLICT` | 用户名、邮箱、收藏等重复 |
| `RATE_LIMITED` | 请求过于频繁 |
| `REGISTRATION_METHOD_UNSUPPORTED` | 不支持的注册方式 |
| `AI_TIMEOUT` | AI 调用超时 |
| `AI_INVALID_RESPONSE` | AI 输出无法解析 |
| `ASR_FAILED` | 语音识别失败 |
| `UPLOAD_INVALID` | 文件不合法 |
| `QUESTION_TOKEN_INVALID` | 训练题目令牌无效或被修改 |
| `QUESTION_TOKEN_EXPIRED` | 训练题目令牌已过期或单词阶段已变化 |
| `INSUFFICIENT_DISTRACTORS` | 无法找到三个安全且唯一的干扰项 |
| `ACTIVE_WORD_PLAN_CONFLICT` | 用户已经有一个 active 词汇计划 |
| `NO_ACTIVE_PLAN` | 用户既没有 active 词汇计划，也没有任何可学单词；存在散收生词时不再返回此码 |
| `NO_DUE_WORDS` | 有可学单词但当前都未到期，且不能再激活新词 |
| `EVALUATION_FAILED` | 答案已保存，但 AI 评价失败，可重试 |
| `EVALUATION_PENDING_TIMEOUT` | 答案已保存，AI 评价超时未完成，可重试 |
| `INTERNAL_ERROR` | 未预期错误 |

错误响应必须带请求追踪 ID，服务端日志使用同一 ID。

## 12. 缓存策略

MVP 可不使用 Redis。性能需要时按以下顺序增加：

1. 缓存热门词典查询，TTL 1-24 小时。
2. 缓存搜索联想，TTL 5-30 分钟。
3. 使用 Redis 做分布式限流。
4. 缓存公共 AI 例句。

用户翻译、作文、语音和纠错结果默认不进入共享缓存。

## 13. 测试策略

### 单元测试

- 注册策略按 `method` 分发，未知方式报错
- 复习间隔计算
- 输入校验和语言方向判断
- AI 结构化输出解析
- 权限判断
- 文件类型和大小校验
- 训练题目令牌签名、过期和用户绑定校验
- 单词答案规范化和判分
- 四阶段晋级、降级与间隔计算
- 中英文干扰项评分、去重和分层降级

### GORM Repository 测试

- 词典精确查询和联想
- 生词唯一约束
- 计划词条和队列位置唯一约束
- CET-4 快照批量插入和固定种子顺序
- 每日新增、活跃窗口和到期复习查询
- 用户数据隔离
- 答题 `submission_id` 唯一约束和幂等返回
- 同一道题多次提交保留多条答题记录
- GORM 错误到业务错误的转换

### API 集成测试

- 普通账号密码注册成功并自动登录
- 重复用户名（或填写的重复邮箱）返回 `CONFLICT`
- 未知 `method` 返回 `REGISTRATION_METHOD_UNSUPPORTED`
- 登录和退出
- 查词、收藏、复习闭环
- 翻译成功、超时和无效 AI 响应
- 上传非法文件
- 用户 A 无法读取用户 B 的记录
- 获取题目后直接下一题不产生答题记录
- 正确答案和错误答案都产生答题记录
- AI 评价失败时用户答案仍保留为 `failed`
- 重试 AI 评价不重复插入答案
- 单词答题记录、用户单词、计划项和计划 `completed_at` 保持事务一致
- 题目令牌阶段与数据库阶段不一致时拒绝提交
- 干扰项不足时不返回残缺题目
- 创建四级计划不批量创建全部 `user_words`
- 创建四级计划时排除词组，`source_snapshot_count` 等于过滤后的单词数
- 退出后重新进入继续原计划、原活跃单词和最新阶段
- 单词答错只更新答题历史和 `user_words`，不创建翻译错题记录
- recognition 使用学习步：新计划首节课内可把词从 recognition 推进到 discrimination，不会立刻返回 `NO_DUE_WORDS`
- 跳过计划词后该词不再被激活或选中，并计入 `completed_at` 判定
- 已在其他来源 mastered 的词激活时直接回填 `first_mastered_at`，不占用每日新增额度
- 既无 active 计划又无任何 `user_words` 时获取下一题返回 `NO_ACTIVE_PLAN`
- 没有 active 计划但有到期的散收生词时，获取下一题正常返回该词的题目，不返回 `NO_ACTIVE_PLAN`
- 并发/重复请求 `next` 触发激活时，当天激活数不超过 `daily_new_word_limit`、活跃数不超过 `active_word_limit`
- 翻译/作文答案评价超时卡在 `pending` 时，可经 `retry-evaluation` 重试且不新增答案行

### 前端测试

- 关键表单校验
- 登录状态和路由守卫
- 录音授权失败提示
- AI 请求加载、失败和重试状态
- 训练提交按钮防重复点击并复用同一 `submission_id`
- 点击下一题时不发送答案提交请求
- 输入框非空时点击下一题会出现放弃确认，确认后仍不落库

### 端到端测试

至少覆盖：

1. 注册 -> 创建四级计划 -> 激活10词 -> 完成部分题目 -> 退出 -> 重新进入继续。
2. 登录 -> 翻译 -> 历史详情。
3. 上传音频 -> 识别 -> 修改文本 -> 翻译。
4. 纠错 -> 查看错误列表 -> 历史记录。
5. 获取单词题 -> 跳过 -> 确认无记录 -> 提交下一题答案 -> 查看答题记录。
6. 提交翻译答案 -> AI 失败 -> 答案仍存在 -> 重试评价成功。

## 14. 配置建议

`.env.example` 至少包含：

```dotenv
APP_ENV=development
HTTP_PORT=8080
DB_ADDR=127.0.0.1:3306
DB_NAME=lingua
DB_USER=
DB_PASSWORD=
JWT_ACCESS_SECRET=
QUESTION_TOKEN_SECRET=

# Redis 缓存与限流，MVP 可暂不启用
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

密钥不得提交到 Git。

## 15. 部署建议

开发环境：

- Vue Vite 开发服务器
- Go API
- 本地或 Docker 数据库
- 本地文件目录

演示或生产环境：

```text
Nginx
├── /        -> Vue 静态文件
└── /api/    -> Go API

Go API
├── Database
├── Upload Storage
└── External AI / ASR
```

Docker Compose 可包含 `nginx`、`api`、`database` 和可选 `redis`。已有远程词典数据库时不要重复启动数据库容器。
