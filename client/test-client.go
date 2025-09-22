package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

type TokenCache struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

func decodeJWT(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode the payload (second part)
	payload := parts[1]
	// Add padding if needed
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", 4-len(payload)%4)
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWT claims: %v", err)
	}

	return claims, nil
}

func printTokenInfo(label string, token *oauth2.Token) {
	fmt.Printf("%s:\n", label)
	fmt.Printf("  Token Type: %s\n", token.TokenType)
	fmt.Printf("  Access Token (first 30 chars): %s...\n", truncateToken(token.AccessToken, 30))
	fmt.Printf("  Refresh Token (first 30 chars): %s...\n", truncateToken(token.RefreshToken, 30))
	fmt.Printf("  Expiry: %s\n", token.Expiry.Format(time.RFC3339))
	fmt.Printf("  Time until expiry: %v\n", time.Until(token.Expiry).Round(time.Second))

	// Refresh token basic info
	fmt.Printf("  Refresh Token Details:\n")
	fmt.Printf("    Length: %d characters\n", len(token.RefreshToken))
	if isJWT(token.RefreshToken) {
		fmt.Printf("    Format: JWT\n")
		if claims, err := decodeJWT(token.RefreshToken); err == nil {
			if exp, ok := claims["exp"].(float64); ok {
				expiry := time.Unix(int64(exp), 0)
				fmt.Printf("    Expires: %s\n", expiry.Format(time.RFC3339))
				if time.Now().After(expiry) {
					fmt.Printf("    Status: EXPIRED\n")
				} else {
					fmt.Printf("    Valid for: %v\n", time.Until(expiry).Round(time.Second))
				}
			}
			if iat, ok := claims["iat"].(float64); ok {
				issued := time.Unix(int64(iat), 0)
				fmt.Printf("    Issued: %s\n", issued.Format(time.RFC3339))
				fmt.Printf("    Age: %v\n", time.Since(issued).Round(time.Second))
			}
		}
	} else {
		fmt.Printf("    Format: Opaque (no expiry info available)\n")
	}

	// Decode and display access token claims
	if claims, err := decodeJWT(token.AccessToken); err == nil {
		fmt.Printf("  Access Token Claims:\n")
		for key, value := range claims {
			switch key {
			case "iss", "sub", "aud", "email", "name", "preferred_username":
				fmt.Printf("    %s: %v\n", key, value)
			case "exp", "iat", "nbf":
				if timestamp, ok := value.(float64); ok {
					fmt.Printf("    %s: %v (%s)\n", key, int64(timestamp), time.Unix(int64(timestamp), 0).Format(time.RFC3339))
				}
			case "groups":
				fmt.Printf("    %s: %v\n", key, value)
			}
		}
	}

	// Decode and display ID token if present
	if idToken, exists := token.Extra("id_token").(string); exists && idToken != "" {
		if claims, err := decodeJWT(idToken); err == nil {
			fmt.Printf("  ID Token Claims:\n")
			for key, value := range claims {
				switch key {
				case "iss", "sub", "aud", "email", "name", "preferred_username":
					fmt.Printf("    %s: %v\n", key, value)
				case "exp", "iat", "nbf":
					if timestamp, ok := value.(float64); ok {
						fmt.Printf("    %s: %v (%s)\n", key, int64(timestamp), time.Unix(int64(timestamp), 0).Format(time.RFC3339))
					}
				case "groups":
					fmt.Printf("    %s: %v\n", key, value)
				}
			}
		}
	}
	fmt.Println()
}

func readTokenFromCache() (*TokenCache, error) {
	fmt.Println("Looking for kubelogin token cache...")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	cacheDir := filepath.Join(homeDir, ".kube", "cache", "oidc-login")
	fmt.Printf("Searching cache directory: %s\n", cacheDir)
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %v", err)
	}

	for _, file := range files {
		fmt.Printf("Checking file: %s\n", file.Name())
		filePath := filepath.Join(cacheDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var tokenCache TokenCache
		if err := json.Unmarshal(data, &tokenCache); err != nil {
			continue
		}

		fmt.Printf("Successfully loaded tokens from: %s\n", file.Name())
		return &tokenCache, nil
	}

	return nil, fmt.Errorf("no valid token cache file found")
}

