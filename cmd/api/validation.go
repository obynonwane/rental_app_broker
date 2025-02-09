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
	minIDLen        = 36
	minCommentLen   = 5
	tokenMinLen     = 30
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

func (app *Config) ValidateCreateStaffInput(req CreateStaffPayload) map[string]string {

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

func (app *Config) ValidateReplyRatingInput(req ReplyRatingPayload) map[string]string {

	errors := map[string]string{}
	if len(req.RatingID) < minFirstNameLen {
		errors["rating_id"] = fmt.Sprintf("rating id length should be at least %d characters", minIDLen)
	}
	if len(req.Comment) < minLastNameLen {
		errors["comment"] = fmt.Sprintf("comment length should be at least %d characters", minCommentLen)
	}

	if req.ParentReplyID != "" {
		if len(req.ParentReplyID) < minLastNameLen {
			errors["parent_reply_id"] = fmt.Sprintf("parent reply id length should be at least %d characters", minIDLen)
		}
	}

	return errors
}

func (app *Config) ValidateResetPasswordEmailInput(req ResetPasswordEmailPayload) map[string]string {
	errors := map[string]string{}
	if len(req.Email) < minEmailLen {
		errors["email"] = fmt.Sprintf("%s is required", "email")
	}

	if !isEmailValid(req.Email) {
		errors["email"] = fmt.Sprintf("%s supplied is invalid", "email")
	}

	return errors
}

func (app *Config) ValidateChangePasswordInput(req ChangePasswordPayload) map[string]string {

	errors := map[string]string{}

	if len(req.Token) < tokenMinLen {
		errors["token"] = fmt.Sprintf("token length should be at least %d characters", tokenMinLen)
	}

	if len(req.Password) < minPassword {
		errors["password"] = fmt.Sprintf("password length should be at least %d characters", minPassword)
	}

	if len(req.ConfirmPassword) < minPassword {
		errors["confirm_password"] = fmt.Sprintf("confirm password length should be at least %d characters", minPassword)
	}

	if req.Password != req.ConfirmPassword {
		errors["confirm_password"] = fmt.Sprintf("confirm password not equal to password supplied")
	}

	return errors
}
