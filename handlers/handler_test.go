package handlers

import (
  "fmt"
  "strings"
  "io/ioutil"
  "os"
  "io"
  "encoding/json"
  "bytes"
  "mime/multipart"
  "time"
  "net/http"
  "testing"
  "log"
  "sync"
  "github.com/gorilla/mux"
  "github.build.ge.com/PredixEdgeOS/container-app-service/config"
  "github.build.ge.com/PredixEdgeOS/container-app-service/utils"
)

var configFilePath string = "../ecs.json"
var appInputFilePath1 string = "../test_artifacts/matlab-sim-app.tar.gz"
var appInputFilePath2 string = "../test_artifacts/helloapp.tar.gz"
var idSlice []string
var handler *Handler

func setupServerInTest(cfg config.Config) *http.Server {
	handler = NewHandler(cfg)
	router := mux.NewRouter()
	router.HandleFunc("/ping", handler.ping).Methods("GET")
	router.HandleFunc("/applications", handler.listApplications).Methods("GET")
	router.HandleFunc("/application/{id}", handler.getApplication).Methods("GET")
	router.HandleFunc("/application/deploy", handler.deployApplication).Methods("POST")
	router.HandleFunc("/application/restart/{id}", handler.restartApplication).Methods("POST")
	router.HandleFunc("/application/start/{id}", handler.startApplication).Methods("POST")
	router.HandleFunc("/application/stop/{id}", handler.stopApplication).Methods("POST")
	router.HandleFunc("/application/status/{id}", handler.statusApplication).Methods("GET")
	router.HandleFunc("/application/purge/{id}", handler.purgeApplication).Methods("POST")

	listenAddr := cfg.ListenAddress
	server := &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
	}
	return server
}

func StartInTest(cfg config.Config) {
	server := setupServerInTest(cfg)
	for {
		once := sync.Once{}
		utils.RetryWithBackoff(utils.NewSimpleBackoff(time.Second, time.Minute, 0.2, 2), func() error {
			err := server.ListenAndServe()
			once.Do(func() {
				log.Println("Error running http api - ", err)
			})
			return err
		})
	}
}

func Setup(t *testing.T, configPath string) error {
  var err error
  cfg, err := config.NewConfig(configPath)
  if err == nil {
    go StartInTest(cfg)
    // Wait 5 seconds to make sure the server starts first before sending the POST request
    time.Sleep(5*time.Second)
  } else {
    t.Error("Failed to create new config! err is: ", err)
    t.Fail()
  }
  return err
}

func Teardown(t *testing.T) error {
  for _, id := range idSlice {
    err := handler.provider.Undeploy(id)
    if (err != nil) {
      t.Error("Failed to undeploy app with id: " + id)
      t.Fail()
      return err
    }
  }
  idSlice = nil
  handler = nil

  return nil
}

func TestNewHandler(t *testing.T) {
  cfg, err := config.NewConfig(configFilePath)
  if err == nil {
      handler := NewHandler(cfg)
      if (handler == nil) {
        t.Error("Failed to create new handler!")
        t.Fail()
      }
      handler = nil
  } else {
    t.Error("Failed to create new config!")
    t.Fail()
  }
}

func TestAllHandlers(t *testing.T) {
  Setup(t, configFilePath)

  PingTest(t)
  DeployApplicationDTRTest(t)
  ListApplicationsTest(t)
  GetApplicationTest(t)
  DeployApplicationTARTest(t)

  Teardown(t)
}

func PingTest(t *testing.T) {
  req, err := http.NewRequest("GET", "http://127.0.0.1:9000/ping", nil)
  if err != nil {
    t.Error("Failed in creating http ping request!")
    t.Fail()
  }

  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
    t.Error("Failed in executing the http ping request! err is: ", err)
    t.Fail()
  }
  defer resp.Body.Close()

  if statusCode := resp.StatusCode; statusCode != http.StatusOK {
    t.Errorf("handler returned wrong status code: got %v want %v",
      statusCode, http.StatusOK)
    t.Fail()
  }

  fmt.Println("Passed Ping Test")
}

func ListApplicationsTest(t *testing.T) {
  req, err := http.NewRequest("GET", "http://127.0.0.1:9000/applications", nil)
  if err != nil {
    t.Error("Failed in creating http ListApplications request!")
    t.Fail()
  }

  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
    t.Error("Failed in executing the http ListApplications request! err is: ", err)
    t.Fail()
  }
  defer resp.Body.Close()

  if statusCode := resp.StatusCode; statusCode != http.StatusOK {
    t.Errorf("handler returned wrong status code: got %v want %v",
      statusCode, http.StatusOK)
    t.Fail()
  }

  bodyBytes, _ := ioutil.ReadAll(resp.Body)
  bodyString := string(bodyBytes)
  expected := idSlice[0]
  if !strings.Contains(bodyString, expected) {
    t.Errorf("handler returned unexpected body: got %v want %v",
        bodyString, expected)
    t.Fail()
  }

  fmt.Println("Passed ListApplications Test")
}

