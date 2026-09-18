package httpapi

import (
	"net/http"

	"github.com/Serpentes-DF/apostolepis/internal/products"
	"github.com/Serpentes-DF/apostolepis/internal/users"
	"github.com/gin-gonic/gin"
)

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
