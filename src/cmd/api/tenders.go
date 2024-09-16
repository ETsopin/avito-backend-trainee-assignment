package main

import (
	"avitotask/internal/data"
	"errors"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var availableServices = map[string]bool{
	"Construction": true,
	"Delivery":     true,
	"Manufacture":  true,
}

var availableStatuses = map[string]bool{
	"Created":   true,
	"Published": true,
	"Closed":    true,
}

type tenderInput struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	ServiceType     string `json:"serviceType"`
	OrganizationID  string `json:"organizationId"`
	CreatorUsername string `json:"creatorUsername"`
}

func (tender tenderInput) validate() error {
	if tender.Name == "" || tender.Description == "" || tender.ServiceType == "" || tender.OrganizationID == "" || tender.CreatorUsername == "" {
		return errors.New("empty fields are not permitted")
	}

	if utf8.RuneCountInString(tender.Name) > 100 {
		return errors.New("tenderName cannot be longer than 100 symbols")
	}

	if utf8.RuneCountInString(tender.Description) > 500 {
		return errors.New("tenderDescription cannot be longer than 500 symbols")
	}
	if utf8.RuneCountInString(tender.OrganizationID) > 100 {
		return errors.New("tenderOrganizationId cannot be longer than 100 symbols")
	}

	if _, ok := availableServices[tender.ServiceType]; !ok {
		return ErrWrongService(tender.ServiceType)
	}

	return nil
}

func tryGetIntQuery(value []string) (int, error) {
	if len(value) != 1 {
		return 0, errors.New("query value must contain only one integer value")
	}
	res, err := strconv.Atoi(value[0])

	if err != nil {
		return 0, err
	}

	if res < 1 {
		return 0, errors.New("query integer value must be positive")
	}

	return res, nil
}

func tryGetUsernameQuery(value []string) (string, error) {
	if len(value) != 1 {
		return "", errors.New("username value must contain only one string value")
	}
	res := string(value[0])

	if res == "" {
		return "", errors.New("username must not be blank")
	}

	return res, nil
}

func validateServiceTypeQuery(value []string) error {
	if len(value) < 1 && len(value) > 3 {
		return errors.New("the amount of service types must be between 1 and 3")
	}

	for _, s := range value {
		if _, ok := availableServices[s]; !ok {
			return ErrWrongService(s)
		}
	}

	return nil
}

func tryGetStatusQuery(value []string) (string, error) {
	if len(value) != 1 {
		return "", errors.New("there can only be 1 status in request")
	}

	for _, s := range value {
		if _, ok := availableStatuses[s]; !ok {
			return "", ErrWrongStatus(s)
		}
	}

	return string(value[0]), nil
}

