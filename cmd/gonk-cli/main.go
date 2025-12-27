package main

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
)

var (
    Version   = "1.1.0"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

var rootCmd = &cobra.Command{
    Use:   "gonk",
    Short: "GONK API Gateway - Edge-Native Gateway with Authorization & mTLS",
    Long: `GONK is an ultra-lightweight API Gateway designed for edge, IoT, 
and industrial environments with built-in authorization, mTLS, and load balancing.`,
    Version: Version,
}

func init() {
    // Server commands
    rootCmd.AddCommand(startCmd)
    rootCmd.AddCommand(validateCmd)
    rootCmd.AddCommand(statusCmd)
    rootCmd.AddCommand(reloadCmd)
    
    // Configuration commands
    rootCmd.AddCommand(initCmd)
    rootCmd.AddCommand(configCmd)
    
    // Route management
    rootCmd.AddCommand(routesCmd)
    
    // Auth management
    rootCmd.AddCommand(authCmd)
    
    // Certificate management
    rootCmd.AddCommand(certsCmd)
    
    // Monitoring commands
    rootCmd.AddCommand(metricsCmd)
    rootCmd.AddCommand(logsCmd)
    rootCmd.AddCommand(healthCmd)
    rootCmd.AddCommand(cacheCmd)
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

// Start command
var startCmd = &cobra.Command{
    Use:   "start",
    Short: "Start GONK server",
    Long:  "Start the GONK API Gateway server with the specified configuration",
    Run: func(cmd *cobra.Command, args []string) {
        configPath, _ := cmd.Flags().GetString("config")
        daemon, _ := cmd.Flags().GetBool("daemon")
        
        if daemon {
            fmt.Println("Starting GONK in daemon mode...")
            // TODO: Implement daemon mode
        } else {
            fmt.Printf("Starting GONK with config: %s\n", configPath)
            // Start server normally
            startServer(configPath)
        }
    },
}

// Validate command
var validateCmd = &cobra.Command{
    Use:   "validate",
    Short: "Validate configuration file",
    Run: func(cmd *cobra.Command, args []string) {
        configPath, _ := cmd.Flags().GetString("config")
        if err := validateConfig(configPath); err != nil {
            fmt.Printf("❌ Configuration invalid: %v\n", err)
            os.Exit(1)
        }
        fmt.Println("✅ Configuration is valid")
    },
}

// Status command
var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Check GONK server status",
    Run: func(cmd *cobra.Command, args []string) {
        checkStatus()
    },
}

// Reload command
var reloadCmd = &cobra.Command{
    Use:   "reload",
    Short: "Hot reload configuration",
    Run: func(cmd *cobra.Command, args []string) {
        reloadConfig()
    },
}

// Init command
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize GONK configuration",
    Run: func(cmd *cobra.Command, args []string) {
        template, _ := cmd.Flags().GetString("template")
        output, _ := cmd.Flags().GetString("output")
        initializeConfig(template, output)
    },
}

// Config command
var configCmd = &cobra.Command{
    Use:   "config",
    Short: "Configuration management",
}

var configShowCmd = &cobra.Command{
    Use:   "show",
    Short: "Show current configuration",
    Run: func(cmd *cobra.Command, args []string) {
        configPath, _ := cmd.Flags().GetString("config")
        showConfig(configPath)
    },
}

// Routes command
var routesCmd = &cobra.Command{
    Use:   "routes",
    Short: "Route management",
}

var routesListCmd = &cobra.Command{
    Use:   "list",
    Short: "List all routes",
    Run: func(cmd *cobra.Command, args []string) {
        listRoutes()
    },
}

var routesAddCmd = &cobra.Command{
    Use:   "add",
    Short: "Add a new route interactively",
    Run: func(cmd *cobra.Command, args []string) {
        addRoute()
    },
}

var routesDescribeCmd = &cobra.Command{
    Use:   "describe [route-name]",
    Short: "Describe a route in detail",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        describeRoute(args[0])
    },
}

