package steamcmd

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/hokaccha/go-prettyjson"
	"github.com/mholt/archiver/v3"
)

// TODO: Add some sort of retry logic for steamcmd, with backoff support, so it'll keep trying to a while

// AppInfo returns a JSON string containing information about the Steam app
func AppInfo(appID string) string {
	return run([]string{"+@sSteamCmdForcePlatformType", steamcmdPlatform, "+login", "anonymous", "+app_info_update", "1", "+app_info_print", appID, "+quit"})
}

func run(args []string) string {
	//log.Println("Running steamcmd with args:", args)

	// Check if steamcmd already exists
	binary, lookErr := exec.LookPath(steamcmdBinaryPath)
	if lookErr != nil {
		// Download steamcmd
		download(steamcmdDownloadPath, steamcmdDownloadURL)

		// Unzip the downloaded file
		// err := archiver.TarGz.Open(steamcmdDownloadPath, steamcmdPath)
		err := archiver.Unarchive(steamcmdDownloadPath, steamcmdPath)
		if err != nil {
			log.Fatal("Extraction failed:", err)
			os.Exit(1)
		}

		// Check that steamcmd exists, but this time fail if it doesn't exist
		_, lookErr := exec.LookPath(steamcmdBinaryPath)
		if lookErr != nil {
			log.Fatal("Installation failed:", lookErr)
			os.Exit(1)
		}
	}

	// FIXME: This will probably only work on macOS and Linux, but not on Windows
	// Remove the "appinfo.vdf" cache file before running any commands
	os.Remove("~/Steam/appcache/appinfo.vdf")

	//	Format the command
	cmd := exec.Command(binary, args...)

	//	Sanity check -- capture stdout and stderr:
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	//	Run the command
	cmd.Run()

	//	Output our results
	result := appInfoFormat(out.String())
	error := stderr.String()
	if error != "" {
		log.Fatal("Command failed:", stderr.String())
		os.Exit(1)
	}
	if result == "" {
		log.Fatal("Command failed: No output or invalid app info")
		os.Exit(1)
	}

	return result
}

func appInfoFormat(appInfo string) string {
	// Remove everything before the first opening curly
	firstIndex := strings.LastIndex(reverse(appInfo), "{")
	if firstIndex > 0 {
		appInfo = reverse(trimLength(reverse(appInfo), firstIndex+1))
	}

	// Remove everything after the last closing curly brace
	lastIndex := strings.LastIndex(appInfo, "}")
	if lastIndex > 0 {
		appInfo = trimLength(appInfo, lastIndex+1)
	}

	// Get the app info part of the incoming data
	result := appInfo

	// Remove tabs
	result = strings.Replace(result, "\t", "", -1)

	// Add missing semicolons
	result = strings.Replace(result, "\"\n{", "\":\n{", -1)
	result = strings.Replace(result, "\"\"", "\":\"", -1)

	// Add missing commas
	result = strings.Replace(result, "}\n\"", "},\n\"", -1)
	result = strings.Replace(result, "\"\n\"", "\",\n\"", -1)

	// Remove newlines last
	result = strings.Replace(result, "\n", "", -1)

	// Validate that we have a proper JSON string
	if !isJSON(result) {
		log.Fatal("Parsing failed, invalid JSON:", result)
		os.Exit(1)
	}

	// Convert to pretty printed JSON
	in := []byte(result)
	var raw map[string]interface{}
	json.Unmarshal(in, &raw)
	prettyJSON, prettyErr := prettyjson.Marshal(raw)
	if prettyErr != nil {
		log.Fatal("Pretty JSON failed:", prettyErr)
		os.Exit(1)
	}

	// Verify that we have a valid JSON string again
	result = string(prettyJSON)
	if result == "" {
		log.Fatal("Pretty JSON failed: Empty JSON string")
		os.Exit(1)
	}

	// Return the parsed result
	return string(result)
}

func download(filepath string, url string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func isJSONString(s string) bool {
	var js string
	return json.Unmarshal([]byte(s), &js) == nil
}

func isJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func trimLength(s string, i int) string {
	runes := []rune(s)
	if len(runes) > i {
		return string(runes[:i])
	}
	return s
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
