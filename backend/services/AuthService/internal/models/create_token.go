package models

import "github.com/go-playground/validator/v10"

// ErrorResponse matches the shared error schema.

type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

var validate = validator.New()

// CreateJWTRequest represents POST /internal/create payload with validation rules.
// username: required, 3-20 chars, alphanumeric plus spaces.
// user_id optional UUID.

type CreateJWTRequest struct {
	UserID   string `json:"user_id" validate:"omitempty,uuid4"`
	Username string `json:"username" validate:"required,min=3,max=20,usernameFmt"`
}

// ValidateJWTRequest represents POST /internal/validate payload.

type ValidateJWTRequest struct {
	Token string `json:"token" validate:"required,jwt"`
}

// JWTResponse unified response for create (token) or validate (claims).

type JWTResponse struct {
	Token    string `json:"token,omitempty"`
	Valid    *bool  `json:"valid,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	IsGuest  *bool  `json:"is_guest,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Register custom validators.
func init() {
	_ = validate.RegisterValidation("usernameFmt", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		// allow letters, numbers, space
		for _, r := range value {
			if r == ' ' {
				continue
			}
			if r >= 'a' && r <= 'z' {
				continue
			}
			if r >= 'A' && r <= 'Z' {
				continue
			}
			if r >= '0' && r <= '9' {
				continue
			}
			return false
		}
		return true
	})
	_ = validate.RegisterValidation("jwt", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		// rudimentary structural check: three dot-separated base64url parts
		parts := 0
		for _, c := range value {
			if c == '.' {
				parts++
			}
		}
		return parts == 2
	})
}

// Validate helper functions returning error map.
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
