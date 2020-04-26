package main

import (
	"github.com/terra-farm/go-virtualbox"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Mixer struct {
	Main string `yaml:"main"`
	Backup string `yaml:"backup"`
	Type string `yaml:"type"`
	Name string `yaml:"name"`
}

type Config struct {
	Mixers []Mixer `yaml:"Mixers"`
}

var (
	mixers Config
	liveMixers []Mixer
	mc56Mixers []Mixer
	mc36Mixers []Mixer
	mc96Mixers []Mixer
	)

func vBoxImport(file string) {
	_ = virtualbox.ImportOV(file)
}

func testConnection(mixer Mixer) {
	log.Print("Testing connectivity to " + mixer.Name + " at " + mixer.Main)
	portNum := "80"
	seconds := 1
	timeOut := time.Duration(seconds) * time.Second
	_, err := net.DialTimeout("tcp", mixer.Main+":"+portNum, timeOut)
	if err != nil {
		log.Println(err)
		return
	}
	liveMixers = append(liveMixers,mixer)
	//fmt.Printf("%+v",liveMixers)
}

func bootstrapMxGUIVMS() {
	dirname := "."
	f, err := os.Open(dirname)
	if err != nil {
		log.Fatal(err)
	}
	files, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.Mode().IsRegular() {
			if filepath.Ext(file.Name()) ==".ova" {
				vBoxImport(file.Name())
				fileName := strings.Split(file.Name(), ".")
				createConfigShareFolders(fileName[0])
				command := "VBoxmanage.exe"
				wd, err := os.Getwd()
				configShareArgs := []string{"sharedfolder", "add", fileName[0], "--name", "mxgui_config_share", "--hostpath", wd+"\\configShares\\" + fileName[0] +"\\mxgui_config_share"}
				userShareArgs := []string{"sharedfolder", "add", fileName[0], "--name", "mxgui_user_share", "--hostpath", wd+"\\mxgui_user_share\\"}
				log.Println(configShareArgs)
				configShareCMD := exec.Command(command, configShareArgs...)
				log.Print("Mounting config share folder...")
				userShareCMD := exec.Command(command, userShareArgs...)
				log.Print("Mounting user share folder...")
				err = configShareCMD.Run()
				if err !=nil {
					log.Fatalf("cmd.Run() failed with %s\n", err)
				}
				err = userShareCMD.Run()
				if err !=nil {
					log.Fatalf("cmd.Run() failed with %s\n", err)
				}
				archiveOVA(file.Name())
				}
			}
		}
	}



func createUserShareFolders(mixer Mixer) {
	dir := "./mxgui_user_share/" + mixer.Name
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(dir, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
	}
	log.Print("Created User Share folder for " + mixer.Name)
}

func archiveOVA(fileName string){
	dir :="./mxguiAppliancesArchive/"
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(dir, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
	}
	err = os.Rename(fileName, dir+fileName)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Archiving " + fileName + "at" + dir+fileName + " for safekeeping...")
}

func createConfigShareFolders(fileName string) {
	log.Print("Creating Config Share Folder...")
	baseDir := "./configShares/" + fileName + "/mxgui_config_share"
	log.Print("Created " + baseDir)
	mc56_mk2 := "./configShares/" + fileName + "/mxgui_config_share/mc56_mk2/config"
	log.Print("Created " + mc56_mk2)
	mc36_40 := "./configShares/" + fileName + "/mxgui_config_share/mc36_40/config"
	log.Print("Created " + mc36_40)
	mc96 := "./configShares/" + fileName + "/mxgui_config_share/mc96/config"
	log.Print("Created " + mc96)

	_, err := os.Stat(baseDir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(baseDir, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
	}
	_, err = os.Stat(mc56_mk2)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(mc56_mk2, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
	}
	_, err = os.Stat(mc36_40)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(mc36_40, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
	}
	_, err = os.Stat(mc96)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(mc96, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
	}

	for _, mixer := range mc96Mixers {
		f, err := os.OpenFile("./configShares/"+fileName+"/mxgui_config_share/mc96/config/gui_hosts.tcl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write([]byte("\n add_gui_host " + `"` + mixer.Name + `"` + ` "` + mixer.Main + `"` + ` "` + mixer.Backup + `"` + " 1"))
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
		log.Print("Bootstrapping mc96 gui_hosts.tcl...")
	}
	for _, mixer := range mc56Mixers {
		f, err := os.OpenFile("./configShares/"+fileName+"/mxgui_config_share/mc56_mk2/config/gui_hosts.tcl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write([]byte("\n add_gui_host " + `"` + mixer.Name + `"` + ` "` + mixer.Main + `"` + ` "` + mixer.Backup + `"` + " 1"))
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
		log.Print("Bootstrapping mc56 gui_hosts.tcl...")
	}
	for _, mixer := range mc36Mixers {
		f, err := os.OpenFile("./configShares/"+fileName+"/mxgui_config_share/mc36_40/config/gui_hosts.tcl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write([]byte("\n add_gui_host " + `"` + mixer.Name + `"` + ` "` + mixer.Main + `"` + ` "` + mixer.Backup + `"` + " 1"))
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
		log.Print("Bootstrapping mc36 gui_hosts.tcl...")
	}
}
func getMixers() {
		yamlFile, err := ioutil.ReadFile("config.yaml")
		if err != nil {
			log.Printf("yamlFile.Get err   #%v ", err)
		}
		log.Print("Reading config.yaml...")
		err = yaml.Unmarshal(yamlFile, &mixers)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}
		log.Print("Successfully read configuration file")
}


func main() {
	log.Print("MXGUI-bootstrapper is starting")
	getMixers()
	for _, mixer := range mixers.Mixers{
		switch mixer.Type {
		case "MC2_96":
			mc96Mixers = append(mc96Mixers, mixer)
		case "MC2_56_MKii":
			mc56Mixers = append(mc56Mixers, mixer)
		case "MC2_36":
			mc36Mixers = append(mc36Mixers, mixer)
		}
		//testConnection(mixer)
		createUserShareFolders(mixer)
	}
	bootstrapMxGUIVMS()
	log.Print("All done! :^)")
}
