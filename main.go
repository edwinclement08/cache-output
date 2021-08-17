package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base32"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func initLog() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func parseArguments() (int, bool, bool, bool, bool, string) {
	var noOfDays int
	var recache, debug, deleteAll, cachePath, list bool

	helpMsg := map[string]string{
		"validity":  "How long the cache is valid in days",
		"recache":   "[TBD] re cache all the results",
		"debug":     "sets log level to debug",
		"deleteAll": "delete all the cached results",
		"cachePath": "show where the cached results are stored",
		"list":      "[TBD] list all cached commands",
	}

	flag.IntVar(&noOfDays, "V", 1, helpMsg["validity"])
	//flag.BoolVar(&recache, "r", false, helpMsg["recache"])
	flag.BoolVar(&debug, "d", false, helpMsg["debug"])
	flag.BoolVar(&deleteAll, "D", false, helpMsg["deleteAll"])
	flag.BoolVar(&cachePath, "C", false, helpMsg["cachePath"])
	//flag.BoolVar(&list, "l", false, helpMsg["list"])

	flag.Parse()
	program := strings.Join(flag.Args(), " ")

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	return noOfDays, recache, deleteAll, cachePath, list, program
}

// TODO
func recacheAllCommands() {
	fmt.Println("Not Implemented")
}

func listAllCommands() {

	fmt.Println("Not Implemented")
}

func deleteAllCached() {
	log.Debug().Msg("Will Delete all Cached results")
	savePath := ensureCacheDirExists()

	if err := os.RemoveAll(savePath); err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("Failed to delete: \n\t%s\n", savePath))
	}
	fmt.Println("Deleted Cache folder")
}

// Ensures that the directory for caching exists
func ensureCacheDirExists() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatal().Msgf("Can't find Cache Dir: %s", err)
	}
	cachePath := path.Join(cacheDir, "cache-output")
	err = os.MkdirAll(cachePath, 0755)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create the cache directory")
	}
	return cachePath
}

//// Ensures that the directory for config exists
//func ensureConfigDirExists() string {
//	configDir, err := os.UserConfigDir()
//	if err != nil {
//		log.Fatal().Msgf("Can't find Config Dir: %s", err)
//	}
//	configPath := path.Join(configDir, "cache-output")
//	err = os.MkdirAll(configPath, 0755)
//	if err != nil {
//		log.Fatal().Err(err).Msg("Failed to create the config directory")
//	}
//	return configPath
//}

func main() {
	initLog()
	noOfDays, recache, deleteAll, cachePath, list, program := parseArguments()

	switch {
	case cachePath:
		fmt.Printf("The Cached results are stored in \n\t%s\n", ensureCacheDirExists())
		os.Exit(0)
	case recache:
		recacheAllCommands()
		os.Exit(0)
	case deleteAll:
		deleteAllCached()
		os.Exit(0)
	case list:
		listAllCommands()
		os.Exit(0)
	}

	if len(strings.Trim(program, " \n\t")) == 0 {
		fmt.Println("No command provided.\n\tUsage: cache-output <command to cache>")
		os.Exit(0)
	}

	savePath := ensureCacheDirExists()

	// Generate Cached path
	hashBytes := sha256.Sum256([]byte(program))
	hashString := base32.HexEncoding.EncodeToString(hashBytes[:])
	filePath := path.Join(savePath, hashString)

	file, err := os.Open(filePath)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal().Err(err).Msgf("Error while opening the cache file: %s", filePath)
	}

	if err == nil {
		log.Debug().Msg("Cache File Exists")
		fileStat, err := file.Stat()
		if err != nil {
			log.Fatal().Err(err).Msgf("Error Reading the Cache file(%s) Modification time", filePath)
		}

		modifiedTime := fileStat.ModTime()

		if modifiedTime.Add(time.Hour * 24 * time.Duration(noOfDays)).After(time.Now()) {
			log.Debug().Msg("Cache File is Valid")

			cachedData := bufio.NewReader(file)

			if _, err := io.Copy(os.Stdout, cachedData); err != nil {
				log.Fatal().Err(err).Msg("")
			}

			if err := file.Close(); err != nil {
				log.Warn().Err(err).Msg("Failed to close the cached file")
			}
			return
		} else {
			// Cache file is invalid
			if err := file.Close(); err != nil {
				log.Warn().Err(err).Msg("Failed to close the cached file")
			}
		}
	}

	// Cache file is either invalid, or doesn't exist
	err = os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal().Err(err).Msgf("Error Deleting the old Cache file: %s", filePath)
	}

	file, err = os.Create(filePath)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create the cache file %s for saving the program output", filePath)
	}
	fileWriter := bufio.NewWriter(file)

	cmdArray := strings.Split(program, " ")
	cmd := exec.Command(cmdArray[0], cmdArray[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to connect to output pipe of program: %s", program)
	}

	duplicateReader := io.TeeReader(stdout, os.Stdout) // Split the pipe to 2 directions: 1) stdout, 2) reader

	log.Debug().Msg("Running the program.")
	if err := cmd.Start(); err != nil {
		log.Fatal().Err(err).Msg("Program Failed to start")
	}

	if _, err := io.Copy(fileWriter, duplicateReader); err != nil {
		log.Fatal().Err(err).Msg("Failed to copy over the cached data.")
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal().Err(err).Msg("Given Program exited with non zero exit code.")
	}

	log.Debug().Msgf("Will cache results to %s of [%s] for %d days.", hashString, program, noOfDays)
}
