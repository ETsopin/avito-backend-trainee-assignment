package data

import "database/sql"

type Models struct {
	Tenders TenderModel
	Bids    BidModel
	Tables  TableModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Tenders: TenderModel{
			DB: db,
		},
		Bids: BidModel{
			DB: db,
		},
		Tables: TableModel{
			DB: db,
		},
	}
}
