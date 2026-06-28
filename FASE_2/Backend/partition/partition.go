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
	if _, ok := params["delete"]; ok {
		DeletePartition(params)
		return
	}
	if _, ok := params["add"]; ok {
		AddPartitionSpace(params)
		return
	}

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

	modo := strings.ToLower(strings.ReplaceAll(params["delete"], "\"", ""))
	if modo != "fast" && modo != "full" {
		fmt.Println("Error: -delete debe ser 'fast' o 'full'")
		return
	}

	archivo, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer archivo.Close()

	mbr := utils.ObtenerMBR(archivo)

	// Buscar primero en particiones primarias/extendidas del MBR
	for i := 0; i < 4; i++ {
		n := utils.BytesToString(mbr.MbrPartitions[i].PartName[:])
		if n != name {
			continue
		}
		partStart := mbr.MbrPartitions[i].PartStart
		partSize := mbr.MbrPartitions[i].PartS
		esExtendida := mbr.MbrPartitions[i].PartType == 'E'

		// Si es extendida, eliminar primero todas las logicas que contenga
		if esExtendida {
			eliminarTodasLasLogicas(archivo, partStart, modo)
		}

		if modo == "full" {
			rellenarConCeros(archivo, partStart, partSize)
		}

		mbr.MbrPartitions[i] = types.Partition{PartS: -1, PartStart: -1}
		utils.EscribirMBR(archivo, mbr)
		fmt.Println("Particion eliminada:", name)
		return
	}


	// No esta en el MBR: buscar en la cadena de EBRs de la extendida
	for i := 0; i < 4; i++ {
		if mbr.MbrPartitions[i].PartType != 'E' {
			continue
		}
		extStart := mbr.MbrPartitions[i].PartStart
		if deleteLogica(archivo, extStart, name) {
			fmt.Println("Particion logica eliminada:", name)
			return
		}
	}

	fmt.Println("Error: particion no encontrada")
}

// rellenarConCeros escribe el caracter \0 en todo el espacio fisico de la
// particion, segun lo exige -delete=full.
func rellenarConCeros(archivo *os.File, inicio, tamanio int64) {
	if tamanio <= 0 {
		return
	}
	const bufSize = 4096
	buffer := make([]byte, bufSize)
	archivo.Seek(inicio, 0)
	restante := tamanio
	for restante > 0 {
		n := int64(bufSize)
		if restante < n {
			n = restante
		}
		archivo.Write(buffer[:n])
		restante -= n
	}
}

// eliminarTodasLasLogicas recorre la cadena de EBR de una extendida y
// limpia (fast o full) cada particion logica encontrada, antes de que la
// extendida misma sea eliminada.
func eliminarTodasLasLogicas(archivo *os.File, extStart int64, modo string) {
	currentOffset := extStart
	for {
		ebr := utils.ObtenerEBR(archivo, currentOffset)
		if ebr.PartS == -1 {
			break
		}
		if modo == "full" {
			rellenarConCeros(archivo, ebr.PartStart, ebr.PartS)
		}
		next := ebr.PartNext
		if next == -1 {
			break
		}
		currentOffset = next
	}
}