// Auth command
var authCmd = &cobra.Command{
    Use:   "auth",
    Short: "Authentication & Authorization management",
}

var authJWTCmd = &cobra.Command{
    Use:   "jwt",
    Short: "JWT management",
}

var authJWTGenerateCmd = &cobra.Command{
    Use:   "generate",
    Short: "Generate JWT token",
    Run: func(cmd *cobra.Command, args []string) {
        role, _ := cmd.Flags().GetString("role")
        scopes, _ := cmd.Flags().GetStringSlice("scopes")
        userID, _ := cmd.Flags().GetString("user-id")
        expiry, _ := cmd.Flags().GetString("expiry")
        
        generateJWT(role, scopes, userID, expiry)
    },
}

var authJWTValidateCmd = &cobra.Command{
    Use:   "validate [token]",
    Short: "Validate JWT token",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        validateJWT(args[0])
    },
}

var authJWTDecodeCmd = &cobra.Command{
    Use:   "decode [token]",
    Short: "Decode JWT token",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        decodeJWT(args[0])
    },
}

var authAPIKeyCmd = &cobra.Command{
    Use:   "apikey",
    Short: "API Key management",
}

var authAPIKeyGenerateCmd = &cobra.Command{
    Use:   "generate",
    Short: "Generate API key",
    Run: func(cmd *cobra.Command, args []string) {
        clientID, _ := cmd.Flags().GetString("client-id")
        roles, _ := cmd.Flags().GetStringSlice("roles")
        scopes, _ := cmd.Flags().GetStringSlice("scopes")
        
        generateAPIKey(clientID, roles, scopes)
    },
}

var authAPIKeyListCmd = &cobra.Command{
    Use:   "list",
    Short: "List all API keys",
    Run: func(cmd *cobra.Command, args []string) {
        listAPIKeys()
    },
}

// Certs command
var certsCmd = &cobra.Command{
    Use:   "certs",
    Short: "Certificate management for mTLS",
}

var certsGenerateCmd = &cobra.Command{
    Use:   "generate",
    Short: "Generate certificates",
    Run: func(cmd *cobra.Command, args []string) {
        cn, _ := cmd.Flags().GetString("cn")
        certType, _ := cmd.Flags().GetString("type")
        output, _ := cmd.Flags().GetString("output")
        
        generateCertificate(cn, certType, output)
    },
}

var certsValidateCmd = &cobra.Command{
    Use:   "validate",
    Short: "Validate certificate",
    Run: func(cmd *cobra.Command, args []string) {
        certFile, _ := cmd.Flags().GetString("cert")
        caFile, _ := cmd.Flags().GetString("ca")
        
        validateCertificate(certFile, caFile)
    },
}

var certsInfoCmd = &cobra.Command{
    Use:   "info",
    Short: "Show certificate information",
    Run: func(cmd *cobra.Command, args []string) {
        certFile, _ := cmd.Flags().GetString("cert")
        showCertInfo(certFile)
    },
}

// Metrics command
var metricsCmd = &cobra.Command{
    Use:   "metrics",
    Short: "Show Prometheus metrics",
    Run: func(cmd *cobra.Command, args []string) {
        route, _ := cmd.Flags().GetString("route")
        showMetrics(route)
    },
}

// Logs command
var logsCmd = &cobra.Command{
    Use:   "logs",
    Short: "Log management",
}

var logsTailCmd = &cobra.Command{
    Use:   "tail",
    Short: "Tail logs in real-time",
    Run: func(cmd *cobra.Command, args []string) {
        route, _ := cmd.Flags().GetString("route")
        follow, _ := cmd.Flags().GetBool("follow")
        
        tailLogs(route, follow)
    },
}

// Health command
var healthCmd = &cobra.Command{
    Use:   "health",
    Short: "Health check",
    Run: func(cmd *cobra.Command, args []string) {
        checkHealth()
    },
}

// Cache command
var cacheCmd = &cobra.Command{
    Use:   "cache",
    Short: "Cache management",
}

