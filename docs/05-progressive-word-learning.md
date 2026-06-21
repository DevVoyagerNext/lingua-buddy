# Lingua Buddy 渐进式单词学习设计

## 1. 目标

单词学习采用固定、可解释的渐进路径：

```text
英文选中文
  -> 中文选英文
  -> 根据中文默写英文
  -> 已掌握后的定期默写复习
```

用户不需要手动选择题型。后端根据这个用户对目标单词的当前 `learning_stage` 自动决定下一道题。这样可以避免用户一直选择简单题型，也避免刚收藏的陌生单词直接进入默写。

本设计替代旧的“`mastery_level` + 用户手动选择题型”方案。由于业务表尚未创建，不需要执行历史数据迁移，直接按新表结构实施。

## 2. 四个学习阶段

| 数据库值 | 页面名称 | 默认题型 | 学习目标 |
| --- | --- | --- | --- |
| `recognition` | 初识/不认识 | `word_to_meaning_choice` | 看到英文能够认出中文意思 |
| `discrimination` | 辨认/模糊 | `meaning_to_word_choice` | 看到中文能够从相近单词中认出英文 |
| `spelling` | 默写/认识 | `meaning_to_word_spelling` | 不依赖选项拼写英文 |
| `mastered` | 已掌握 | `meaning_to_word_spelling` | 通过间隔默写维持记忆 |

接口可以派生一个兼容页面展示的 `mastery_label`：

| `learning_stage` | `mastery_label` |
| --- | --- |
| `recognition` | unknown |
| `discrimination` | fuzzy |
| `spelling` | known |
| `mastered` | mastered |

`mastery_label` 不单独存入数据库，避免两个状态字段出现矛盾。

## 3. 初始阶段

用户收藏单词时可以描述自己的熟悉程度，后端映射为初始阶段：

| 用户选择 | 初始阶段 | 首次训练时间 |
| --- | --- | --- |
| 不认识 | `recognition` | 立即 |
| 有点印象/模糊 | `discrimination` | 1 天后 |
| 认识但不会写 | `spelling` | 3 天后 |
| 已经熟练 | `mastered` | 30 天后 |

系统默认选择“不认识”。用户只在首次收藏时选择初始阶段；之后的阶段由训练和复习结果自动维护。

## 4. 阶段推进和降级

### 4.0 学习步与复习间隔

间隔分为两类，必须区分，否则新计划的第一节课会被长间隔卡死：

- **学习步（learning step）**：用于同一次会话内的短期重现。它不是真正的间隔记忆，只是把单词排到当前活跃窗口队列末尾、约 1 分钟后再次出现，让用户在同一坐姿内能完成识别阶段的多次验证。默认 `learning_step=1 分钟`，可配置。
- **复习间隔（review interval）**：用于跨会话、跨天的间隔记忆，例如 1 天、3 天、7 天、30 天。

规则：`recognition` 阶段内部（首次答对保持 recognition、以及 recognition 答错重练）使用**学习步**；一旦离开 recognition（进入 discrimination 及以后），全部使用真正的**复习间隔**。

这样，新计划激活 10 个词后：用户先对 10 个词各答 1 遍（每个词进入学习步，约 1 分钟后重现）；答完一轮通常已超过 1 分钟，于是这些词陆续再次到期，第 2 次答对即可进入 discrimination（1 天后）。用户无需空等 10 分钟，也不会在第一节课就遇到 `NO_DUE_WORDS`。

### 4.1 阶段规则