func main() {
	fmt.Println("Starting OAuth2 Token Refresh Test")
	fmt.Println("===================================")

	// Configure HTTP client to skip TLS verification for self-signed certificates
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// Check which Dex flow is running by looking for environment variable
	// or defaulting to password flow (with secret)
	clientSecret := "kubelogin-test-secret"
	flowType := "password"

	if os.Getenv("DEX_FLOW") == "device-code" {
		clientSecret = "" // Public client for device-code flow
		flowType = "device-code"
	}

	config := &oauth2.Config{
		ClientID:     "kubelogin-test",
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://dex.127.0.0.1.nip.io:32000/auth",
			TokenURL: "https://dex.127.0.0.1.nip.io:32000/token",
		},
		Scopes: []string{"openid", "profile", "email"},
	}

	fmt.Printf("OAuth2 Config:\n")
	fmt.Printf("  Flow Type: %s\n", flowType)
	fmt.Printf("  TLS Verification: DISABLED (trusting self-signed certificates)\n")
	fmt.Printf("  Client ID: %s\n", config.ClientID)
	if clientSecret != "" {
		fmt.Printf("  Client Secret: %s\n", config.ClientSecret)
	} else {
		fmt.Printf("  Client Type: Public (no secret)\n")
	}
	fmt.Printf("  Auth URL: %s\n", config.Endpoint.AuthURL)
	fmt.Printf("  Token URL: %s\n", config.Endpoint.TokenURL)
	fmt.Printf("  Scopes: %v\n", config.Scopes)
	fmt.Println()

	// Read tokens from kubelogin cache
	tokenCache, err := readTokenFromCache()
	if err != nil {
		fmt.Printf("Failed to read token cache: %v\n", err)
		fmt.Println("\nTroubleshooting tips:")
		fmt.Println("1. Make sure you've authenticated with kubectl using kubelogin first:")
		fmt.Println("   kubectl --user oidc get pods -A")
		fmt.Println("2. Check if cache directory exists:")
		fmt.Println("   ls -la ~/.kube/cache/oidc-login/")
		return
	}

	// Validate tokens before proceeding
	if tokenCache.IDToken == "" {
		fmt.Println("Error: ID token is empty in cache")
		return
	}
	if tokenCache.RefreshToken == "" {
		fmt.Println("Error: Refresh token is empty in cache")
		fmt.Println("Note: Some OIDC configurations may not provide refresh tokens")
		return
	}

	// Check if tokens look like JWTs
	if !strings.Contains(tokenCache.IDToken, ".") {
		fmt.Println("Warning: ID token doesn't appear to be a JWT")
	}
	if !strings.Contains(tokenCache.RefreshToken, ".") && len(tokenCache.RefreshToken) < 10 {
		fmt.Println("Warning: Refresh token appears to be very short")
	}

	fmt.Printf("Token validation:\n")
	fmt.Printf("  ID Token length: %d characters\n", len(tokenCache.IDToken))
	fmt.Printf("  Refresh Token length: %d characters\n", len(tokenCache.RefreshToken))

	// Try to decode ID token to check if it's expired
	if claims, err := decodeJWT(tokenCache.IDToken); err == nil {
		if exp, ok := claims["exp"].(float64); ok {
			expiry := time.Unix(int64(exp), 0)
			fmt.Printf("  ID Token expires: %s\n", expiry.Format(time.RFC3339))
			if time.Now().After(expiry) {
				fmt.Printf("  WARNING: ID token is already expired!\n")
			} else {
				fmt.Printf("  ID Token valid for: %v\n", time.Until(expiry).Round(time.Second))
			}
		}
	}
	fmt.Println()

	// After getting initial token...
	token := &oauth2.Token{
		AccessToken:  tokenCache.IDToken,
		RefreshToken: tokenCache.RefreshToken,
		Expiry:       time.Now().Add(30 * time.Second),
	}

	printTokenInfo("Initial Token Information", token)

	cycle := 1
	for {
		fmt.Printf("=== REFRESH CYCLE %d ===\n", cycle)
		fmt.Printf("Waiting 35 seconds for token to expire...\n")

		for i := 35; i > 0; i-- {
			fmt.Printf("\rCountdown: %2d seconds remaining", i)
			time.Sleep(1 * time.Second)
		}

		fmt.Printf("Attempting token refresh (cycle %d)...\n", cycle)
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

		startTime := time.Now()
		newToken, err := config.TokenSource(ctx, token).Token()
		refreshDuration := time.Since(startTime)

		if err != nil {
			fmt.Printf("Refresh failed after %v: %v\n", refreshDuration.Round(time.Millisecond), err)

			// Provide specific guidance based on error type
			errorStr := err.Error()
			if strings.Contains(errorStr, "invalid_request") && strings.Contains(errorStr, "Refresh token") {
				fmt.Println("\nDiagnosis: Refresh token has been consumed or expired")
				fmt.Println("Common causes:")
				fmt.Println("1. Another client (like kubectl) already used this refresh token")
				fmt.Println("2. Refresh token has expired (check Dex configuration)")
				fmt.Println("3. Dex server was restarted, invalidating tokens")
				fmt.Println("\nSolutions:")
				fmt.Println("1. Re-authenticate with kubectl to get fresh tokens:")
				fmt.Println("   kubectl --user oidc get pods -A")
				fmt.Println("2. Clear token cache and re-authenticate:")
				fmt.Println("   rm -rf ~/.kube/cache/oidc-login && kubectl --user oidc get pods -A")
			} else if strings.Contains(errorStr, "invalid_client") {
				fmt.Println("\nDiagnosis: Client credentials are invalid")
				fmt.Println("Solutions:")
				fmt.Println("1. Check if using correct flow type (set DEX_FLOW=device-code if needed)")
				fmt.Println("2. Verify Dex client configuration matches test client settings")
			} else {
				fmt.Println("\nGeneral troubleshooting:")
				fmt.Println("1. Check Dex server logs: make logs")
				fmt.Println("2. Verify network connectivity to Dex")
				fmt.Println("3. Check if certificates are trusted")
			}

			fmt.Printf("\nCompleted %d successful refresh cycles.\n", cycle-1)
			break
		}

		fmt.Printf("Refresh successful! (took %v)\n", refreshDuration.Round(time.Millisecond))
		printTokenInfo(fmt.Sprintf("Refreshed Token Information (Cycle %d)", cycle), newToken)

		// Update token for next cycle
		token = newToken
		// Set expiry to 30 seconds for testing (normally would use actual expiry)
		token.Expiry = time.Now().Add(30 * time.Second)

		cycle++

		// Optional: Add a small delay before starting next cycle
		fmt.Println("Starting next refresh cycle in 3 seconds...")
		time.Sleep(3 * time.Second)
	}

	fmt.Println("Token refresh loop completed.")
}

func truncateToken(token string, length int) string {
	if len(token) <= length {
		return token
	}
	return token[:length]
}

func isJWT(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3
}