var cacheStatsCmd = &cobra.Command{
    Use:   "stats",
    Short: "Show cache statistics",
    Run: func(cmd *cobra.Command, args []string) {
        showCacheStats()
    },
}

var cacheClearCmd = &cobra.Command{
    Use:   "clear",
    Short: "Clear cache",
    Run: func(cmd *cobra.Command, args []string) {
        clearCache()
    },
}

func init() {
    // Start flags
    startCmd.Flags().StringP("config", "c", "gonk.yaml", "Configuration file path")
    startCmd.Flags().BoolP("daemon", "d", false, "Run in daemon mode")
    
    // Validate flags
    validateCmd.Flags().StringP("config", "c", "gonk.yaml", "Configuration file path")
    
    // Init flags
    initCmd.Flags().StringP("template", "t", "basic", "Template type (basic, industrial, microservices)")
    initCmd.Flags().StringP("output", "o", "gonk.yaml", "Output file path")
    
    // Config flags
    configShowCmd.Flags().StringP("config", "c", "gonk.yaml", "Configuration file path")
    configCmd.AddCommand(configShowCmd)
    
    // Routes subcommands
    routesCmd.AddCommand(routesListCmd)
    routesCmd.AddCommand(routesAddCmd)
    routesCmd.AddCommand(routesDescribeCmd)
    
    // Auth JWT flags and subcommands
    authJWTGenerateCmd.Flags().StringP("role", "r", "", "Role for the token")
    authJWTGenerateCmd.Flags().StringSliceP("scopes", "s", []string{}, "Scopes for the token")
    authJWTGenerateCmd.Flags().StringP("user-id", "u", "", "User ID")
    authJWTGenerateCmd.Flags().StringP("expiry", "e", "24h", "Token expiry duration")
    
    authJWTCmd.AddCommand(authJWTGenerateCmd)
    authJWTCmd.AddCommand(authJWTValidateCmd)
    authJWTCmd.AddCommand(authJWTDecodeCmd)
    
    // Auth API Key flags and subcommands
    authAPIKeyGenerateCmd.Flags().StringP("client-id", "c", "", "Client ID")
    authAPIKeyGenerateCmd.Flags().StringSliceP("roles", "r", []string{}, "Roles")
    authAPIKeyGenerateCmd.Flags().StringSliceP("scopes", "s", []string{}, "Scopes")
    
    authAPIKeyCmd.AddCommand(authAPIKeyGenerateCmd)
    authAPIKeyCmd.AddCommand(authAPIKeyListCmd)
    
    // Auth subcommands
    authCmd.AddCommand(authJWTCmd)
    authCmd.AddCommand(authAPIKeyCmd)
    
    // Certs flags and subcommands
    certsGenerateCmd.Flags().StringP("cn", "n", "localhost", "Common Name")
    certsGenerateCmd.Flags().StringP("type", "t", "server", "Certificate type (server, client, ca)")
    certsGenerateCmd.Flags().StringP("output", "o", ".", "Output directory")
    
    certsValidateCmd.Flags().StringP("cert", "c", "", "Certificate file")
    certsValidateCmd.Flags().StringP("ca", "a", "", "CA certificate file")
    
    certsInfoCmd.Flags().StringP("cert", "c", "", "Certificate file")
    
    certsCmd.AddCommand(certsGenerateCmd)
    certsCmd.AddCommand(certsValidateCmd)
    certsCmd.AddCommand(certsInfoCmd)
    
    // Metrics flags
    metricsCmd.Flags().StringP("route", "r", "", "Filter by route")
    
    // Logs flags
    logsTailCmd.Flags().StringP("route", "r", "", "Filter by route")
    logsTailCmd.Flags().BoolP("follow", "f", true, "Follow log output")
    logsCmd.AddCommand(logsTailCmd)
    
    // Cache subcommands
    cacheCmd.AddCommand(cacheStatsCmd)
    cacheCmd.AddCommand(cacheClearCmd)
}
