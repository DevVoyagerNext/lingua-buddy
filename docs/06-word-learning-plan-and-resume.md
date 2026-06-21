# Lingua Buddy 词汇学习计划与断点续学设计

## 1. 设计结论

用户选择“四级词汇”后，不采用以下两种极端方式：

- 不在每次请求时临时从 `ecdict` 随机取一个词，因为顺序无法稳定，难以断点续学，也可能重复或遗漏。
- 不把全部四级词一次发送给前端，也不一次把全部词创建成 `user_words`，因为数据量大且大多数词尚未开始学习。

采用四层结构：

```text
四级词库快照
  -> 用户学习计划和固定队列
  -> 小规模活跃学习窗口
  -> 每次只返回一道题
```

当前词库实测有 3,849 个 CET-4 标签词条。创建计划时先**过滤掉词组**（`word` 含空格或连字符的多词条目），只保留单词，再固定剩余单词的队列顺序；真正开始学习时，才把少量单词激活为用户单词进度。

过滤词组的原因是 `ecdict.word` 同时包含“单词或词组”，而 spelling 阶段要求用户根据中文默写整串、干扰项又用编辑距离选拼写相近的词，这两者对词组都不成立。因此 `source_snapshot_count` 是过滤后的单词数，可能少于 3,849；具体数量以创建计划时的实际查询为准。如果以后需要支持词组，再单独设计词组题型，不在首版混入单词队列。

## 2. 批量与单题的关系

### 2.1 数据库层面是一批

创建计划时一次生成完整的计划词条队列，例如 3,849 条 `word_learning_plan_items`。这些记录只表示“这个词属于计划以及它排在什么位置”，不代表用户已经开始学习。

为了避免纯随机把冷僻词排在最前面，默认使用 `frequency_shuffled`：

1. 根据 `frq`、`bnc` 把常用词排在较前范围。
2. 每 50 个词组成一个词频桶。
3. 使用计划的 `shuffle_seed` 在桶内执行 Fisher-Yates 打乱。
4. 把最终顺序保存为 `queue_position`。

因此队列既有一定难度顺序，又不会每个用户看到完全机械相同的排列。重新登录不会重新洗牌。

### 2.2 学习层面是活跃窗口

默认配置：

- `daily_new_word_limit=10`：每天最多激活10个新词。
- `active_word_limit=20`：同时处于首次掌握前的单词最多20个。

后端只把进入活跃窗口的计划项关联到 `user_words`。尚未激活的几千个词只存在计划队列表中，不占用用户进度表。

### 2.3 接口层面每次一道题

`GET /api/v1/word-learning/next` 每次只返回一道题。

前端答完后再次请求下一道。前端可以预取一题，但不能把整批答案或标准答案下载到客户端。

## 3. 单词不是整组一起晋级

一个活跃窗口中的每个单词独立推进：

```text
recognition 英文选中文
  -> discrimination 中文选英文
  -> spelling 中文默写英文
  -> mastered 已掌握维护复习
```

系统不会要求“这一批20个词全部完成英文选中文后，整批一起进入中文选英文”。

例如：

| 单词 | 当前阶段 | 下次到期 |
| --- | --- | --- |
| accomplish | discrimination | 今天 10:30 |
| benefit | recognition | 现在 |
| maintain | spelling | 明天 |
| adequate | mastered | 20 天后 |

下一题只选择当前已经到期、优先级最高的词。因此同一个学习页面中以后可能出现不同阶段的题型。

这样做可以让较熟悉的词更快进入默写，困难词继续停留在识别阶段，不互相拖累。

## 4. 下一题选择顺序

用户打开学习页时，后端按以下顺序处理：

