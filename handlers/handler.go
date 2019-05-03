package handlers

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
	"os/exec"
	"path/filepath"
	"io/ioutil"
	"strconv"
	"strings"
	"fmt"

	"github.com/gorilla/mux"

	"github.build.ge.com/PredixEdgeOS/container-app-service/config"
	"github.build.ge.com/PredixEdgeOS/container-app-service/provider"
	"github.build.ge.com/PredixEdgeOS/container-app-service/types"
	"github.build.ge.com/PredixEdgeOS/container-app-service/utils"
)

//Constants ...
const (
	Ok       = "Ok"
	Fail     = "Fail"
	Deployed = "Deployed"
	Running  = "Running"
	Stopped  = "Stopped"
	NoID     = "No ID in request"
	GenTPMKeyCommandFmt = "tpm2tss-genkey -a rsa -s 2048 %s"
	GenOpensslKeyCommandFmt = "openssl genrsa -out %s 2048"
	GenTPMPubKeyCommandFmt = "openssl rsa -engine tpm2tss -inform engine -in %s -pubout -outform pem"
	GenOpensslPubKeyCommandFmt = "openssl rsa -in %s -pubout -outform pem"
)

// key name body
type KeyName struct {
	Name string `json:"name"`
}

//BasicResponse ...
type BasicResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

//HasKeyResponse ...
type HasKeyResponse struct {
	HasKey bool `json:"hasKey"`
}

//PubKeyResponse ...
type PubKeyResponse struct {
	PubKey string `json:"pubKey"`
}

//DeployResponse ...
type DeployResponse struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Error   string `json:"error"`
}

//AppDetailsResponse ...
type AppDetailsResponse struct {
	UUID       string            `json:"uuid"`
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Containers []types.Container `json:"containers"`
	Status     string            `json:"status"`
	Error      string            `json:"error"`
}

//Handler ...
type Handler struct {
	cfg      config.Config
	provider provider.Provider
}

//NewHandler ...
func NewHandler(c config.Config) *Handler {
	return &Handler{
		cfg:      c,
		provider: provider.NewProvider(c)}
}

func (h *Handler) ping(w http.ResponseWriter, r *http.Request) {
	response := BasicResponse{Status: Ok, Error: ""}
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) listApplications(w http.ResponseWriter, r *http.Request) {
	response := h.provider.ListApplications()
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) listPersistentApplications(w http.ResponseWriter, r *http.Request) {
        response := h.provider.ListPersistentApplications()
        json.NewEncoder(w).Encode(response)
}

