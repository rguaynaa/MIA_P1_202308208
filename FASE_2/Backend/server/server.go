package server

import (
	"encoding/json"
	"fmt"
	"io"
	"mia/commands"
	"mia/mount"
	"mia/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// cmdMutex serializa la ejecucion de comandos, ya que el motor EXT2 usa
// variables globales (sesion activa, particiones montadas) que no son
// seguras para acceso concurrente.
var cmdMutex sync.Mutex

// Start levanta el servidor HTTP en el puerto indicado (ej. ":8080").
func Start(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", withCORS(handleHealth))
	mux.HandleFunc("/api/command", withCORS(handleCommand))
	mux.HandleFunc("/api/script", withCORS(handleScript))
	mux.HandleFunc("/api/script/upload", withCORS(handleScriptUpload))
	mux.HandleFunc("/api/session", withCORS(handleSession))
	mux.HandleFunc("/api/mounted", withCORS(handleMounted))
	mux.HandleFunc("/api/disk/partitions", withCORS(handleDiskPartitions))
	mux.HandleFunc("/api/reports/file", withCORS(handleReportFile))

	fmt.Println("Servidor MIA escuchando en", addr)
	return http.ListenAndServe(addr, mux)
}

// withCORS habilita CORS para que el frontend React (servido en otro
// puerto, p.ej. localhost:5173 con Vite) pueda llamar a esta API sin
// bloqueos del navegador.
func withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// captureOutput redirige os.Stdout a una tuberia mientras ejecuta fn, y
// devuelve todo lo impreso por fn como string. Esto permite reusar
// commands.Execute / commands.RunScript sin tener que tocar su logica
// interna (que esta escrita en terminos de fmt.Println).
func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		fn()
		return ""
	}
	os.Stdout = w

	outCh := make(chan string)
	go func() {
		buf, _ := io.ReadAll(r)
		outCh <- string(buf)
	}()

	fn()

	w.Close()
	os.Stdout = old
	return <-outCh
}

// ---------- /api/command ----------

type commandRequest struct {
	Line string `json:"line"`
}

type commandResponse struct {
	Output string `json:"output"`
}

// handleCommand ejecuta una sola linea de comando (ej. "mkdisk -size=3000
// -path=/home/user/Disco1.mia") y devuelve la salida tal como se veria en
// la consola interactiva de la Fase 1.
func handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "metodo no permitido"})
		return
	}
	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON invalido"})
		return
	}

	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	output := captureOutput(func() {
		cmd, params := commands.ParseLine(req.Line)
		if cmd != "" {
			commands.Execute(cmd, params)
		}
	})

	writeJSON(w, http.StatusOK, commandResponse{Output: output})
}

// ---------- /api/script ----------

type scriptRequest struct {
	Content string `json:"content"`
}

// handleScript recibe el contenido completo de un script .smia como texto
// (por ejemplo pegado en un textarea del frontend), lo guarda temporalmente
// y lo ejecuta linea por linea reusando commands.RunScript, devolviendo
// toda la salida acumulada.
func handleScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "metodo no permitido"})
		return
	}
	var req scriptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON invalido"})
		return
	}

	tmpFile, err := os.CreateTemp("", "script-*.smia")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "no se pudo crear archivo temporal"})
		return
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(req.Content)
	tmpFile.Close()

	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	output := captureOutput(func() {
		commands.RunScript(tmpFile.Name())
	})

	writeJSON(w, http.StatusOK, commandResponse{Output: output})
}

// handleScriptUpload acepta un archivo .smia subido via multipart/form-data
// (campo "file") y lo ejecuta igual que handleScript.
func handleScriptUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "metodo no permitido"})
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "formulario invalido"})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "campo 'file' requerido"})
		return
	}
	defer file.Close()

	tmpFile, err := os.CreateTemp("", "upload-*.smia")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "no se pudo crear archivo temporal"})
		return
	}
	defer os.Remove(tmpFile.Name())
	io.Copy(tmpFile, file)
	tmpFile.Close()

	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	output := captureOutput(func() {
		commands.RunScript(tmpFile.Name())
	})

	writeJSON(w, http.StatusOK, commandResponse{Output: output})
}

