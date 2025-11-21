package service

import (
	"context"
	"database/sql"
	"math/rand"

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
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var exists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_requests_id = $1)`, req.PullRequestId).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConflict
	}

	var authorTeam string
	err = tx.QueryRowContext(ctx, `SELECT team_name FROM users WHERE user_id = $1 FOR UPDATE`, req.AuthorId).Scan(&authorTeam)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) 
		VALUES ($1, $2, $3, $4)`,
		req.PullRequestId, req.PullRequestName, req.AuthorId, models.PullRequestStatusOPEN)
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
	return s.getPullRequest(ctx, req.PullRequestId)
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (*models.PullRequest, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM pull_requests WHERE pull_request_id = $1`, prID).Scan(&status)
	if err != nil {
		return nil, err
	}

	if status == string(models.PullRequestStatusMERGED) {
		//идемптоичнсть
		return s.getPullRequest(ctx, prID)
	}

	_, err = tx.ExecContext(ctx, `UPDATE pull_requests SET status = $1, merged_at = CURRENT_TIMESTAMP 
		WHERE pull_request_id = $2`, models.PullRequestShortStatusMERGED, prID)
	if err != nil {
		return nil, err
	}

	return s.getPullRequest(ctx, prID)
}

func (s *Service) ReassignReviewer(ctx context.Context, req models.PostPullRequestReassignJSONRequestBody) (*models.PullRequest, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM pull_request WHERE pull_request_id = $1`, req.PullRequestId).Scan(&status)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	if status == string(models.PullRequestStatusMERGED) {
		return nil, ErrPrecondition
	}

	var assigned bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pr_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2)`, req.PullRequestId, req.OldUserId).Scan(&assigned)
	if err != nil {
		return nil, err
	}
	if !assigned {
		return nil, ErrInvalidInput //нет юзера
	}

	var oldTeam string
	err = tx.QueryRowContext(ctx, `SELECT team_name FROM users WHERE user_id = $1`, req.OldUserId).Scan(&oldTeam)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT u.user_id FROM users u
		WHERE u.team_name = $1 AND u.is_active = TRUE AND u.user_id != $2 AND u.user_id NOT IN 
		(SELECT reviewer_id FROM pr_reviewers WHERE pull_request_id = $3)
	`, oldTeam, req.OldUserId, req.PullRequestId)
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

	if len(candidates) == 0 {
		return nil, ErrConflict
	}

	newRev := candidates[rand.Intn(len(candidates))]

	_, err = tx.ExecContext(ctx, `DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2`, req.PullRequestId, req.OldUserId)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)`, req.PullRequestId, newRev)
	if err != nil {
		return nil, err
	}

	return s.getPullRequest(ctx, req.PullRequestId)
}

func (s *Service) AddTeam(ctx context.Context, team models.Team) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	_, err = tx.ExecContext(ctx, "INSERT INTO teams (team_name) VALUES ($1) ON CONFLICT (team_name) DO NOTHING", team.TeamName)
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

	return nil
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id, username, is_active FROM users WHERE team_name  = $1`, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.TeamMember
	for rows.Next() {
		var m models.TeamMember
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	if len(members) == 0 {
		return nil, ErrNotFound
	}

	return &models.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}

func (s *Service) GetUsersReviews(ctx context.Context, userID string) ([]models.PullRequestShort, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status 
        FROM pull_requests pr 
        JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id 
        WHERE prr.reviewer_id = $1
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []models.PullRequestShort
	for rows.Next() {
		var pr models.PullRequestShort
		var statusStr string
		if err := rows.Scan(&pr.PullRequestId, &pr.PullRequestName, &pr.AuthorId, &statusStr); err != nil {
			return nil, err
		}
		pr.Status = models.PullRequestShortStatus(statusStr)
		prs = append(prs, pr)
	}
	if prs == nil {
		prs = []models.PullRequestShort{}
	}
	return prs, nil
}

func (s *Service) SetUserActive(ctx context.Context, req models.PostUsersSetIsActiveJSONRequestBody) (*models.User, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET is_active = $1 WHERE user_id = $2`, req.IsActive, req.UserId)
	if err != nil {
		return nil, err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	var user models.User
	err = s.db.QueryRowContext(ctx, `SELECT user_id, username, team_name, is_active FROM users WHERE user_id = $1`, req.UserId).
		Scan(&user.UserId, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Service) getPullRequest(ctx context.Context, prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	var statusStr string

	err := s.db.QueryRowContext(ctx, `
        SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at 
        FROM pull_requests WHERE pull_request_id = $1
    `, prID).Scan(&pr.PullRequestId, &pr.PullRequestName, &pr.AuthorId, &statusStr, &pr.CreatedAt, &pr.MergedAt)
	if err != nil {
		return nil, err
	}
	pr.Status = models.PullRequestStatus(statusStr)

	rows, err := s.db.QueryContext(ctx, `SELECT reviewer_id FROM pr_reviewers WHERE pull_request_id = $1`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var rev string
		if err := rows.Scan(&rev); err != nil {
			return nil, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, rev)
	}

	return &pr, nil
}
