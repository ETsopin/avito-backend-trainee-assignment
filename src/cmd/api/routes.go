package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (app *application) routes() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/api/ping", pingHandler).Methods("GET")
	router.HandleFunc("/api/tenders", app.getTendersHandler).Methods("GET")
	router.HandleFunc("/api/tenders/new", app.createNewTenderHandler).Methods("POST")
	router.HandleFunc("/api/tenders/my", app.getMyTendersHandler).Methods("GET")
	router.HandleFunc("/api/tenders/{tenderId}/status", app.getStatusHandler).Methods("GET")
	router.HandleFunc("/api/tenders/{tenderId}/status", app.changeStatusHandler).Methods("PUT")
	router.HandleFunc("/api/tenders/{tenderId}/edit", app.updateTenderHandler).Methods("PATCH")
	router.HandleFunc("/api/tenders/{tenderId}/rollback/{version}", app.rollbackTenderHandler).Methods("PUT")

	router.HandleFunc("/api/bids/new", app.createBidHandler).Methods("POST")
	router.HandleFunc("/api/bids/my", app.getMyBidsHandler).Methods("GET")
	router.HandleFunc("/api/bids/{bidId}/status", app.changeBidStatusHandler).Methods("PUT")
	router.HandleFunc("/api/bids/{bidId}/status", app.getBidStatusHandler).Methods("GET")
	router.HandleFunc("/api/bids/{tenderId}/list", app.getBidsForTenderHandler).Methods("GET")
	router.HandleFunc("/api/bids/{bidId}/edit", app.updateBidHandler).Methods("PATCH")
	router.HandleFunc("/api/bids/{bidId}/rollback/{version}", app.rollbackBidHandler).Methods("PUT")
	router.HandleFunc("/api/bids/{bidId}/submit_decision", app.submitDecisionHandler).Methods("PUT")
	return router
}
