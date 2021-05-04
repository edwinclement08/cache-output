package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base32"
	"flag"
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

func parseArguements() (bool, int, string) {
	noOfDays := flag.Int("validity", 1, "How long the cache is valid in days")
	recache := flag.Bool("recache", false, "Re cache all the results")
	debug := flag.Bool("debug", false, "sets log level to debug")

	flag.Parse()
	program := strings.Join(flag.Args(), " ")

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	return *recache, *noOfDays, program
}

// TODO
func recacheAllCommands() {
	log.Debug().Msg("Will Recache all")
	os.Exit(0)
}

func ensureCacheDirExists() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatal().Msgf("Can't find Cache Dir: %s", err)
	}
	savePath := path.Join(cacheDir, "cache-output")
	os.MkdirAll(savePath, 0755)
	return savePath
}

func main() {
	initLog()
	recache, noOfDays, program := parseArguements()

	if recache {
		recacheAllCommands()
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

			file.Close()
			return
		} else {
			// Cache file is invalid
			file.Close()
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
	log.Debug().Msg("Reached herez")

	duplicateReader := io.TeeReader(stdout, os.Stdout) // Split the pipe to 2 directions: 1) stdout, 2) reader
	log.Debug().Msg("Reached herez 2")

	log.Debug().Msg("Running the program.")
	if err := cmd.Start(); err != nil {
		log.Fatal().Err(err).Msg("Program Failed to start")
	}

	io.Copy(fileWriter, duplicateReader)

	if err := cmd.Wait(); err != nil {
		log.Fatal().Err(err).Msg("Program exited with non zero exit code.")
	}

	log.Debug().Msgf("Will cache results to %s of [%s] for %d days.", hashString, program, noOfDays)

	// reader := bufio.NewReader(os.Stdin)
	// fmt.Print("Enter text: ")
	// text, _ := reader.ReadString('\n')
	// fmt.Println(text)

	// fmt.Println("Enter text: ")
	// text2 := ""
	// fmt.Scanln(text2)
	// fmt.Println(text2)

	// ln := ""
	// fmt.Sscanln("%v", ln)
	// fmt.Println(ln)
}
