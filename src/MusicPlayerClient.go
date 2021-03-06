package main

// Imports
import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/alexanderklapdor/RaspberryPi_Go_Audioplayer/logger"
	"github.com/alexanderklapdor/RaspberryPi_Go_Audioplayer/screener"
	"github.com/alexanderklapdor/RaspberryPi_Go_Audioplayer/sender"
	"github.com/alexanderklapdor/RaspberryPi_Go_Audioplayer/structs"
	"github.com/alexanderklapdor/RaspberryPi_Go_Audioplayer/util"
	"github.com/tkanos/gonfig"
)

// Global var declaration
var configuration = structs.ClientConfiguration{}

// Main function
func main() {
	// Set up configuration
	err := gonfig.GetConf("config.json", &configuration)
	util.Check(err, "Client")

	// Check if Log directory exists
	if _, err := os.Stat(configuration.Log_Dir); os.IsNotExist(err) {
		os.Mkdir(configuration.Log_Dir, 0777)
	}
	// Set up Logger
	logger.Setup(path.Join(configuration.Log_Dir, configuration.Client_Log), configuration.Debug_Infos)

	// Start Screen
	screener.StartScreen()
	logger.Notice("Starting MusicPlayerClient...")

	socket_path := configuration.Socket_Path

	// Check if server is running
	if checkServerStatus() {
		logger.Info("Server is running")
	} else {
		logger.Info("Server is not running")
		fmt.Println("Server is not running")
		// Start Server
		logger.Info("Starting server")
		fmt.Println("Starting server...")
		startServer()
		// Wait for server has been started
		ind := 0
		for {
			if _, err := os.Stat(socket_path); err == nil ||
				ind >= configuration.Server_Connection_Attempts {
				break
			} // End of if
			logger.Info("Waiting for server")
			time.Sleep(1 * time.Second)
			ind++
		} // End of for

		// Check Server Stat
		if _, err2 := os.Stat(socket_path); err2 == nil {
			logger.Info("Server started succesfully")
		} else if os.IsNotExist(err2) {
			logger.Info("Server not started succesfully")
			os.Exit(304)
		} else {
			logger.Info("Something unexpected happened")
			os.Exit(777)
		}
	} // End of else

	// Check if no argument is given
	if len(os.Args) < 2 {
		logger.Error("Missing required argument")
		return
	}

	// Define flags
	command := flag.String("c", configuration.Default_Command, "command for the server (default "+
		configuration.Default_Command+")")
	input := flag.String("i", configuration.Default_Input, "input music file/folder (default "+
		configuration.Default_Input+")")
	volume := flag.Int("v", configuration.Default_Volume, "music volume in percent (default "+
		strconv.Itoa(configuration.Default_Volume)+")")
	depth := flag.Int("d", configuration.Default_Depth, "audio file searching depth (default/recommended "+
		strconv.Itoa(configuration.Default_Depth)+")")
	shuffle := flag.Bool("s", configuration.Default_Shuffle, "shuffle (default "+
		strconv.FormatBool(configuration.Default_Shuffle)+")")
	loop := flag.Bool("l", configuration.Default_Loop, "loop (default "+
		strconv.FormatBool(configuration.Default_Loop)+")")
	fadeIn := flag.Int("fi", configuration.Default_FadeIn, "fadein in milliseconds (default "+
		strconv.Itoa(configuration.Default_FadeIn)+")")
	fadeOut := flag.Int("fo", configuration.Default_FadeOut, "fadeout in milliseconds (default "+
		strconv.Itoa(configuration.Default_FadeOut)+")")

	// Parsing flags
	logger.Notice("Start Parsing cli parameters")
	flag.Parse()

	var values []string
	// If argument without flagname is given parse it as command
	if flag.NArg() > 1 {
		// Command argument
		*command = flag.Arg(0)
		// Value arguments
		for id, arg := range flag.Args() {
			if id != 0 {
				values = append(values, arg)
			} //End of if
		} // End of for
	} else {
		if flag.NArg() == 1 && *command == "default" {
			*command = flag.Arg(0)
		} // End of if
	} // End of else

	// Check received arguments
	logger.Notice("Check received arguments")
	if *volume < 0 || *depth < 0 || *fadeIn < 0 || *fadeOut < 0 {
		logger.Error("no negative values allowed")
		return
	}

	// Check volume
	if *volume > 100 {
		logger.Info("No volume above 100 allowed")
		logger.Info("Set volume to 100")
		*volume = 100
	}

	// Print received argument
	logger.Notice("Given arguments:")
	logger.Info("Command   " + *command)
	logger.Info("Input:    " + *input)
	logger.Info("Volume:   " + strconv.Itoa(*volume))
	logger.Info("Depth:    " + strconv.Itoa(*depth))
	logger.Info("Shuffle:  " + strconv.FormatBool(*shuffle))
	logger.Info("Loop:     " + strconv.FormatBool(*loop))
	logger.Info("Fade in:  " + strconv.Itoa(*fadeIn))
	logger.Info("Fade out: " + strconv.Itoa(*fadeOut))

	// Parsing to json
	logger.Notice("Parsing argument to json")
	dataInfo := &structs.Data{
		Depth:   *depth,
		FadeIn:  *fadeIn,
		FadeOut: *fadeOut,
		Shuffle: *shuffle,
		Loop:    *loop,
		Path:    *input,
		Values:  values,
		Volume:  *volume}
	requestInfo := &structs.Request{
		Command: string(*command),
		Data:    *dataInfo}
	requestJson, _ := json.Marshal(requestInfo)
	logger.Info("JSON String : " + string(requestJson))

	// Check if Command is Shutdown Command
	if requestInfo.Command == "exit" {
		fmt.Println("The server will shut down...")
	}

	// Send command
	sender.SetSocketPath(configuration.Socket_Path)
	sender.Send(requestJson)

	// Closing Client
	logger.Info("Closing MusicPlayerClient...\n")
	screener.EndScreen()

}

// CheckServerStatus function
func checkServerStatus() bool {
	// Get socket_path
	socket_path := configuration.Socket_Path
	// Check if socket exists
	if _, err := os.Stat(socket_path); err != nil {
		return false // Unix socket does not exists
	} else {
		// Check if process exists
		cmd := "ps -ef | grep MusicPlayerServer"
		output, err := exec.Command("bash", "-c", cmd).Output()
		util.Check(err, "Client")
		for _, pi := range strings.Split(string(output), "\n") {
			if strings.Contains(pi, "go run") {
				return true
			} // end of if
		} // end of for
		return false
	} // end of else
} // end of checkServerStatus

// StartServer function
func startServer() {
	logger.Info("Starting Server process")
	var attr = os.ProcAttr{
		Dir: ".",
		Env: os.Environ(),
		Files: []*os.File{
			os.Stdin,
			nil,
			nil,
		},
	}

	// Start process
	process, err := os.StartProcess(util.GetGoExPath(), []string{"go", "run", "MusicPlayerServer.go"}, &attr)
	util.Check(err, "Client")
	logger.Info("Detaching process")
	err = process.Release()
	util.Check(err, "Client")
}