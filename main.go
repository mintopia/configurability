package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"plugin"

	"github.com/1and1internet/configurability/file_helpers"
	"github.com/1and1internet/configurability/plugins"
	"github.com/go-ini/ini"
)

func getPluginFolder() string {
	pluginFolder, ok := os.LookupEnv("CONF_PLUGIN_FOLDER")
	if ok {
		return pluginFolder
	}
	return "/opt/configurability/goplugins"
}

func LoadCustomisationData(customisorSymbol plugin.Symbol, etcConfigSections []*ini.Section) bool {
	var customisationFilePathMap map[string]string = file_helpers.MapCustomisationFolder()
	customised := false
	for _, section := range etcConfigSections {
		var configuration_file_name = section.Key("configuration_file_name")
		customisationFilePath, ok := customisationFilePathMap[configuration_file_name.String()]
		if ok {
			content, err := ioutil.ReadFile(customisationFilePath)
			if err != nil {
				log.Printf("There was a problem reading %s: %s\n", configuration_file_name.String(), err)
				log.Println("Continuing without it...")
				continue
			}

			customised = customisorSymbol.(func([]byte, *ini.Section, string) bool)(content, section, configuration_file_name.String())
			if customised {
				break
			}
		}
	}
	return customised
}

func readAllEtcConfigSections() []*ini.Section {
	var sections []*ini.Section
	for _, etcConfigrationFilePath := range file_helpers.ListEtcConfigFolder() {
		section, err := plugins.ReadEtcConfiguration(etcConfigrationFilePath)
		if err == nil && section != nil {
			sections = append(sections, section)
		}
	}
	return sections
}

func main() {
	loggingFilename := "/dev/stdout"
	f, err := os.OpenFile(loggingFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	pluginFolder := getPluginFolder()
	fileglob := path.Join(pluginFolder, "*.so")
	files, err := filepath.Glob(fileglob)
	if err == nil {
		etcConfigSections := readAllEtcConfigSections()
		for _, file := range files {
			log.Printf("Loading plugin %s\n", file)
			configuratorPlugin, err := plugin.Open(file)
			if err != nil {
				log.Printf("Could not load plugin %s: %s\n", file, err)
				continue
			}
			customisorSymbol, err := configuratorPlugin.Lookup("Customise")
			if err != nil {
				log.Printf("Could not lookup 'Customise' in %s\n", file)
			}
			if !LoadCustomisationData(customisorSymbol, etcConfigSections) {
				log.Printf("WARNING: No customisation by %s", file)
			}
		}
	} else {
		log.Printf("Fileglob error: %s\n", err)
	}
}
