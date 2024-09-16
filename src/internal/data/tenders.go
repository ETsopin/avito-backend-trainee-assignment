package data

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"
)

var (
	ErrUsernameNotFound      = errors.New("user does not exist")
	ErrNoRights              = errors.New("user does not have rights for this action")
	ErrOrganizationNotFound  = errors.New("username is incorrect or organization does not exist")
	ErrTenderNotFound        = errors.New("tender does not exist")
	ErrTenderVersionNotFound = errors.New("tender or version does not exist")
)

type Tender struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	ServiceType    string `json:"serviceType"`
	Status         string `json:"status"`
	OrganizationId string `json:"-"`
	Version        int    `json:"version"`
	CreatedAt      string `json:"createdAt"`
}

type TenderModel struct {
	DB *sql.DB
}

func (m TenderModel) GetTenderById(tenderId string) (*Tender, error) {
	query := `
		SELECT * FROM tenders WHERE id=$1
	`

	var tender Tender

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := m.DB.QueryRowContext(ctx, query, tenderId)
	err := row.Scan(&tender.Id, &tender.Name, &tender.CreatedAt, &tender.ServiceType, &tender.Status, &tender.OrganizationId, &tender.Version, &tender.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenderNotFound
		}
		return nil, err
	}
	return &tender, nil
}

func (m TenderModel) ChangeTenderStatus(tenderId string, status string) (*Tender, error) {
	changeStatusQuery := `
		UPDATE tenders SET status=$1 WHERE id=$2 
		RETURNING id, name, description, status, service_type, version, created_at
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var tender Tender

	err := m.DB.QueryRowContext(ctx, changeStatusQuery, status, tenderId).Scan(
		&tender.Id, &tender.Name, &tender.Description, &tender.Status, &tender.ServiceType, &tender.Version, &tender.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenderNotFound
		}
		return nil, err
	}
	return &tender, nil
}

func (m TenderModel) GetTenderStatus(tenderId string) (string, error) {
	tenderStatusQuery := `
		SELECT status FROM tenders WHERE id=$1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	var status string

	err := m.DB.QueryRowContext(ctx, tenderStatusQuery, tenderId).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrTenderNotFound
		}
		return "", err
	}

	return status, nil

}

func (m TenderModel) GetUserID(username string) (string, error) {
	userIdQuery := `
		SELECT id FROM employee WHERE username=$1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	var userId string

	err := m.DB.QueryRowContext(ctx, userIdQuery, username).Scan(&userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrUsernameNotFound
		}
		return "", err
	}

	return userId, nil
}

func (m TenderModel) GetUserOrganization(userId string) (string, error) {
	organizationIdQuery := `
		SELECT organization_id FROM organization_responsible WHERE user_id=$1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	var organizationId string

	err := m.DB.QueryRowContext(ctx, organizationIdQuery, userId).Scan(&organizationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrOrganizationNotFound
		}
		return "", err
	}

	return organizationId, nil
}
func (m TenderModel) GetOrganizationUsers(organizationId string) ([]string, error) {
	query := `
		SELECT user_id FROM organization_responsible WHERE organization_id=$1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, organizationId)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err)
		}
	}()
	var userIds []string
	for rows.Next() {
		var userId string

		err := rows.Scan(&userId)
		if err != nil {
			return nil, err
		}

		userIds = append(userIds, userId)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return userIds, nil
}

func (m TenderModel) InsertTender(tender *Tender) error {
	query := `
		INSERT INTO tenders (id, name, description, service_type, status, organization_id, version, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
		`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{tender.Id, tender.Name, tender.Description, tender.ServiceType, tender.Status, tender.OrganizationId, tender.Version, tender.CreatedAt}

	_, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (m TenderModel) GetTenderOrganization(tenderId string) (string, error) {
	organizationIdQuery := `
		SELECT organization_id FROM tenders WHERE id=$1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	var organizationId string

	err := m.DB.QueryRowContext(ctx, organizationIdQuery, tenderId).Scan(&organizationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrTenderNotFound
		}
		return "", err
	}

	return organizationId, nil
}

