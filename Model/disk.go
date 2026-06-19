package disk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"mia/types"
	"mia/utils"
	"os"
	"os/exec"
	"strings"
	"time"
)

func CreateDisk(params map[string]string) {
	// Validar obligatorios
	sizeStr, ok1 := params["size"]
	path, ok2 := params["path"]
	if !ok1 || !ok2 {
		fmt.Println("Error: MKDISK requiere -size y -path")
		return
	}

	var tamanio int64
	fmt.Sscanf(sizeStr, "%d", &tamanio)
	if tamanio <= 0 {
		fmt.Println("Error: size debe ser mayor a 0")
		return
	}

	unit := "m"
	if u, ok := params["unit"]; ok {
		unit = strings.ToLower(u)
	}

	fit := "F"
	if f, ok := params["fit"]; ok {
		switch strings.ToUpper(f) {
		case "BF":
			fit = "B"
		case "WF":
			fit = "W"
		default:
			fit = "F"
		}
	}

	tamanioBytes := utils.Tamanio(tamanio, unit)
	if tamanioBytes <= 0 {
		fmt.Println("Error: unidad invalida")
		return
	}

	path = strings.ReplaceAll(path, "\"", "")

	// Crear directorios
	dir := path[:strings.LastIndex(path, "/")]
	if dir != "" {
		exec.Command("mkdir", "-p", dir).Output()
	}

	if _, err := os.Stat(path); err == nil {
		fmt.Println("Error: el disco ya existe:", path)
		return
	}

	rand.Seed(time.Now().Unix())
	particionVacia := types.Partition{PartS: -1, PartStart: -1}
	mbr := types.MBR{
		MbrTamanio:      tamanioBytes,
		MbrDskSignature: int32(rand.Intn(1000)),
		DskFit:          fit[0],
		MbrPartitions: [4]types.Partition{
			particionVacia,
			particionVacia,
			particionVacia,
			particionVacia,
		},
	}
	copy(mbr.MbrFechaCreacion[:], utils.FechaActual())

	file, err := os.Create(path)
	if err != nil {
		fmt.Println("Error al crear el archivo:", err)
		return
	}
	defer file.Close()

	// Llenar con ceros
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint8(0))
	file.Seek(tamanioBytes-1, 0)
	file.Write(buffer.Bytes())

	// Escribir MBR al inicio
	file.Seek(0, 0)
	buf2 := bytes.NewBuffer([]byte{})
	binary.Write(buf2, binary.BigEndian, &mbr)
	file.Write(buf2.Bytes())

	fmt.Println("Disco creado exitosamente:", path)
}

func DeleteDisk(params map[string]string) {
	path, ok := params["path"]
	if !ok {
		fmt.Println("Error: RMDISK requiere -path")
		return
	}
	path = strings.ReplaceAll(path, "\"", "")

	fmt.Print("¿Desea eliminar el disco? (si/no): ")
	resp := "no"
	fmt.Scanln(&resp)

	if strings.ToLower(resp) == "si" {
		err := os.Remove(path)
		if err != nil {
			fmt.Println("Error al eliminar el disco:", err)
		} else {
			fmt.Println("Disco eliminado exitosamente")
		}
	} else {
		fmt.Println("Operacion cancelada")
	}
}
