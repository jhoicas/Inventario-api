package dto

// AdminCreateUserRequest entrada para que super_admin cree un usuario de empresa.
type AdminCreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=1,max=200"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Status   string `json:"status" validate:"required,oneof=active inactive suspended"`
}

// AdminUpdateUserRequest entrada para que super_admin actualice un usuario de empresa.
type AdminUpdateUserRequest struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
	Password *string `json:"password,omitempty" validate:"omitempty,min=8"`
	Status   *string `json:"status,omitempty" validate:"omitempty,oneof=active inactive suspended"`
}
