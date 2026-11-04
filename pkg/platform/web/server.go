package web

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"harns-platform/cmd/platform/config"
	"harns-platform/cmd/platform/options"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/audit"
	"harns-platform/pkg/emqx"
	"harns-platform/pkg/generic"
	"harns-platform/pkg/logging"
	"harns-platform/pkg/platform/auditapi"
	"harns-platform/pkg/platform/auth"
	"harns-platform/pkg/platform/files"
	"harns-platform/pkg/platform/httpx"
	"harns-platform/pkg/platform/identity"
	"harns-platform/pkg/platform/thingmodel"
	"harns-platform/pkg/platform/things"
	"harns-platform/pkg/telemetry"
	"harns-platform/pkg/timeseries"
)

type Server struct {
	*generic.Server
	*config.Config
	audit                      *audit.Module
	maintenance                *audit.Maintenance
	messagePipeline            *things.MessagePipeline
	propertyTriggerDispatchers []*things.PropertyTriggerDispatcher
	timeseriesSvc              *timeseries.Service
}

func NewServer(router *gin.Engine, o *options.Options, config *config.Config) (*Server, error) {
	if config.Telemetry.Enabled {
		router.Use(telemetry.Middleware(config.Telemetry.ServiceName))
	}
	router.Use(generic.CORS(config.CORS))
	router.Use(httpx.Recovery())
	router.HandleMethodNotAllowed = true
	router.NoRoute(func(c *gin.Context) {
		response.WriteError(c, http.StatusNotFound, response.ErrNotFound())
	})
	router.NoMethod(func(c *gin.Context) {
		response.WriteError(c, http.StatusMethodNotAllowed, response.ErrMethodNotAllowed())
	})

	s := &generic.Server{
		Router: router,
		Port:   o.Port,
	}

	server := &Server{
		Server: s,
		Config: config,
	}

	if err := server.InstallHandlers(); err != nil {
		return nil, err
	}

	return server, nil
}