func (m TenderModel) GetTenders(limit, offset int32, serviceTypes []string) ([]*Tender, error) {
	query := `
		SELECT id, name, description, service_type, status, organization_id, version, created_at
		FROM tenders
		WHERE ((service_type = $1 OR service_type = $2 OR service_type = $3) 
		OR ($1='' AND $2='' AND $3=''))
		AND status='Published'
		ORDER BY name
		LIMIT NULLIF($4, 0) OFFSET $5
		`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{serviceTypes[0], serviceTypes[1], serviceTypes[2], limit, offset}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err)
		}
	}()

	tenders := []*Tender{}
	for rows.Next() {
		var tender Tender

		err := rows.Scan(
			&tender.Id,
			&tender.Name,
			&tender.Description,
			&tender.ServiceType,
			&tender.Status,
			&tender.OrganizationId,
			&tender.Version,
			&tender.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		tenders = append(tenders, &tender)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tenders, nil
}

func (m TenderModel) GetMyTenders(limit, offset int32, organization_id string) ([]*Tender, error) {
	query := `
		SELECT id, name, description, service_type, status, organization_id, version, created_at
		FROM tenders
		WHERE (organization_id=$1)
		ORDER BY name
		LIMIT NULLIF($2, 0) OFFSET $3
		`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{organization_id, limit, offset}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err)
		}
	}()

	tenders := []*Tender{}
	for rows.Next() {
		var tender Tender

		err := rows.Scan(
			&tender.Id,
			&tender.Name,
			&tender.Description,
			&tender.ServiceType,
			&tender.Status,
			&tender.OrganizationId,
			&tender.Version,
			&tender.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		tenders = append(tenders, &tender)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tenders, nil
}

func (m *TenderModel) UpdateTender(tenderId string, newTender Tender) (*Tender, error) {

	///MULTIPLE CONTEXTS????
	updateQuery := `
		UPDATE tenders SET name=coalesce(NULLIF($1,''), name), description=coalesce(NULLIF($2,''), description), 
		service_type=coalesce(NULLIF($3,''), service_type),version=$4 WHERE id=$5
		RETURNING id, name, description, status, service_type, version, created_at
	`
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	currentTender := Tender{}

	err = tx.QueryRow("SELECT id, name, description, service_type, version FROM tenders WHERE id = $1", tenderId).Scan(
		&currentTender.Id, &currentTender.Name, &currentTender.Description, &currentTender.ServiceType, &currentTender.Version)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenderNotFound
		}
		return nil, err
	}

	_, err = tx.Exec(`INSERT INTO tenders_history (tender_id, name, description, service_type, version) 
	VALUES ($1, $2, $3, $4, $5)`, currentTender.Id, currentTender.Name, currentTender.Description, currentTender.ServiceType, currentTender.Version)

	if err != nil {
		tx.Rollback()
		return nil, err
	}
	row := tx.QueryRow(updateQuery, newTender.Name, newTender.Description, newTender.ServiceType, currentTender.Version+1, tenderId)

	err = row.Scan(&newTender.Id, &newTender.Name, &newTender.Description, &newTender.Status, &newTender.ServiceType, &newTender.Version, &newTender.CreatedAt)

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &newTender, nil

}

func (m *TenderModel) RollbackTender(targetVersion int, tenderId string) (*Tender, error) {
	getHistoryTenderQuery :=
		`
		SELECT name, description, service_type FROM tenders_history 
		WHERE tender_id=$1 AND version=$2
	`

	rollbackTenderQuery :=
		`
		UPDATE tenders SET name=$1, description=$2, service_type=$3, version=version+1
		WHERE id=$4
		RETURNING id, name, description, status, service_type, version, created_at
	`
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}

	currentParams := struct {
		name        string
		description string
		serviceType string
		version     string
	}{}
	err = tx.QueryRow("SELECT name, description, service_type, version FROM tenders WHERE id = $1", tenderId).Scan(&currentParams.name, &currentParams.description, &currentParams.serviceType, &currentParams.version)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	_, err = tx.Exec("INSERT INTO tenders_history (tender_id, name, description, service_type,version) VALUES ($1, $2, $3,$4,$5)", tenderId, currentParams.name, currentParams.description, currentParams.serviceType, currentParams.version)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	historyParams := struct {
		name         string
		description  string
		service_type string
	}{}

	row := tx.QueryRow(getHistoryTenderQuery, tenderId, targetVersion)
	err = row.Scan(&historyParams.name, &historyParams.description, &historyParams.service_type)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenderVersionNotFound
		}
		return nil, err
	}

	tender := Tender{}

	row = tx.QueryRow(rollbackTenderQuery, historyParams.name, historyParams.description, historyParams.service_type, tenderId)
	err = row.Scan(&tender.Id, &tender.Name, &tender.Description, &tender.Status, &tender.ServiceType, &tender.Version, &tender.CreatedAt)

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
	return &tender, nil

}
