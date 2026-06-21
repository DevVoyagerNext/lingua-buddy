// Package models 定义全部 GORM 持久化模型。
// ecdict 为已有只读词典表，不参与 AutoMigrate；其余 15 张业务表由迁移命令创建。
package models

import "time"

// ====== 只读词典 ======

// ECDICTEntry 映射已有词典表 ecdict（只读，不迁移）。
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

// TableName 固定为已有表名 ecdict。
func (ECDICTEntry) TableName() string { return "ecdict" }

// ====== 用户 ======

// User 账号、注册方式与英语水平。
type User struct {
	ID                 uint64    `gorm:"primaryKey" json:"id"`
	Username           string    `gorm:"size:30;not null;uniqueIndex:uk_users_username" json:"username"`
	Email              *string   `gorm:"size:120;uniqueIndex:uk_users_email" json:"email"`
	PasswordHash       *string   `gorm:"size:255" json:"-"`
	RegistrationMethod string    `gorm:"size:32;not null;default:username_password" json:"registration_method"`
	EnglishLevel       string    `gorm:"size:16;not null;default:''" json:"english_level"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ====== 词汇计划 ======

// WordLearningPlan 四级、六级等词汇学习计划。
type WordLearningPlan struct {
	ID                  uint64     `gorm:"primaryKey" json:"id"`
	UserID              uint64     `gorm:"not null;index" json:"user_id"`
	Name                string     `gorm:"size:64;not null" json:"name"`
	SourceType          string     `gorm:"size:24;not null" json:"source_type"`
	SourceValue         string     `gorm:"size:32;not null" json:"source_value"`
	OrderingMode        string     `gorm:"size:32;not null;default:frequency_shuffled" json:"ordering_mode"`
	ShuffleSeed         int64      `gorm:"not null" json:"-"`
	SourceSnapshotCount int        `gorm:"not null;default:0" json:"source_snapshot_count"`
	DailyNewWordLimit   int        `gorm:"not null;default:10" json:"daily_new_word_limit"`
	ActiveWordLimit     int        `gorm:"not null;default:20" json:"active_word_limit"`
	Status              string     `gorm:"size:16;not null;default:active;index" json:"status"`
	StartedAt           *time.Time `json:"started_at"`
	CompletedAt         *time.Time `json:"completed_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// WordLearningPlanItem 计划的固定单词队列与激活状态。
type WordLearningPlanItem struct {
	ID              uint64     `gorm:"primaryKey" json:"id"`
	PlanID          uint64     `gorm:"not null;uniqueIndex:uk_plan_entry,priority:1;uniqueIndex:uk_plan_position,priority:1" json:"plan_id"`
	ECDICTEntryID   uint64     `gorm:"not null;uniqueIndex:uk_plan_entry,priority:2" json:"ecdict_entry_id"`
	Word            string     `gorm:"size:255;not null" json:"word"`
	QueuePosition   int        `gorm:"not null;uniqueIndex:uk_plan_position,priority:2" json:"queue_position"`
	UserWordID      *uint64    `gorm:"index" json:"user_word_id"`
	ActivatedAt     *time.Time `json:"activated_at"`
	FirstMasteredAt *time.Time `json:"first_mastered_at"`
	SkippedAt       *time.Time `json:"skipped_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// UserWord 用户已收藏或已激活单词的全局学习状态。
type UserWord struct {
	ID                 uint64     `gorm:"primaryKey" json:"id"`
	UserID             uint64     `gorm:"not null;uniqueIndex:uk_user_word,priority:1" json:"user_id"`
	ECDICTEntryID      *uint64    `gorm:"index" json:"ecdict_entry_id"`
	Word               string     `gorm:"size:255;not null;uniqueIndex:uk_user_word,priority:2" json:"word"`
	LearningStage      string     `gorm:"size:24;not null;index" json:"learning_stage"`
	StageCorrectStreak int        `gorm:"not null;default:0" json:"stage_correct_streak"`
	NextReviewAt       time.Time  `gorm:"not null;index" json:"next_review_at"`
	LastTrainedAt      *time.Time `json:"last_trained_at"`
	StageChangedAt     time.Time  `gorm:"not null" json:"stage_changed_at"`
	FirstMasteredAt    *time.Time `json:"first_mastered_at"`
	TotalCorrectCount  int        `gorm:"not null;default:0" json:"total_correct_count"`
	TotalWrongCount    int        `gorm:"not null;default:0" json:"total_wrong_count"`
	LastAnswerCorrect  *bool      `json:"last_answer_correct"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// UserWordNote 用户为单词添加的笔记，同一单词可多条。
type UserWordNote struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `gorm:"not null;index:idx_user_word_note,priority:1" json:"user_id"`
	Word      string    `gorm:"size:255;not null;index:idx_user_word_note,priority:2" json:"word"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserSentence 用户收藏的英文句子，按 (user_id, sentence_hash) 去重。
type UserSentence struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	UserID       uint64    `gorm:"not null;uniqueIndex:uk_user_sentence,priority:1" json:"user_id"`
	Sentence     string    `gorm:"type:text;not null" json:"sentence"`
	SentenceHash string    `gorm:"size:64;not null;uniqueIndex:uk_user_sentence,priority:2" json:"-"`
	Translation  *string   `gorm:"type:text" json:"translation"`
	Note         *string   `gorm:"type:text" json:"note"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ====== 外刊 ======

// Article 可阅读的外刊文章。
type Article struct {
	ID          uint64     `gorm:"primaryKey" json:"id"`
	Title       string     `gorm:"size:512;not null" json:"title"`
	Summary     *string    `gorm:"type:text" json:"summary"`
	Content     *string    `gorm:"type:mediumtext" json:"content"`
	Difficulty  string     `gorm:"size:16;not null;default:intermediate;index" json:"difficulty"`
	SourceName  string     `gorm:"size:128;not null" json:"source_name"`
	SourceURL   string     `gorm:"size:768;not null;uniqueIndex:uk_article_url" json:"source_url"`
	Attribution *string    `gorm:"size:512" json:"attribution"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// UserArticleRead 用户文章阅读记录。
type UserArticleRead struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	UserID     uint64    `gorm:"not null;uniqueIndex:uk_user_article,priority:1" json:"user_id"`
	ArticleID  uint64    `gorm:"not null;uniqueIndex:uk_user_article,priority:2" json:"article_id"`
	IsFinished bool      `gorm:"not null;default:false" json:"is_finished"`
	LastReadAt time.Time `json:"last_read_at"`
}

// ====== 查词历史 ======

// DictionaryQueryRecord 查词历史，按 (user_id, word) 去重累加。
type DictionaryQueryRecord struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	UserID        uint64    `gorm:"not null;uniqueIndex:uk_dict_query,priority:1" json:"user_id"`
	Word          string    `gorm:"size:255;not null;uniqueIndex:uk_dict_query,priority:2" json:"word"`
	QueryCount    int       `gorm:"not null;default:1" json:"query_count"`
	LastQueriedAt time.Time `json:"last_queried_at"`
}

// ====== 统一历史 ======

// HistoryRecord 翻译、语音、语法分析、纠错、作文共用的简单历史。
type HistoryRecord struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	UserID      uint64    `gorm:"not null;index:idx_history_user_time,priority:1" json:"user_id"`
	RecordType  string    `gorm:"size:24;not null;index" json:"record_type"`
	InputText   string    `gorm:"type:mediumtext;not null" json:"input_text"`
	ResultText  string    `gorm:"type:mediumtext;not null" json:"result_text"`
	AudioFileID *uint64   `gorm:"index" json:"audio_file_id"`
	CreatedAt   time.Time `gorm:"index:idx_history_user_time,priority:2" json:"created_at"`
}

// ====== AI 对话 ======

// Conversation AI 对话会话。
type Conversation struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	UserID     uint64    `gorm:"not null;index" json:"user_id"`
	Title      string    `gorm:"size:128;not null" json:"title"`
	Scene      string    `gorm:"size:64;not null" json:"scene"`
	Difficulty string    `gorm:"size:16;not null" json:"difficulty"`
	Status     string    `gorm:"size:16;not null;default:active" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ConversationMessage 会话内的用户与 AI 消息。
type ConversationMessage struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	ConversationID uint64    `gorm:"not null;index" json:"conversation_id"`
	Role           string    `gorm:"size:16;not null" json:"role"`
	Content        string    `gorm:"type:mediumtext;not null" json:"content"`
	Feedback       *string   `gorm:"type:mediumtext" json:"feedback"`
	CreatedAt      time.Time `json:"created_at"`
}

// ====== 音频 ======

// AudioFile 原始音频文件信息（本体存 OSS）。
type AudioFile struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	UserID       uint64    `gorm:"not null;index" json:"user_id"`
	FilePath     string    `gorm:"size:768;not null" json:"file_path"`
	OriginalName string    `gorm:"size:255;not null" json:"original_name"`
	MimeType     string    `gorm:"size:64;not null" json:"mime_type"`
	FileSize     int64     `gorm:"not null" json:"file_size"`
	CreatedAt    time.Time `json:"created_at"`
}

// ====== 专项训练答题记录 ======

// TrainingAnswerRecord 用户每一次已提交的专项训练答案（追加式历史）。
type TrainingAnswerRecord struct {
	ID                     uint64     `gorm:"primaryKey" json:"id"`
	UserID                 uint64     `gorm:"not null;uniqueIndex:uk_training_submission,priority:1;index:idx_training_user_time,priority:1;index:idx_training_question,priority:1" json:"user_id"`
	SubmissionID           string     `gorm:"size:36;not null;uniqueIndex:uk_training_submission,priority:2" json:"submission_id"`
	TrainingType           string     `gorm:"size:20;not null;index:idx_training_question,priority:2" json:"training_type"`
	QuestionType           string     `gorm:"size:40;not null" json:"question_type"`
	QuestionKey            string     `gorm:"size:255;not null;index:idx_training_question,priority:3" json:"question_key"`
	AnswerSource           string     `gorm:"size:32;not null" json:"answer_source"`
	UserWordID             *uint64    `gorm:"index" json:"user_word_id"`
	WordLearningPlanID     *uint64    `gorm:"index" json:"word_learning_plan_id"`
	WordLearningPlanItemID *uint64    `gorm:"index" json:"word_learning_plan_item_id"`
	QuestionText           string     `gorm:"type:text;not null" json:"question_text"`
	Options                []byte     `gorm:"type:json" json:"options"`
	UserAnswer             string     `gorm:"type:mediumtext;not null" json:"user_answer"`
	ReferenceAnswer        *string    `gorm:"type:mediumtext" json:"reference_answer"`
	IsCorrect              *bool      `json:"is_correct"`
	UsedHint               bool       `gorm:"not null;default:false" json:"used_hint"`
	LearningStageBefore    *string    `gorm:"size:24" json:"learning_stage_before"`
	LearningStageAfter     *string    `gorm:"size:24" json:"learning_stage_after"`
	GeneratorVersion       *string    `gorm:"size:40" json:"generator_version"`
	EvaluationStatus       string     `gorm:"size:20;not null" json:"evaluation_status"`
	EvaluationResult       []byte     `gorm:"type:json" json:"evaluation_result"`
	HistoryRecordID        *uint64    `json:"history_record_id"`
	SubmittedAt            time.Time  `gorm:"not null;index:idx_training_user_time,priority:2" json:"submitted_at"`
	EvaluatedAt            *time.Time `json:"evaluated_at"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// UserTranslationWrongQuestion 用户确认仍需重练的翻译错题。
type UserTranslationWrongQuestion struct {
	ID                 uint64    `gorm:"primaryKey" json:"id"`
	UserID             uint64    `gorm:"not null;uniqueIndex:uk_trans_wrong,priority:1" json:"user_id"`
	Direction          string    `gorm:"size:16;not null" json:"direction"`
	QuestionKey        string    `gorm:"size:64;not null;uniqueIndex:uk_trans_wrong,priority:2" json:"-"`
	QuestionText       string    `gorm:"type:text;not null" json:"question_text"`
	ReferenceAnswer    *string   `gorm:"type:text" json:"reference_answer"`
	UserAnswer         *string   `gorm:"type:text" json:"user_answer"`
	WrongCount         int       `gorm:"not null;default:1" json:"wrong_count"`
	LastAnswerRecordID *uint64   `json:"last_answer_record_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// BusinessModels 返回需要 AutoMigrate 的业务模型（不含只读 ecdict）。
func BusinessModels() []any {
	return []any{
		&User{},
		&WordLearningPlan{},
		&WordLearningPlanItem{},
		&UserWord{},
		&UserWordNote{},
		&UserSentence{},
		&Article{},
		&UserArticleRead{},
		&DictionaryQueryRecord{},
		&HistoryRecord{},
		&Conversation{},
		&ConversationMessage{},
		&AudioFile{},
		&TrainingAnswerRecord{},
		&UserTranslationWrongQuestion{},
	}
}
