package main

import (
	"fmt"
	"log"
	"net/http"
)

type ErrWrongService string

func (err ErrWrongService) Error() string {
	return fmt.Sprintf("service %s does not exist", string(err))
}

type ErrWrongStatus string

func (err ErrWrongStatus) Error() string {
	return fmt.Sprintf("status %s does not exist", string(err))
}

func errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	writeJSON(w, r, status, envelope{"reason": message}, nil)
}

func badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Println(err)
	errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Println(err)
	errorResponse(w, r, http.StatusInternalServerError, "the server encountered a problem and could not handle request")
}

func unauthorizedResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Println(err)
	errorResponse(w, r, http.StatusUnauthorized, "username is incorrect or does not exist")
}
func forbiddenResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Println(err)
	errorResponse(w, r, http.StatusForbidden, "user does not have rights for this action")
}

func notFoundError(w http.ResponseWriter, r *http.Request, err error) {
	log.Println(err)
	errorResponse(w, r, http.StatusNotFound, err.Error())
}