1. 先收集该用户全部到期的活跃 `user_words`（`next_review_at <= now`，未被计划标记跳过），既包含 active 计划激活的词，也包含从查词页散收、没有计划项的生词。这一步不要求存在 active 计划。
2. 上一步既含已激活的学习中单词，也含已经首次掌握但当前到期的维护复习单词，统一参与候选。
3. 如果有到期词，跳到第 7 步选题。
4. 如果没有到期词且用户有 active 计划，检查活跃窗口是否不足20个、今日新增是否少于10个。
5. 满足新增条件时，在对计划行加锁的事务内从最小 `queue_position` 开始激活下一批**未激活且未跳过**（`activated_at IS NULL AND skipped_at IS NULL`）的计划词（见第 8 节）。
6. 既无到期词又无法激活新词时：有未到期的词返回 `NO_DUE_WORDS` 与下次到期时间；完全没有任何 `user_words` 且无 active 计划时返回 `NO_ACTIVE_PLAN`。
7. 根据选中词的 `learning_stage` 生成一道题。
8. 返回题目，不保存答题记录。

到期词排序：

```text
逾期时间更长优先
-> recognition
-> discrimination
-> spelling
-> mastered
-> 最近训练时间更早优先
```

如果没有到期词且今日新增额度已经用完，接口返回 `NO_DUE_WORDS` 和下一次到期时间，不为了让页面一直有题而破坏间隔规则。

## 5. 退出与继续

### 5.1 已经提交的答案

提交后会保存：

- 本次题目和选项快照。
- 用户答案和判分。
- 作答前、作答后的学习阶段。
- 更新后的下次复习时间。

用户退出后重新打开，系统从数据库读取最新阶段继续。

### 5.2 没有提交的题目

用户只是看到题目、输入后放弃、刷新或关闭页面：

- 不创建答题记录。
- 不更新 `user_words`。
- 不移动计划队列。
- 不增加错误次数。

重新打开时，这个单词仍处于原阶段。如果已经到期，后端可能再次选中它，但选择题干扰项可以重新生成。

因此“继续学习”保证的是同一批活跃单词和相同学习阶段，不保证恢复一个从未提交过的临时题目页面。

### 5.3 跳过单词

用户在学习页可以对“我已经会了/不想学这个词”执行跳过：

`POST /api/v1/word-learning/plan-items/{id}/skip`

处理规则：

1. 校验该计划项属于当前用户的 active 计划。
2. 在事务中设置 `skipped_at = now`。
3. 跳过后该计划项不再被激活，也不再被到期选择命中（见第 4 节第 2、5 步的 `skipped_at IS NULL` 过滤）。
4. 如果该词已经激活并占用活跃窗口，跳过后即释放槽位，允许激活下一批新词。`user_words` 不删除，因为同一个词可能仍属于其他计划或生词本。
5. 跳过计入计划完成度：`completed_at` 的判定是“全部计划项都已 `first_mastered_at` 或 `skipped_at`”。

跳过是针对**计划项**而非全局单词的操作，因此跳过四级计划中的某词不会影响六级计划或生词本中的同一个词。可选提供 `DELETE …/skip`（撤销跳过）把 `skipped_at` 清空，让词重新回到等待队列；首版可不实现撤销。

注意区分“跳过”和“点击下一题”：点击下一题只是丢弃当前未提交的 `question_token`，不写任何数据库记录，该词下次仍可能被选中；跳过则永久把该计划项移出队列。

## 6. 数据库表

### 6.1 `word_learning_plans`

保存用户选择的词汇学习目标。

| 字段 | 说明 |
| --- | --- |
| `id` | 计划 ID |
| `user_id` | 用户 ID |
| `name` | 计划名称，例如“四级词汇” |
| `source_type` | `ecdict_tag` 或 `manual` |
| `source_value` | `cet4`、`cet6` 等 |
| `ordering_mode` | `frequency_shuffled` 等 |
| `shuffle_seed` | 固定随机种子 |
| `source_snapshot_count` | 创建计划时匹配的总词数 |
| `daily_new_word_limit` | 每日新增上限，默认10 |
| `active_word_limit` | 活跃学习词上限，默认20 |
| `status` | active/paused/archived；是否完成由 `completed_at` 表示 |
| `started_at` | 开始时间 |
| `completed_at` | 全部词首次掌握时间，可空 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

