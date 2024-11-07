package main

import (
	"fmt"
	"regexp"
)

const (
	minPassword     = 6
	minEmailLen     = 2
	minFirstNameLen = 2
	minLastNameLen  = 2
	minPhoneLen     = 10
	iniqueIDLen     = 36
)

func (app *Config) ValidateLoginInput(req LoginPayload) map[string]string {

	errors := map[string]string{}
	if len(req.Email) < minEmailLen {
		errors["email"] = fmt.Sprintf("%s is required", "email")
	}

	if len(req.Password) < minPassword {
		errors["password"] = fmt.Sprintf("password length should be at least %d characters", minPassword)
	}

	if !isEmailValid(req.Email) {
		errors["email"] = fmt.Sprintf("%s supplied is invalid", "email")
	}

	return errors
}

func (app *Config) ValidataSignupInput(req SignupPayload) map[string]string {

	errors := map[string]string{}
	if len(req.FirstName) < minFirstNameLen {
		errors["first_name"] = fmt.Sprintf("first name length should be at least %d characters", minFirstNameLen)
	}

	if len(req.LastName) < minLastNameLen {
		errors["last_name"] = fmt.Sprintf("last name length should be at least %d characters", minLastNameLen)
	}

	if len(req.Phone) < minPhoneLen {
		errors["phone"] = fmt.Sprintf("phone length should be at least %d characters", minPhoneLen)
	}

	if len(req.Email) < minEmailLen {
		errors["email"] = fmt.Sprintf("%s is required", "email")
	}

	if !isEmailValid(req.Email) {
		errors["email"] = fmt.Sprintf("%s supplied is invalid", "email")
	}

	return errors
}

func isEmailValid(e string) bool {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return emailRegex.MatchString(e)
}
