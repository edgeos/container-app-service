package handlers

import (
  "fmt"
  "strings"
  "encoding/json"
  "io/ioutil"
  "time"
  "io"
  "os"
  "bytes"
  "mime/multipart"
  "testing"
  "net/http"
  "net/http/httptest"
  "github.build.ge.com/PredixEdgeOS/container-app-service/config"
  "github.build.ge.com/PredixEdgeOS/container-app-service/utils"
  "github.build.ge.com/PredixEdgeOS/container-app-service/types"

  "log"
  "sync"
  "github.com/gorilla/mux"
)

var configFilePath string = "../ecs.json"
var configFilePath1 string = "../test_artifacts/ecsTest1.json"
var configFilePath2 string = "../test_artifacts/ecsTest2.json"
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
          return err
func TestPing(t *testing.T) {
  cfg, err := config.NewConfig(configFilePath)
  if err == nil {
      handler := NewHandler(cfg)
      handlerPing := http.HandlerFunc(handler.ping)

      req, err := http.NewRequest("GET", "/ping", nil)
      if err != nil {
        t.Error("Failed in creating http ping request!")          return err
        t.Fail()
      }
      rr := httptest.NewRecorder()

      handlerPing.ServeHTTP(rr, req)
      handler = nil
      if status := rr.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
          status, http.StatusOK)
      }
      expected := "{\"status\":\"Ok\",\"error\":\"\"}\n"
      if rr.Body.String() != expected {
        t.Errorf("handler returned unexpected body: got %v want %v",
            rr.Body.String(), expected)
      }
  } else {          return err
    t.Error("Failed to create new config!")
    t.Fail()
  }
}

func TestListApplications(t *testing.T) {
  cfg, err := config.NewConfig(configFilePath)
  if err == nil {
      h := NewHandler(cfg)
      handlerListApplications := http.HandlerFunc(h.listApplications)

      // First need to deploy an applications
      metadata := types.Metadata{"testapp", "1.0"}
      file, _ := os.Open(appInputFilePath1)          return err
      if app, err := h.provider.Deploy(metadata, file); err == nil {
        req, err := http.NewRequest("GET", "/applications", nil)
        if err != nil {
          t.Error("Failed in creating http GetApplications request!")
          t.Fail()
        }
        rr := httptest.NewRecorder()
        appId := app.UUID

        handlerListApplications.ServeHTTP(rr, req)
        if status := rr.Code; status != http.StatusOK {
          t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusOK)
        }
  fmt.Printf("rr.Body.String()=%s\n", rr.Body.String())
        expected := appId //"{\"applications\":"
        if !strings.Contains(rr.Body.String(), expected) {
          t.Errorf("handler returned unexpected body: got %v want %v",
              rr.Body.String(), expected)
        }

        err = h.provider.Undeploy(appId)
        if (err != nil) {
          t.Error("Failed to undeploy app with id: " + appId)
          t.Fail()
        }

      } else {
        t.Error("Failed in deploying the application!")
        t.Fail()
      }
  } else {
    t.Error("Failed to create new config!")
    t.Fail()
  }
}

func TestDeployApplicationDTR(t *testing.T) {
  err := Setup(t, configFilePath1)
  if err == nil {
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
    _ = writer.WriteField("metadata", "{\"Name\":\"testapp2\", \"Version\":\"1.0\"}")

    writer.Close()

    req, err := http.NewRequest("POST", "http://127.0.0.1:9001/application/deploy", &body)
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
        fmt.Println("resBody error:", readErr)
        t.Error("Failed in reading response body!")
        t.Fail()
    } else {
      resBodyString := string(resBody)
      fmt.Printf("\nresBodyString=%v\n\n", resBodyString)

      var deployResponse DeployResponse
      unmarshalErr := json.Unmarshal([]byte(resBodyString), &deployResponse)
      if unmarshalErr != nil {
        t.Error("Failed in unmarshalling the response!")
        t.Fail()
      }
      idSlice = append(idSlice, deployResponse.UUID)
    }

    teardownErr := Teardown(t)
    if teardownErr != nil {
      t.Error("Failed in teardown, err is: ", teardownErr)
      t.Fail()
    }
  }
}

func TestDeployApplicationTAR(t *testing.T) {
  err := Setup(t, configFilePath2)
  if err == nil {
    // Add file to POST request body
    var body bytes.Buffer
    writer := multipart.NewWriter(&body)
    file, err := os.Open(appInputFilePath2)
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

    req, err := http.NewRequest("POST", "http://127.0.0.1:9002/application/deploy", &body)
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
        fmt.Println("resBody error:", readErr)
        t.Error("Failed in reading response body!")
        t.Fail()
    } else {
      resBodyString := string(resBody)
      fmt.Printf("\nresBodyString=%v\n\n", resBodyString)

      var deployResponse DeployResponse
      unmarshalErr := json.Unmarshal([]byte(resBodyString), &deployResponse)
      if unmarshalErr != nil {
        t.Error("Failed in unmarshalling the response!")
        t.Fail()
      }
      idSlice = append(idSlice, deployResponse.UUID)
    }

    teardownErr := Teardown(t)
    if teardownErr != nil {
      t.Error("Failed in teardown, err=%v\n", teardownErr)
      t.Fail()
    }
  }
}



// func TestStart(t *testing.T) {
//   var filePath string = "../ecs.json"
//   cfg, err := config.NewConfig(filePath)
//   if err == nil {
//     fmt.Println("Before Start(cfg)...")
//     go Start(cfg)
//
//     fmt.Println("After Start(cfg)...")
//     //if (errStart == nil) {
//       _, errPing := http.Get("http://localhost:9000/applications")
//       if errPing != nil {
//         t.Error("Failed with ping request!")
//         t.Fail()
//       }
//     // } else {
//     //   t.Error("Failed to start server!")
//     //   t.Fail()
//     // }
//   } else {
//     t.Error("Failed to create new config!")
//     t.Fail()
//   }
// }