MVP 同一用户最多一个 active 计划，可以保留多个 paused 或 archived 计划。达到 `completed_at` 的计划仍可保持 active 进行长期维护复习；切换计划时必须显式暂停或归档当前计划。

### 6.2 `word_learning_plan_items`

保存计划的固定词条队列和激活状态。

| 字段 | 说明 |
| --- | --- |
| `id` | 计划词条 ID |
| `plan_id` | 所属计划 |
| `ecdict_entry_id` | 对应 `ecdict.id` |
| `word` | 单词快照 |
| `queue_position` | 固定队列位置 |
| `user_word_id` | 激活后关联 `user_words.id`，未激活为空 |
| `activated_at` | 首次进入活跃窗口的时间，可空 |
| `first_mastered_at` | 第一次进入 mastered 的时间，可空 |
| `skipped_at` | 用户永久跳过该词的时间，可空 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

约束：

- 唯一索引 `plan_id + ecdict_entry_id`。
- 唯一索引 `plan_id + queue_position`。
- `activated_at IS NULL` 表示仍在等待队列。
- 已激活且 `first_mastered_at IS NULL` 表示正在首次学习。
- `first_mastered_at IS NOT NULL` 表示至少掌握过一次，但仍可能在未来维护复习中降级。

不额外保存 queued/learning/mastered 状态字符串，避免状态与时间字段矛盾。

### 6.3 `user_words`

保存一个用户对一个单词的全局当前学习状态。一个词即使同时属于四级和六级计划，也只保留一条 `user_words`。

| 字段 | 说明 |
| --- | --- |
| `id` | 用户单词 ID |
| `user_id` | 用户 ID |
| `ecdict_entry_id` | 词典 ID，可空 |
| `word` | 标准化单词快照 |
| `learning_stage` | recognition/discrimination/spelling/mastered |
| `stage_correct_streak` | 当前阶段连续正确次数 |
| `next_review_at` | 下次训练时间 |
| `last_trained_at` | 最近提交时间 |
| `stage_changed_at` | 最近阶段变化时间 |
| `first_mastered_at` | 第一次达到 mastered 的时间，可空 |
| `total_correct_count` | 累计正确次数 |
| `total_wrong_count` | 累计错误次数 |
| `last_answer_correct` | 最近一次答案是否正确，可空 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

唯一索引：`user_id + word`。

`user_words` 不是四级词表副本，只保存已经收藏或已经激活学习的词。

### 6.4 `training_answer_records`

保存每一次已提交答案，单词答题增加以下关联：

| 字段 | 说明 |
| --- | --- |
| `user_word_id` | 对应用户单词 |
| `word_learning_plan_id` | 本次来自哪个学习计划，可空 |
| `word_learning_plan_item_id` | 对应计划词条，可空 |
| `learning_stage_before` | 作答前阶段 |
| `learning_stage_after` | 作答后阶段 |
| `is_correct` | 是否正确 |
| `question_text`、`options` | 题目快照 |
| `user_answer`、`reference_answer` | 用户答案和正确答案 |

每次提交新增一条记录。答错记录不能被后来的正确答案覆盖。

### 6.5 `user_translation_wrong_questions`

只保存用户确认仍需重练的翻译训练错题。

单词不再写入这张表。原因是：

- 单词的每次错误已经永久保存在 `training_answer_records`。
- 单词当前是否需要补学由 `user_words.learning_stage` 和 `next_review_at` 表达。
- 再复制一条“当前单词错题”会与阶段状态重复，容易不一致。

如果页面需要“我的错词”：

- 当前薄弱词：查询 `user_words` 中 recognition/discrimination 或最近答案错误的词。
- 历史错词：查询 `training_answer_records WHERE training_type='word' AND is_correct=false`。

