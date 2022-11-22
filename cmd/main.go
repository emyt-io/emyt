package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/emyt-io/emyt/config"
	"github.com/emyt-io/emyt/db"
	"github.com/emyt-io/emyt/models"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm"
)

const AppYamlFilename = "app.yaml"

func load() config.Config {
	var cfg config.Config
	// read configuration from the file and environment variables
	if err := cleanenv.ReadConfig(AppYamlFilename, &cfg); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	return cfg
}

var hosts = map[string]*models.Host{}
var redirectUrls = map[string]*models.RedirectUrl{}

func customHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	c.Logger().Error(err)

	if code == 401 {
		c.Response().Header().Add("X-Authorized", "false")
		c.Response().Header().Add("X-Next-Page", c.Request().RequestURI)
	}

	errorPage := fmt.Sprintf("views/%d.html", code)
	if err := c.File(errorPage); err != nil {
		c.Logger().Error(err)
	}
}

func main() {
	doSeed := flag.Bool("seed", false, "Seeds the database")
	flag.Parse()

	if *doSeed {
		fmt.Println("About to seed the database.")
		models.JsonSeed()

		fmt.Println("Database seeded. Exiting.")
		os.Exit(0)
	}

	// Load ENV
	cfg := load()
	// Init DB
	db.Init()
	// Hosts
	for _, service := range cfg.Services {
		// Service Target
		tenant := echo.New()
		var targets []*middleware.ProxyTarget
		skip := !service.UseAuth
		serviceName := service.Name

		tenant.HTTPErrorHandler = customHTTPErrorHandler
		tenant.Use(loadAuthorizationHeader)
		tenant.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
			Skipper: func(ctx echo.Context) bool { return skip },
			Realm:   serviceName,
			Validator: func(username, password string, ctx echo.Context) (bool, error) {
				user, err := models.GetUserByUsername(username)
				if err != nil {
					errors.Is(err, gorm.ErrRecordNotFound)
					return false, echo.ErrUnauthorized
				}

				err = models.VerifyPassword(user.Password, password)
				if err != nil {
					return false, echo.ErrUnauthorized
				}

				return true, nil
			},
		}))

		// Service Config
		if service.Type == "proxy" {
			// Web endpoint
			urlS, err := url.Parse(service.EgressUrl)
			if err != nil {
				tenant.Logger.Fatal(err)
			}
			targets = append(targets, &middleware.ProxyTarget{
				URL: urlS,
			})
			tenant.Use(middleware.Proxy(middleware.NewRoundRobinBalancer(targets)))
			tenant.GET("/*", func(c echo.Context) error {
				return c.String(http.StatusOK, "Tenant:"+c.Request().Host)
			})
			hosts[service.IngressUrl] = &models.Host{Echo: tenant}
		} else if service.Type == "static" {
			// Static endpoint
			tenant.Use(middleware.GzipWithConfig(middleware.GzipConfig{
				Level: 5,
			}))
			tenant.Use(expiresServerHeader)
			tenant.Use(middleware.BodyLimit("10M"))
			tenant.Use(middleware.StaticWithConfig(middleware.StaticConfig{
				Root:   service.EgressUrl,
				Browse: true,
				HTML5:  true,
			}))
			// Add to Hosts
			hosts[service.IngressUrl] = &models.Host{Echo: tenant}
		} else if service.Type == "redirect" {
			// Redirect endpoint
			urlS, err := url.Parse(service.EgressUrl)
			if err != nil {
				tenant.Logger.Fatal(err)
			}
			redirectUrls[service.IngressUrl] = &models.RedirectUrl{
				URL: urlS,
			}
			tenant.Any("/*", func(c echo.Context) error {
				redirectUrl := redirectUrls[c.Request().Host].URL.String()
				return c.Redirect(http.StatusMovedPermanently, redirectUrl)
			})
			hosts[service.IngressUrl] = &models.Host{Echo: tenant}
		}
	}

	//---------
	// ROOT
	//---------
	server := echo.New()
	server.Use(middleware.Recover())
	server.GET("/status", func(c echo.Context) error {
		return c.String(http.StatusOK, "{\"success\":\"ok\"}")
	})
	hosts[cfg.StatusHost+":"+cfg.ProxyListenPort] = &models.Host{Echo: server}

	// Server
	e := echo.New()
	e.Use(middleware.Logger())
	e.Logger.SetOutput(&lumberjack.Logger{
		Filename:   cfg.Logfile,
		MaxSize:    100, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	})
	e.Any("/*", func(c echo.Context) (err error) {
		req := c.Request()
		res := c.Response()
		host := hosts[req.Host]
		if host == nil {
			err = echo.ErrNotFound
			e.Logger.Info("Resource Not found - " + req.Host)
		} else {
			host.Echo.ServeHTTP(res, req)
		}

		return
	})
	// 4 Terabyte limit
	e.Use(middleware.BodyLimit("4T"))

	// Start server with Graceful Shutdown WITH CERT
	//go func() {
	//	if err := e.StartTLS(":"+cfg.SSLPort,
	//		"/etc/emty/ssl/server.crt",
	//		"/etc/emyt/ssl/server.key"); err != nil && err != http.ErrServerClosed {
	//		e.Logger.Fatal("shutting down the server")
	//	}
	//}()

	// Start server with Graceful Shutdown WITHOUT CERT
	go func() {
		if err := e.Start(":9000"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

// If the Authorization is not sent as header but present as a cookie, loads it among the headers
func loadAuthorizationHeader(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		auth := c.Request().Header.Get(echo.HeaderAuthorization)
		if auth != "" {
			// Skip if found.
			return next(c)
		}

		cookie, err := c.Cookie(echo.HeaderAuthorization)
		if err != nil {
			// The soley prupose is copy the header; if we can't then skip.
			return next(c)
		}

		if time.Now().Before(cookie.Expires) {
			return next(c)
		}
		c.Request().Header.Set(echo.HeaderAuthorization, cookie.Value)

		return next(c)
	}
}

// ServerHeader middleware adds a `Server` header to the response.
func expiresServerHeader(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "public, max-age=3600")
		return next(c)
	}
}