| 当前阶段 | 本次结果 | 数据更新 | 下次时间 |
| --- | --- | --- | --- |
| recognition | 第 1 次连续答对 | 保持 recognition，`stage_correct_streak=1` | 学习步（约 1 分钟后，同会话重现） |
| recognition | 第 2 次连续答对 | 进入 discrimination，连续次数清零 | 1 天后 |
| recognition | 答错 | 保持 recognition，连续次数清零 | 学习步（约 1 分钟后，同会话重现） |
| discrimination | 第 1 次连续答对 | 保持 discrimination，`stage_correct_streak=1` | 1 天后 |
| discrimination | 第 2 次连续答对 | 进入 spelling，连续次数清零 | 3 天后 |
| discrimination | 答错 | 降为 recognition，连续次数清零 | 10 分钟后 |
| spelling | 第 1 次连续答对 | 保持 spelling，连续次数为 1 | 3 天后 |
| spelling | 第 2 次连续答对 | 保持 spelling，连续次数为 2 | 7 天后 |
| spelling | 第 3 次连续答对 | 进入 mastered，连续次数清零 | 30 天后 |
| spelling | 答错或使用提示后答对 | 降为 discrimination，连续次数清零 | 1 天后 |
| mastered | 无提示默写答对 | 保持 mastered | 30 天后 |
| mastered | 答错或使用提示 | 降为 discrimination，连续次数清零 | 1 天后 |

连续正确必须来自不同的已提交答案。跳过题目不改变阶段、连续正确次数和复习时间。

### 4.2 为什么选择两次选择题、三次默写

- 选择题用于建立初步识别，两次连续正确足以进入下一层验证。
- 默写比选择题更能证明主动回忆，因此要求三次连续正确，并通过 3 天、7 天的间隔验证。
- 已掌握不代表永久结束，仍按 30 天间隔进行默写复习。

这些阈值是首版固定规则，后续可以通过配置调整，但不能在同一版本中对不同用户随机改变。

## 5. 三种题型

### 5.1 英文选中文

示例：

```text
accomplish

A. 完成，实现
B. 拒绝，谢绝
C. 观察，注意
D. 原谅，宽恕
```

正确释义来自目标词的 `ecdict.translation`。另外三个中文释义由干扰项模块生成。

### 5.2 中文选英文

示例：

```text
完成，实现

A. accompany
B. accomplish
C. accommodate
D. accumulate
```

正确单词是目标词。另外三个英文单词由干扰项模块生成，优先选择拼写相近、难度接近且中文意思不同的词。

### 5.3 中文默写英文

示例：

```text
完成，实现

请输入英文：__________
```

判分前统一执行：去除首尾空格、转换为小写。首版不接受近义词替代，因为题目明确关联一个目标词。

可选提示包括首字母、字母数量或显示一个字母。使用提示后即使答案正确，也不能推进到下一阶段。`used_hint` 由前端上报，后端首版不强校验，属于已知完整性边界，详见[词汇学习计划与断点续学设计](06-word-learning-plan-and-resume.md)第 13.1 节。

## 5.4 规范中文释义 `canonicalGloss`（出题、判分、去重共用）

`ecdict.translation` 不是一个干净的短释义，而是按字面量 `\n` 分隔的**多行多义串**（如 `n. 完成；实现\nvt. 达到\n...`），实测 768,739 条均为此格式。三个动作都要用到“这个词的中文意思”，如果各用各的处理方式，会直接导致选择题判分对不上：

- **出题展示**：`word_to_meaning_choice` 的正确选项、`meaning_to_word_choice` 与 `meaning_to_word_spelling` 的中文题干。
- **判分比对**：提交时服务端“从 `ecdict` 重新读取正确答案”，必须得到与出题时**完全一致**的字符串才能比对成功。
- **干扰项去重**：判断候选释义是否与正确释义相同或高度重叠。

因此后端定义**唯一的确定性函数** `canonicalGloss(translation) -> string`，三处共用同一实现：

1. 按字面量 `\n` 拆分为多行，丢弃空行。
2. 取第一条非空释义。
3. 去掉开头词性前缀（`n.`、`v.`、`vt.`、`vi.`、`adj.`、`adv.`、`prep.`、`conj.`、`pron.`、`num.`、`art.`、`int.` 等）。
4. 按 `；;，,` 截取前 1–2 个义项片段，去首尾空白，归一化标点。
5. 输出一个稳定的简短中文 gloss。

约定：

