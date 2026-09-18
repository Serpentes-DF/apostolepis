package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/Serpentes-DF/apostolepis/internal/inventory"
	"github.com/Serpentes-DF/apostolepis/internal/products"
	"github.com/Serpentes-DF/apostolepis/internal/users"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ProductService interface {
	Create(context.Context, products.CreateInput) (products.Product, error)
	Get(context.Context, bson.ObjectID) (products.Product, error)
	List(context.Context) ([]products.Product, error)
	Update(context.Context, bson.ObjectID, products.UpdateInput) (products.Product, error)
	Delete(context.Context, bson.ObjectID) error
}

type InventoryService interface {
	Adjust(context.Context, bson.ObjectID, int64) (inventory.Inventory, error)
	Get(context.Context, bson.ObjectID) (inventory.Inventory, error)
}

type UserService interface {
	Create(context.Context, users.CreateInput) (users.User, error)
	Get(context.Context, bson.ObjectID) (users.User, error)
	GetByEmail(context.Context, string) (users.User, error)
	List(context.Context) ([]users.User, error)
}

type Dependencies struct {
	Products  ProductService
	Inventory InventoryService
	Users     UserService
	Ping      func(context.Context) error
}

type handler struct{ dependencies Dependencies }

func NewRouter(dependencies Dependencies) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	handler := handler{dependencies: dependencies}

	router.GET("/health", handler.health)
	api := router.Group("/api/v1")
	adminProducts := api.Group("/products")
	adminProducts.Use(handler.requireAdmin())
	adminProducts.POST("", handler.createProduct)
	adminProducts.PUT("/:id", handler.updateProduct)
	adminProducts.DELETE("/:id", handler.deleteProduct)
	adminProducts.POST("/:id/stock-adjustments", handler.adjustStock)
	api.GET("/products", handler.listProducts)
	api.GET("/products/:id", handler.getProduct)
	api.GET("/products/:id/stock", handler.getStock)
	api.GET("/users", handler.listUsers)
	api.GET("/users/:id", handler.getUser)

	return router
}

func (handler handler) health(c *gin.Context) {
	if err := handler.dependencies.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func objectID(c *gin.Context) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid object ID"})
		return bson.NilObjectID, false
	}
	return id, true
}

func bindJSON(c *gin.Context, destination any) bool {
	if err := c.ShouldBindJSON(destination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON request"})
		return false
	}
	return true
}

func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, products.ErrNotFound), errors.Is(err, users.ErrNotFound), errors.Is(err, inventory.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, products.ErrDuplicate), errors.Is(err, users.ErrDuplicate):
		status = http.StatusConflict
	case errors.Is(err, products.ErrInvalid), errors.Is(err, users.ErrInvalid),
		errors.Is(err, inventory.ErrInvalidDelta), errors.Is(err, inventory.ErrInsufficientStock):
		status = http.StatusUnprocessableEntity
	}
	message := err.Error()
	if status == http.StatusInternalServerError {
		message = "internal server error"
	}
	c.JSON(status, gin.H{"error": message})
}

func (handler handler) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		email := c.GetHeader("X-User-Email")
		if email == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing X-User-Email header"})
			return
		}
		user, err := handler.dependencies.Users.GetByEmail(c.Request.Context(), email)
		if err != nil {
			if errors.Is(err, users.ErrNotFound) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "administrator privileges required"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		if !user.IsAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "administrator privileges required"})
			return
		}
		c.Next()
	}
}