// ---------- /api/session ----------

type sessionResponse struct {
	LoggedIn bool   `json:"loggedIn"`
	User     string `json:"user,omitempty"`
	Id       string `json:"id,omitempty"`
	IsRoot   bool   `json:"isRoot,omitempty"`
}

// handleSession devuelve la sesion (LOGIN) activa actualmente, para que el
// frontend sepa si debe mostrar la pantalla de login o el explorador de
// archivos.
func handleSession(w http.ResponseWriter, r *http.Request) {
	if mount.CurrentSession == nil {
		writeJSON(w, http.StatusOK, sessionResponse{LoggedIn: false})
		return
	}
	s := mount.CurrentSession
	writeJSON(w, http.StatusOK, sessionResponse{
		LoggedIn: true,
		User:     s.User,
		Id:       s.Id,
		IsRoot:   s.IsRoot,
	})
}

// ---------- /api/mounted ----------

type mountedPartitionDTO struct {
	Id          string `json:"id"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Correlative int32  `json:"correlative"`
}

// handleMounted lista todas las particiones actualmente montadas, para que
// el frontend pueda ofrecer un selector al momento de hacer LOGIN.
func handleMounted(w http.ResponseWriter, r *http.Request) {
	list := mount.ListMountsStruct()
	out := make([]mountedPartitionDTO, 0, len(list))
	for _, mp := range list {
		out = append(out, mountedPartitionDTO{
			Id:          mp.Id,
			Path:        mp.Path,
			Name:        mp.Name,
			Correlative: mp.Correlative,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------- /api/disk/partitions ----------

type partitionDTO struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Fit    string `json:"fit"`
	Start  int64  `json:"start"`
	Size   int64  `json:"size"`
	Status string `json:"status"`
}

// handleDiskPartitions recibe ?path=/ruta/al/disco.mia y devuelve la lista
// de particiones primarias/extendida (leidas del MBR) junto con las
// logicas (leidas de la cadena de EBR dentro de la extendida, si existe).
// Le permite al frontend mostrar que particiones tiene un disco antes de
// hacer FDISK/MOUNT sobre el.
func handleDiskPartitions(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parametro 'path' requerido"})
		return
	}

	archivo, err := os.Open(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no se pudo abrir el disco: " + err.Error()})
		return
	}
	defer archivo.Close()

	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	mbr := utils.ObtenerMBR(archivo)
	var result []partitionDTO

	for i := 0; i < 4; i++ {
		p := mbr.MbrPartitions[i]
		if p.PartS <= 0 {
			continue
		}
		result = append(result, partitionDTO{
			Name:   utils.BytesToString(p.PartName[:]),
			Type:   string(p.PartType),
			Fit:    string(p.PartFit),
			Start:  p.PartStart,
			Size:   p.PartS,
			Status: string(p.PartStatus),
		})

		if p.PartType == 'E' {
			result = append(result, listarLogicas(archivo, p.PartStart)...)
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// listarLogicas recorre la cadena de EBR de una particion extendida y
// devuelve cada particion logica encontrada como partitionDTO (con
// Type="L" para distinguirlas en el frontend).
func listarLogicas(archivo *os.File, extStart int64) []partitionDTO {
	var out []partitionDTO
	currentOffset := extStart
	for {
		ebr := utils.ObtenerEBR(archivo, currentOffset)
		if ebr.PartS == -1 {
			break
		}
		out = append(out, partitionDTO{
			Name:   utils.BytesToString(ebr.PartName[:]),
			Type:   "L",
			Fit:    string(ebr.PartFit),
			Start:  ebr.PartStart,
			Size:   ebr.PartS,
			Status: string(ebr.PartMount),
		})
		if ebr.PartNext == -1 {
			break
		}
		currentOffset = ebr.PartNext
	}
	return out
}

// ---------- /api/reports/file ----------

// handleReportFile sirve el archivo generado por el comando REP (imagen
// PNG/JPG o texto) para que el frontend pueda mostrarlo, dado ?path=la
// misma ruta de salida que se uso en REP -path=...
func handleReportFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parametro 'path' requerido"})
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "reporte no encontrado: " + err.Error()})
		return
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Write(data)
}