- `word_to_meaning_choice` 判分：提交时服务端重算 `canonicalGloss(target)`，与用户提交的所选选项文本**规范化后做等值比较**；正确选项不写入客户端可读令牌字段。
- 干扰项的“释义不同”以 `canonicalGloss` 结果比较，避免两个选项实际同义；正确释义为空（极少数仅英文释义词条）时该词不进入 `word_to_meaning_choice`，改用其他可出题阶段或跳过。
- `canonicalGloss` 的版本并入 `generator_version`（如 `word-v1`），算法升级后旧题仍可凭 `training_answer_records.options` 快照还原。

`meaning_to_word_choice` / `meaning_to_word_spelling` 的判分对象是**英文单词本身**（与 `ecdict.word` 等值比较），不依赖 `canonicalGloss`；`canonicalGloss` 只用于这两类题的**中文题干展示**。

## 6. 独立干扰项模块

### 6.1 模块职责

后端新增独立模块：

```text
backend/internal/worddistractor/
├── service.go
├── repository.go
├── scorer.go
├── normalizer.go
├── model.go
└── errors.go
```

它只负责：

1. 接收一个目标词、题型和需要的干扰项数量。
2. 从内部 `ecdict` 查询候选词。
3. 对候选词进行评分、去重和过滤。
4. 返回三个干扰项以及生成策略版本。

它不负责：

- 选择用户下一步学习哪个单词。
- 决定学习阶段。
- 组装完整题目和正确答案。
- 保存训练题目。
- 判分、更新生词状态或维护错题本。

因此，干扰项模块可以独立进行单元测试，并被两种选择题共同复用。

### 6.2 接口

```go
type Service interface {
    FindMeaningDistractors(
        ctx context.Context,
        target DictionaryEntry,
        count int,
    ) ([]MeaningDistractor, GenerationMeta, error)

    FindWordDistractors(
        ctx context.Context,
        target DictionaryEntry,
        count int,
    ) ([]WordDistractor, GenerationMeta, error)
}
```

问题生成模块负责把“正确选项 + 三个干扰项”合并、打乱并签发 `question_token`。

### 6.3 中文释义干扰项

用于 `word_to_meaning_choice`，选择顺序：

1. 候选词必须有有效中文释义，不能是目标词或目标词形变化。
2. 优先与目标词共享考试标签，例如都属于 CET-4。
3. 优先处于相近词频区间，例如 BNC/FRQ 排名在同一数量级。
4. 尝试从释义前缀提取词性，优先选择相同词性。
5. 排除与正确释义标准化后相同或高度重叠的释义；相同性以第 5.4 节的 `canonicalGloss` 结果比较。
6. 中文选项去重，避免两个选项实际上表达同一个意思（同样以 `canonicalGloss` 比较）。

中文释义选择不要求候选英文拼写相近，因为前端只展示中文选项。这里更重要的是词性和难度接近。

### 6.4 英文单词干扰项

用于 `meaning_to_word_choice`，候选评分因素：

| 因素 | 说明 | 建议权重 |
| --- | --- | ---: |
| 编辑距离 | Levenshtein 距离 1-3 优先 | 35 |
| 长度差 | 长度相差 0-2 优先 | 15 |
| 前缀相似 | 相同首字母或共享 2-4 个前缀字符 | 15 |
| 后缀相似 | 相同常见后缀，例如 -tion、-ment | 10 |
| 考试标签 | CET-4、CET-6 等标签重合 | 10 |
| 词频接近 | BNC/FRQ 排名在相近区间 | 10 |
| 词性相同 | 可从释义前缀推断时使用 | 5 |

必须排除：

- 目标词本身及大小写变体。
- 目标词的复数、过去式等直接词形。
- 中文核心释义与目标词相同或近乎相同的词。
- 空释义、异常长词条和包含无关符号的候选。
- 同一组选项中重复或仅大小写不同的单词。

### 6.5 分层降级

严格条件不一定总能找到三个候选，所以按层降级：