// AddPartitionSpace implementa fdisk -add: agrega o quita espacio a una
// particion existente. Valor positivo agrega (verificando que el espacio
// libre inmediatamente despues de la particion alcance), valor negativo
// quita (verificando que la particion no quede en 0 o negativo).
func AddPartitionSpace(params map[string]string) {
	addStr, ok0 := params["add"]
	path, ok1 := params["path"]
	name, ok2 := params["name"]
	if !ok0 || !ok1 || !ok2 {
		fmt.Println("Error: FDISK -add requiere -add, -path y -name")
		return
	}
	path = strings.ReplaceAll(path, "\"", "")
	name = strings.ReplaceAll(name, "\"", "")

	var addVal int64
	fmt.Sscanf(addStr, "%d", &addVal)
	if addVal == 0 {
		fmt.Println("Error: -add no puede ser 0")
		return
	}

	unit := "k"
	if u, ok := params["unit"]; ok {
		unit = strings.ToLower(u)
	}
	addBytes := utils.Tamanio(absInt64(addVal), unit)
	if addBytes <= 0 {
		fmt.Println("Error: unidad invalida")
		return
	}
	if addVal < 0 {
		addBytes = -addBytes
	}

	archivo, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer archivo.Close()

	mbr := utils.ObtenerMBR(archivo)
	mbrSize := int64(unsafe.Sizeof(mbr))

	for i := 0; i < 4; i++ {
		n := utils.BytesToString(mbr.MbrPartitions[i].PartName[:])
		if n != name {
			continue
		}
		p := &mbr.MbrPartitions[i]
		nuevoTamanio := p.PartS + addBytes
		if nuevoTamanio <= 0 {
			fmt.Println("Error: no se puede dejar la particion con tamanio negativo o cero")
			return
		}

		if addBytes > 0 {
			// Verificar espacio libre disponible inmediatamente despues
			finActual := p.PartStart + p.PartS
			limite := siguienteInicioOFinDisco(mbr, mbrSize, i, finActual)
			if finActual+addBytes > limite {
				fmt.Println("Error: no hay espacio libre suficiente despues de la particion para agregar")
				return
			}
		} else {
			// addBytes negativo: solo se reduce el tamanio logico
			// (el espacio liberado queda disponible para nuevas particiones).
		}

		p.PartS = nuevoTamanio
		utils.EscribirMBR(archivo, mbr)
		fmt.Printf("Particion %s actualizada, nuevo tamanio=%d bytes\n", name, nuevoTamanio)
		return
	}

	fmt.Println("Error: particion no encontrada (las logicas no soportan -add)")
}

// siguienteInicioOFinDisco calcula el limite superior disponible para que
// una particion (indice idx) pueda extenderse: el inicio de la particion
// mas cercana que empiece despues de 'desde', o el final del disco si no
// hay ninguna.
func siguienteInicioOFinDisco(mbr types.MBR, mbrSize int64, idx int, desde int64) int64 {
	limite := mbr.MbrTamanio
	for j := 0; j < 4; j++ {
		if j == idx {
			continue
		}
		p := mbr.MbrPartitions[j]
		if p.PartStart >= desde && p.PartStart < limite {
			limite = p.PartStart
		}
	}
	return limite
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// deleteLogica recorre la cadena de EBRs desde extStart buscando 'name'.
// Si la encuentra, la desenlaza de la cadena (el EBR anterior salta al
// PartNext de la eliminada) y la marca como libre.
func deleteLogica(archivo *os.File, extStart int64, name string) bool {
	currentOffset := extStart
	var prevOffset int64 = -1

	for {
		ebr := utils.ObtenerEBR(archivo, currentOffset)
		if ebr.PartS == -1 {
			return false
		}
		ename := utils.BytesToString(ebr.PartName[:])
		if ename == name {
			if prevOffset == -1 {
				// Es el primer EBR de la cadena: lo dejamos vacio pero
				// conservamos el enlace al siguiente, copiando sus datos
				// hacia esta posicion (la cadena no puede empezar "vacia"
				// en medio si hay mas logicas detras).
				next := ebr.PartNext
				if next != -1 {
					siguienteEbr := utils.ObtenerEBR(archivo, next)
					utils.EscribirEBR(archivo, siguienteEbr, currentOffset)
					// Liberar el bloque del EBR que se movio
					vacio := types.EBR{PartS: -1, PartStart: -1, PartNext: -1}
					utils.EscribirEBR(archivo, vacio, next)
				} else {
					vacio := types.EBR{PartS: -1, PartStart: -1, PartNext: -1}
					utils.EscribirEBR(archivo, vacio, currentOffset)
				}
			} else {
				prevEbr := utils.ObtenerEBR(archivo, prevOffset)
				prevEbr.PartNext = ebr.PartNext
				utils.EscribirEBR(archivo, prevEbr, prevOffset)
				vacio := types.EBR{PartS: -1, PartStart: -1, PartNext: -1}
				utils.EscribirEBR(archivo, vacio, currentOffset)
			}
			return true
		}
		if ebr.PartNext == -1 {
			return false
		}
		prevOffset = currentOffset
		currentOffset = ebr.PartNext
	}
}
