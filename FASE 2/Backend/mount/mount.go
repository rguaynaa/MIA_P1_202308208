package mount

import (
	"fmt"
	"mia/types"
	"mia/utils"
	"os"
	"strings"
)

// Carnet - ultimos 2 digitos
const Carnet = "08"

var MountedPartitions []types.MountedPartition
var CurrentSession *types.Session

// letras para IDs
var letters = []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}

func Mount(params map[string]string) {
	path, ok1 := params["path"]
	name, ok2 := params["name"]
	if !ok1 || !ok2 {
		fmt.Println("Error: MOUNT requiere -path y -name")
		return
	}
	path = strings.ReplaceAll(path, "\"", "")
	name = strings.ReplaceAll(name, "\"", "")

	archivo, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir disco:", err)
		return
	}
	defer archivo.Close()

	mbr := utils.ObtenerMBR(archivo)

	// Buscar particion con ese nombre
	for i := 0; i < 4; i++ {
		pName := utils.BytesToString(mbr.MbrPartitions[i].PartName[:])
		if pName == name && mbr.MbrPartitions[i].PartType == 'P' {
			// Calcular correlativo
			correlative := int32(1)
			letterIndex := 0
			for _, mp := range MountedPartitions {
				if mp.Path == path {
					correlative++
				}
				letterIndex++
			}

			id := fmt.Sprintf("%s%d%s", Carnet, correlative, letters[letterIndex%10])

			// Actualizar particion
			mbr.MbrPartitions[i].PartStatus = '1'
			mbr.MbrPartitions[i].PartCorrelative = correlative
			copy(mbr.MbrPartitions[i].PartId[:], id)
			utils.EscribirMBR(archivo, mbr)

			mp := types.MountedPartition{
				Path:        path,
				Name:        name,
				Id:          id,
				Correlative: correlative,
			}
			MountedPartitions = append(MountedPartitions, mp)
			fmt.Printf("Particion montada: ID=%s Path=%s Name=%s\n", id, path, name)
			return
		}
	}
	fmt.Println("Error: particion no encontrada o no es primaria")
}

func GetMountedPartition(id string) (*types.MountedPartition, bool) {
	for i := range MountedPartitions {
		if MountedPartitions[i].Id == id {
			return &MountedPartitions[i], true
		}
	}
	return nil, false
}

func ListMounts() {
	if len(MountedPartitions) == 0 {
		fmt.Println("No hay particiones montadas")
		return
	}
	fmt.Println("Particiones montadas:")
	for _, mp := range MountedPartitions {
		fmt.Printf("ID: %-6s | Disco: %-30s | Particion: %s-16 | Correlativo: %d \n",
		 mp.Id, mp.Path, mp.Name, mp.Correlative)
	}

//devuelve la lista de particiones montadas
func ListMountsStruct() []types.MountedPartition {
	return MountedPartitions

}