func (s *Server) InstallHandlers() error {
	v1 := s.Router.Group("/api/v1")
	v1.Use(generic.BodyLimit(s.Config.MaxBodyBytes))
	v1.GET("/health", func(c *gin.Context) {
		response.WriteOK(c, gin.H{"status": "ok"})
	})
	v1.GET("/ready", readiness(s.Config))
	installOpenAPI(s.Router, v1)

	if s.Config.DB == nil {
		logging.Warnf("database not configured, identity endpoints disabled")
		return nil
	}

	if s.Config.Timeseries.Enabled {
		tsSvc, err := timeseries.NewService(s.Config.Timeseries)
		if err != nil {
			return fmt.Errorf("init timeseries service failed: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err = tsSvc.Start(ctx)
		cancel()
		if err != nil {
			return fmt.Errorf("start timeseries service failed: %w", err)
		}
		s.timeseriesSvc = tsSvc
	}

	identitySvc := identity.NewService(identity.NewStore(s.Config.DB))
	authSvc, err := auth.NewService(identitySvc, s.Config.Redis, s.Config.MinIO, s.Config.MinIOBucket, s.Config.Security)
	if err != nil {
		return fmt.Errorf("init auth service failed: %w", err)
	}
	identitySvc.SetSessionInvalidator(authSvc)

	var auditMW gin.HandlerFunc = func(c *gin.Context) { c.Next() }
	if s.Config.Audit.Enabled {
		s.audit = audit.NewModule(s.Config.DB, s.Config.Audit, audit.AppIOT,
			audit.WithMetrics(auditapi.PlatformMetrics{}))
		auditMW = auditapi.Middleware(s.audit, s.Config.Audit)
	}
	s.maintenance = audit.NewMaintenance(s.Config.DB, s.Config.Audit)
	reader := audit.NewReader(s.Config.DB, "")

	authGroup := v1.Group("/auth")
	authGroup.POST("/login", auditMW, httpx.WithOperation(httpx.OpLogin, "用户登录"), authSvc.Login)
	authGroup.POST("/logout", auditMW, httpx.WithOperation(httpx.OpLogout, "用户退出"), authSvc.Logout)
	authGroup.POST("/verificationCode", authSvc.VerificationCode)
	authGroup.GET("/publicKey", authSvc.PublicKey)
	authGroup.GET("/index", authSvc.Auth(), authSvc.Index)

	iot := v1.Group("/iot", authSvc.Auth(), auditMW)
	identity.InstallHandler(iot, identitySvc)
	thingModelSvc := thingmodel.NewService(thingmodel.NewStore(s.Config.DB), s.Config.Redis)
	thingmodel.InstallHandler(iot, thingModelSvc)
	emqxSvc := emqx.NewService(s.Config.EMQX)
	thingsSvc := things.NewService(
		things.NewStore(s.Config.DB),
		s.Config.Redis,
		time.Duration(s.Config.Things.OnlineTTLSeconds)*time.Second,
		emqxSvc,
	)
	things.InstallHandler(iot, thingsSvc)

	var propertyTS things.PropertyTimeseries
	var propertyLogWriter things.PropertyLogWriter
	if s.timeseriesSvc != nil {
		propertyTS = s.timeseriesSvc
		propertyLogWriter = s.timeseriesSvc
	}
	alarmTriggerDispatcher := things.NewPropertyTriggerDispatcher(things.PropertyTriggerConfig{
		KafkaEnabled: s.Config.Things.Message.KafkaEnabled,
		Brokers:      s.Config.Things.Message.Kafka.Brokers,
		Topic:        s.Config.Things.Message.Kafka.AlarmPropertyTriggerTopic,
		QueueSize:    s.Config.Things.Message.TriggerQueueSize,
	}, &things.LoggingPropertyTriggerHandler{Name: "alarm"})
	ruleTriggerDispatcher := things.NewPropertyTriggerDispatcher(things.PropertyTriggerConfig{
		KafkaEnabled: s.Config.Things.Message.KafkaEnabled,
		Brokers:      s.Config.Things.Message.Kafka.Brokers,
		Topic:        s.Config.Things.Message.Kafka.RulePropertyTriggerTopic,
		QueueSize:    s.Config.Things.Message.TriggerQueueSize,
	}, &things.LoggingPropertyTriggerHandler{Name: "rule"})
	s.propertyTriggerDispatchers = []*things.PropertyTriggerDispatcher{
		alarmTriggerDispatcher,
		ruleTriggerDispatcher,
	}
	triggerRouter := things.NewPropertyTriggerRouter(
		things.AllowAllPropertyTriggerMatcher{},
		alarmTriggerDispatcher,
		ruleTriggerDispatcher,
	)

	propertyProcessor := things.NewPropertyProcessor(thingModelSvc, propertyTS, thingsSvc, triggerRouter)
	propertyHandler := things.NewPropertyHandlerWithLogWriter(propertyProcessor, propertyLogWriter)
	messageRouter := things.NewMessageRouter(map[things.MessageKind]things.MessageHandler{
		things.MessageKindProperty:     propertyHandler,
		things.MessageKindEvent:        things.NewEventHandler(&things.LoggingEventProcessor{}),
		things.MessageKindCommandReply: things.NewCommandReplyHandler(&things.LoggingCommandReplyProcessor{}),
	})

	s.messagePipeline = things.NewMessagePipeline(things.MessagePipelineConfig{
		KafkaEnabled: s.Config.Things.Message.KafkaEnabled,
		Brokers:      s.Config.Things.Message.Kafka.Brokers,
		Topic:        s.Config.Things.Message.Kafka.Topic,
		GroupID:      s.Config.Things.Message.Kafka.GroupID,
		BatchSize:    s.Config.Things.Message.Kafka.BatchSize,
		MaxWait:      time.Duration(s.Config.Things.Message.Kafka.MaxWaitMs) * time.Millisecond,
	}, messageRouter)
	emqx := v1.Group("/iot")
	things.InstallEMQXHandler(emqx, thingsSvc, s.messagePipeline, s.Config.Things.AuthSecret)
	deviceHTTP := v1.Group("/iot")
	things.InstallHTTPReportHandler(deviceHTTP, thingsSvc, s.messagePipeline)

	if s.Config.MinIO != nil {
		filesSvc := files.NewService(s.Config.MinIO, s.Config.MinIOBucket, s.Config.Files)
		filesGroup := s.Router.Group("/api/v1/iot")
		filesGroup.Use(authSvc.Auth(), auditMW, generic.BodyLimit(s.Config.Files.MaxImageBytes))
		files.InstallHandler(filesGroup, filesSvc)
	}
	iot.GET("/auditLogs", httpx.WithOperation(httpx.OpQuery, "查询审计日志"), auditapi.List(reader))
	return nil
}

func (s *Server) Serve() (func(ctx context.Context), error) {
	var srv *http.Server
	if len(s.Config.CertFile) != 0 && len(s.Config.KeyFile) != 0 {
		x509KeyPair, err := tls.LoadX509KeyPair(s.Config.CertFile, s.Config.KeyFile)
		if err != nil {
			return nil, err
		}
		c := &tls.Config{
			Certificates: []tls.Certificate{x509KeyPair},
		}

		srv = &http.Server{
			Addr:              fmt.Sprintf(":%s", s.Port),
			Handler:           s.Router,
			TLSConfig:         c,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    1 << 20,
		}
		go func() {
			logging.Error(srv.ListenAndServeTLS("", ""))
		}()
	} else {
		srv = &http.Server{
			Addr:              fmt.Sprintf(":%s", s.Port),
			Handler:           s.Router,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    1 << 20,
		}
		go func() {
			logging.Error(srv.ListenAndServe())
		}()
	}

	pipelineCtx, pipelineCancel := context.WithCancel(context.Background())
	if s.messagePipeline != nil {
		s.messagePipeline.Start(pipelineCtx)
	}

	return func(ctx context.Context) {
		pipelineCancel()
		if s.messagePipeline != nil {
			_ = s.messagePipeline.Close()
		}
		for _, dispatcher := range s.propertyTriggerDispatchers {
			_ = dispatcher.Close()
		}
		if s.timeseriesSvc != nil {
			_ = s.timeseriesSvc.Close()
		}
		srv.SetKeepAlivesEnabled(false)
		// Stop accepting new requests and drain in-flight ones first.
		if err := srv.Shutdown(ctx); err != nil {
			logging.Error(err)
		}
		// Audit writer may still be persisting logs, so close it before the DB.
		if s.audit != nil {
			_ = s.audit.Close(ctx)
		}
		if s.maintenance != nil {
			s.maintenance.Close(ctx)
		}
		if s.Config.TelemetryShutdown != nil {
			tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := s.Config.TelemetryShutdown(tctx); err != nil {
				logging.Error(err)
			}
			cancel()
		}
		if s.Config.DB != nil {
			if err := s.Config.DB.Close(); err != nil {
				logging.Error(err)
			}
		}
		if s.Config.Redis != nil {
			if err := s.Config.Redis.Close(); err != nil {
				logging.Error(err)
			}
		}
	}, nil
}
