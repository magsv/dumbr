
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

var (
	Version string
	Build   string
)

var responseTemplates *template.Template

type Configuration struct {
	RequestConfigurations []RequestConfig `json:"requestConfig"`
}

type RequestConfig struct {
	ResponseTemplate string `json:"responseTemplateName"`
	Resource         string `json:"resource"` //the resource url to map to
	Method           string `json:"method"`   //the http method to respond to GET or POST
}

func getGuuid() string {
	id := uuid.NewV4()
	return id.String()
}

func logHTTPDump(data []byte) {
	zap.S().Debugf("REQUEST:%s", string(data))
}
func logRequest(r *http.Request, message string, uuid string) {
	zap.S().Infof("REQUEST,remoteAdd:%s, refered:%s,userAgent:%s,uuid:%s,requestURI:%s", r.RemoteAddr, r.Referer(), r.UserAgent(), uuid, r.RequestURI)
}

func (RCnfg *RequestConfig) HandlePost(response http.ResponseWriter, r *http.Request) {
	uuid := getGuuid()
	logRequest(r, "POST CALLED", uuid)
	dumpRequest(r, RCnfg, "POST", uuid)
	respondWithTemplate(response, RCnfg.ResponseTemplate, r)

}

func respondWithTemplate(w http.ResponseWriter, templateName string, r *http.Request) {
	var err error
	if err = responseTemplates.ExecuteTemplate(w, templateName, r); err != nil {
		zap.S().Errorf("Failed in execution of template with name:%s, error:%s", templateName, err.Error())
		http.Error(w, "500 Internal Server Error", 500)
		return
	}
}

func dumpRequest(r *http.Request, RCnfg *RequestConfig, mode string, uuid string) {
	var rDump []byte
	var err error
	if rDump, err = httputil.DumpRequest(r, true); err != nil {
		zap.S().Errorf("Failed in dump request:%s", err.Error())
		return
	}
	logHTTPDump(rDump)

}

func (RCnfg *RequestConfig) HandleGet(w http.ResponseWriter, r *http.Request) {

	var err error
	uuid := getGuuid()
	logRequest(r, "GET Called - query", uuid)
	dumpRequest(r, RCnfg, "GET", uuid)
	if err = responseTemplates.ExecuteTemplate(w, RCnfg.ResponseTemplate, r); err != nil {
		zap.S().Errorf("Failed in handle get:%s", err.Error())
		http.Error(w, "500 Internal Server Error", 500)
		return
	}
}

func parseTemplates(templateFolder string, templateExtension string) (*template.Template, error) {
	var err error
	templ := template.New("")
	err = filepath.Walk(templateFolder, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, templateExtension) {
			_, err = templ.ParseFiles(path)
			if err != nil {
				zap.S().Errorf("Failed in parsing template:%s, error;%s", path, err)
			}
			zap.S().Infof("Parsed template from templateFolder:%s, template:%s", templateFolder, path)
		}

		return err
	})

	if err != nil {
		return templ, err
	}

	return templ, nil
}

func parseConfigFile(configFile string) (Configuration, error) {
	var jsonFile *os.File
	var err error
	var cnfg Configuration
	if jsonFile, err = os.Open(configFile); err != nil {
		return cnfg, err
	}
	defer jsonFile.Close()
	if byteValue, err := ioutil.ReadAll(jsonFile); err != nil {
		return cnfg, err
	} else {
		if err := json.Unmarshal(byteValue, &cnfg); err != nil {
			return cnfg, err
		} else {
			return cnfg, nil
		}

	}
}

func startServer(port string, templatePath,
	serverKey, serverCrt, configurationFile, logConfig string) {
	var err error
	var cfg zap.Config
	var requestCnfg Configuration

	//configure the logging
	if logConfig == "" {
		cfg = zap.Config{

			Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
			Development:      false,
			Encoding:         "console",
			EncoderConfig:    zap.NewProductionEncoderConfig(),
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}
	} else {
		//need to parse the config from the file
		if rawJSON, err := ioutil.ReadFile(logConfig); err != nil {
			panic(err)
		} else {
			if err := json.Unmarshal(rawJSON, &cfg); err != nil {
				panic(err)
			}
		}

	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(logger)
	zap.S().Infof("Starting server listening on port:%s", port)

	//parse all of the templates
	if responseTemplates, err = parseTemplates(templatePath, ".template"); err != nil {
		zap.S().Errorf("Failed in reading templates, error:%s", err.Error())
		return
	}
	zap.S().Infof("Parsed templates in folder:%s", templatePath)
	if requestCnfg, err = parseConfigFile(configurationFile); err != nil {
		zap.S().Errorf("Failed in reading config file:%s", err.Error())
		return
	}
	router := httprouter.New()
	for i := 0; i < len(requestCnfg.RequestConfigurations); i++ {
		zap.S().Infof("Adding request handler for resource:%s, method:%s, templatateName:%s", requestCnfg.RequestConfigurations[i].Resource,
			requestCnfg.RequestConfigurations[i].Method, requestCnfg.RequestConfigurations[i].ResponseTemplate)
		if strings.ToLower(requestCnfg.RequestConfigurations[i].Method) == "get" {
			router.HandlerFunc("GET", requestCnfg.RequestConfigurations[i].Resource, requestCnfg.RequestConfigurations[i].HandleGet)
		} else if strings.ToLower(requestCnfg.RequestConfigurations[i].Method) == "post" {
			router.HandlerFunc("POST", requestCnfg.RequestConfigurations[i].Resource, requestCnfg.RequestConfigurations[i].HandlePost)
		}

	}
	if serverCrt == "" && serverKey == "" {
		zap.S().Infof("Starting normal http server..")
		zap.S().Fatal(http.ListenAndServe(":"+port, router))
	} else {
		//need to start a tls server
		zap.S().Info("Starting ssl/tls enables server")
		zap.S().Fatal(http.ListenAndServeTLS(":"+port, serverCrt, serverKey, router))
	}
}

func main() {
	showVersion := flag.Bool("version", false, "If specified will print out the version information and then exit")
	templatePath := flag.String("templates", "./templates", "Specifies the path to the templates folder")
	port := flag.String("port", "", "Specifies the port to listen on")
	serverKey := flag.String("serverKey", "", "Specifies the server ssl key")
	serverCrt := flag.String("serverCrt", "", "Specifies the server SSL/TLS certificate")
	configuration := flag.String("configuration", "", "Path to json file containing configuration for service")
	logConfig := flag.String("logconfig", "", "The path to the zap logger configuration file")
	flag.Parse()
	if *showVersion {
		fmt.Println("Version:", Version)
		fmt.Println("Build time:", Build)
		return
	}
	if *templatePath != "" && *port != "" && *configuration != "" {
		startServer(*port, *templatePath, *serverKey, *serverCrt, *configuration, *logConfig)
	} else {
		fmt.Println("Missing required input parameters...")
		flag.PrintDefaults()
		return
	}

}
