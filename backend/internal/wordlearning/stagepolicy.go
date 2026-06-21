// Package wordlearning 实现渐进式单词学习：阶段规则、出题、令牌与答题事务。
package wordlearning

import "time"

// 学习阶段。
const (
	StageRecognition    = "recognition"
	StageDiscrimination = "discrimination"
	StageSpelling       = "spelling"
	StageMastered       = "mastered"
)

// 题型代码。
const (
	QTypeWordToMeaningChoice = "word_to_meaning_choice"
	QTypeMeaningToWordChoice = "meaning_to_word_choice"
	QTypeMeaningToWordSpell  = "meaning_to_word_spelling"
)

// 间隔常量。recognition 阶段内部用“学习步”，其余用复习间隔。
const (
	LearningStep = time.Minute
	interval10m  = 10 * time.Minute
	interval1d   = 24 * time.Hour
	interval3d   = 3 * 24 * time.Hour
	interval7d   = 7 * 24 * time.Hour
	interval30d  = 30 * 24 * time.Hour
)

// StageForQuestionType 返回阶段对应的默认题型。
func StageForQuestionType(stage string) string {
	switch stage {
	case StageRecognition:
		return QTypeWordToMeaningChoice
	case StageDiscrimination:
		return QTypeMeaningToWordChoice
	default: // spelling / mastered
		return QTypeMeaningToWordSpell
	}
}

// MasteryLabel 由阶段派生页面掌握标签。
func MasteryLabel(stage string) string {
	switch stage {
	case StageRecognition:
		return "unknown"
	case StageDiscrimination:
		return "fuzzy"
	case StageSpelling:
		return "known"
	case StageMastered:
		return "mastered"
	default:
		return "unknown"
	}
}

// InitialStageFor 把用户首次收藏时的熟悉度映射为初始阶段与首次训练延迟。
func InitialStageFor(familiarity string) (stage string, delay time.Duration) {
	switch familiarity {
	case "fuzzy", "discrimination":
		return StageDiscrimination, interval1d
	case "known", "spelling":
		return StageSpelling, interval3d
	case "mastered":
		return StageMastered, interval30d
	default: // unknown / recognition / 默认
		return StageRecognition, 0
	}
}

// StageDecision 是 StagePolicy 计算出的阶段更新结果。
type StageDecision struct {
	Stage          string
	Streak         int
	NextReviewAt   time.Time
	BecameMastered bool // 本次是否首次进入 mastered（由调用方结合 first_mastered_at 判断是否真正首次）
}

// Apply 是纯规则：根据当前阶段、连续正确数、本次是否正确与是否用提示，计算新状态。
// usedHint 仅对 spelling/mastered（默写题）生效：用提示即视为未通过，降级。
func Apply(stage string, streak int, correct bool, usedHint bool, now time.Time) StageDecision {
	switch stage {
	case StageRecognition:
		if !correct {
			return StageDecision{Stage: StageRecognition, Streak: 0, NextReviewAt: now.Add(LearningStep)}
		}
		if streak+1 >= 2 {
			return StageDecision{Stage: StageDiscrimination, Streak: 0, NextReviewAt: now.Add(interval1d)}
		}
		return StageDecision{Stage: StageRecognition, Streak: 1, NextReviewAt: now.Add(LearningStep)}

	case StageDiscrimination:
		if !correct {
			return StageDecision{Stage: StageRecognition, Streak: 0, NextReviewAt: now.Add(interval10m)}
		}
		if streak+1 >= 2 {
			return StageDecision{Stage: StageSpelling, Streak: 0, NextReviewAt: now.Add(interval3d)}
		}
		return StageDecision{Stage: StageDiscrimination, Streak: 1, NextReviewAt: now.Add(interval1d)}

	case StageSpelling:
		if !correct || usedHint {
			return StageDecision{Stage: StageDiscrimination, Streak: 0, NextReviewAt: now.Add(interval1d)}
		}
		switch streak {
		case 0:
			return StageDecision{Stage: StageSpelling, Streak: 1, NextReviewAt: now.Add(interval3d)}
		case 1:
			return StageDecision{Stage: StageSpelling, Streak: 2, NextReviewAt: now.Add(interval7d)}
		default: // streak >= 2 → 第三次正确
			return StageDecision{Stage: StageMastered, Streak: 0, NextReviewAt: now.Add(interval30d), BecameMastered: true}
		}

	case StageMastered:
		if !correct || usedHint {
			return StageDecision{Stage: StageDiscrimination, Streak: 0, NextReviewAt: now.Add(interval1d)}
		}
		return StageDecision{Stage: StageMastered, Streak: 0, NextReviewAt: now.Add(interval30d)}

	default:
		return StageDecision{Stage: StageRecognition, Streak: 0, NextReviewAt: now.Add(LearningStep)}
	}
}
