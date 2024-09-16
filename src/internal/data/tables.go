package data

import "database/sql"

type TableModel struct {
	DB *sql.DB
}

func (m *TableModel) CreateTables() error {
	bidsQuery :=
		`
	create table if not exists bids
	(
		id          uuid    default uuid_generate_v4() not null,
		name        varchar(100)                       not null,
		description varchar(500)                       not null,
		status      varchar(50)                        not null,
		tender_id   uuid                               not null,
		author_type varchar(50)                        not null,
		author_id   uuid                               not null,
		version     integer default 1                  not null,
		created_at  timestamp with time zone
	);
	`
	bidsHistoryQuery :=
		`
	create table if not exists bids_history
	(
		id          uuid default uuid_generate_v4(),
		bid_id      uuid         not null,
		name        varchar(100) not null,
		description varchar(500) not null,
		version     integer      not null
	);
	`
	tendersQuery :=
		`
	create table if not exists tenders
	(
		id              uuid                     not null
			primary key,
		name            varchar(100)             not null,
		description     varchar(500)             not null,
		service_type    varchar(100)             not null,
		status          varchar(50)              not null,
		organization_id uuid
			references organization
				on delete cascade,
		version         integer default 1        not null
			constraint tenders_version_check
				check (version >= 1),
		created_at      timestamp with time zone not null
	);
	`
	tendersHistoryQuery :=
		`
	create table if not exists tenders_history
	(
		id           uuid    default uuid_generate_v4() not null
			primary key,
		tender_id    uuid,
		name         varchar(100),
		description  varchar(500),
		service_type varchar(100),
		version      integer default 1
	);
	`

	bidsApprovalsQuery :=
		`
	CREATE TABLE IF NOT EXISTS bids_approvals (
		id UUID DEFAULT uuid_generate_v4(),
		bid_id UUID NOT NULL,
		user_id UUID NOT NULL
	)
	`

	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(bidsQuery)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(bidsHistoryQuery)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(tendersQuery)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(tendersHistoryQuery)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(bidsApprovalsQuery)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil

}
