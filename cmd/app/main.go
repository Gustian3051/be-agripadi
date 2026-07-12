package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gustian305/backend/config"
	"github.com/gustian305/backend/internal/handler"
	"github.com/gustian305/backend/internal/repository"
	"github.com/gustian305/backend/internal/router"
	"github.com/gustian305/backend/internal/service/app"
	authservice "github.com/gustian305/backend/internal/service/auth"
	"github.com/gustian305/backend/internal/service/cnn"
	"github.com/gustian305/backend/internal/service/consultation"
	expertSystem "github.com/gustian305/backend/internal/service/expert_system"
	"github.com/gustian305/backend/internal/service/expert_system/diagnose"
	expertLLM "github.com/gustian305/backend/internal/service/expert_system/llm"
	"github.com/gustian305/backend/internal/service/expert_system/pesticide"
	"github.com/gustian305/backend/internal/service/expert_system/rule"
	"github.com/gustian305/backend/internal/service/expert_system/session"
	"github.com/gustian305/backend/internal/service/expert_system/symptom"
	"github.com/gustian305/backend/internal/service/expert_system/workflow"
	profileservice "github.com/gustian305/backend/internal/service/profile"
)

// @title AgriPadi Backend API
// @version 1.0
// @description Dokumentasi REST API backend AgriPadi untuk autentikasi, profil, upload gambar, percakapan, dan alur diagnosis hama padi.
// @termsOfService http://swagger.io/terms/
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Masukkan token JWT dengan format: Bearer <token>
func main() {
	cfg, err := config.LoadConfig(
		"./config.yml",
	)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if strings.EqualFold(strings.TrimSpace(cfg.App.Env), "production") {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := config.ConnectPostgres(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	attachmentRepo := repository.NewAttachmentRepository(db)
	conversationRepo := repository.NewConversationRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	userRepo := repository.NewUserRepository(db)
	expertSessionRepo := repository.NewExpertSessionRepository(db)
	expertCatalogRepo := repository.NewExpertCatalogRepository(db)

	attachmentService := app.NewAttachmentAppService(attachmentRepo)
	_ = attachmentService

	conversationService := app.NewConversationAppService(conversationRepo)
	messageService := app.NewMessageAppService(messageRepo, conversationRepo)
	authService := authservice.NewService(userRepo, cfg.Auth.JWTSecret)
	profileService := profileservice.NewService(userRepo)
	var cnnService *cnn.Service
	cnnMode := strings.ToLower(strings.TrimSpace(cfg.CNN.Mode))
	cnnURL := strings.TrimSpace(cfg.CNN.URL)
	if cnnMode == "" {
		cnnMode = "server_api"
	}
	if cnnMode != "on_device_flutter" && cnnURL != "" {
		cnnService = cnn.NewService(cnnURL)
	}
	consultationLLMClient := consultation.NewOpenAICompatibleClient(
		cfg.LLM.APIURL,
		cfg.LLM.APIKey,
		cfg.LLM.Model,
		cfg.LLM.Timeout,
	)
	consultationService := consultation.NewConsultationService(
		consultationLLMClient,
	)
	expertRecommendationLLMService := expertLLM.NewRecommendationService(
		consultationLLMClient,
	)

	sessionService := session.NewSessionService(expertSessionRepo)
	sessionLoader := session.NewSessionLoader(expertSessionRepo)
	sessionUpdater := session.NewSessionUpdater(expertSessionRepo)
	symptomNormalizer := symptom.NewNormalizerService()
	symptomMatcher := symptom.NewMatcherService(
		symptomNormalizer,
		expertCatalogRepo,
	)

	diagnoseService := diagnose.NewDiagnoseService(
		diagnose.NewStartService(sessionUpdater, sessionService),
		diagnose.NewImageService(),
		diagnose.NewSymptomService(
			sessionLoader,
			sessionUpdater,
			symptomNormalizer,
			symptomMatcher,
		),
		diagnose.NewSeverityService(sessionLoader, sessionUpdater),
		diagnose.NewGrowthStageService(sessionLoader, sessionUpdater),
		diagnose.NewFinalizeService(sessionLoader, sessionUpdater),
	)

	pesticideSafetyService := pesticide.NewSafetyService()
	pesticideResolver := pesticide.NewResolverService(
		expertCatalogRepo,
		pesticideSafetyService,
	)

	ruleScorer := rule.NewScorerService()
	ruleService := rule.NewService(
		rule.NewMatcherService(
			expertCatalogRepo,
			nil,
		),
		ruleScorer,
		rule.NewResolverService(
			ruleScorer,
		),
	)

	recommendationService := expertSystem.NewRecommendationService(
		expertCatalogRepo,
		expertCatalogRepo,
		expertCatalogRepo,
		ruleService,
		pesticideResolver,
		pesticideSafetyService,
		expertRecommendationLLMService,
	)

	expertService := expertSystem.NewExpertSystemService(
		workflow.NewRouterService(),
		expertCatalogRepo,
		sessionService,
		sessionLoader,
		sessionUpdater,
		diagnoseService,
		recommendationService,
	)

	chatService := app.NewChatAppService(
		expertService,
		consultationService,
		messageService,
		conversationService,
		cnnService,
		"./uploads",
	)

	messageHandler := handler.NewMessageHandler(chatService)
	conversationHandler := handler.NewConversationHandler(conversationService)
	profileHandler := handler.NewProfileHandler(profileService)
	authHandler := handler.NewAuthHandler(authService)

	uploadHandler := handler.NewUploadHandler("./uploads", cnnService, cfg.CNN.Mode)

	r := router.NewRouter(
		cfg,
		messageHandler,
		conversationHandler,
		uploadHandler,
		profileHandler,
		authHandler,
	)

	server := &http.Server{
		Addr:              buildAddress(cfg.App.Port),
		Handler:           r,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("server started on %s", server.Addr)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func buildAddress(port int) string {
	if port <= 0 {
		port = 8080
	}

	return fmt.Sprintf(":%d", port)
}
