package main

import (
	"avitotask/internal/data"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var NULL_UUID = "00000000-0000-0000-0000-000000000000"

type BidInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TenderId    string `json:"tenderId"`
	AuthorType  string `json:"authorType"`
	AuthorId    string `json:"authorId"`
}

var availableBidStatuses = map[string]bool{
	"Created":   true,
	"Published": true,
	"Canceled":  true,
}

func tryGetBidStatusQuery(value []string) (string, error) {
	if len(value) != 1 {
		return "", errors.New("there can only be 1 status in request")
	}

	for _, s := range value {
		if _, ok := availableBidStatuses[s]; !ok {
			return "", ErrWrongStatus(s)
		}
	}

	return string(value[0]), nil
}

func tryGetDecisionQuery(value []string) (string, error) {
	if len(value) != 1 {
		return "", errors.New("there can only be 1 decision in request")
	}

	if value[0] == "Approved" || value[0] == "Rejected" {
		return value[0], nil
	}

	return "", errors.New("incorrect decision")
}

func (app *application) createBidHandler(w http.ResponseWriter, r *http.Request) {
	var bidInput BidInput

	err := readJSON(w, r, &bidInput)

	if err != nil {
		badRequestResponse(w, r, err)
		return
	}

	if bidInput.Name == "" || bidInput.Description == "" || bidInput.TenderId == "" || bidInput.AuthorType == "" || bidInput.AuthorId == "" {
		badRequestResponse(w, r, errors.New("all the values must be provided"))
		return
	}

	if bidInput.AuthorType != "User" && bidInput.AuthorType != "Organization" {
		badRequestResponse(w, r, errors.New("incorrect author type"))
		return
	}

	if _, err := uuid.Parse(bidInput.TenderId); err != nil {
		notFoundError(w, r, data.ErrTenderNotFound)
		return
	}

	if _, err := uuid.Parse(bidInput.AuthorId); err != nil {
		unauthorizedResponse(w, r, err)
		return

	}

	tender, err := app.models.Tenders.GetTenderById(bidInput.TenderId)

	if err != nil {
		if errors.Is(err, data.ErrTenderNotFound) {
			notFoundError(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}
	if tender.Status != "Published" {
		forbiddenResponse(w, r, errors.New("trying to bid on tender that is not published"))
		return
	}

	if bidInput.AuthorType == "User" {
		isInOrganization := true
		organizationId, err := app.models.Tenders.GetUserOrganization(bidInput.AuthorId)
		if err != nil {
			if errors.Is(err, data.ErrOrganizationNotFound) {
				isInOrganization = false
			} else {
				serverErrorResponse(w, r, err)
				return
			}
		}

		//CHECK IF USER IN THE SAME ORGANIZATION AS TENDER

		if isInOrganization && (organizationId == tender.OrganizationId) {
			forbiddenResponse(w, r, errors.New("trying to bid on your own tender"))
			return
		}
	} else {
		_, err = app.models.Tenders.GetOrganizationUsers(bidInput.AuthorId)
		if err != nil {
			if errors.Is(err, data.ErrUsernameNotFound) {
				unauthorizedResponse(w, r, err)
				return
			}
			serverErrorResponse(w, r, err)
			return
		}
		if bidInput.AuthorId == tender.OrganizationId {
			forbiddenResponse(w, r, errors.New("trying to bid on your own tender"))
			return
		}

	}

	bid := data.Bid{
		Id:          uuid.New().String(),
		Description: bidInput.Description,
		Name:        bidInput.Name,
		Status:      "Created",
		TenderId:    bidInput.TenderId,
		AuthorType:  bidInput.AuthorType,
		AuthorId:    bidInput.AuthorId,
		Version:     1,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	err = app.models.Bids.InsertBid(&bid)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

	err = writeJSON(w, r, http.StatusOK, bid, nil)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

}

func (app *application) getMyBidsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := struct {
		limit    int32
		offset   int32
		username string
	}{
		limit: 5,
	}
	limit, found := q["limit"]
	if found {
		parsedLimit, err := tryGetIntQuery(limit)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.limit = int32(parsedLimit)
	}

	offset, found := q["offset"]
	if found {
		parsedOffset, err := tryGetIntQuery(offset)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.offset = int32(parsedOffset)
	}
	username, found := q["username"]
	if found {
		parsedUsername, err := tryGetUsernameQuery(username)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.username = parsedUsername

	} else {
		badRequestResponse(w, r, errors.New("username must be provided"))
		return
	}

	userId, err := app.models.Tenders.GetUserID(params.username)
	if err != nil {
		if errors.Is(err, data.ErrUsernameNotFound) {
			unauthorizedResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	isInOrganization := true
	organizationId, err := app.models.Tenders.GetUserOrganization(userId)
	if err != nil {
		if errors.Is(err, data.ErrOrganizationNotFound) {
			isInOrganization = false
		} else {
			serverErrorResponse(w, r, err)
			return
		}

	}
	var bids []*data.Bid

	if isInOrganization {
		bids, err = app.models.Bids.GetMyBids(params.limit, params.offset, organizationId, userId)
	} else {
		bids, err = app.models.Bids.GetMyBids(params.limit, params.offset, NULL_UUID, userId)
	}

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

	err = writeJSON(w, r, http.StatusOK, bids, nil)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) changeBidStatusHandler(w http.ResponseWriter, r *http.Request) {
	params := struct {
		status   string
		username string
	}{}
	q := r.URL.Query()
	vars := mux.Vars(r)
	bidId := vars["bidId"]
	_, err := uuid.Parse(bidId)

	if err != nil {
		notFoundError(w, r, data.ErrBidNotFound)
		return
	}

	status, found := q["status"]
	if found {
		parsedStatus, err := tryGetBidStatusQuery(status)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.status = parsedStatus
	} else {
		badRequestResponse(w, r, errors.New("status must be provided"))
		return
	}

	username, found := q["username"]

	if found {
		parsedUsername, err := tryGetUsernameQuery(username)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.username = parsedUsername
	} else {
		badRequestResponse(w, r, errors.New("username must be provided"))
		return
	}

	currentBid, err := app.models.Bids.GetBidById(bidId)
	if err != nil {
		if errors.Is(err, data.ErrBidNotFound) {
			notFoundError(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	userId, err := app.models.Tenders.GetUserID(params.username)

	if err != nil {
		if errors.Is(err, data.ErrUsernameNotFound) {
			unauthorizedResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}
	isInOrganization := true
	organizationId, err := app.models.Tenders.GetUserOrganization(userId)

	if err != nil {
		if errors.Is(err, data.ErrOrganizationNotFound) {
			isInOrganization = false
		} else {
			serverErrorResponse(w, r, err)
			return
		}
	}
	validIds := []string{userId}
	if isInOrganization {
		validUsers, err := app.models.Tenders.GetOrganizationUsers(organizationId)
		if err != nil {
			serverErrorResponse(w, r, err)
			return
		}
		validIds = append(validIds, organizationId)
		validIds = append(validIds, validUsers...)
	}

	if !containsString(validIds, currentBid.AuthorId) {
		forbiddenResponse(w, r, errors.New("user is not responsible for this bid"))
		return
	}

	newBid, err := app.models.Bids.ChangeBidStatus(bidId, params.status)

	if err != nil {
		if errors.Is(err, data.ErrBidNotFound) {
			notFoundError(w, r, err)
			return
		}
		badRequestResponse(w, r, err)
		return
	}
	err = writeJSON(w, r, http.StatusOK, newBid, nil)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getBidStatusHandler(w http.ResponseWriter, r *http.Request) {
	params := struct {
		username string
	}{}
	q := r.URL.Query()
	vars := mux.Vars(r)
	bidId := vars["bidId"]
	_, err := uuid.Parse(bidId)

	if err != nil {
		notFoundError(w, r, data.ErrBidNotFound)
		return
	}
	username, found := q["username"]
	if found {
		parsedUsername, err := tryGetUsernameQuery(username)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.username = parsedUsername
	} else {
		badRequestResponse(w, r, errors.New("username must be provided"))
		return
	}

	userId, err := app.models.Tenders.GetUserID(params.username)

	if err != nil {
		if errors.Is(err, data.ErrUsernameNotFound) {
			unauthorizedResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	isInOrganization := true
	organizationId, err := app.models.Tenders.GetUserOrganization(userId)

	if err != nil {
		if errors.Is(err, data.ErrOrganizationNotFound) {
			isInOrganization = false
		} else {
			serverErrorResponse(w, r, err)
			return
		}
	}
	validIds := []string{userId}
	if isInOrganization {
		validUsers, err := app.models.Tenders.GetOrganizationUsers(organizationId)
		if err != nil {
			serverErrorResponse(w, r, err)
			return
		}
		validIds = append(validIds, organizationId)
		validIds = append(validIds, validUsers...)
	}

	currentBid, err := app.models.Bids.GetBidById(bidId)
	if err != nil {
		if errors.Is(err, data.ErrBidNotFound) {
			notFoundError(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	if !containsString(validIds, currentBid.AuthorId) {
		forbiddenResponse(w, r, errors.New("user is not responsible for this bid"))
		return
	}

	status, err := app.models.Bids.GetBidStatus(bidId)

	if err != nil {
		if errors.Is(err, data.ErrBidNotFound) {
			notFoundError(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(status))
}

func (app *application) getBidsForTenderHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := struct {
		limit    int32
		offset   int32
		username string
	}{
		limit: 5,
	}
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]
	limit, found := q["limit"]
	if found {
		parsedLimit, err := tryGetIntQuery(limit)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.limit = int32(parsedLimit)
	}

	offset, found := q["offset"]
	if found {
		parsedOffset, err := tryGetIntQuery(offset)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.offset = int32(parsedOffset)
	}
	username, found := q["username"]
	if found {
		parsedUsername, err := tryGetUsernameQuery(username)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.username = parsedUsername

	} else {
		badRequestResponse(w, r, errors.New("username must be provided"))
		return
	}

	userId, err := app.models.Tenders.GetUserID(params.username)
	if err != nil {
		if errors.Is(err, data.ErrUsernameNotFound) {
			unauthorizedResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	tenderOrganizationId, err := app.models.Tenders.GetTenderOrganization(tenderId)

	if err != nil {
		if errors.Is(err, data.ErrTenderNotFound) {
			unauthorizedResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	userOrganizationId, err := app.models.Tenders.GetUserOrganization(userId)

	if err != nil {
		if errors.Is(err, data.ErrOrganizationNotFound) {
			forbiddenResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	if userOrganizationId != tenderOrganizationId {
		forbiddenResponse(w, r, errors.New("only users responsible for organizations can view bids"))
		return
	}

	bids, err := app.models.Bids.GetBidsByTenderId(params.limit, params.offset, tenderId)

	if err != nil {
		if errors.Is(err, data.ErrBidOrTenderNotFound) {
			notFoundError(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	err = writeJSON(w, r, http.StatusOK, bids, nil)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

}

func (app *application) updateBidHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := struct {
		username string
	}{}
	vars := mux.Vars(r)
	bidId := vars["bidId"]
	username, found := q["username"]
	if found {
		parsedUsername, err := tryGetUsernameQuery(username)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.username = parsedUsername
	} else {
		badRequestResponse(w, r, errors.New("username must be provided"))
		return
	}

	var input struct {
		Name       string `json:"name"`
		Desription string `json:"description"`
	}
	err := readJSON(w, r, &input)
	if err != nil {
		badRequestResponse(w, r, err)
		return
	}

	bidInput := data.Bid{
		Name:        input.Name,
		Description: input.Desription,
	}

	userId, err := app.models.Tenders.GetUserID(params.username)

	if err != nil {
		if errors.Is(err, data.ErrUsernameNotFound) {
			unauthorizedResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	isInOrganization := true
	organizationId, err := app.models.Tenders.GetUserOrganization(userId)

	if err != nil {
		if errors.Is(err, data.ErrOrganizationNotFound) {
			isInOrganization = false
		} else {
			serverErrorResponse(w, r, err)
			return
		}
	}
	validIds := []string{userId}
	if isInOrganization {
		validUsers, err := app.models.Tenders.GetOrganizationUsers(organizationId)
		if err != nil {
			serverErrorResponse(w, r, err)
			return
		}
		validIds = append(validIds, organizationId)
		validIds = append(validIds, validUsers...)
	}
	currentBid, err := app.models.Bids.GetBidById(bidId)
	if err != nil {
		if errors.Is(err, data.ErrBidNotFound) {
			notFoundError(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	if !containsString(validIds, currentBid.AuthorId) {
		forbiddenResponse(w, r, errors.New("user is not responsible for this bid"))
		return
	}

	updatedBid, err := app.models.Bids.EditBid(bidId, bidInput)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

	err = writeJSON(w, r, http.StatusOK, updatedBid, nil)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

}

func (app *application) rollbackBidHandler(w http.ResponseWriter, r *http.Request) {
	params := struct {
		username string
	}{}
	q := r.URL.Query()
	vars := mux.Vars(r)
	bidId := vars["bidId"]
	_, err := uuid.Parse(bidId)
	if err != nil {
		notFoundError(w, r, data.ErrBidNotFound)
		return
	}
	version, err := strconv.Atoi(vars["version"])

	if err != nil || version < 1 {
		badRequestResponse(w, r, errors.New("version can only be an integer greater than 0"))
		return
	}

	username, found := q["username"]

	if found {
		parsedUsername, err := tryGetUsernameQuery(username)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.username = parsedUsername
	} else {
		badRequestResponse(w, r, errors.New("username must be provided"))
		return
	}

	userId, err := app.models.Tenders.GetUserID(params.username)

	if err != nil {
		if errors.Is(err, data.ErrUsernameNotFound) {
			unauthorizedResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	isInOrganization := true
	organizationId, err := app.models.Tenders.GetUserOrganization(userId)

	if err != nil {
		if errors.Is(err, data.ErrOrganizationNotFound) {
			isInOrganization = false
		} else {
			serverErrorResponse(w, r, err)
			return
		}
	}
	validIds := []string{userId}
	if isInOrganization {
		validUsers, err := app.models.Tenders.GetOrganizationUsers(organizationId)
		if err != nil {
			serverErrorResponse(w, r, err)
			return
		}
		validIds = append(validIds, organizationId)
		validIds = append(validIds, validUsers...)
	}

	currentBid, err := app.models.Bids.GetBidById(bidId)
	if err != nil {
		if errors.Is(err, data.ErrBidNotFound) {
			notFoundError(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	if !containsString(validIds, currentBid.AuthorId) {
		forbiddenResponse(w, r, errors.New("user is not responsible for this bid"))
		return
	}

	updatedBid, err := app.models.Bids.RollbackBid(version, bidId)

	if err != nil {
		if errors.Is(err, data.ErrBidVersionNotFound) {
			notFoundError(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	err = writeJSON(w, r, http.StatusOK, updatedBid, nil)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

}

func (app *application) submitDecisionHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := struct {
		username string
		decision string
	}{}
	vars := mux.Vars(r)
	bidId := vars["bidId"]
	_, err := uuid.Parse(bidId)
	if err != nil {
		notFoundError(w, r, data.ErrBidNotFound)
	}

	username, found := q["username"]
	if found {
		parsedUsername, err := tryGetUsernameQuery(username)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.username = parsedUsername
	} else {
		badRequestResponse(w, r, errors.New("username must be provided"))
		return
	}
	decision, found := q["decision"]
	if found {
		parsedDecision, err := tryGetDecisionQuery(decision)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		params.decision = parsedDecision
	} else {
		badRequestResponse(w, r, errors.New("decision must be provided"))
		return
	}

	bid, err := app.models.Bids.GetBidById(bidId)
	if err != nil {
		if errors.Is(err, data.ErrBidNotFound) {
			notFoundError(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}
	if bid.Status != "Published" {
		forbiddenResponse(w, r, errors.New("trying to send decision on inactive bid"))
		return
	}

	userId, err := app.models.Tenders.GetUserID(params.username)
	if err != nil {
		if errors.Is(err, data.ErrUsernameNotFound) {
			unauthorizedResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	userOrganizationId, err := app.models.Tenders.GetUserOrganization(userId)
	if err != nil {
		if errors.Is(err, data.ErrOrganizationNotFound) {
			unauthorizedResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	tender, err := app.models.Tenders.GetTenderById(bid.TenderId)

	if err != nil {
		if errors.Is(err, data.ErrTenderNotFound) {
			notFoundError(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
	}

	if tender.Status != "Published" {
		forbiddenResponse(w, r, errors.New("user trying to send decision for inactive tender"))
	}

	if tender.OrganizationId != userOrganizationId {
		forbiddenResponse(w, r, errors.New("user trying to send decision for tender he is not responsible for"))
		return
	}

	if params.decision == "Approved" {
		approvers, err := app.models.Bids.ApprovalCount(bidId)
		if err != nil {
			serverErrorResponse(w, r, err)
			return
		}
		if containsString(approvers, userId) {
			forbiddenResponse(w, r, errors.New("user has already approved"))
			return
		}
		users, err := app.models.Tenders.GetOrganizationUsers(userOrganizationId)
		if err != nil {
			serverErrorResponse(w, r, err)
			return
		}
		err = app.models.Bids.ApproveDecision(bidId, userId)
		if err != nil {
			serverErrorResponse(w, r, err)
			return
		}
		fmt.Println(len(approvers))
		if len(approvers)+1 >= 3 || len(approvers)+1 >= len(users) {
			_, err := app.models.Tenders.ChangeTenderStatus(tender.Id, "Closed")
			if err != nil {
				serverErrorResponse(w, r, err)
				return
			}

		}

	} else {
		newBid, err := app.models.Bids.RejectDecision(bidId)
		if err != nil {
			if errors.Is(err, data.ErrBidNotFound) {
				notFoundError(w, r, err)
				return
			}
			serverErrorResponse(w, r, err)
			return
		}
		err = writeJSON(w, r, http.StatusOK, newBid, nil)
		if err != nil {
			serverErrorResponse(w, r, err)
			return
		}
		return
	}

	err = writeJSON(w, r, http.StatusOK, bid, nil)
	if err != nil {
		serverErrorResponse(w, r, err)
	}

}
