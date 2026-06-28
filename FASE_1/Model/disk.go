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
	"time"
)

func CreateDisk(tamanio int64, ajuste, unit, path string) {
	rand.Seed(time.Now().Unix())
	particionVacia := types.Partition{Tamanio: -1}
	mbr := types.MBR{
		Tamanio: utils.Tamanio(tamanio, unit),
		Id:      int16(rand.Intn(100)),
		Ajuste:  ajuste[0],
		Particiones: [4]types.Partition{
			particionVacia, particionVacia, particionVacia, particionVacia,
		},
	}
	copy(mbr.FechaCreacion[:], date())

	exec.Command("mkdir", "-p", path).Output()
	exec.Command("rmdir", path).Output()

	if _, err := os.Stat(path); err == nil {
		fmt.Println("El archivo ya existe, vuelva a intentarlo....")
		return
	}

	file, err := os.Create(path)
	if err != nil {
		fmt.Println("Ocurrio un error al crear el archivo")
		return
	}
	defer file.Close()

	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint8(0))
	file.Write(buffer.Bytes())

	file.Seek(mbr.Tamanio-int64(1), 0)
	file.Write(buffer.Bytes())

	file.Seek(0, 0)
	buffer.Reset()
	binary.Write(buffer, binary.BigEndian, &mbr)
	file.Write(buffer.Bytes())

	fmt.Println("Disco creado exitosamente en:", path)
}

func DeleteDisk(path string) {
	fmt.Println("¿Quieres eliminar el disco? (si/no)")
	var mess string
	fmt.Scanln(&mess)

	if mess == "si" {
		err := os.Remove(path)
		if err != nil {
			fmt.Println("Error al eliminar el disco")
		} else {
			fmt.Println("Disco eliminado exitosamente")
		}
	} else {
		fmt.Println("Operacion cancelada")
	}
}

func date() string {
	t := time.Now()
	return fmt.Sprintf("%02d-%02d-%d %02d:%02d:%02d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute(), t.Second())
}