## 7. 创建四级计划的数据库流程

`POST /api/v1/word-learning/plans` 请求：

```json
{
  "name": "四级词汇",
  "source_type": "ecdict_tag",
  "source_value": "cet4",
  "daily_new_word_limit": 10,
  "active_word_limit": 20
}
```

处理流程：

1. 验证用户没有其他 active 计划。
2. 新增 `word_learning_plans`。
3. 从内部 `ecdict` 查询带独立 `cet4` 标签的词条。`tag` 是空格分隔串（如 `cet4 cet6 ky`），必须按**词边界**匹配单个标签，不能用裸 `LIKE '%cet4%'`（避免误伤，也便于校验）：可对补齐空格后的字段判断，例如 `CONCAT(' ', tag, ' ') LIKE '% cet4 %'`，或 `FIND_IN_SET('cet4', REPLACE(tag,' ',','))`。`tag` 无索引，这一步是 77 万行全表扫描，但**建计划是一次性低频操作**，耗时可接受；不要把这种 tag 扫描放进实时出题路径（实时出题的候选拉取见[技术设计](02-technical-design.md)第 6.8 节，走 `frq`/`bnc` 索引）。
4. **过滤掉词组**：排除 `word` 含空格或连字符的多词条目，只保留单词。
5. 按词频分桶，并使用 `shuffle_seed` 在桶内打乱。
6. 按500条一批批量插入 `word_learning_plan_items`。
7. 写入 `source_snapshot_count`，等于过滤后入队的单词数（少于或等于 3,849，以实际查询为准）。
8. 提交事务并返回计划信息。

创建计划时不为入队单词新增 `user_words`。

## 8. 激活新词的数据库流程

每日新增和活跃上限衡量的是**真正的首次学习负担**，因此“已经在别处掌握的词”不占用额度。当没有足够到期词时，**在对该 `word_learning_plans` 行加锁（`SELECT ... FOR UPDATE`）的事务中**执行激活，保证并发的取题请求串行结算名额：

1. 统计计划中：
   - 今日因进入首次学习而激活的数量（`activated_at` 为今日且激活时 `first_mastered_at IS NULL`）；
   - 已激活但尚未首次掌握的活跃数量。
   - “今日”以 `Asia/Shanghai` 自然日为界，避免跨午夜把额度算错。
2. 计算还能新增的首次学习名额：

```text
min(
  daily_new_word_limit - 今日已新增,
  active_word_limit - 当前活跃数
)
```

3. 按 `queue_position` 顺序扫描 `activated_at IS NULL AND skipped_at IS NULL` 的计划项，对每个候选词执行 user+word upsert：
   - 该词已有全局 `user_words` 且已是 mastered：直接复用，回填计划项的 `user_word_id`、`activated_at` 和 `first_mastered_at`，**不占用首次学习名额**，继续扫描下一个候选。
   - 该词已有 `user_words` 但未掌握：复用，占用一个首次学习名额。
   - 该词没有 `user_words`：创建 recognition 阶段的 `user_words`，占用一个首次学习名额。
4. 名额用完即停止激活新的首次学习词（已掌握词的直接回填不受名额限制）。
5. 回填计划项的 `user_word_id` 和 `activated_at`。
6. 提交事务。

这样，从别的来源已经 mastered 的词不会挤占当天的新词额度，也不要求用户重复从第一阶段学习；真正的新词仍按 `daily_new_word_limit` 和 `active_word_limit` 控制节奏。

**并发与幂等**：`GET /word-learning/next` 虽是读取语义，但“激活新词”这一步会写库（改计划项、建 `user_words`、消耗当日额度）。前端双击、预取或网络重试可能在极短时间内触发多次 `next`。因此激活必须满足：

