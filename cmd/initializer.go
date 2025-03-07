package main

import (
	"context"
	"fmt"
	"github.com/Libra-security/devops-server/internal/config"
	"github.com/Libra-security/devops-server/internal/db"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-metrics"
	"github.com/hashicorp/go-metrics/datadog"
	healthGo "github.com/hellofresh/health-go/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"strconv"
	"time"

	"net/http"
)

func initialize() {
	// Initialize zerolog logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msgf("Initializing application")
	// Load the configuration
	c, err := config.LoadConfig()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load configuration ")
		panic(err)
	}
	// Initialize metrics with statsd sink
	sink, err := datadog.NewDogStatsdSink(fmt.Sprintf("%s:%s", c.Statsd.Host, c.Statsd.Port), "")
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize statsd sink")
		panic(err)
	}
	sink.SetTags([]string{"service:" + c.Statsd.ServiceName})
	metricsConfig := metrics.DefaultConfig("")
	metricsConfig.EnableHostnameLabel = true
	_, err = metrics.NewGlobal(metricsConfig, sink)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize metrics")
		panic(err)
	}
	// Initialize the DB client.
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s", c.DB.Host, c.DB.User, c.DB.Password, c.DB.Database, c.DB.Port)
	client, err := db.NewClient(dsn)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize DB client")
		panic(err)
	}

	// Initialize the DB handler (this will auto-migrate the Grade model).
	handler, err := db.NewHandler(client)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize DB handler")
		panic(err)
	}
	// Initialize Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// Add the logger middleware to log the requests
	r.Use(logger.SetLogger(logger.WithLogger(func(_ *gin.Context, l zerolog.Logger) zerolog.Logger {
		return l.Output(gin.DefaultWriter).With().Logger()
	}), logger.WithSkipPath([]string{"/actuator/health"})))

	// Initialize health probe
	health, _ := healthGo.New(healthGo.WithComponent(healthGo.Component{
		Name: "devops-server",
	}), healthGo.WithChecks(healthGo.Config{
		Name:      "postgres",
		SkipOnErr: false,
		Check: func(ctx context.Context) error {
			return handler.Ping()
		},
	},
	))
	r.GET("/actuator/health", gin.WrapH(health.Handler()))

	// GET /grades - Retrieve all grades.
	r.GET("/api/v1/grades", func(c *gin.Context) {
		defer metrics.MeasureSince([]string{"get_grades_time"}, time.Now())
		grades, err := handler.GetGrades()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, grades)
	})

	// GET /grades/:id - Retrieve a grade by ID.
	r.GET("/api/v1/grades/:id", func(c *gin.Context) {
		tags := []metrics.Label{{Name: "layer", Value: "api"}, {Name: "operation", Value: "get_grades_id"}}
		defer metrics.MeasureSinceWithLabels([]string{"get_grades_id_time"}, time.Now(), tags)
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid grade ID"})
			return
		}
		if id < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID must be a positive number"})
			return
		}
		grade, err := handler.GetGradeByID(uint(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Grade not found"})
			return
		}
		c.JSON(http.StatusOK, grade)
	})

	// POST /grades - Create a new grade.
	r.POST("/api/v1/grades", func(c *gin.Context) {
		tags := []metrics.Label{{Name: "layer", Value: "api"}, {Name: "operation", Value: "post_grades"}}
		defer metrics.MeasureSinceWithLabels([]string{"post_grades_time"}, time.Now(), tags)
		var grade db.Grade
		if err := c.ShouldBindJSON(&grade); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := handler.CreateGrade(&grade); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, grade)
	})

	// PUT /grades/:id - Update an existing grade.
	r.PUT("/api/v1/grades/:id", func(c *gin.Context) {
		tags := []metrics.Label{{Name: "layer", Value: "api"}, {Name: "operation", Value: "put_grades_id"}}
		defer metrics.MeasureSinceWithLabels([]string{"put_grades_id_time"}, time.Now(), tags)
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid grade ID"})
			return
		}
		if id < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID must be a positive number"})
			return
		}
		existingGrade, err := handler.GetGradeByID(uint(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Grade not found"})
			return
		}
		var input db.Grade
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update the fields.
		existingGrade.StudentName = input.StudentName
		existingGrade.Email = input.Email
		existingGrade.Class = input.Class
		existingGrade.Grade = input.Grade

		if err := handler.UpdateGrade(existingGrade); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, existingGrade)
	})

	// DELETE /grades/:id - Delete a grade.
	r.DELETE("/api/v1/grades/:id", func(c *gin.Context) {
		tags := []metrics.Label{{Name: "layer", Value: "api"}, {Name: "operation", Value: "delete_grades_id"}}
		defer metrics.MeasureSinceWithLabels([]string{"delete_grades_id_time"}, time.Now(), tags)
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid grade ID"})
			return
		}
		if id < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID must be a positive number"})
			return
		}
		grade, err := handler.GetGradeByID(uint(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Grade not found"})
			return
		}
		if err := handler.DeleteGrade(grade); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Grade deleted"})
	})

	// GET /api/v1/grades/avg - Compute and return the average grade.
	r.GET("/api/v1/grades/avg", func(c *gin.Context) {
		tags := []metrics.Label{{Name: "layer", Value: "api"}, {Name: "operation", Value: "get_grades_avg"}}
		defer metrics.MeasureSinceWithLabels([]string{"get_grades_avg_time"}, time.Now(), tags)
		avg, err := handler.GetAverageGrade()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"average": avg})
	})

	// Run the server
	err = r.Run(":" + c.Server.Port)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start server")
		panic(err)
	}
}
