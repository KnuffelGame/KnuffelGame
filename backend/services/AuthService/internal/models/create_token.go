package models

import (
	"github.com/go-playground/validator/v10"
	"regexp"
)

// ErrorResponse matches the shared error schema.

type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

var (
	validate       = validator.New()
	usernameRegex  = regexp.MustCompile(`^(?=.*[A-Za-z0-9])[A-Za-z0-9 ]+$`)
	jwtStructRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)
)

// CreateJWTRequest now requires user_id and username.
// Guest flag removed; service only issues guest tokens implicitly.

type CreateJWTRequest struct {
	UserID   string `json:"user_id" validate:"required,uuid4"`
	Username string `json:"username" validate:"required,min=3,max=20,usernameFmt"`
}

type ValidateJWTRequest struct {
	Token string `json:"token" validate:"required,jwt"`
}

// CreateTokenResponse response for create token.

type CreateTokenResponse struct {
	Token string `json:"token"`
}

// ValidateTokenResponse response for validate token.

type ValidateTokenResponse struct {
	Valid    bool   `json:"valid"`
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	IsGuest  bool   `json:"is_guest,omitempty"`
	Error    string `json:"error,omitempty"`
}

func init() {
	_ = validate.RegisterValidation("usernameFmt", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		// Allows letters, digits and spaces; requires at least one non-space alphanumeric.
		return usernameRegex.MatchString(value)
	})
	_ = validate.RegisterValidation("jwt", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		return jwtStructRegex.MatchString(value)
	})
}

func (r *CreateJWTRequest) Validate() map[string]string {
	errMap := map[string]string{}
	if err := validate.Struct(r); err != nil {
		for _, fe := range err.(validator.ValidationErrors) {
			errMap[fe.Field()] = fe.Tag()
		}
	}
	return errMap
}

func (r *ValidateJWTRequest) Validate() map[string]string {
	errMap := map[string]string{}
	if err := validate.Struct(r); err != nil {
		for _, fe := range err.(validator.ValidationErrors) {
			errMap[fe.Field()] = fe.Tag()
		}
	}
	return errMap
}
