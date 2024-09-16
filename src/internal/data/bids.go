package data

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"
)

//
//DESCRIPTION????
//

type Bid struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"-"`
	Status      string `json:"status"`
	TenderId    string `json:"-"`
	AuthorType  string `json:"authorType"`
	AuthorId    string `json:"authorId"`
	Version     int    `json:"version"`
	CreatedAt   string `json:"createdAt"`
}

type BidModel struct {
	DB *sql.DB
}

var (
	ErrBidNotFound         = errors.New("bid does not exist")
	ErrBidOrTenderNotFound = errors.New("bid or tender does not exist")
	ErrBidVersionNotFound  = errors.New("bid version does not exist")
)

func (m BidModel) GetBidById(bidId string) (*Bid, error) {
	query :=
		`
		SELECT * FROM bids WHERE id=$1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	var bid Bid
	row := m.DB.QueryRowContext(ctx, query, bidId)

	err := row.Scan(&bid.Id, &bid.Name, &bid.Description, &bid.Status, &bid.TenderId, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBidNotFound
		}
		return nil, err
	}
	return &bid, nil

}

func (m BidModel) InsertBid(bid *Bid) error {
	query := `
		INSERT INTO bids (id, name, description, status, tender_id, author_type, author_id, version, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
		`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []interface{}{bid.Id, bid.Name, bid.Description, bid.Status, bid.TenderId, bid.AuthorType, bid.AuthorId, bid.Version, bid.CreatedAt}

	_, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (m BidModel) GetMyBids(limit, offset int32, groupId, userId string) ([]*Bid, error) {
	query :=
		`
		SELECT * FROM bids 
		WHERE author_id=$1 OR author_id=$2 
		ORDER BY name
		LIMIT $3 OFFSET $4
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	args := []any{userId, groupId, limit, offset}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err)
		}
	}()

	bids := []*Bid{}
	for rows.Next() {
		var bid Bid

		err := rows.Scan(
			&bid.Id,
			&bid.Name,
			&bid.Description,
			&bid.Status,
			&bid.TenderId,
			&bid.AuthorType,
			&bid.AuthorId,
			&bid.Version,
			&bid.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		bids = append(bids, &bid)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return bids, nil

}

func (m BidModel) ChangeBidStatus(bidId, status string) (*Bid, error) {

	query :=
		`
		UPDATE bids SET status=$1
		WHERE id=$2
		RETURNING *
	`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var bid Bid

	err := m.DB.QueryRowContext(ctx, query, status, bidId).Scan(
		&bid.Id, &bid.Name, &bid.Description, &bid.Status, &bid.TenderId, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBidNotFound
		}
		return nil, err
	}
	return &bid, nil

}

func (m BidModel) GetBidStatus(bidId string) (string, error) {
	query :=
		`
		SELECT status FROM bids WHERE id=$1
	`

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var status string

	row := m.DB.QueryRowContext(ctx, query, bidId)

	err := row.Scan(&status)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrBidNotFound
		}
		return "", err

	}
	return status, nil

}

func (m BidModel) GetBidsByTenderId(limit, offset int32, tenderId string) ([]*Bid, error) {
	query := `
		SELECT * FROM bids 
		WHERE tender_id=$1 AND status='Published'
		ORDER BY name
		LIMIT $2 OFFSET $3
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []any{tenderId, limit, offset}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err)
		}
	}()

	bids := []*Bid{}
	for rows.Next() {
		var bid Bid

		err := rows.Scan(
			&bid.Id,
			&bid.Name,
			&bid.Description,
			&bid.Status,
			&bid.TenderId,
			&bid.AuthorType,
			&bid.AuthorId,
			&bid.Version,
			&bid.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		bids = append(bids, &bid)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(bids) == 0 {
		return nil, ErrBidOrTenderNotFound
	}

	return bids, nil

}

func (m BidModel) EditBid(bidId string, newBid Bid) (*Bid, error) {
	updateQuery := `
		UPDATE bids SET name=coalesce(NULLIF($1,''), name), description=coalesce(NULLIF($2,''), description), version=$3 
		WHERE id=$4
		RETURNING id, name, description, status, tender_id, author_type,author_id, version, created_at
	`
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	currentBid := Bid{}

	err = tx.QueryRow("SELECT * FROM bids WHERE id = $1", bidId).Scan(
		&currentBid.Id, &currentBid.Name, &currentBid.Description, &currentBid.Status, &currentBid.TenderId, &currentBid.AuthorType, &currentBid.AuthorId, &currentBid.Version, &currentBid.CreatedAt)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBidNotFound
		}
		return nil, err
	}

	_, err = tx.Exec(`INSERT INTO bids_history (bid_id, name, description, version) 
	VALUES ($1, $2, $3, $4)`, currentBid.Id, currentBid.Name, currentBid.Description, currentBid.Version)

	if err != nil {
		tx.Rollback()
		return nil, err
	}
	row := tx.QueryRow(updateQuery, newBid.Name, newBid.Description, currentBid.Version+1, bidId)

	err = row.Scan(&newBid.Id, &newBid.Name, &newBid.Description, &newBid.Status, &newBid.TenderId, &newBid.AuthorType, &newBid.AuthorId, &newBid.Version, &newBid.CreatedAt)

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &newBid, nil
}

