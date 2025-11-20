package service

import (
	"context"
	"database/sql"
	"math/rand/v2"

	"pull-request-api.com/internal/models"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreatePullRequest(ctx context.Context, req models.PostPullRequestCreateJSONRequestBody) (*models.PullRequest, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	//check existing
	var exists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_requests_id = $1)`, req.PullRequestId).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConflict
	}

	var authorTeam string
	err = tx.QueryRowContext(ctx, `SELECT team_name FROM users WHERE user_id = $1`, req.AuthorId).Scan(&authorTeam)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) 
		VALUES ($1, $2, $3, 'OPEN')`,
		req.PullRequestId, req.PullRequestName, req.AuthorId)
	if err != nil {
		return nil, err
	}

	rows, err := tx.QueryContext(ctx, `SELECT user_id FROM users 
		WHERE team_name = $1 AND is_active = TRUE AND user_id != $2`, authorTeam, req.AuthorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		candidates = append(candidates, uid)
	}
	rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
	if len(candidates) > 2 {
		candidates = candidates[:2]
	}

	for _, rev := range candidates {
		_, err := tx.ExecContext(ctx, `INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)`, req.PullRequestId, rev)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return _, _
}

func (s *Service) AddTeam(ctx context.Context, team models.Team) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO teams (team_name) VALUES ($1) ON CONFLICT (team_name) DO NOTHING", team.TeamName)
	if err != nil {
		return err
	}

	for _, m := range team.Members {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO users (user_id, username, team_name, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE SET username = $2, team_name = $3, is_active = $4
		`, m.UserId, m.Username, team.TeamName, m.IsActive)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
