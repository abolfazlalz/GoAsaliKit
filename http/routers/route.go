package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goasali/toolkit/config"
	"github.com/goasali/toolkit/database"
	"github.com/goasali/toolkit/http/controllers"
	middlewares "github.com/goasali/toolkit/http/middleware"
	"github.com/goasali/toolkit/http/validations"
	"github.com/goasali/toolkit/storage"
	log "github.com/sirupsen/logrus"
	"strconv"
)

type Interface interface {
	Listen(*RouteModuleParams)
}

type RouteModule struct {
	Interface
}

func NewRouteModule() *RouteModule {
	return &RouteModule{}
}

type RouteModuleParams struct {
	Router *gin.RouterGroup
}

type Route struct {
	*gin.Engine
	appConfig *config.App
}

func SetupRouter(configFunctions ...ConfigFunc) *Route {
	c := getConfig(configFunctions...)
	appConfig, _ := config.GetApp()

	if c.mode != "" {
		appConfig.Mode = c.mode
	}
	if c.host != "" {
		appConfig.Host = c.host
	}
	if c.port != 0 {
		appConfig.Port = strconv.Itoa(c.port)
	}

	if c.mode != "" {
		gin.SetMode(c.mode)
	}
	router := gin.Default()

	for _, disk := range config.PublicDisks() {
		storage.DiskFromConfig(disk).ServeOnRoute(disk.Route, router)
	}

	router.Use(middlewares.Logging())
	router.Use(middlewares.Recovery())

	r := &Route{
		router,
		appConfig,
	}
	if db := c.db; db != nil {
		r.loadValidations()
	}

	return r
}

func (r *Route) loadValidations() {
	db, _ := database.Database()
	if err := validations.AddDatabase(db); err != nil {
		log.Fatalf("error during load database validation: %v", err)
	}
}

func (r *Route) AddApiRoutes(routes ...Interface) {
	group := r.Group("/api")
	group.Use(middlewares.JSONMiddleware())
	r.AddRoutesGroup(group, routes...)
}

func AddRouteView(route *gin.RouterGroup, controller controllers.ResourceController) {
	route.GET("/", controller.Index)
	route.GET("/create", controller.Create)
	route.POST("/", controller.Store)
	route.GET("/:id", controller.Show)
	route.GET("/:id/edit", controller.Edit)
	route.PUT("/:id", controller.Update)
	route.DELETE("/:id", controller.Delete)
	route.DELETE("/", controller.Destroy)
}

func (r *Route) AddRoutes(routePath string, routes ...Interface) {
	grp := r.Group(routePath)
	r.AddRoutesGroup(grp, routes...)
}

func (r *Route) AddRoutesGroup(group *gin.RouterGroup, routes ...Interface) {
	for _, route := range routes {
		route.Listen(&RouteModuleParams{group})
	}
}

func (r *Route) Listen() error {
	addr := fmt.Sprintf("%s:%s", r.appConfig.Host, r.appConfig.Port)
	if err := r.Run(addr); err != nil {
		return err
	}

	return nil
}
