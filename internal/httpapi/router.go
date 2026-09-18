package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/Serpentes-DF/apostolepis/internal/auth"
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

type TokenService interface {
	Issue(bson.ObjectID, string) (string, error)
	Parse(string) (auth.Identity, error)
}

type GoogleIdentityTokenValidator interface {
	Validate(context.Context, string) (users.User, error)
}

type Dependencies struct {
	Products     ProductService
	Inventory    InventoryService
	Users        UserService
	Tokens       TokenService
	GoogleTokens GoogleIdentityTokenValidator
	Ping         func(context.Context) error
}

type handler struct{ dependencies Dependencies }

const authenticatedUserKey = "authenticatedUser"

func NewRouter(dependencies Dependencies) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	handler := handler{dependencies: dependencies}

	router.GET("/health", handler.health)
	router.GET("/docs", serveDocs)
	router.GET("/docs/*path", serveDocs)
	api := router.Group("/v1")
	api.POST("/auth/login", handler.login)
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
	api.GET("/users/:id/orders", handler.requireAuthenticatedUser(), handler.requireSameUserOrAdmin(), handler.getUserOrders)

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

func (handler handler) authenticateUser(c *gin.Context) (users.User, bool) {
	token, err := auth.BearerToken(c.GetHeader("Authorization"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
		return users.User{}, false
	}
	identity, err := handler.dependencies.Tokens.Parse(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return users.User{}, false
	}
	user, err := handler.dependencies.Users.Get(c.Request.Context(), identity.UserID)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user credentials"})
			return users.User{}, false
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return users.User{}, false
	}
	if user.Email != identity.Email {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user credentials"})
		return users.User{}, false
	}
	return user, true
}

func (handler handler) requireAuthenticatedUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := handler.authenticateUser(c)
		if !ok {
			return
		}
		c.Set(authenticatedUserKey, user)
		c.Next()
	}
}

func (handler handler) requireSameUserOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := bson.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid object ID"})
			return
		}
		authenticated, ok := c.Get(authenticatedUserKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		user, ok := authenticated.(users.User)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		if !user.IsAdmin && user.ID != id {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user access denied"})
			return
		}
		c.Next()
	}
}

func (handler handler) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		authenticated, ok := c.Get(authenticatedUserKey)
		if !ok {
			user, authenticatedOK := handler.authenticateUser(c)
			if !authenticatedOK {
				return
			}
			authenticated = user
			ok = true
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		user, ok := authenticated.(users.User)
		if !ok {
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