func GetApplicationTest(t *testing.T) {
  appId := idSlice[0]
  req, err := http.NewRequest("GET", "http://127.0.0.1:9000/application/" + appId, nil)
  if err != nil {
    t.Error("Failed in creating http GetApplication request!")
    t.Fail()
  }

  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
    t.Error("Failed in executing the http GetApplication request! err is: ", err)
    t.Fail()
  }
  defer resp.Body.Close()

  if statusCode := resp.StatusCode; statusCode != http.StatusOK {
    t.Errorf("handler returned wrong status code: got %v want %v",
      statusCode, http.StatusOK)
    t.Fail()
  }

  bodyBytes, _ := ioutil.ReadAll(resp.Body)
  bodyString := string(bodyBytes)
  expected := appId
  if !strings.Contains(bodyString, expected) {
    t.Errorf("handler returned unexpected body: got %v want %v",
        bodyString, expected)
    t.Fail()
  }

  fmt.Println("Passed GetApplication Test")
}

func DeployApplicationDTRTest(t *testing.T) {
    // Add file to POST request body
    var body bytes.Buffer
    writer := multipart.NewWriter(&body)
    file, err := os.Open(appInputFilePath1)
    if err != nil {
      t.Error("Failed to open app tar file!")
      t.Fail()
    }
    defer file.Close()
    formWriter, err := writer.CreateFormFile("artifact", appInputFilePath1)
    if err != nil {
      t.Error("Failed to create form file!")
      t.Fail()
    }
    if _, err = io.Copy(formWriter, file); err != nil {
      t.Error("Failed to copy file to formWriter!")
      t.Fail()
    }

    // Add other form fields
    _ = writer.WriteField("metadata", "{\"Name\":\"testapp1\", \"Version\":\"1.0\"}")

    writer.Close()

    req, err := http.NewRequest("POST", "http://127.0.0.1:9000/application/deploy", &body)
    if err != nil {
      t.Error("Failed in creating http application deploy request!")
      t.Fail()
    }
    req.Header.Set("Content-Type", writer.FormDataContentType())

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
      t.Error("Failed in executing the http request! err is: ", err)
      t.Fail()
    }
    defer resp.Body.Close()

    resBody, readErr := ioutil.ReadAll(resp.Body)
    if readErr != nil {
        t.Error("Failed in reading response body!")
        t.Fail()
    } else {
      resBodyString := string(resBody)

      var deployResponse DeployResponse
      unmarshalErr := json.Unmarshal([]byte(resBodyString), &deployResponse)
      if unmarshalErr != nil {
        t.Error("Failed in unmarshalling the response!")
        t.Fail()
      }
      idSlice = append(idSlice, deployResponse.UUID)
    }

    fmt.Println("Passed DeployApplicationDTR Test")
}

func DeployApplicationTARTest(t *testing.T) {
  // Add file to POST request body
  var body bytes.Buffer
  writer := multipart.NewWriter(&body)
  file, err := os.Open(appInputFilePath2)
  if err != nil {
    t.Error("Failed to open app tar file!")
    t.Fail()
  }
  defer file.Close()
  formWriter, err := writer.CreateFormFile("artifact", appInputFilePath2)
  if err != nil {
    t.Error("Failed to create form file!")
    t.Fail()
  }
  if _, err = io.Copy(formWriter, file); err != nil {
    t.Error("Failed to copy file to formWriter!")
    t.Fail()
  }

  // Add other form fields
  _ = writer.WriteField("metadata", "{\"Name\":\"testapp2\", \"Version\":\"1.0\"}")

  writer.Close()

  req, err := http.NewRequest("POST", "http://127.0.0.1:9000/application/deploy", &body)
  if err != nil {
    t.Error("Failed in creating http application deploy request!")
    t.Fail()
  }
  req.Header.Set("Content-Type", writer.FormDataContentType())

  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
    t.Error("Failed in executing the http request! err is: ", err)
    t.Fail()
  }
  defer resp.Body.Close()

  resBody, readErr := ioutil.ReadAll(resp.Body)
  if readErr != nil {
      t.Error("Failed in reading response body!")
      t.Fail()
  } else {
    resBodyString := string(resBody)
    var deployResponse DeployResponse
    unmarshalErr := json.Unmarshal([]byte(resBodyString), &deployResponse)
    if unmarshalErr != nil {
      t.Error("Failed in unmarshalling the response!")
      t.Fail()
    }
    idSlice = append(idSlice, deployResponse.UUID)
  }

  fmt.Println("Passed DeployApplicationTAR Test")
}
