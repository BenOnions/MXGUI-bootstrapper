package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/secsy/goftp"
	"gopkg.in/yaml.v2"
)

type Mixer struct {
	Main   string `yaml:"main"`
	Backup string `yaml:"backup"`
	Type   string `yaml:"type"`
	Name   string `yaml:"name"`
}

type Config struct {
	Mixers []Mixer `yaml:"Mixers"`
}

var (
	mixers     Config
	liveMixers []Mixer
	mc56Mixers []Mixer
	mc36Mixers []Mixer
	mc96Mixers []Mixer
	nepLogo    string
	wg         = &sync.WaitGroup{}
)

func copy(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo
	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

func vBoxImport(file string) {
	command := "C:\\Program Files\\Oracle\\VirtualBox\\VBoxmanage.exe"
	args := []string{"import", "--vsys", "0", "--eula", "accept", file}
	importCommand := exec.Command(command, args...)
	err := importCommand.Run()
	if err != nil {
		log.Fatalf("cmd.Run() import failed with %s\n", err)
	}
	//_ = virtualbox.ImportOV(file)
	//return Manage().run("import --vsys 0 --eula accept", file)
}

func testConnection(mixer Mixer) {
	wg.Add(1)
	defer wg.Done()
	log.Print("Testing connectivity to " + mixer.Name + " at " + mixer.Main)
	portNum := "80"
	seconds := 1
	timeOut := time.Duration(seconds) * time.Second
	_, err := net.DialTimeout("tcp", mixer.Main+":"+portNum, timeOut)
	if err != nil {
		log.Println(err)
		return
	}
	liveMixers = append(liveMixers, mixer)
	for _, mixer := range liveMixers {
		backupFiles(mixer)
	}
	//fmt.Printf("%+v", liveMixers)
}

func backupFiles(mixer Mixer) {
	config := goftp.Config{
		User:               "root",
		Password:           "hong",
		ConnectionsPerHost: 10,
		Timeout:            10 * time.Second,
		Logger:             os.Stderr,
	}

	log.Print("Backing up " + mixer.Name + "...")
	client, err := goftp.DialConfig(config, mixer.Main)
	if err != nil {
		panic(err)
	}

	files, err := client.ReadDir("/data/productions")
	for _, file := range files {

		fmt.Println("Downloading " + file.Name() + "...")

		targetDir := "./mxgui_user_share/" + mixer.Name + "/productions/"
		_, err := os.Stat(targetDir)
		if os.IsNotExist(err) {
			errDir := os.MkdirAll(targetDir, 0755)
			if errDir != nil {
				log.Fatal(err)
			}
		}

		if file.Name() != ".backup" {
			newFile, err := os.Create(targetDir + file.Name() + ".lpn")
			if err != nil {
				panic(err)
			}

			log.Print("creating " + file.Name())

			err = client.Retrieve("/data/productions/"+file.Name(), newFile)
			if err != nil {
				panic(err)
			}


		}

	}
}

func bootstrapMxGUIVMS() {
	dirname := "."
	f, err := os.Open(dirname)
	if err != nil {
		log.Fatal(err)
	}
	files, err := f.Readdir(-1)
	_ = f.Close()
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.Mode().IsRegular() {
			if filepath.Ext(file.Name()) == ".ova" {
				vBoxImport(file.Name())
				fileName := strings.Split(file.Name(), ".")
				createConfigShareFolders(fileName[0])
				command := "VBoxmanage.exe"
				wd, err := os.Getwd()
				configShareArgs := []string{"sharedfolder", "add", fileName[0], "--name", "mxgui_config_share", "--hostpath", wd + "\\configShares\\" + fileName[0] + "\\mxgui_config_share"}
				userShareArgs := []string{"sharedfolder", "add", fileName[0], "--name", "mxgui_user_share", "--hostpath", wd + "\\mxgui_user_share\\"}
				log.Println(configShareArgs)
				configShareCMD := exec.Command(command, configShareArgs...)
				log.Print("Mounting config share folder...")
				log.Print(command, configShareArgs)
				userShareCMD := exec.Command(command, userShareArgs...)
				log.Print("Mounting user share folder...")
				err = configShareCMD.Run()
				if err != nil {
					log.Fatalf("cmd.Run() config share failed with %s\n", err)
				}
				err = userShareCMD.Run()
				if err != nil {
					log.Fatalf("cmd.Run() user share failed with %s\n", err)
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

func archiveOVA(fileName string) {
	dir := "./mxguiAppliancesArchive/"
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
	log.Print("Archiving " + fileName + "at" + dir + fileName + " for safekeeping...")
}

func createConfigShareFolders(fileName string) {
	log.Print("Creating Config Share Folder...")
	baseDir := "./configShares/" + fileName + "/mxgui_config_share"
	log.Print("Created " + baseDir)
	mc56Mk2 := "./configShares/" + fileName + "/mxgui_config_share/mc56_mk2/config/bayserver_config/mxGUI/"
	log.Print("Created " + mc56Mk2)
	mc3640 := "./configShares/" + fileName + "/mxgui_config_share/mc36_40/config/bayserver_config/mxGUI/"
	log.Print("Created " + mc3640)
	mc96 := "./configShares/" + fileName + "/mxgui_config_share/mc96/config/bayserver_config/mxGUI/"
	log.Print("Created " + mc96)

	_, err := os.Stat(baseDir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(baseDir, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
	}
	_, err = os.Stat(mc56Mk2)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(mc56Mk2, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
		copy("mcx_gui_global.tcl", mc56Mk2+"mcx_gui_global.tcl")
	}
	_, err = os.Stat(mc3640)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(mc3640, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
		copy("mcx_gui_global.tcl", mc3640+"mcx_gui_global.tcl")

	}
	_, err = os.Stat(mc96)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(mc96, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
		copy("mcx_gui_global.tcl", mc96+"mcx_gui_global.tcl")
	}

	for _, mixer := range mc96Mixers {
		f, err := os.OpenFile("./configShares/"+fileName+"/mxgui_config_share/mc96/config/gui_hosts.tcl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write([]byte("\n add_gui_host " + `"` + mixer.Name + `"` + ` "` + mixer.Main + `"` + ` "` + mixer.Backup + `"` + " 0"))
		if err != nil {
			log.Fatal(err)
		}
		_ = f.Close()
		log.Print("Bootstrapping mc96 gui_hosts.tcl...")
	}
	for _, mixer := range mc56Mixers {
		f, err := os.OpenFile("./configShares/"+fileName+"/mxgui_config_share/mc56_mk2/config/gui_hosts.tcl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write([]byte("\n add_gui_host " + `"` + mixer.Name + `"` + ` "` + mixer.Main + `"` + ` "` + mixer.Backup + `"` + " 0"))
		if err != nil {
			log.Fatal(err)
		}
		_ = f.Close()
		log.Print("Bootstrapping mc56 gui_hosts.tcl...")
	}
	for _, mixer := range mc36Mixers {
		f, err := os.OpenFile("./configShares/"+fileName+"/mxgui_config_share/mc36_40/config/gui_hosts.tcl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write([]byte("\n add_gui_host " + `"` + mixer.Name + `"` + ` "` + mixer.Main + `"` + ` "` + mixer.Backup + `"` + " 0"))
		if err != nil {
			log.Fatal(err)
		}
		_ = f.Close()
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
	for _, mixer := range mixers.Mixers {
		switch mixer.Type {
		case "MC2_96":
			mc96Mixers = append(mc96Mixers, mixer)
		case "MC2_56_MKii":
			mc56Mixers = append(mc56Mixers, mixer)
		case "MC2_36":
			mc36Mixers = append(mc36Mixers, mixer)
		}
		createUserShareFolders(mixer)
		go testConnection(mixer)
	}
	bootstrapMxGUIVMS()
	wg.Wait()
	nepLogo = ` *//*//.                                                            
       ,,,,,//*/*/**%%%%%                                                       
    .,,,,,,, ///*/*%%%%%%%%%                                                    
   //,.,,,,,,*//**%%%%%%% %%%%                                                  
 *////// ,,,, //,%%%% %%%%%%%%%                                                 
 /////////.,,,/.#..#%%%%%%%%%%%%                                                
//(*   /*////                                                                   
 //////////////  ( @@@@@.    @@@   @@@@@@@@@@@  @@@@@@@@@@@                     
 //////// .(((( #( @@@,@@@   @@@                @@@&     @@@                    
  / ,(((((((( ###( @@@  @@@  @@@   @@@@@@@@@@   @@@&    @@@@                    
   ,(((((((*%####( @@@   @@@ @@@                @@@@@@@@@@                      
      (((( ######( @@@    @@@@@@                @@@&                            
         .#######( @@@     @@@@@   @@@@@@@@@@   @@@&     `
	log.Print(nepLogo)
	log.Print("

	Project Maintainer: Ben Onions, Email:bonions@nepgroup.com")
}
