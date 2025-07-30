package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/goodwaysIT/go-oracle-dr-dashboard/handlers"
	"github.com/goodwaysIT/go-oracle-dr-dashboard/models"
	"github.com/goodwaysIT/go-oracle-dr-dashboard/util"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// --- Cached index.html content ---
var indexHTMLContent []byte
var indexHTMLModTime time.Time

func i18nMiddleware(bundle *i18n.Bundle) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.Query("lang")
		accept := c.GetHeader("Accept-Language")
		localizer := i18n.NewLocalizer(bundle, lang, accept)
		c.Set("localizer", localizer)
		c.Next()
	}
}

// watchConfig monitors the config file for changes and reloads it.
func watchConfig(configFile string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		util.Logger.Fatalf("Failed to create file watcher: %v", err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					util.Logger.Println("Configuration file change detected, reloading...")
					if err := models.LoadConfig(configFile); err != nil {
						util.Logger.Printf("Failed to hot-reload config file: %v", err)
					} else {
						util.Logger.Println("Configuration file hot-reloaded successfully.")
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				util.Logger.Printf("File watcher error: %v", err)
			}
		}
	}()

	err = watcher.Add(configFile)
	if err != nil {
		util.Logger.Fatalf("Failed to add file to watcher: %v", err)
	}

	// Block forever
	<-make(chan struct{})
}