- 对计划行加锁，使并发请求按顺序进入激活逻辑，各自基于最新的“今日已新增数”和“当前活跃数”重新计算名额。
- 名额计算后才写入，避免两个请求都读到旧计数、各激活一批，导致当天激活数突破 `daily_new_word_limit` 或活跃数突破 `active_word_limit`。
- 激活对“计划项是否已激活”判等幂等：已 `activated_at IS NOT NULL` 的计划项不再重复激活，也不重复 upsert `user_words`。
- 仅返回题目、未激活任何新词的 `next` 请求不写库，天然无并发问题。

**预取与额度**：前端预取下一题会真实触发激活，因而即使用户从未作答，被激活的词也已占用当天 `daily_new_word_limit` 名额并写入 `user_words`（保持 recognition、到期时间为即时）。这不影响数据正确性——这些词只是提前进入活跃窗口，后续仍会被到期选中学习；但要明确这是预期行为，前端不应无节制预取（建议至多预取 1 题）。若希望“预取绝不消耗额度”，可让预取只读已到期词、遇到需要激活时不激活而返回 `NO_DUE_WORDS`，由用户的显式取题动作再激活；首版采用前者（预取可激活）。

## 9. 提交答案的数据库事务

单词答案提交时：

1. 锁定 `user_words`；计划词再锁定对应 `word_learning_plan_items`，散收生词无计划项则只锁 `user_words`。
2. 校验题目令牌中的（计划词的）计划、词条和当前阶段仍有效；散收生词只校验 `user_word_id` 与阶段。
3. 重新从 `ecdict` 获取正确答案并判分。
4. StagePolicy 计算阶段变化和下次时间。
5. 在同一事务中：
   - 插入 `training_answer_records`；
   - 更新 `user_words` 阶段、次数、累计正确/错误和时间；
   - 第一次达到 mastered 时，更新所有关联该 `user_word_id` 且 `first_mastered_at IS NULL` 的计划项；
   - 如果计划全部词条都已经首次掌握或跳过，写入 `completed_at`，但不强制改变 active 状态，以便继续维护复习。
6. 提交后返回结果。

答错不会创建单独的单词错题行；错误答案记录和阶段降级已经在同一事务中完成。

## 10. 典型断点续学示例

用户创建四级计划，系统激活10个词：

```text
计划总词数：3849
等待队列：3839
首次学习中：10
首次掌握：0
```

用户完成6道题后关闭页面：

- 6道已提交答案写入 `training_answer_records`。
- 对应单词的阶段和下次时间已更新。
- 另外4个已经激活但没提交的词保持原阶段。
- 未展示或未激活的3839个词仍在固定队列中。

第二天打开：

1. 先返回已到期的这10个活跃词。
2. 某些词可能仍是英文选中文，某些已经进入中文选英文。
3. 活跃词首次掌握后腾出窗口，系统再从3839个等待词中激活新词。
4. 队列位置、学习阶段和历史答案都不会丢失。

## 11. 不需要的表

首版不建立：

- `word_learning_sessions`：退出续学依靠计划、计划项和单词进度，不需要保存页面会话。
- `word_questions`：题目动态生成，提交后保存快照即可。
- `word_distractors`：干扰项动态生成，提交后保存在答案记录选项中。
- `user_word_wrong_questions`：与 `user_words` 当前阶段以及答题历史重复。

这样既能完整续学，又避免维护四套重复状态。

## 12. API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/word-learning/plans` | 创建四级、六级等学习计划 |
| GET | `/word-learning/plans` | 查询个人计划 |
| GET | `/word-learning/plans/{id}` | 查询总数、等待、学习中和首次掌握数量 |
| POST | `/word-learning/plans/{id}/activate` | 激活或切换当前计划 |
| POST | `/word-learning/plans/{id}/pause` | 暂停计划 |
| GET | `/word-learning/next` | 获取下一道到期题，必要时激活新词 |
| POST | `/word-learning/answer` | 提交答案并更新进度 |
| POST | `/word-learning/plan-items/{id}/skip` | 跳过该计划词，写入 `skipped_at` |
| GET | `/word-learning/words` | 查询计划内单词及当前阶段 |
| GET | `/word-learning/wrong-answers` | 查询历史单词错误答案 |