func (h *Handler) getApplication(w http.ResponseWriter, r *http.Request) {
	response := AppDetailsResponse{Status: Fail, Error: ""}
	vars := mux.Vars(r)
	id, exists := vars["id"]
	if exists {
		if details, err := h.provider.GetApplication(id); err == nil {
			response.Status = Ok
			response.UUID = details.UUID
			response.Name = details.Name
			response.Version = details.Version
			response.Containers = details.Containers
		} else {
			response.Status = Fail
			response.Error = err.Error()
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		response.Status = Fail
		response.Error = NoID
		w.WriteHeader(http.StatusBadRequest)
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) deployAppGeneric(w http.ResponseWriter, r *http.Request, persistent bool) {
	response := DeployResponse{Status: Fail, Error: ""}
	var metadata types.Metadata
	if err := r.ParseMultipartForm(0); err == nil {
		if err := json.Unmarshal([]byte(r.FormValue("metadata")), &metadata); err == nil {
			m := r.MultipartForm
			artifacts := m.File["artifact"]
			for i := range artifacts {
				if file, err := artifacts[i].Open(); err == nil {
					defer file.Close()
					if app, err := h.provider.Deploy(metadata, file, persistent); err == nil {
						response.UUID = app.UUID
						response.Name = app.Name
						response.Version = app.Version
						response.Status = Ok
					} else {
						response.Error = err.Error()
						w.WriteHeader(http.StatusInternalServerError)
					}
				} else {
					response.Error = err.Error()
					w.WriteHeader(http.StatusBadRequest)
				}
			}
			m.RemoveAll()
		} else {
			response.Error = err.Error()
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		response.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) deployApplication(w http.ResponseWriter, r *http.Request) {
	h.deployAppGeneric(w, r, false)
}

func (h *Handler) deployPersistentApplication(w http.ResponseWriter, r *http.Request) {
	h.deployAppGeneric(w, r, true)
}

func (h *Handler) restartApplication(w http.ResponseWriter, r *http.Request) {
	response := BasicResponse{Status: Ok, Error: ""}

	vars := mux.Vars(r)
	id, exists := vars["id"]
	if exists {
		if err := h.provider.Restart(id); err != nil {
			response.Status = Fail
			response.Error = err.Error()
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		response.Status = Fail
		response.Error = NoID
		w.WriteHeader(http.StatusBadRequest)
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) startApplication(w http.ResponseWriter, r *http.Request) {
	response := BasicResponse{Status: Ok, Error: ""}

	vars := mux.Vars(r)
	id, exists := vars["id"]
	if exists {
		if err := h.provider.Start(id); err != nil {
			response.Status = Fail
			response.Error = err.Error()
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		response.Status = Fail
		response.Error = NoID
		w.WriteHeader(http.StatusBadRequest)
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) stopApplication(w http.ResponseWriter, r *http.Request) {
	response := BasicResponse{Status: Ok, Error: ""}

	vars := mux.Vars(r)
	id, exists := vars["id"]
	if exists {
		if err := h.provider.Stop(id); err != nil {
			response.Status = Fail
			response.Error = err.Error()
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		response.Status = Fail
		response.Error = NoID
		w.WriteHeader(http.StatusBadRequest)
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) statusApplication(w http.ResponseWriter, r *http.Request) {
	response := BasicResponse{Status: Ok, Error: ""}

	vars := mux.Vars(r)
	id, exists := vars["id"]
	if exists {
		if details, err := h.provider.GetApplication(id); err == nil {
			running := true
			UP := regexp.MustCompile("^Up")
			for c := range details.Containers {
				container := details.Containers[c]
				if !UP.MatchString(container.State) {
					running = false
				}
			}
			if running && len(details.Containers) > 0 {
				response.Status = Running
			} else {
				response.Status = Stopped
			}
		} else {
			response.Status = Fail
			response.Error = err.Error()
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		response.Status = Fail
		response.Error = NoID
		w.WriteHeader(http.StatusBadRequest)
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) purgeApplication(w http.ResponseWriter, r *http.Request) {
	response := BasicResponse{Status: Ok, Error: ""}

	vars := mux.Vars(r)
	id, exists := vars["id"]
	if exists {
		if err := h.provider.Undeploy(id); err != nil {
			response.Status = Fail
			response.Error = err.Error()
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		response.Status = Fail
		response.Error = NoID
		w.WriteHeader(http.StatusBadRequest)
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) purgePersistentApplication(w http.ResponseWriter, r *http.Request) {
	response := BasicResponse{Status: Ok, Error: ""}

	vars := mux.Vars(r)
	name, exists := vars["name"]
	if exists {
		if err := h.provider.PurgePersistent(name); err != nil {
			response.Status = Fail
			response.Error = err.Error()
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		response.Status = Fail
		response.Error = NoID
		w.WriteHeader(http.StatusBadRequest)
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) killApplication(w http.ResponseWriter, r *http.Request) {
	response := BasicResponse{Status: Ok, Error: ""}

	vars := mux.Vars(r)
	id, exists := vars["id"]
	if exists {
		if err := h.provider.Kill(id); err != nil {
			response.Status = Fail
			response.Error = err.Error()
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		response.Status = Fail
		response.Error = NoID
		w.WriteHeader(http.StatusBadRequest)
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) createKey(w http.ResponseWriter, r *http.Request) {
	response := BasicResponse{Status: Ok, Error: ""}
	log.Println("Provisioning new decryption key!")
	//todo: make key intead of touching a file

	decoder := json.NewDecoder(r.Body)
	var nameJson KeyName
	err := decoder.Decode(&nameJson)
	if err != nil {
		log.Printf("Could not proccess request body:\n%v", err)
		response.Status = "FAIL"
		response.Error = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	var name = nameJson.Name

	err = os.MkdirAll(filepath.Dir(h.cfg.KeyLocation), 0744)
	if err != nil {
		log.Printf("Could not create key directory (%s) due to error: \n%v\n",
			filepath.Dir(h.cfg.KeyLocation), err)
		response.Status = "FAIL"
		response.Error = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	err = ioutil.WriteFile(h.cfg.KeyName, []byte(name), 0644)
	if err != nil {
		log.Printf("Failed writing key name to file (%s): %v\n", h.cfg.KeyName, err)
		response.Status = "FAIL"
		response.Error = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	// result, err := exec.Command("sh", "-c", "systemctl is-active tpm2-abrmd | grep -o inactive | wc -l").Output()
	// if err != nil {
	// 	log.Printf("TPM test command finished with error: %v\n", err)
	// 	response.Status = "FAIL"
	// 	response.Error = err.Error()
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	json.NewEncoder(w).Encode(response)
	// 	return
	// }
	// lineCount, err := strconv.Atoi(strings.Trim(string(result), "\n"))
	// if err != nil {
	// 	log.Printf("Error parsing result of TPM test (%s): %v\n", result, err)
	// 	response.Status = "FAIL"
	// 	response.Error = err.Error()
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	json.NewEncoder(w).Encode(response)
	// 	return
	// }
	//hasTPM := lineCount == 0
	hasTPM, err := hasTPM2()
	if err != nil {
		log.Printf("Error while attempting to detect TPM2.0 presence: %v\n", err)
		response.Status = "FAIL"
		response.Error = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	genKeyCommand := fmt.Sprintf(GenOpensslKeyCommandFmt, h.cfg.KeyLocation)
	if hasTPM {
		log.Println("TPM detected, locking private key with TPM")
		genKeyCommand = fmt.Sprintf(GenTPMKeyCommandFmt, h.cfg.KeyLocation)
	} else {
		log.Println("NO TPM FOUND, generating private key using openssl tools")
	}
	cmd := exec.Command("sh", "-c", genKeyCommand)
	err = cmd.Run()
	if err != nil {
		log.Printf("Error generating private key: \n%v",  err)
		response.Status = "FAIL"
		response.Error = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) hasKey(w http.ResponseWriter, r *http.Request) {
	log.Println("Checking if key has been generated due to API request")
	if _, err := os.Stat(h.cfg.KeyLocation); err == nil {
		log.Println("  Have key")
		json.NewEncoder(w).Encode(HasKeyResponse{HasKey: true})
	} else if os.IsNotExist(err) {
		log.Println("  Do not have key")
		json.NewEncoder(w).Encode(HasKeyResponse{HasKey: false})
	} else {
		log.Printf("hasKey returned an error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BasicResponse{Status: Fail, Error: err.Error()})
	}
}

func (h *Handler) getKey(w http.ResponseWriter, r *http.Request) {
	log.Println("Responding to request for public key from API")
	if _, err := os.Stat(h.cfg.KeyLocation); err == nil {
		genPubKeyCommand := fmt.Sprintf(GenOpensslPubKeyCommandFmt, h.cfg.KeyLocation)
		hasTPM, err := hasTPM2()
		if hasTPM {
			log.Println("TPM detected, generating public key from TPM locked private key.")
			genPubKeyCommand = fmt.Sprintf(GenTPMPubKeyCommandFmt, h.cfg.KeyLocation)
		} else {
			log.Printf("NO TPM FOUND, generating public key from openssl generated private key.")
		}
		pubKeyBytes, err := exec.Command("sh", "-c", genPubKeyCommand).Output()
		if err != nil {
			log.Printf("Error generating public key: \n%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(BasicResponse{Status: Fail, Error: err.Error()})			
		}
		json.NewEncoder(w).Encode(PubKeyResponse{PubKey: string(pubKeyBytes)})
	} else if os.IsNotExist(err) {
		log.Printf("getKey returned an error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BasicResponse{Status: Fail, Error: "Key does not exist"})
	} else {
		log.Printf("getKey returned an error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(BasicResponse{Status: Fail, Error: err.Error()})
	}
}

func hasTPM2() (bool, error) {
	result, err := exec.Command("sh", "-c", "systemctl is-active tpm2-abrmd | grep -o inactive | wc -l").Output()
	if err != nil {
		return false, err
	}
	lineCount, err := strconv.Atoi(strings.Trim(string(result), "\n"))
	if err != nil {
		return false, err
	}
	hasTPM := lineCount == 0
	return hasTPM, nil
}

func setupServer(cfg config.Config) *http.Server {
	handler := NewHandler(cfg)
	router := mux.NewRouter()
	router.HandleFunc("/ping", handler.ping).Methods("GET")
	router.HandleFunc("/applications", handler.listApplications).Methods("GET")
	router.HandleFunc("/persistent-applications", handler.listPersistentApplications).Methods("GET")
	router.HandleFunc("/application/{id}", handler.getApplication).Methods("GET")
	router.HandleFunc("/application/deploy", handler.deployApplication).Methods("POST")
	router.HandleFunc("/application/deploy-persistent", handler.deployPersistentApplication).Methods("POST")
	router.HandleFunc("/application/restart/{id}", handler.restartApplication).Methods("POST")
	router.HandleFunc("/application/start/{id}", handler.startApplication).Methods("POST")
	router.HandleFunc("/application/stop/{id}", handler.stopApplication).Methods("POST")
	router.HandleFunc("/application/status/{id}", handler.statusApplication).Methods("GET")
	router.HandleFunc("/application/purge/{id}", handler.purgeApplication).Methods("POST")
	router.HandleFunc("/application/purge-persistent/{name}", handler.purgePersistentApplication).Methods("POST")
	router.HandleFunc("/application/kill/{id}", handler.killApplication).Methods("POST")
	router.HandleFunc("/provision/createKey", handler.createKey).Methods("POST")
	router.HandleFunc("/provision/hasKey", handler.hasKey).Methods("GET")
	router.HandleFunc("/provision/getKey", handler.getKey).Methods("GET")
	server := &http.Server{
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
	}
	return server
}

// Start the HTTP server to handle client requests
func Start(cfg config.Config) {
	server := setupServer(cfg)
	for {
		once := sync.Once{}
		utils.RetryWithBackoff(utils.NewSimpleBackoff(time.Second, time.Minute, 0.2, 2), func() error {
			// Intentionally ignore error, socket might not exist
			_ = os.Remove(cfg.ListenAddress)

			cappsdSock, err := net.Listen("unix", cfg.ListenAddress)

			if err != nil {
				log.Println("Error binding socket - ", err)
				return err
			}
			err = os.Chmod(cfg.ListenAddress, 0760)
			if err != nil {
				log.Println("Error setting socket permissions - ", err)
				return err
			}
			err = server.Serve(cappsdSock)

			once.Do(func() {
				log.Println("Error running http api - ", err)
			})

			return err
		})
	}
}