func (app *application) createNewTenderHandler(w http.ResponseWriter, r *http.Request) {
	tenderInput := tenderInput{}

	err := readJSON(w, r, &tenderInput)

	if err != nil {
		badRequestResponse(w, r, err)
		return
	}

	err = tenderInput.validate()

	if err != nil {
		badRequestResponse(w, r, err)
		return
	}

	userId, err := app.models.Tenders.GetUserID(tenderInput.CreatorUsername)

	if err != nil {
		if errors.Is(err, data.ErrUsernameNotFound) {
			unauthorizedResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	organizationId, err := app.models.Tenders.GetUserOrganization(userId)

	if err != nil {
		if errors.Is(err, data.ErrOrganizationNotFound) {
			forbiddenResponse(w, r, data.ErrNoRights)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	if organizationId != tenderInput.OrganizationID {
		forbiddenResponse(w, r, data.ErrNoRights)
		return
	}

	tenderOutput := data.Tender{
		Id:             uuid.New().String(),
		Name:           tenderInput.Name,
		Description:    tenderInput.Description,
		Status:         "Created",
		ServiceType:    tenderInput.ServiceType,
		OrganizationId: tenderInput.OrganizationID,
		Version:        1,
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	err = app.models.Tenders.InsertTender(&tenderOutput)
	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

	err = writeJSON(w, r, http.StatusOK, tenderOutput, nil)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

}

func (app *application) getTendersHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := struct {
		limit       int32
		offset      int32
		serviceType []string
	}{
		limit:       5,
		offset:      0,
		serviceType: []string{"", "", ""},
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
	serviceType, found := q["service_type"]
	if found {
		err := validateServiceTypeQuery(serviceType)
		if err != nil {
			badRequestResponse(w, r, err)
			return
		}
		for i, v := range serviceType {
			params.serviceType[i] = v
		}
	}

	tenders, err := app.models.Tenders.GetTenders(params.limit, params.offset, params.serviceType)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

	err = writeJSON(w, r, http.StatusOK, tenders, nil)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getMyTendersHandler(w http.ResponseWriter, r *http.Request) {
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
	organizationId, err := app.models.Tenders.GetUserOrganization(userId)
	if err != nil {
		if errors.Is(err, data.ErrOrganizationNotFound) {
			forbiddenResponse(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}

	tenders, err := app.models.Tenders.GetMyTenders(params.limit, params.offset, organizationId)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

	err = writeJSON(w, r, http.StatusOK, tenders, nil)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getStatusHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]

	_, err := uuid.Parse(tenderId)

	if err != nil {
		notFoundError(w, r, data.ErrTenderNotFound)
		return
	}

	params := struct {
		username string
	}{}

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

	tenderOrganizationId, err := app.models.Tenders.GetTenderOrganization(tenderId)

	if err != nil {
		if errors.Is(err, data.ErrTenderNotFound) {
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
		forbiddenResponse(w, r, data.ErrNoRights)
		return
	}

	status, err := app.models.Tenders.GetTenderStatus(tenderId)

	if err != nil {
		if errors.Is(err, data.ErrTenderNotFound) {
			notFoundError(w, r, err)
			return
		} else {
			serverErrorResponse(w, r, err)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(status))

}

func (app *application) changeStatusHandler(w http.ResponseWriter, r *http.Request) {
	params := struct {
		status   string
		username string
	}{}
	q := r.URL.Query()
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]
	_, err := uuid.Parse(tenderId)

	if err != nil {
		notFoundError(w, r, data.ErrTenderNotFound)
		return
	}

	status, found := q["status"]
	if found {
		parsedStatus, err := tryGetStatusQuery(status)
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

	tenderOrganizationId, err := app.models.Tenders.GetTenderOrganization(tenderId)

	if err != nil {
		if errors.Is(err, data.ErrTenderNotFound) {
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
		forbiddenResponse(w, r, data.ErrNoRights)
		return
	}

	tender, err := app.models.Tenders.ChangeTenderStatus(tenderId, params.status)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

	err = writeJSON(w, r, http.StatusOK, tender, nil)
	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

}

func (app *application) updateTenderHandler(w http.ResponseWriter, r *http.Request) {
	params := struct {
		username string
	}{}
	q := r.URL.Query()
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]
	_, err := uuid.Parse(tenderId)

	if err != nil {
		notFoundError(w, r, data.ErrTenderNotFound)
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

	tenderOrganizationId, err := app.models.Tenders.GetTenderOrganization(tenderId)

	if err != nil {
		if errors.Is(err, data.ErrTenderNotFound) {
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
		forbiddenResponse(w, r, data.ErrNoRights)
		return
	}

	tenderChanges := data.Tender{}
	err = readJSON(w, r, &tenderChanges)
	if err != nil {
		badRequestResponse(w, r, err)
		return
	}

	if _, ok := availableServices[tenderChanges.ServiceType]; len(tenderChanges.Name) > 100 || len(tenderChanges.Description) > 500 ||
		(!ok && tenderChanges.ServiceType != "") {
		badRequestResponse(w, r, errors.New("invalid parameters"))
		return
	}

	newTender, err := app.models.Tenders.UpdateTender(tenderId, tenderChanges)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

	err = writeJSON(w, r, http.StatusOK, newTender, nil)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

}

func (app *application) rollbackTenderHandler(w http.ResponseWriter, r *http.Request) {
	params := struct {
		username string
	}{}
	q := r.URL.Query()
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]
	_, err := uuid.Parse(tenderId)
	if err != nil {
		notFoundError(w, r, data.ErrTenderNotFound)
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

	tenderOrganizationId, err := app.models.Tenders.GetTenderOrganization(tenderId)

	if err != nil {
		if errors.Is(err, data.ErrTenderNotFound) {
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
		forbiddenResponse(w, r, data.ErrNoRights)
		return
	}

	tender, err := app.models.Tenders.RollbackTender(version, tenderId)

	if err != nil {
		if errors.Is(err, data.ErrTenderVersionNotFound) {
			notFoundError(w, r, err)
			return
		}
		serverErrorResponse(w, r, err)
		return
	}
	err = writeJSON(w, r, http.StatusOK, tender, nil)

	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

}