所有路径位于 `/api/v1` 下。

## 13. 边界与已知漏洞处理

结合前端预取、刷新、双击和用户自报等真实操作，明确以下边界：

### 13.1 `used_hint` 由客户端上报，服务端不强校验

提示（首字母、字母数、显示一个字母）在前端发生，后端无法独立判断用户是否真的用了提示。因此 `used_hint` 采信客户端上报：上报为 true 时，即使答案正确也不晋级（spelling/mastered 按降级或保持处理）。这是已知完整性边界——在学习场景中谎报“没用提示”只会损害自己的学习效果，首版不为此引入服务端验证。若以后需要，可改为“点击提示即由后端签发带 `used_hint=true` 的新题目令牌”，让提示状态进入受签名保护的字段。

### 13.2 题目令牌只编码阶段，不编码连续正确次数

`question_token` 包含 `learning_stage`，提交时只校验“令牌阶段 == 数据库当前阶段”，不校验 `stage_correct_streak`。含义：

- 同一个词在同一阶段的两次**真实**提交都会累加 `stage_correct_streak`，这是正常的连续答对。
- 前端预取（一次取两题）或网络重试可能对同一个词产生两次提交。网络重试必须复用同一个 `submission_id`，后端凭 `uk_training_submission(user_id, submission_id)` 幂等返回原结果，不重复推进。
- 防止“双击产生两个不同 `submission_id`”靠前端纪律（提交按钮防抖、复用 `submission_id`）。后端可选增强：对同一 `user_word_id` 在极短时间窗口（例如 1 秒）内、不同 `submission_id` 的重复提交按重复点击处理；首版可不实现，但前端测试必须覆盖“防重复点击复用同一 `submission_id`”。

阶段已变化时（例如另一个标签页已经把该词推进），旧令牌阶段与数据库不一致，提交返回 `QUESTION_TOKEN_EXPIRED`，前端需重新获取题目。

### 13.3 没有 active 计划时的取题

“没有 active 计划”不等于“没有可学的词”。用户可能从查词页散收过生词（`VOCAB-01`），这些词在 `user_words` 中、有阶段和到期时间，应当能被复习。因此 `GET /word-learning/next` 按如下分支处理：

- 有到期的散收生词或计划词：正常出题，不要求 active 计划。
- 有词但当前都未到期：返回 `NO_DUE_WORDS` 和下一次到期时间。
- 既无 active 计划，也没有任何 `user_words`（真正“无事可做”）：返回 `NO_ACTIVE_PLAN`，引导用户创建/激活计划或先收藏生词。

只有最后一种情况才返回 `NO_ACTIVE_PLAN`；它表示“没有任何可学的词”，而不是“没有计划”。散收生词的复习不依赖计划。

### 13.4 时区与“今日”

`daily_new_word_limit` 的“今日”以 `Asia/Shanghai` 自然日为界（与外刊导入任务时区一致），避免用户跨午夜学习时把每日新增额度算错。`next_review_at` 统一使用 UTC 存储，展示时再转换。

### 13.5 词典词条缺失时的出题

按词典数据规则（[产品需求](01-product-requirements.md)第 10.3 节），词条未来被更新或删除时，用户仍保留其 `user_words`。但出题需要从 `ecdict` 读取释义/正确答案，词条缺失时无法组题。处理：

- `GET /word-learning/next` 选中某词后，若其 `ecdict` 词条已不存在（或正确释义为空、`canonicalGloss` 取不到有效值），**跳过该词**继续选下一个到期词，并把该 `user_words` 的 `next_review_at` 顺延一小段时间，避免每次都卡在它上面。
- 若全部到期词都因词条缺失而无法出题，按“无可出题的到期词”返回 `NO_DUE_WORDS`。
- 生词本/复习列表展示该词时，提示“词典数据不可用”，但不删除用户进度。
