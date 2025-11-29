package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"openvdo/internal/models"
	"openvdo/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type UserHandler struct {
	db          *sql.DB
	redisClient *redis.Client
}

func NewUserHandler(db *sql.DB, redisClient *redis.Client) *UserHandler {
	return &UserHandler{
		db:          db,
		redisClient: redisClient,
	}
}

// CreateUser creates a new user
// @Summary Create a new user
// @Description Create a new user with email and password
// @Tags users
// @Accept json
// @Produce json
// @Param user body models.CreateUserRequest true "User data"
// @Success 201 {object} response.Response{data=models.UserResponse}
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var existingEmail string
	checkQuery := "SELECT email FROM users WHERE email = $1"
	err := h.db.QueryRow(checkQuery, req.Email).Scan(&existingEmail)
	if err == nil {
		response.Error(c, http.StatusConflict, "User with this email already exists")
		return
	}

	passwordHash := req.Password

	query := `
		INSERT INTO users (email, password_hash, first_name, last_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, first_name, last_name, created_at, updated_at`

	var user models.User
	err = h.db.QueryRow(query, req.Email, passwordHash, req.FirstName, req.LastName, time.Now(), time.Now()).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		response.InternalServerError(c, "Failed to create user")
		return
	}

	response.SuccessWithMessage(c, http.StatusCreated, user.ToResponse(), "User created successfully")
}

// GetUsers returns all users
// @Summary Get all users
// @Description Get a list of all users
// @Tags users
// @Produce json
// @Success 200 {object} response.Response{data=[]models.UserResponse}
// @Failure 500 {object} response.Response
// @Router /users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	query := "SELECT id, email, first_name, last_name, created_at, updated_at FROM users ORDER BY created_at DESC"
	rows, err := h.db.Query(query)
	if err != nil {
		response.InternalServerError(c, "Failed to fetch users")
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			response.InternalServerError(c, "Failed to scan user")
			return
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		response.InternalServerError(c, "Error after scanning users")
		return
	}

	var userResponses []models.UserResponse
	for _, user := range users {
		userResponses = append(userResponses, user.ToResponse())
	}

	response.Success(c, http.StatusOK, userResponses)
}

// GetUser returns a user by ID
// @Summary Get user by ID
// @Description Get a single user by their ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} response.Response{data=models.UserResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	query := "SELECT id, email, first_name, last_name, created_at, updated_at FROM users WHERE id = $1"
	var user models.User
	err = h.db.QueryRow(query, id).Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		response.NotFound(c, "User not found")
		return
	}
	if err != nil {
		response.InternalServerError(c, "Failed to fetch user")
		return
	}

	response.Success(c, http.StatusOK, user.ToResponse())
}

// UpdateUser updates a user by ID
// @Summary Update user
// @Description Update user information by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body models.UpdateUserRequest true "User update data"
// @Success 200 {object} response.Response{data=models.UserResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, updated_at = $3
		WHERE id = $4
		RETURNING id, email, first_name, last_name, created_at, updated_at`

	var user models.User
	err = h.db.QueryRow(query, req.FirstName, req.LastName, time.Now(), id).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		response.NotFound(c, "User not found")
		return
	}
	if err != nil {
		response.InternalServerError(c, "Failed to update user")
		return
	}

	response.SuccessWithMessage(c, http.StatusOK, user.ToResponse(), "User updated successfully")
}

// DeleteUser deletes a user by ID
// @Summary Delete user
// @Description Delete a user by their ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	query := "DELETE FROM users WHERE id = $1"
	result, err := h.db.Exec(query, id)
	if err != nil {
		response.InternalServerError(c, "Failed to delete user")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		response.NotFound(c, "User not found")
		return
	}

	response.SuccessWithMessage(c, http.StatusOK, nil, "User deleted successfully")
}