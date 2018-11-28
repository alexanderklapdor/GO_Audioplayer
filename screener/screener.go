package screener

//Imports
import (
	"fmt"
	"strconv"
	"github.com/alexanderklapdor/RaspberryPi_Go_Audioplayer/logger"
)

// Start Screen
func StartScreen() {

	fmt.Println("############################################")
	fmt.Println("#               Welcome to                 #")
	fmt.Println("#              Music Player                #")
	fmt.Println("############################################")
	fmt.Println("")

}

// End Screen
func EndScreen() {

	fmt.Println("")
	fmt.Println("********************************************")
	fmt.Println("*               Shutdown                   *")
	fmt.Println("********************************************")
}

// Print Files
func PrintFiles(fileList []string, printFiles bool) {

	logger.Log.Info("Found " + strconv.Itoa(len(fileList)) + " supported files")
	if printFiles {
		logger.Log.Info("*********Filelist (Supported Files)*********")
		for _, fileElement := range fileList {
			logger.Log.Info(fileElement)
		}
	}

}