func (m BidModel) RollbackBid(targetVersion int, bidId string) (*Bid, error) {
	getHistoryTenderQuery :=
		`
		SELECT name, description FROM bids_history 
		WHERE bid_id=$1 AND version=$2
	`

	rollbackTenderQuery :=
		`
		UPDATE bids SET name=$1, description=$2, version=version+1
		WHERE id=$3
		RETURNING *
	`
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}

	currentParams := struct {
		name        string
		description string
		version     string
	}{}
	err = tx.QueryRow("SELECT name, description, version FROM bids WHERE id = $1", bidId).Scan(&currentParams.name, &currentParams.description, &currentParams.version)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	_, err = tx.Exec("INSERT INTO bids_history (bid_id, name, description, version) VALUES ($1, $2, $3,$4)", bidId, currentParams.name, currentParams.description, currentParams.version)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	historyParams := struct {
		name        string
		description string
	}{}

	row := tx.QueryRow(getHistoryTenderQuery, bidId, targetVersion)
	err = row.Scan(&historyParams.name, &historyParams.description)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBidVersionNotFound
		}
		return nil, err
	}

	bid := Bid{}

	row = tx.QueryRow(rollbackTenderQuery, historyParams.name, historyParams.description, bidId)
	err = row.Scan(&bid.Id, &bid.Name, &bid.Description, &bid.Status, &bid.TenderId, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt)

	if err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenderVersionNotFound
		}
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &bid, nil

}

/*func (m *BidModel) ApproveDecision(bidId string) error {
	closeTenderQuery := `UPDATE tenders SET status='Closed'
	WHERE id=(SELECT tender_id FROM bids WHERE id=$1)`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, closeTenderQuery, bidId)
	return err
}*/

func (m BidModel) RejectDecision(bidId string) (*Bid, error) {
	cancelBidQuery := `UPDATE bids SET status='Canceled' 
	WHERE id=$1
	RETURNING *`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var bid Bid
	row := m.DB.QueryRowContext(ctx, cancelBidQuery, bidId)
	err := row.Scan(&bid.Id, &bid.Name, &bid.Description, &bid.Status, &bid.TenderId, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBidNotFound
		}
		return nil, err
	}
	return &bid, nil
}

func (m BidModel) ApproveDecision(bidId, userId string) error {
	query := `
		INSERT INTO bids_approvals (bid_id, user_id)
		VALUES ($1, $2)
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, bidId, userId)

	return err
}

func (m BidModel) ApprovalCount(bidId string) ([]string, error) {
	query :=
		`
		SELECT user_id FROM bids_approvals
		WHERE bid_id=$1
	`
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, bidId)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err)
		}
	}()

	ids := []string{}
	for rows.Next() {
		var id string

		err := rows.Scan(
			&id,
		)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}