1. **strict**：拼写、标签、词频和词性都尽量接近。
2. **balanced**：放宽编辑距离和词频范围，仍要求有有效且不同的释义。
3. **fallback**：从相同考试标签或常用词中选择不同释义的候选。

无论降级到哪一层，都必须满足“不是正确答案、选项不重复、释义不相同”的硬性条件。找不到三个安全选项时返回错误，不生成残缺或明显错误的题目。

### 6.6 是否建立干扰项数据库表

首版不建立。

理由：

- 干扰项可以从已有 `ecdict` 动态计算。
- 题目只有用户提交后才需要保留，当时的四个选项已经保存在 `training_answer_records.options`。
- 单独保存所有“单词 -> 干扰项”会产生大量可重建数据和失效维护问题。

可选优化：对热门目标词的候选结果做进程内缓存；以后性能不足时再使用 Redis。缓存不是业务真相，失效后可以重新计算。

## 7. 数据库设计

### 7.1 `user_words`

旧设计删除 `mastery_level` 和 `correct_streak`，改为以下字段：

| 字段 | 说明 |
| --- | --- |
| `id` | 用户单词 ID |
| `user_id` | 用户 ID |
| `word` | 标准化后的目标词 |
| `learning_stage` | recognition/discrimination/spelling/mastered |
| `stage_correct_streak` | 当前阶段连续正确次数 |
| `next_review_at` | 下次应训练或复习时间 |
| `last_trained_at` | 最近一次提交单词训练答案的时间，可空 |
| `stage_changed_at` | 最近一次阶段变化时间 |
| `created_at` | 收藏时间 |
| `updated_at` | 更新时间 |

唯一索引：`user_id + word`。

只保存当前阶段，不保存重复的 `mastery_level`。完整变化过程由 `training_answer_records` 中每次答案的阶段前后值还原。

### 7.2 `training_answer_records`

保留通用答题历史表，并为单词训练增加：

| 字段 | 说明 |
| --- | --- |
| `learning_stage_before` | 作答前阶段，非单词训练为空 |
| `learning_stage_after` | 作答后阶段，非单词训练为空 |
| `generator_version` | 题目/干扰项算法版本，例如 word-v1 |

单词题型枚举统一为：

```text
word_to_meaning_choice
meaning_to_word_choice
meaning_to_word_spelling
```

`options` 保存用户当时看到的完整四个选项快照。即使后续干扰项算法升级，也能还原原题。

### 7.3 学习计划关联

当单词来自四级、六级等计划时，答题记录保存：

- `user_word_id`
- `word_learning_plan_id`
- `word_learning_plan_item_id`

计划本身由 `word_learning_plans` 保存来源、固定种子和每日/活跃上限；完整词条队列由 `word_learning_plan_items` 保存。详细设计见[词汇学习计划与断点续学设计](06-word-learning-plan-and-resume.md)。

### 7.4 单词错误记录

单词不建立 `user_word_wrong_questions`：

- 每次错误保存在 `training_answer_records`。
- 当前需要补学的状态由 `user_words.learning_stage`、`next_review_at` 和 `last_answer_correct` 表达。
- 当前薄弱词可以直接查询 recognition/discrimination 或最近答案错误的 `user_words`。
- 历史错词从答案记录中过滤 `training_type=word AND is_correct=false`。

通用错题表改为只保存翻译训练中用户确认的错题，表名为 `user_translation_wrong_questions`。

## 8. 获取下一题流程

`GET /api/v1/word-learning/next` 不接收用户指定的题型，服务“计划词”和“散收生词”两类来源，流程如下：

