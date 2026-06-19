package partition

import (
	"fmt"
	"mia/types"
	"mia/utils"
	"os"
	"strings"
	"unsafe"
)

func CreatePartition(params map[string]string) {
	sizeStr, ok1 := params["size"]
	path, ok2 := params["path"]
	name, ok3 := params["name"]
	if !ok1 || !ok2 || !ok3 {
		fmt.Println("Error: FDISK requiere -size, -path y -name")
		return
	}

	var tamanio int64
	fmt.Sscanf(sizeStr, "%d", &tamanio)

	unit := "k"
	if u, ok := params["unit"]; ok {
		unit = strings.ToLower(u)
	}
	ptype := "P"
	if t, ok := params["type"]; ok {
		ptype = strings.ToUpper(t)
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

	path = strings.ReplaceAll(path, "\"", "")
	name = strings.ReplaceAll(name, "\"", "")

	tamanioBytes := utils.Tamanio(tamanio, unit)
	if tamanioBytes <= 0 {
		fmt.Println("Error: tamanio invalido")
		return
	}

	archivo, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer archivo.Close()

	mbr := utils.ObtenerMBR(archivo)
	mbrSize := int64(unsafe.Sizeof(mbr))

	// Verificar nombre unico
	for i := 0; i < 4; i++ {
		n := utils.BytesToString(mbr.MbrPartitions[i].PartName[:])
		if n == name {
			fmt.Println("Error: ya existe una particion con ese nombre")
			return
		}
	}

	// Solo una extendida por disco
	if ptype == "E" {
		for i := 0; i < 4; i++ {
			if mbr.MbrPartitions[i].PartType == 'E' {
				fmt.Println("Error: ya existe una particion extendida")
				return
			}
		}
	}

	switch ptype {
	case "P", "E":
		crearPrimariaOExtendida(archivo, mbr, mbrSize, tamanioBytes, ptype, fit, name)
	case "L":
		crearLogica(archivo, mbr, tamanioBytes, fit, name)
	default:
		fmt.Println("Error: tipo de particion invalido")
	}
}

func crearPrimariaOExtendida(archivo *os.File, mbr types.MBR, mbrSize, tamanioBytes int64, ptype, fit, name string) {
	// Encontrar espacio libre con el ajuste
	inicio := calcularInicio(mbr, mbrSize, tamanioBytes, fit)
	if inicio == -1 {
		fmt.Println("Error: no hay espacio suficiente en el disco")
		return
	}

	// Buscar slot libre en MBR
	for i := 0; i < 4; i++ {
		if mbr.MbrPartitions[i].PartStart == -1 || mbr.MbrPartitions[i].PartStart == 0 {
			newPartition := types.Partition{
				PartStatus: '0',
				PartType:   ptype[0],
				PartFit:    fit[0],
				PartStart:  inicio,
				PartS:      tamanioBytes,
			}
			copy(newPartition.PartName[:], name)
			mbr.MbrPartitions[i] = newPartition
			utils.EscribirMBR(archivo, mbr)

			fmt.Printf("Particion %s creada: inicio=%d tamanio=%d nombre=%s\n", ptype, inicio, tamanioBytes, name)

			// Si es extendida, escribir primer EBR
			if ptype == "E" {
				ebr := types.EBR{
					PartMount: '0',
					PartFit:   fit[0],
					PartStart: inicio,
					PartS:     -1,
					PartNext:  -1,
				}
				utils.EscribirEBR(archivo, ebr, inicio)
			}
			return
		}
	}
	fmt.Println("Error: no hay slots libres en el MBR (max 4 particiones)")
}

func calcularInicio(mbr types.MBR, mbrSize, tamanio int64, fit string) int64 {
	type Espacio struct {
		inicio int64
		fin    int64
		size   int64
	}

	// Recolectar particiones existentes ordenadas por inicio
	type PInfo struct {
		inicio int64
		fin    int64
	}
	var partes []PInfo
	for i := 0; i < 4; i++ {
		p := mbr.MbrPartitions[i]
		if p.PartStart > 0 {
			partes = append(partes, PInfo{p.PartStart, p.PartStart + p.PartS})
		}
	}

	// Ordenar por inicio (bubble sort simple)
	for i := 0; i < len(partes); i++ {
		for j := i + 1; j < len(partes); j++ {
			if partes[j].inicio < partes[i].inicio {
				partes[i], partes[j] = partes[j], partes[i]
			}
		}
	}

	// Calcular espacios libres
	var espacios []Espacio
	cursor := mbrSize
	for _, p := range partes {
		if p.inicio > cursor {
			espacios = append(espacios, Espacio{cursor, p.inicio, p.inicio - cursor})
		}
		cursor = p.fin
	}
	// Espacio al final
	if mbr.MbrTamanio > cursor {
		espacios = append(espacios, Espacio{cursor, mbr.MbrTamanio, mbr.MbrTamanio - cursor})
	}

	// Filtrar espacios con tamanio suficiente
	var validos []Espacio
	for _, e := range espacios {
		if e.size >= tamanio {
			validos = append(validos, e)
		}
	}
	if len(validos) == 0 {
		return -1
	}

	switch fit {
	case "B": // Best Fit - espacio mas pequeno que alcance
		mejor := validos[0]
		for _, e := range validos[1:] {
			if e.size < mejor.size {
				mejor = e
			}
		}
		return mejor.inicio
	case "W": // Worst Fit - espacio mas grande
		peor := validos[0]
		for _, e := range validos[1:] {
			if e.size > peor.size {
				peor = e
			}
		}
		return peor.inicio
	default: // First Fit
		return validos[0].inicio
	}
}

func crearLogica(archivo *os.File, mbr types.MBR, tamanioBytes int64, fit, name string) {
	// Encontrar particion extendida
	extStart := int64(-1)
	extEnd := int64(-1)
	for i := 0; i < 4; i++ {
		if mbr.MbrPartitions[i].PartType == 'E' {
			extStart = mbr.MbrPartitions[i].PartStart
			extEnd = extStart + mbr.MbrPartitions[i].PartS
			break
		}
	}
	if extStart == -1 {
		fmt.Println("Error: no existe particion extendida para crear logica")
		return
	}

	// Recorrer EBRs para encontrar el ultimo y el espacio disponible
	ebrSize := int64(unsafe.Sizeof(types.EBR{}))
	currentOffset := extStart
	var lastEBR types.EBR
	var lastOffset int64 = -1

	for {
		ebr := utils.ObtenerEBR(archivo, currentOffset)
		if ebr.PartS == -1 {
			// Primer EBR vacio
			lastOffset = currentOffset
			lastEBR = ebr
			break
		}
		lastOffset = currentOffset
		lastEBR = ebr
		if ebr.PartNext == -1 {
			break
		}
		currentOffset = ebr.PartNext
	}

	// Calcular inicio de nueva particion logica
	var nuevoInicio int64
	if lastEBR.PartS == -1 {
		// Primer logica dentro de extendida
		nuevoInicio = extStart + ebrSize
	} else {
		nuevoInicio = lastEBR.PartStart + lastEBR.PartS + ebrSize
	}

	if nuevoInicio+tamanioBytes > extEnd {
		fmt.Println("Error: no hay espacio en la particion extendida")
		return
	}

	// Crear nuevo EBR
	nuevoEBROffset := nuevoInicio - ebrSize
	if lastEBR.PartS == -1 {
		nuevoEBROffset = extStart
	}

	nuevoEBR := types.EBR{
		PartMount: '0',
		PartFit:   fit[0],
		PartStart: nuevoInicio,
		PartS:     tamanioBytes,
		PartNext:  -1,
	}
	copy(nuevoEBR.PartName[:], name)

	if lastEBR.PartS != -1 {
		// Apuntar el EBR anterior al nuevo
		lastEBR.PartNext = nuevoEBROffset
		utils.EscribirEBR(archivo, lastEBR, lastOffset)
	}

	utils.EscribirEBR(archivo, nuevoEBR, nuevoEBROffset)
	fmt.Printf("Particion logica creada: inicio=%d tamanio=%d nombre=%s\n", nuevoInicio, tamanioBytes, name)
}

func DeletePartition(params map[string]string) {
	path, ok1 := params["path"]
	name, ok2 := params["name"]
	if !ok1 || !ok2 {
		fmt.Println("Error: FDISK -delete requiere -path y -name")
		return
	}
	path = strings.ReplaceAll(path, "\"", "")
	name = strings.ReplaceAll(name, "\"", "")

	archivo, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer archivo.Close()

	mbr := utils.ObtenerMBR(archivo)
	for i := 0; i < 4; i++ {
		n := utils.BytesToString(mbr.MbrPartitions[i].PartName[:])
		if n == name {
			mbr.MbrPartitions[i] = types.Partition{PartS: -1, PartStart: -1}
			utils.EscribirMBR(archivo, mbr)
			fmt.Println("Particion eliminada:", name)
			return
		}
	}
	fmt.Println("Error: particion no encontrada")
}
