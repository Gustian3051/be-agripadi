package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gustian305/backend/config"
	docs "github.com/gustian305/backend/docs"
	"github.com/gustian305/backend/internal/handler"
	"github.com/gustian305/backend/internal/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(
	cfg *config.Config,
	messageHandler *handler.MessageHandler,
	conversationHandler *handler.ConversationHandler,
	uploadHandler *handler.UploadHandler,
	profileHandler *handler.ProfileHandler,
	authHandler *handler.AuthHandler,
) *gin.Engine {

	r := gin.New()

	// =====================================================
	// GLOBAL MIDDLEWARE
	// =====================================================

	r.Use(
		gin.Logger(),
		gin.Recovery(),
		corsMiddleware(cfg),
	)

	// =====================================================
	// SWAGGER DOCUMENTATION
	// =====================================================

	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	r.GET(
		"/",
		func(c *gin.Context) {
			c.Redirect(http.StatusTemporaryRedirect, "/swagger/index.html")
		},
	)

	r.GET(
		"/swagger/*any",
		ginSwagger.WrapHandler(
			swaggerFiles.Handler,
			ginSwagger.URL("/swagger/doc.json"),
			ginSwagger.DocExpansion("none"),
			ginSwagger.PersistAuthorization(true),
		),
	)

	// =====================================================
	// STATIC FILES
	// =====================================================

	r.Static(
		"/uploads",
		"./uploads",
	)

	// =====================================================
	// HEALTH CHECK
	// =====================================================

	r.GET(
		"/health",
		func(c *gin.Context) {

			c.JSON(
				http.StatusOK,
				gin.H{
					"status":  "success",
					"message": "ok",
				},
			)
		},
	)

	// =====================================================
	// API ROOT
	// =====================================================

	api := r.Group("/api")

	{
		authRequired := middleware.JWTAuth(cfg.Auth.JWTSecret)

		// =================================================
		// CHAT
		// =================================================

		api.POST(
			"/upload",
			authRequired,
			uploadHandler.UploadImage,
		)

		messages := api.Group(
			"/conversations/:conversation_id/messages",
		)
		messages.Use(authRequired)
		{
			messages.POST(
				"",
				messageHandler.CreateUserMessage,
			)

			messages.GET(
				"",
				messageHandler.ListConversationMessages,
			)
		}

		// =================================================
		// CONVERSATIONS
		// =================================================

		conversation := api.Group("/conversations")
		conversation.Use(authRequired)
		{
			conversation.GET("", conversationHandler.ListConversations)

			conversation.POST(
				"/delete-many",
				conversationHandler.DeleteManyConversations,
			)

			conversation.POST(
				"/delete-all",
				conversationHandler.DeleteAllConversations,
			)

			conversation.GET(
				"/:conversation_id",
				conversationHandler.GetOrCreateConversation,
			)

			conversation.POST(
				"",
				conversationHandler.GetOrCreateConversation,
			)

			conversation.DELETE(
				"/:conversation_id",
				conversationHandler.DeleteConversation,
			)

			conversation.DELETE(
				"",
				conversationHandler.DeleteManyConversations,
			)
		}

		// =================================================
		// PROFILE
		// =================================================

		profile := api.Group("/profile")
		profile.Use(authRequired)
		{
			profile.GET("", profileHandler.GetProfile)
			profile.PATCH("", profileHandler.UpdateProfile)
			profile.PATCH("/password", profileHandler.ChangePassword)
		}

		// =================================================
		// AUTH
		// =================================================

		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		protectedAuth := api.Group("/auth")
		protectedAuth.Use(authRequired)
		{
			protectedAuth.POST("/logout", authHandler.Logout)
		}
	}

	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {

	allowedOrigins := make(map[string]struct{})

	if cfg != nil {
		for _, origin := range cfg.CORS.AllowedOrigins {
			allowedOrigins[strings.TrimSpace(origin)] = struct{}{}
		}
	}

	allowAnyOrigin := cfg == nil ||
		!strings.EqualFold(
			strings.TrimSpace(cfg.App.Env),
			"production",
		)

	return func(c *gin.Context) {

		origin := strings.TrimSpace(
			c.GetHeader("Origin"),
		)

		if origin != "" {

			if allowAnyOrigin {

				c.Header(
					"Access-Control-Allow-Origin",
					origin,
				)

			} else {

				if _, ok := allowedOrigins[origin]; ok {
					c.Header(
						"Access-Control-Allow-Origin",
						origin,
					)
				}
			}
		}

		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header(
			"Access-Control-Allow-Headers",
			"Content-Type, Accept, Authorization",
		)
		c.Header(
			"Access-Control-Allow-Methods",
			"GET, POST, PUT, PATCH, DELETE, OPTIONS",
		)

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
