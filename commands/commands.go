package commands

import (
	"bufio"
	"fmt"
	"mia/auth"
	"mia/disk"
	"mia/filesystem"
	"mia/format"
	"mia/mount"
	"mia/partition"
	"mia/report"
	"os"
	"strings"
)

var IsScript bool = false

// ParseLine parsea una linea de comando y extrae comando + parametros
func ParseLine(line string) (string, map[string]string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", nil
	}

	params := make(map[string]string)
	tokens := tokenize(line)
	if len(tokens) == 0 {
		return "", nil
	}

	cmd := strings.ToUpper(tokens[0])
	i := 1
	for i < len(tokens) {
		token := tokens[i]
		if strings.HasPrefix(token, "-") {
			key := strings.ToLower(strings.TrimPrefix(token, "-"))
			if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
				params[key] = tokens[i+1]
				i += 2
			} else {
				params[key] = "true"
				i++
			}
		} else {
			i++
		}
	}
	return cmd, params
}

// tokenize divide la linea respetando comillas
func tokenize(line string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false

	for _, ch := range line {
		switch {
		case ch == '"':
			inQuote = !inQuote
			current.WriteRune(ch)
		case (ch == ' ' || ch == '\t') && !inQuote:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// Execute ejecuta un comando
func Execute(cmd string, params map[string]string) {
	switch cmd {
	case "MKDISK":
		disk.CreateDisk(params)
	case "RMDISK":
		disk.DeleteDisk(params)
	case "FDISK":
		if _, ok := params["delete"]; ok {
			partition.DeletePartition(params)
		} else {
			partition.CreatePartition(params)
		}
	case "MOUNT":
		mount.Mount(params)
	case "MOUNTED":
		mount.ListMounts()
	case "MKFS":
		format.Mkfs(params)
	case "LOGIN":
		auth.Login(params)
	case "LOGOUT":
		auth.Logout()
	case "MKGRP":
		auth.MkGrp(params)
	case "RMGRP":
		auth.RmGrp(params)
	case "MKUSR":
		auth.MkUsr(params)
	case "RMUSR":
		auth.RmUsr(params)
	case "CHGRP":
		auth.ChGrp(params)
	case "MKDIR":
		execMkdir(params)
	case "MKFILE":
		execMkfile(params)
	case "REP":
		report.Rep(params)
	case "PAUSE":
		fmt.Println("Presione ENTER para continuar...")
		bufio.NewReader(os.Stdin).ReadString('\n')
	case "EXEC":
		execScript(params)
	case "":
		// comentario o linea vacia
	default:
		fmt.Println("Comando desconocido:", cmd)
	}
}

func execMkdir(params map[string]string) {
	if mount.CurrentSession == nil {
		fmt.Println("Error: no hay sesion activa")
		return
	}
	path, ok := params["path"]
	if !ok {
		fmt.Println("Error: MKDIR requiere -path")
		return
	}
	path = strings.ReplaceAll(path, "\"", "")
	_, createParents := params["p"]

	mp, ok2 := mount.GetMountedPartition(mount.CurrentSession.Id)
	if !ok2 {
		fmt.Println("Error: particion no montada")
		return
	}
	filesystem.MkDir(mp, path, createParents, mount.CurrentSession.Uid, mount.CurrentSession.Gid)
}

func execMkfile(params map[string]string) {
	if mount.CurrentSession == nil {
		fmt.Println("Error: no hay sesion activa")
		return
	}
	path, ok := params["path"]
	if !ok {
		fmt.Println("Error: MKFILE requiere -path")
		return
	}
	path = strings.ReplaceAll(path, "\"", "")

	size := 0
	if s, ok2 := params["size"]; ok2 {
		fmt.Sscanf(s, "%d", &size)
	}
	cont := strings.ReplaceAll(params["cont"], "\"", "")

	mp, ok3 := mount.GetMountedPartition(mount.CurrentSession.Id)
	if !ok3 {
		fmt.Println("Error: particion no montada")
		return
	}
	filesystem.MkFile(mp, path, size, cont, mount.CurrentSession.Uid, mount.CurrentSession.Gid)
}

func execScript(params map[string]string) {
	path, ok := params["path"]
	if !ok {
		fmt.Println("Error: EXEC requiere -path")
		return
	}
	path = strings.ReplaceAll(path, "\"", "")
	RunScript(path)
}

// RunScript ejecuta un archivo .smia
func RunScript(path string) {
	IsScript = true
	defer func() { IsScript = false }()
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error al abrir script:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fmt.Printf("[Script:%d] %s\n", lineNum, line)
		cmd, p := ParseLine(line)
		if cmd != "" {
			Execute(cmd, p)
		}
	}
}
