package wordlearning

import (
	"testing"
	"time"
)

func TestStagePolicy(t *testing.T) {
	now := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		stage      string
		streak     int
		correct    bool
		usedHint   bool
		wantStage  string
		wantStreak int
		wantNext   time.Duration
		wantMaster bool
	}{
		{"recognition 第1次对", StageRecognition, 0, true, false, StageRecognition, 1, LearningStep, false},
		{"recognition 第2次对→辨认", StageRecognition, 1, true, false, StageDiscrimination, 0, interval1d, false},
		{"recognition 答错", StageRecognition, 1, false, false, StageRecognition, 0, LearningStep, false},
		{"discrimination 第1次对", StageDiscrimination, 0, true, false, StageDiscrimination, 1, interval1d, false},
		{"discrimination 第2次对→默写", StageDiscrimination, 1, true, false, StageSpelling, 0, interval3d, false},
		{"discrimination 答错→识别", StageDiscrimination, 0, false, false, StageRecognition, 0, interval10m, false},
		{"spelling 第1次对", StageSpelling, 0, true, false, StageSpelling, 1, interval3d, false},
		{"spelling 第2次对", StageSpelling, 1, true, false, StageSpelling, 2, interval7d, false},
		{"spelling 第3次对→掌握", StageSpelling, 2, true, false, StageMastered, 0, interval30d, true},
		{"spelling 答错→辨认", StageSpelling, 2, false, false, StageDiscrimination, 0, interval1d, false},
		{"spelling 用提示答对也降级", StageSpelling, 2, true, true, StageDiscrimination, 0, interval1d, false},
		{"mastered 答对维持", StageMastered, 0, true, false, StageMastered, 0, interval30d, false},
		{"mastered 答错→辨认", StageMastered, 0, false, false, StageDiscrimination, 0, interval1d, false},
		{"mastered 用提示→辨认", StageMastered, 0, true, true, StageDiscrimination, 0, interval1d, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Apply(tc.stage, tc.streak, tc.correct, tc.usedHint, now)
			if got.Stage != tc.wantStage || got.Streak != tc.wantStreak {
				t.Fatalf("stage/streak = %s/%d, want %s/%d", got.Stage, got.Streak, tc.wantStage, tc.wantStreak)
			}
			if !got.NextReviewAt.Equal(now.Add(tc.wantNext)) {
				t.Fatalf("next = %v, want +%v", got.NextReviewAt.Sub(now), tc.wantNext)
			}
			if got.BecameMastered != tc.wantMaster {
				t.Fatalf("becameMastered = %v, want %v", got.BecameMastered, tc.wantMaster)
			}
		})
	}
}