1. 收集当前用户全部到期的活跃 `user_words`（`next_review_at <= now`，未被计划标记跳过），不区分来源；散收生词没有计划项也参与候选，因此**没有 active 计划也能取到到期复习题**。
2. 有到期词时，先按逾期时间排序，再按阶段优先级 `recognition -> discrimination -> spelling -> mastered` 排序，选出一道。
3. 没有到期词且用户存在 active 计划时，按活跃窗口规则在加锁事务内激活新计划项（见[计划与断点续学设计](06-word-learning-plan-and-resume.md)第 8 节），再从新激活词中选一道。
4. 既无到期词、也无可激活的计划词时：有未到期的词返回 `NO_DUE_WORDS` 与下一次到期时间；完全没有任何 `user_words` 且无 active 计划时返回 `NO_ACTIVE_PLAN`。
5. 读取选中单词的 `ecdict` 词条。
6. 根据 `learning_stage` 决定题型。
7. 选择题调用 `worddistractor.Service` 获取三个干扰项。
8. 合并正确选项、随机打乱，生成题目快照；`word_to_meaning_choice` 的正确选项与干扰项中文文本统一由第 5.4 节 `canonicalGloss` 产出，保证提交时可重算比对。
9. 签发包含用户、`user_word_id`、题型、选项、算法版本和过期时间的 `question_token`；计划词附带 `word_learning_plan_id` 与 `word_learning_plan_item_id`，散收生词这两个字段为空。
10. 返回前端，不写数据库。

如果用户选择“自由练习某个单词”，可以指定 `user_word_id`，但题型仍由该单词当前阶段决定。

## 9. 提交答案事务

`POST /api/v1/word-learning/answer`：

1. 校验登录状态、`submission_id`、答案和 `question_token`。
2. 加锁读取 `user_words`，确认令牌中的阶段仍与数据库一致；阶段已经变化时返回题目过期。
3. 从 `ecdict` 重新读取正确答案并判分，不能信任客户端：`word_to_meaning_choice` 重算 `canonicalGloss(target)` 与所选选项比较，`meaning_to_word_choice` 比较所选英文是否为目标词，`meaning_to_word_spelling` 对去空白小写后的英文输入与目标词等值比较。
4. 根据阶段规则计算 `learning_stage_after`、连续正确次数和 `next_review_at`。
5. 在同一事务中：
   - 插入 `training_answer_records`；
   - 更新 `user_words` 阶段、累计正确/错误次数；
   - 首次 mastered 时更新关联计划项 `first_mastered_at`；
   - 必要时写入整个计划的 `completed_at`。
6. 提交事务后返回正确答案、阶段变化和下次时间。

相同 `submission_id` 重试时返回原结果，不重复推进阶段。

## 10. 模块边界

```text
WordTrainingService
├── WordSelector          选择本次训练哪个用户单词
├── StagePolicy           阶段 -> 题型、晋级、降级、间隔
├── QuestionBuilder       组装题目和签名令牌
├── worddistractor.Service 只生成干扰项
├── AnswerJudge           规范化并判分
└── Repository            保存答案、单词进度和计划进度
```

边界原则：

- `worddistractor` 不读取 `user_words`，只处理目标词和 `ecdict` 候选。
- `StagePolicy` 是纯规则模块，不访问数据库。
- `QuestionBuilder` 不保存题目。
- Repository 不自行决定晋级规则，只执行 Service 计算后的事务更新。

## 11. 测试要求

### 干扰项模块

- 不返回目标词和其词形变化。
- 三个英文干扰项唯一且释义不同。
- 三个中文释义唯一且不等于正确释义。
- 严格候选不足时正确降级。
- 所有候选不足时明确失败，而不是返回少于四个选项。
- 固定随机种子时结果可复现，便于测试。

### 阶段规则

- recognition 连续两次正确进入 discrimination。
- discrimination 连续两次正确进入 spelling。
- spelling 连续三次正确进入 mastered。
- 任一答错按照规则降级并清零。
- 跳过不改变任何数据库字段。
- 幂等重试不重复增加连续正确次数。

### 事务

- 答题历史写入失败时不更新用户阶段。
- 计划项或用户单词更新失败时整个单词答题事务回滚。
- 同一单词并发提交时只有持有最新阶段锁的请求成功。
- 题目令牌阶段与数据库阶段不一致时拒绝提交。
