package server

import "testing"

func TestValidateVoteRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		if err := validateVoteRequest(voteRequest{UserID: 1, CandidateID: 2}); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("invalid user_id", func(t *testing.T) {
		if err := validateVoteRequest(voteRequest{UserID: 0, CandidateID: 2}); err == nil {
			t.Fatal("expected error for invalid user_id")
		}
	})

	t.Run("invalid candidate_id", func(t *testing.T) {
		if err := validateVoteRequest(voteRequest{UserID: 1, CandidateID: -1}); err == nil {
			t.Fatal("expected error for invalid candidate_id")
		}
	})
}
