package service

import (
	"testing"

	"meteorx/internal/modules/wiki/model"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"draft -> pending_review", model.PublishStatusDraft, model.PublishStatusPendingReview, true},
		{"draft -> published", model.PublishStatusDraft, model.PublishStatusPublished, false},
		{"draft -> archived", model.PublishStatusDraft, model.PublishStatusArchived, false},
		{"pending_review -> published", model.PublishStatusPendingReview, model.PublishStatusPublished, true},
		{"pending_review -> rejected", model.PublishStatusPendingReview, model.PublishStatusRejected, true},
		{"pending_review -> draft", model.PublishStatusPendingReview, model.PublishStatusDraft, true},
		{"pending_review -> archived", model.PublishStatusPendingReview, model.PublishStatusArchived, false},
		{"published -> archived", model.PublishStatusPublished, model.PublishStatusArchived, true},
		{"published -> draft", model.PublishStatusPublished, model.PublishStatusDraft, true},
		{"published -> pending_review", model.PublishStatusPublished, model.PublishStatusPendingReview, false},
		{"rejected -> pending_review", model.PublishStatusRejected, model.PublishStatusPendingReview, true},
		{"rejected -> draft", model.PublishStatusRejected, model.PublishStatusDraft, true},
		{"rejected -> published", model.PublishStatusRejected, model.PublishStatusPublished, false},
		{"archived -> draft", model.PublishStatusArchived, model.PublishStatusDraft, true},
		{"archived -> published", model.PublishStatusArchived, model.PublishStatusPublished, false},
		{"archived -> pending_review", model.PublishStatusArchived, model.PublishStatusPendingReview, false},
		{"same state", model.PublishStatusDraft, model.PublishStatusDraft, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canTransition(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("canTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestValidTransitions_Complete(t *testing.T) {
	allStatuses := []string{
		model.PublishStatusDraft,
		model.PublishStatusPendingReview,
		model.PublishStatusPublished,
		model.PublishStatusRejected,
		model.PublishStatusArchived,
	}

	for _, from := range allStatuses {
		if _, exists := validTransitions[from]; !exists {
			t.Errorf("status %q not defined in validTransitions", from)
		}
	}
}