func Run(staticFS, localeFS fs.FS) {
	configFile := "config.yaml"
	// ... (initConfig, initLogger) ...
	err := models.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}
	err = util.InitLogger(models.GetConfig().Logging)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	go watchConfig(configFile)

	// --- Pre-read and cache index.html ---
	file, err := staticFS.Open("index.html")
	if err != nil {
		util.Logger.Fatalf("Failed to open embedded index.html: %v", err)
	}
	indexHTMLContent, err = io.ReadAll(file)
	file.Close()
	if err != nil {
		util.Logger.Fatalf("Failed to read embedded index.html: %v", err)
	}
	info, err := fs.Stat(staticFS, "index.html")
	if err == nil {
		indexHTMLModTime = info.ModTime()
	} else {
		indexHTMLModTime = time.Now()
	}

	// --- i18n Setup ---
	bundle := i18n.NewBundle(language.English) // Set English as the default language
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load translation files directly from the embedded filesystem.
	// We still need this for the i18n middleware to work, which detects language.
	_, err = bundle.LoadMessageFileFS(localeFS, "en.json")
	if err != nil {
		util.Logger.Fatalf("failed to load en.json: %v", err)
	}
	_, err = bundle.LoadMessageFileFS(localeFS, "zh.json")
	if err != nil {
		util.Logger.Fatalf("failed to load zh.json: %v", err)
	}

	util.Logger.Println("Starting server...")
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Register i18n middleware
	router.Use(i18nMiddleware(bundle))

	// Register mock routes if the 'mock' build tag is enabled.
	registerMockRoutes(router)

	// --- New I18n API Endpoint ---
	router.GET("/api/i18n/:lang", func(c *gin.Context) {
		lang := c.Param("lang")
		// Basic validation to prevent path traversal
		if lang != "en" && lang != "zh" && lang != "ja" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Language not supported"})
			return
		}

		filePath := fmt.Sprintf("%s.json", lang)
		file, err := localeFS.Open(filePath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Translation file not found"})
			return
		}
		defer file.Close()

		c.Header("Content-Type", "application/json; charset=utf-8")
		io.Copy(c.Writer, file)
	})

	// --- API Route (remains unchanged at /api/data) ---
	router.GET("/api/data", func(c *gin.Context) {
		dbStatuses := handlers.GetAllDatabaseStatus()
		response := models.ApiResponse{Code: 200, Data: dbStatuses, Message: "success", Timestamp: time.Now().Unix()}
		c.JSON(http.StatusOK, response)
	})

	// --- Static File Serving Setup ---

	// *** Modified handler for the root "/" ***
	router.GET("/", func(c *gin.Context) {
		currentConfig := models.GetConfig()
		// 1. Determine Base Tag
		baseTag := ""
		if currentConfig.Server.PublicBasePath != "" && currentConfig.Server.PublicBasePath != "/" {
			// Use template.HTMLEscapeString for safety, although simple paths are usually fine
			escapedBasePath := template.HTMLEscapeString(currentConfig.Server.PublicBasePath)
			baseTag = fmt.Sprintf(`<base href="%s">`, escapedBasePath)
		}

		// 2. Prepare Frontend Config Script Tag
		frontendConfig := struct {
			BasePath string                  `json:"basePath"`
			Layout   models.LayoutConfig     `json:"layout"`
			Frontend models.FrontendSettings `json:"frontend"`
		}{
			BasePath: currentConfig.Server.PublicBasePath,
			Layout:   currentConfig.Layout,
			Frontend: currentConfig.Frontend,
		}
		configJSON, err := json.Marshal(frontendConfig)
		if err != nil {
			util.Logger.Printf("Error marshalling frontend config: %v", err)
			// Fallback to a default config
			configJSON = []byte(`{"basePath":"","layout":{"columns":2}}`)
		}
		configScriptTag := fmt.Sprintf(`<script>window.APP_CONFIG = %s;</script>`, string(configJSON))

		// 3. Prepare Titles Script Tag
		titlesJSON, err := json.Marshal(currentConfig.Titles)
		if err != nil {
			// Fallback or log error if marshalling fails
			util.Logger.Printf("Error marshalling titles: %v", err)
			titlesJSON = []byte("{}") // Send empty object on error
		}
		titlesScriptTag := fmt.Sprintf(`<script>window.APP_TITLES = %s;</script>`, string(titlesJSON))

		// 4. Replace placeholders
		content := indexHTMLContent // Start with cached content
		content = bytes.Replace(content, []byte("<!-- BASE_HREF_PLACEHOLDER -->"), []byte(baseTag), 1)
		content = bytes.Replace(content, []byte("<!-- CONFIG_SCRIPT_PLACEHOLDER -->"), []byte(configScriptTag), 1)
		content = bytes.Replace(content, []byte("<!-- TITLES_SCRIPT_PLACEHOLDER -->"), []byte(titlesScriptTag), 1)

		// REMOVED I18N INJECTION LOGIC

		// 4. Set headers and serve
		// Add cache-control headers to prevent browser caching issues.
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Header("Content-Type", "text/html; charset=utf-8")
		reader := bytes.NewReader(content)
		http.ServeContent(c.Writer, c.Request, "index.html", indexHTMLModTime, reader)
	})

	// Serve other specific root files using StaticFileFS
	router.StaticFileFS("/favicon.ico", "favicon.ico", http.FS(staticFS))
	router.StaticFileFS("/dashboard.html", "dashboard.html", http.FS(staticFS))

	// Middleware to handle language selection from URL query parameter
	router.Use(func(c *gin.Context) {
		lang := c.Query("lang")
		if lang == "en" || lang == "zh" {
			// Set cookie for i18n middleware to consume.
			// The default cookie name for many i18n libraries is 'lang'.
			c.SetCookie("lang", lang, 3600*24*30, "/", "", false, true)
		}
		c.Next()
	})

	// Serve assets under the '/static' path using StaticFS
	router.StaticFS("/static", http.FS(staticFS))

	// --- Server Startup (logging remains the same) ---
	port := models.GetConfig().Server.Port
	if port == "" {
		port = "8080"
	}
	util.Logger.Printf("Server started, listening on port: %s\n", port)
	currentConfig := models.GetConfig()
	util.Logger.Printf("Public base path: %s\n", currentConfig.Server.PublicBasePath)
	util.Logger.Printf("Access URL: http://localhost:%s%s\n", port, currentConfig.Server.PublicBasePath)
	fmt.Printf("Server started, listening on port: %s\n", port)
	fmt.Printf("Access URL: http://localhost:%s%s\n", port, currentConfig.Server.PublicBasePath)

	err = router.Run(":" + port)
	if err != nil {
		util.Logger.Fatalf("Failed to start server: %v", err)
	}
}
