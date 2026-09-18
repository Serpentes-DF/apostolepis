package http

import (
	"errors"
	"net/http"

	"github.com/Serpentes-DF/apostolepis/internal/auth"
	"github.com/Serpentes-DF/apostolepis/internal/products"
	"github.com/Serpentes-DF/apostolepis/internal/users"
	"github.com/gin-gonic/gin"
)

type loginInput struct {
	IdentityToken string `json:"identity_token"`
}

type loginResponse struct {
	Token string     `json:"token"`
	User  users.User `json:"user"`
}

func (handler handler) login(c *gin.Context) {
	var input loginInput
	if !bindJSON(c, &input) {
		return
	}
	identity, err := handler.dependencies.GoogleTokens.Validate(c.Request.Context(), input.IdentityToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidIdentityToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid google identity token"})
			return
		}
		writeError(c, err)
		return
	}
	user, err := handler.dependencies.Users.GetByEmail(c.Request.Context(), identity.Email)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			user, err = handler.dependencies.Users.Create(c.Request.Context(), users.CreateInput{
				Name:  identity.Name,
				Email: identity.Email,
			})
			if err != nil && !errors.Is(err, users.ErrDuplicate) {
				writeError(c, err)
				return
			}
			if errors.Is(err, users.ErrDuplicate) {
				user, err = handler.dependencies.Users.GetByEmail(c.Request.Context(), identity.Email)
				if err != nil {
					writeError(c, err)
					return
				}
			}
		} else {
			writeError(c, err)
			return
		}
	}
	token, err := handler.dependencies.Tokens.Issue(user.ID, user.Email)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, loginResponse{Token: token, User: user})
}

func (handler handler) createProduct(c *gin.Context) {
	var input products.CreateInput
	if !bindJSON(c, &input) {
		return
	}
	product, err := handler.dependencies.Products.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, product)
}

func (handler handler) listProducts(c *gin.Context) {
	result, err := handler.dependencies.Products.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (handler handler) getProduct(c *gin.Context) {
	id, ok := objectID(c)
	if !ok {
		return
	}
	product, err := handler.dependencies.Products.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

func (handler handler) updateProduct(c *gin.Context) {
	id, ok := objectID(c)
	if !ok {
		return
	}
	var input products.UpdateInput
	if !bindJSON(c, &input) {
		return
	}
	product, err := handler.dependencies.Products.Update(c.Request.Context(), id, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

func (handler handler) deleteProduct(c *gin.Context) {
	id, ok := objectID(c)
	if !ok {
		return
	}
	if err := handler.dependencies.Products.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (handler handler) adjustStock(c *gin.Context) {
	id, ok := objectID(c)
	if !ok {
		return
	}
	var input struct {
		Delta int64 `json:"delta"`
	}
	if !bindJSON(c, &input) {
		return
	}
	stock, err := handler.dependencies.Inventory.Adjust(c.Request.Context(), id, input.Delta)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (handler handler) getStock(c *gin.Context) {
	id, ok := objectID(c)
	if !ok {
		return
	}
	stock, err := handler.dependencies.Inventory.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (handler handler) createUser(c *gin.Context) {
	var input users.CreateInput
	if !bindJSON(c, &input) {
		return
	}
	user, err := handler.dependencies.Users.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (handler handler) listUsers(c *gin.Context) {
	result, err := handler.dependencies.Users.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (handler handler) getUser(c *gin.Context) {
	id, ok := objectID(c)
	if !ok {
		return
	}
	user, err := handler.dependencies.Users.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (handler handler) getUserOrders(c *gin.Context) {
	id, ok := objectID(c)
	if !ok {
		return
	}
	user, err := handler.dependencies.Users.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": user.Orders})
}
