package main

import (
	"fmt"
	"log"
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
	descLen         = 100
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

func (app *Config) ValidateCreateInventoryInput(category_id, sub_category_id, name, description, country_id, state_id, lga_id string, offer_price float64) map[string]string {
	errors := map[string]string{}

	if len(category_id) < iniqueIDLen {
		errors["category_id"] = fmt.Sprintf("category id length should be at least %d characters", minIDLen)
	}

	if len(sub_category_id) < iniqueIDLen {
		errors["sub_category_id"] = fmt.Sprintf("subcategory id length should be at least %d characters", minIDLen)
	}

	if len(description) < descLen {
		errors["description"] = fmt.Sprintf("description length should be at least %d characters", descLen)
	}

	if len(name) < minLastNameLen {
		errors["name"] = fmt.Sprintf("name length should be at least %d characters", minCommentLen)
	}

	if len(country_id) < iniqueIDLen {
		errors["country_id"] = fmt.Sprintf("country id length should be at least %d characters", minIDLen)
	}

	if len(state_id) < iniqueIDLen {
		errors["state_id"] = fmt.Sprintf("state id length should be at least %d characters", minCommentLen)
	}

	if len(lga_id) < iniqueIDLen {
		errors["lga_id"] = fmt.Sprintf("lga id length should be at least %d characters", minCommentLen)
	}

	// Offer price validation
	if offer_price <= 0 {
		errors["offer_price"] = "offer price must be greater than zero"
	} else if offer_price > 10000000 {
		errors["offer_price"] = "offer price seems too high"
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

	log.Printf("%v", errors)

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

func (app *Config) ValidateEmailRequestInput(req RequestPasswordVerificationEmailPayload) map[string]string {

	errors := map[string]string{}

	if len(req.Email) < minEmailLen {
		errors["email"] = fmt.Sprintf("%s is required", "email")
	}

	if !isEmailValid(req.Email) {
		errors["email"] = fmt.Sprintf("%s supplied is invalid", "email")
	}

	return errors
}

func (app *Config) ValidateSearchInput(req SearchPayload) map[string]string {

	errors := map[string]string{}
	if len(req.CountryID) < iniqueIDLen {
		errors["country_id"] = fmt.Sprintf("country id length should be at least %d characters", minIDLen)
	}

	if len(req.StateID) < iniqueIDLen {
		errors["state_id"] = fmt.Sprintf("state id length should be at least %d characters", minIDLen)
	}
	if len(req.LgaID) < iniqueIDLen {
		errors["lga_id"] = fmt.Sprintf("lgs id length should be at least %d characters", minIDLen)
	}

	return errors
}
