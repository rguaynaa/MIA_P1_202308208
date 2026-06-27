package filesystem

import (
	"fmt"
	"mia/types"
	"mia/utils"
	"os"
	"strings"
	"unsafe"
)

const (
	NumDirectos    = 12
	IdxIndSimple   = 12
	IdxIndDoble    = 13
	IdxIndTriple   = 14
	PuntPorBloque  = 16
)

// ---------- users.txt ----------

func ReadUsersFile(mp *types.MountedPartition) string {
	archivo, err := os.OpenFile(mp.Path, os.O_RDWR, 0644)
	if err != nil {
		return ""
	}
	defer archivo.Close()

	partStart := getPartStart(archivo, mp)
	if partStart == -1 {
		return ""
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))

	// El inodo 1 es siempre users.txt
	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+1*inodoSize)
	return readFileContent(archivo, sb, inodo)
}

func WriteUsersFile(mp *types.MountedPartition, content string) {
	archivo, err := os.OpenFile(mp.Path, os.O_RDWR, 0644)
	if err != nil {
		return
	}
	defer archivo.Close()

	partStart := getPartStart(archivo, mp)
	if partStart == -1 {
		return
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	inodoOffset := sb.SInodeStart + 1*inodoSize
	inodo := utils.ObtenerInodo(archivo, inodoOffset)

	sb = writeContentToInodo(archivo, sb, partStart, &inodo, []byte(content))
	copy(inodo.IMtime[:], utils.FechaActual())
	utils.EscribirInodo(archivo, inodo, inodoOffset)
	utils.EscribirSuperBloque(archivo, sb, partStart)
}

// ---------- helpers generales ----------

func getPartStart(archivo *os.File, mp *types.MountedPartition) int64 {
	mbr := utils.ObtenerMBR(archivo)
	for i := 0; i < 4; i++ {
		n := utils.BytesToString(mbr.MbrPartitions[i].PartName[:])
		if n == mp.Name {
			return mbr.MbrPartitions[i].PartStart
		}
	}
	return -1
}

func splitPath(path string) []string {
	parts := strings.Split(path, "/")
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func allocBlock(archivo *os.File, sb *types.SuperBloque) int32 {
	for i := int32(0); i < sb.SBlocksCount; i++ {
		bm := make([]byte, 1)
		archivo.Seek(sb.SBmBlockStart+int64(i), 0)
		archivo.Read(bm)
		if bm[0] == '0' || bm[0] == 0 {
			utils.EscribirByte(archivo, sb.SBmBlockStart+int64(i), '1')
			sb.SFreeBlocksCount--
			return i
		}
	}
	return -1
}

func freeBlock(archivo *os.File, sb *types.SuperBloque, blk int32) {
	if blk < 0 {
		return
	}
	utils.EscribirByte(archivo, sb.SBmBlockStart+int64(blk), '0')
	sb.SFreeBlocksCount++
}

func allocInode(archivo *os.File, sb *types.SuperBloque) int32 {
	for i := int32(0); i < sb.SInodesCount; i++ {
		bm := make([]byte, 1)
		archivo.Seek(sb.SBmInodeStart+int64(i), 0)
		archivo.Read(bm)
		if bm[0] == '0' || bm[0] == 0 {
			utils.EscribirByte(archivo, sb.SBmInodeStart+int64(i), '1')
			sb.SFreeInodesCount--
			return i
		}
	}
	return -1
}

func freeInode(archivo *os.File, sb *types.SuperBloque, ino int32) {
	if ino < 0 {
		return
	}
	utils.EscribirByte(archivo, sb.SBmInodeStart+int64(ino), '0')
	sb.SFreeInodesCount++
}

// ---------- lectura de contenido (directos + indirecto simple/doble/triple) ----------

func readFileContent(archivo *os.File, sb types.SuperBloque, inodo types.Inodo) string {
	blockSize := int64(sb.SBlockS)
	var sb_ strings.Builder
	remaining := int(inodo.IS)

	leerBloque := func(blk int32) {
		if remaining <= 0 || blk == -1 {
			return
		}
		fb := utils.ObtenerFileBlock(archivo, sb.SBlockStart+int64(blk)*blockSize)
		chunk := string(fb.BContent[:])
		if remaining < len(chunk) {
			chunk = chunk[:remaining]
		}
		sb_.WriteString(chunk)
		remaining -= len(chunk)
	}

	// Directos
	for i := 0; i < NumDirectos && remaining > 0; i++ {
		if inodo.IBlock[i] == -1 {
			return sb_.String()
		}
		leerBloque(inodo.IBlock[i])
	}

	// Indirecto simple
	if remaining > 0 && inodo.IBlock[IdxIndSimple] != -1 {
		pb := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[IdxIndSimple])*blockSize)
		for _, ptr := range pb.BPointers {
			if ptr == -1 || remaining <= 0 {
				break
			}
			leerBloque(ptr)
		}
	}

	// Indirecto doble
	if remaining > 0 && inodo.IBlock[IdxIndDoble] != -1 {
		pb1 := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[IdxIndDoble])*blockSize)
		for _, ptr1 := range pb1.BPointers {
			if ptr1 == -1 || remaining <= 0 {
				break
			}
			pb2 := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(ptr1)*blockSize)
			for _, ptr2 := range pb2.BPointers {
				if ptr2 == -1 || remaining <= 0 {
					break
				}
				leerBloque(ptr2)
			}
		}
	}

	// Indirecto triple
	if remaining > 0 && inodo.IBlock[IdxIndTriple] != -1 {
		pb1 := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[IdxIndTriple])*blockSize)
		for _, ptr1 := range pb1.BPointers {
			if ptr1 == -1 || remaining <= 0 {
				break
			}
			pb2 := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(ptr1)*blockSize)
			for _, ptr2 := range pb2.BPointers {
				if ptr2 == -1 || remaining <= 0 {
					break
				}
				pb3 := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(ptr2)*blockSize)
				for _, ptr3 := range pb3.BPointers {
					if ptr3 == -1 || remaining <= 0 {
						break
					}
					leerBloque(ptr3)
				}
			}
		}
	}

	return sb_.String()
}

// ---------- escritura de contenido (crea/reusa/libera bloques segun crezca o disminuya) ----------

// writeContentToInodo escribe 'data' en el inodo, manejando directos + indirectos
// 1/2/3, liberando bloques sobrantes si el archivo disminuye y reusando si el
// tamanio se mantiene igual. Devuelve el superbloque actualizado (contadores).
func writeContentToInodo(archivo *os.File, sb types.SuperBloque, partStart int64, inodo *types.Inodo, data []byte) types.SuperBloque {
	blockSize := int64(sb.SBlockS)
	numBloquesNuevo := 0
	if len(data) > 0 {
		numBloquesNuevo = (len(data) + int(blockSize) - 1) / int(blockSize)
	}

	// Recolectar bloques de datos actualmente usados (en orden) y sus punteros indirectos
	bloquesActuales := obtenerListaBloques(archivo, sb, *inodo)
	numBloquesActual := len(bloquesActuales)

	// Asegurar bloques de datos suficientes: reusar los existentes, alocar los que falten
	var bloquesFinales []int32
	for i := 0; i < numBloquesNuevo; i++ {
		if i < numBloquesActual {
			bloquesFinales = append(bloquesFinales, bloquesActuales[i])
		} else {
			nb := allocBlock(archivo, &sb)
			if nb == -1 {
				break
			}
			bloquesFinales = append(bloquesFinales, nb)
		}
	}

	// Liberar bloques de datos sobrantes (archivo disminuyo)
	for i := numBloquesNuevo; i < numBloquesActual; i++ {
		freeBlock(archivo, &sb, bloquesActuales[i])
	}

	// Escribir contenido en los bloques finales
	for i, blk := range bloquesFinales {
		offset := i * int(blockSize)
		end := offset + int(blockSize)
		if end > len(data) {
			end = len(data)
		}
		chunk := make([]byte, blockSize)
		copy(chunk, data[offset:end])
		fb := types.FileBlock{}
		copy(fb.BContent[:], chunk)
		utils.EscribirFileBlock(archivo, fb, sb.SBlockStart+int64(blk)*blockSize)
	}

	// Reconstruir apuntadores (directos + indirectos), liberando punteros que ya no se usan
	asignarApuntadores(archivo, &sb, inodo, bloquesFinales)

	inodo.IS = int32(len(data))
	return sb
}

// obtenerListaBloques devuelve, en orden, todos los bloques de datos (no punteros)
// actualmente asignados al inodo, recorriendo directos + indirecto 1/2/3.
func obtenerListaBloques(archivo *os.File, sb types.SuperBloque, inodo types.Inodo) []int32 {
	var lista []int32
	blockSize := int64(sb.SBlockS)

	for i := 0; i < NumDirectos; i++ {
		if inodo.IBlock[i] == -1 {
			return lista
		}
		lista = append(lista, inodo.IBlock[i])
	}

	if inodo.IBlock[IdxIndSimple] != -1 {
		pb := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[IdxIndSimple])*blockSize)
		for _, p := range pb.BPointers {
			if p == -1 {
				return lista
			}
			lista = append(lista, p)
		}
	} else {
		return lista
	}

	if inodo.IBlock[IdxIndDoble] != -1 {
		pb1 := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[IdxIndDoble])*blockSize)
		for _, p1 := range pb1.BPointers {
			if p1 == -1 {
				return lista
			}
			pb2 := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(p1)*blockSize)
			for _, p2 := range pb2.BPointers {
				if p2 == -1 {
					return lista
				}
				lista = append(lista, p2)
			}
		}
	} else {
		return lista
	}

	if inodo.IBlock[IdxIndTriple] != -1 {
		pb1 := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[IdxIndTriple])*blockSize)
		for _, p1 := range pb1.BPointers {
			if p1 == -1 {
				return lista
			}
			pb2 := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(p1)*blockSize)
			for _, p2 := range pb2.BPointers {
				if p2 == -1 {
					return lista
				}
				pb3 := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(p2)*blockSize)
				for _, p3 := range pb3.BPointers {
					if p3 == -1 {
						return lista
					}
					lista = append(lista, p3)
				}
			}
		}
	}

	return lista
}

// asignarApuntadores asigna bloquesFinales (datos) a los slots directos/indirectos
// del inodo, alocando bloques de punteros segun se necesiten y liberando los que
// queden sin uso.
func asignarApuntadores(archivo *os.File, sb *types.SuperBloque, inodo *types.Inodo, bloques []int32) {
	blockSize := int64(sb.SBlockS)
	idx := 0
	total := len(bloques)

	// Directos
	for i := 0; i < NumDirectos; i++ {
		if idx < total {
			inodo.IBlock[i] = bloques[idx]
			idx++
		} else {
			inodo.IBlock[i] = -1
		}
	}

	// Indirecto simple
	if idx < total {
		if inodo.IBlock[IdxIndSimple] == -1 {
			nb := allocBlock(archivo, sb)
			inodo.IBlock[IdxIndSimple] = nb
		}
		pb := types.PointerBlock{}
		for k := 0; k < PuntPorBloque; k++ {
			if idx < total {
				pb.BPointers[k] = bloques[idx]
				idx++
			} else {
				pb.BPointers[k] = -1
			}
		}
		utils.EscribirPointerBlock(archivo, pb, sb.SBlockStart+int64(inodo.IBlock[IdxIndSimple])*blockSize)
	} else if inodo.IBlock[IdxIndSimple] != -1 {
		freeBlock(archivo, sb, inodo.IBlock[IdxIndSimple])
		inodo.IBlock[IdxIndSimple] = -1
	}

	// Indirecto doble
	if idx < total {
		if inodo.IBlock[IdxIndDoble] == -1 {
			inodo.IBlock[IdxIndDoble] = allocBlock(archivo, sb)
		}
		pb1 := types.PointerBlock{}
		for k1 := 0; k1 < PuntPorBloque; k1++ {
			if idx >= total {
				pb1.BPointers[k1] = -1
				continue
			}
			pb2 := types.PointerBlock{}
			nb2 := allocBlock(archivo, sb)
			for k2 := 0; k2 < PuntPorBloque; k2++ {
				if idx < total {
					pb2.BPointers[k2] = bloques[idx]
					idx++
				} else {
					pb2.BPointers[k2] = -1
				}
			}
			utils.EscribirPointerBlock(archivo, pb2, sb.SBlockStart+int64(nb2)*blockSize)
			pb1.BPointers[k1] = nb2
		}
		utils.EscribirPointerBlock(archivo, pb1, sb.SBlockStart+int64(inodo.IBlock[IdxIndDoble])*blockSize)
	} else if inodo.IBlock[IdxIndDoble] != -1 {
		freeBlock(archivo, sb, inodo.IBlock[IdxIndDoble])
		inodo.IBlock[IdxIndDoble] = -1
	}

	// Indirecto triple
	if idx < total {
		if inodo.IBlock[IdxIndTriple] == -1 {
			inodo.IBlock[IdxIndTriple] = allocBlock(archivo, sb)
		}
		pb1 := types.PointerBlock{}
		for k1 := 0; k1 < PuntPorBloque; k1++ {
			if idx >= total {
				pb1.BPointers[k1] = -1
				continue
			}
			nb2 := allocBlock(archivo, sb)
			pb2 := types.PointerBlock{}
			for k2 := 0; k2 < PuntPorBloque; k2++ {
				if idx >= total {
					pb2.BPointers[k2] = -1
					continue
				}
				nb3 := allocBlock(archivo, sb)
				pb3 := types.PointerBlock{}
				for k3 := 0; k3 < PuntPorBloque; k3++ {
					if idx < total {
						pb3.BPointers[k3] = bloques[idx]
						idx++
					} else {
						pb3.BPointers[k3] = -1
					}
				}
				utils.EscribirPointerBlock(archivo, pb3, sb.SBlockStart+int64(nb3)*blockSize)
				pb2.BPointers[k2] = nb3
			}
			utils.EscribirPointerBlock(archivo, pb2, sb.SBlockStart+int64(nb2)*blockSize)
			pb1.BPointers[k1] = nb2
		}
		utils.EscribirPointerBlock(archivo, pb1, sb.SBlockStart+int64(inodo.IBlock[IdxIndTriple])*blockSize)
	} else if inodo.IBlock[IdxIndTriple] != -1 {
		freeBlock(archivo, sb, inodo.IBlock[IdxIndTriple])
		inodo.IBlock[IdxIndTriple] = -1
	}
}

// ---------- directorios ----------

func findInDir(archivo *os.File, sb types.SuperBloque, dirInodo int32, name string) int32 {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)
	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(dirInodo)*inodoSize)
	for i := 0; i < NumDirectos; i++ {
		if inodo.IBlock[i] == -1 {
			break
		}
		fb := utils.ObtenerFolderBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[i])*blockSize)
		for j := 0; j < 4; j++ {
			n := utils.BytesToString(fb.BContent[j].BName[:])
			if n == name && fb.BContent[j].BInodo != -1 {
				return fb.BContent[j].BInodo
			}
		}
	}
	return -1
}

// resolvePath navega desde la raiz (inodo 0) siguiendo 'parts'. Si crearFaltantes
// es true, crea las carpetas intermedias que no existan (equivalente a -p en MKDIR
// o -r en MKFILE). Devuelve el inodo final o -1 si fallo.
func resolvePath(archivo *os.File, sb *types.SuperBloque, partStart int64, parts []string, crearFaltantes bool, uid, gid int32) int32 {
	current := int32(0)
	for _, part := range parts {
		next := findInDir(archivo, *sb, current, part)
		if next == -1 {
			if !crearFaltantes {
				return -1
			}
			nuevo := createDir(archivo, sb, partStart, current, part, uid, gid)
			if nuevo == -1 {
				return -1
			}
			current = nuevo
		} else {
			current = next
		}
	}
	return current
}

func createDir(archivo *os.File, sb *types.SuperBloque, partStart int64, parentInodo int32, name string, uid, gid int32) int32 {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)

	newInodoNum := allocInode(archivo, sb)
	newBlockNum := allocBlock(archivo, sb)
	if newInodoNum == -1 || newBlockNum == -1 {
		return -1
	}

	fb := types.FolderBlock{}
	copy(fb.BContent[0].BName[:], ".")
	fb.BContent[0].BInodo = newInodoNum
	copy(fb.BContent[1].BName[:], "..")
	fb.BContent[1].BInodo = parentInodo
	fb.BContent[2].BInodo = -1
	fb.BContent[3].BInodo = -1
	utils.EscribirFolderBlock(archivo, fb, sb.SBlockStart+int64(newBlockNum)*blockSize)

	inodo := types.Inodo{
		IUid:  uid,
		IGid:  gid,
		IS:    int32(blockSize),
		IType: '0',
	}
	copy(inodo.IAtime[:], utils.FechaActual())
	copy(inodo.ICtime[:], utils.FechaActual())
	copy(inodo.IMtime[:], utils.FechaActual())
	copy(inodo.IPerm[:], "664")
	for i := range inodo.IBlock {
		inodo.IBlock[i] = -1
	}
	inodo.IBlock[0] = newBlockNum
	utils.EscribirInodo(archivo, inodo, sb.SInodeStart+int64(newInodoNum)*inodoSize)

	addEntryToDir(archivo, sb, partStart, parentInodo, name, newInodoNum)
	utils.EscribirSuperBloque(archivo, *sb, partStart)
	return newInodoNum
}

// addEntryToDir agrega una entrada (name -> childInodo) al directorio dirInodo,
// alocando un nuevo bloque de carpeta si los existentes ya estan llenos.
func addEntryToDir(archivo *os.File, sb *types.SuperBloque, partStart int64, dirInodo int32, name string, childInodo int32) {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)

	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(dirInodo)*inodoSize)

	// Buscar slot libre en bloques de carpeta ya asignados
	for i := 0; i < NumDirectos; i++ {
		if inodo.IBlock[i] == -1 {
			break
		}
		fb := utils.ObtenerFolderBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[i])*blockSize)
		for j := 0; j < 4; j++ {
			if fb.BContent[j].BInodo == -1 {
				copy(fb.BContent[j].BName[:], name)
				fb.BContent[j].BInodo = childInodo
				utils.EscribirFolderBlock(archivo, fb, sb.SBlockStart+int64(inodo.IBlock[i])*blockSize)
				return
			}
		}
	}

	// No hay slot libre: alocar un nuevo bloque de carpeta en el siguiente directo libre
	for i := 0; i < NumDirectos; i++ {
		if inodo.IBlock[i] == -1 {
			newBlock := allocBlock(archivo, sb)
			if newBlock == -1 {
				return
			}
			fb := types.FolderBlock{}
			copy(fb.BContent[0].BName[:], name)
			fb.BContent[0].BInodo = childInodo
			for k := 1; k < 4; k++ {
				fb.BContent[k].BInodo = -1
			}
			utils.EscribirFolderBlock(archivo, fb, sb.SBlockStart+int64(newBlock)*blockSize)
			inodo.IBlock[i] = newBlock
			utils.EscribirInodo(archivo, inodo, sb.SInodeStart+int64(dirInodo)*inodoSize)
			return
		}
	}
	// Carpeta con sus 12 bloques directos llenos: no se soportan indirectos para carpetas
	fmt.Println("Error: directorio sin espacio para mas entradas (limite de bloques directos alcanzado)")
}

func removeEntryFromDir(archivo *os.File, sb types.SuperBloque, dirInodo int32, name string) bool {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)
	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(dirInodo)*inodoSize)

	for i := 0; i < NumDirectos; i++ {
		if inodo.IBlock[i] == -1 {
			break
		}
		fb := utils.ObtenerFolderBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[i])*blockSize)
		for j := 0; j < 4; j++ {
			n := utils.BytesToString(fb.BContent[j].BName[:])
			if n == name && fb.BContent[j].BInodo != -1 {
				fb.BContent[j].BInodo = -1
				for k := range fb.BContent[j].BName {
					fb.BContent[j].BName[k] = 0
				}
				utils.EscribirFolderBlock(archivo, fb, sb.SBlockStart+int64(inodo.IBlock[i])*blockSize)
				return true
			}
		}
	}
	return false
}

// ---------- API publica ----------

// MkDir crea una carpeta en la ruta dada. createParents equivale a -p.
func MkDir(mp *types.MountedPartition, path string, createParents bool, uid, gid int32) {
	archivo, err := os.OpenFile(mp.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir disco")
		return
	}
	defer archivo.Close()

	partStart := getPartStart(archivo, mp)
	if partStart == -1 {
		return
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)

	parts := splitPath(path)
	if len(parts) == 0 {
		return
	}

	// Verificar que los padres existan si no se permite -p
	if !createParents && len(parts) > 1 {
		current := int32(0)
		for _, part := range parts[:len(parts)-1] {
			next := findInDir(archivo, sb, current, part)
			if next == -1 {
				fmt.Println("Error: directorio padre no existe (use -p para crearlo):", part)
				return
			}
			current = next
		}
	}

	final := resolvePath(archivo, &sb, partStart, parts, true, uid, gid)
	if final == -1 {
		fmt.Println("Error: no se pudo crear el directorio:", path)
		return
	}
	utils.EscribirSuperBloque(archivo, sb, partStart)
	fmt.Println("Directorio creado:", path)
}

// MkFile crea o sobrescribe un archivo en la ruta dada.
// r=true crea carpetas padre faltantes (equivalente a -r).
func MkFile(mp *types.MountedPartition, path string, size int, cont string, r bool, uid, gid int32) {
	archivo, err := os.OpenFile(mp.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir disco")
		return
	}
	defer archivo.Close()

	partStart := getPartStart(archivo, mp)
	if partStart == -1 {
		return
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))

	parts := splitPath(path)
	if len(parts) == 0 {
		return
	}
	dirParts := parts[:len(parts)-1]
	fileName := parts[len(parts)-1]

	currentInodo := resolvePath(archivo, &sb, partStart, dirParts, r, uid, gid)
	if currentInodo == -1 {
		fmt.Println("Error: directorio padre no existe (use -r para crearlo):", path)
		return
	}

	// Generar contenido
	var content string
	if cont != "" {
		data, err2 := os.ReadFile(cont)
		if err2 == nil {
			content = string(data)
		} else {
			fmt.Println("Error al leer archivo de contenido:", err2)
			return
		}
	} else if size > 0 {
		digits := "0123456789"
		var b strings.Builder
		for i := 0; i < size; i++ {
			b.WriteByte(digits[i%10])
		}
		content = b.String()
	}

	// Si el archivo ya existe: sobrescribir (crece, disminuye o queda igual)
	existente := findInDir(archivo, sb, currentInodo, fileName)
	if existente != -1 {
		inodoOffset := sb.SInodeStart + int64(existente)*inodoSize
		inodo := utils.ObtenerInodo(archivo, inodoOffset)
		if inodo.IType != '1' {
			fmt.Println("Error: la ruta existe y no es un archivo:", path)
			return
		}
		if !utils.TienePermiso(utils.BytesToString(inodo.IPerm[:]), inodo.IUid, inodo.IGid, uid, gid, uid == 1, 'w') {
			fmt.Println("Error: permiso denegado para escribir:", path)
			return
		}
		sb = writeContentToInodo(archivo, sb, partStart, &inodo, []byte(content))
		copy(inodo.IMtime[:], utils.FechaActual())
		utils.EscribirInodo(archivo, inodo, inodoOffset)
		utils.EscribirSuperBloque(archivo, sb, partStart)
		fmt.Println("Archivo actualizado:", path)
		return
	}

	// Crear archivo nuevo
	newInodoNum := allocInode(archivo, &sb)
	if newInodoNum == -1 {
		fmt.Println("Error: no hay inodos libres")
		return
	}

	newInodo := types.Inodo{
		IUid:  uid,
		IGid:  gid,
		IType: '1',
	}
	copy(newInodo.IAtime[:], utils.FechaActual())
	copy(newInodo.ICtime[:], utils.FechaActual())
	copy(newInodo.IMtime[:], utils.FechaActual())
	copy(newInodo.IPerm[:], "664")
	for i := range newInodo.IBlock {
		newInodo.IBlock[i] = -1
	}

	sb = writeContentToInodo(archivo, sb, partStart, &newInodo, []byte(content))
	utils.EscribirInodo(archivo, newInodo, sb.SInodeStart+int64(newInodoNum)*inodoSize)
	addEntryToDir(archivo, &sb, partStart, currentInodo, fileName, newInodoNum)
	utils.EscribirSuperBloque(archivo, sb, partStart)

	fmt.Println("Archivo creado:", path)
}

// RmFile elimina un archivo o carpeta vacia, liberando todos sus bloques e inodo.
func RmFile(mp *types.MountedPartition, path string, uid, gid int32, isRoot bool) {
	archivo, err := os.OpenFile(mp.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir disco")
		return
	}
	defer archivo.Close()

	partStart := getPartStart(archivo, mp)
	if partStart == -1 {
		return
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))

	parts := splitPath(path)
	if len(parts) == 0 {
		fmt.Println("Error: no se puede eliminar la raiz")
		return
	}
	dirParts := parts[:len(parts)-1]
	targetName := parts[len(parts)-1]

	parentInodo := resolvePath(archivo, &sb, partStart, dirParts, false, uid, gid)
	if parentInodo == -1 {
		fmt.Println("Error: ruta no encontrada:", path)
		return
	}

	targetInodoNum := findInDir(archivo, sb, parentInodo, targetName)
	if targetInodoNum == -1 {
		fmt.Println("Error: archivo o carpeta no encontrada:", path)
		return
	}

	inodoOffset := sb.SInodeStart + int64(targetInodoNum)*inodoSize
	inodo := utils.ObtenerInodo(archivo, inodoOffset)

	if !utils.TienePermiso(utils.BytesToString(inodo.IPerm[:]), inodo.IUid, inodo.IGid, uid, gid, isRoot, 'w') {
		fmt.Println("Error: permiso denegado para eliminar:", path)
		return
	}

	if inodo.IType == '0' {
		// Verificar que la carpeta este vacia (solo . y ..)
		entries := lsInodo(archivo, sb, targetInodoNum)
		if len(entries) > 0 {
			fmt.Println("Error: el directorio no esta vacio:", path)
			return
		}
		// Liberar bloques de carpeta
		for i := 0; i < NumDirectos; i++ {
			if inodo.IBlock[i] == -1 {
				break
			}
			freeBlock(archivo, &sb, inodo.IBlock[i])
		}
	} else {
		// Liberar todos los bloques de datos + punteros del archivo
		bloques := obtenerListaBloques(archivo, sb, inodo)
		for _, b := range bloques {
			freeBlock(archivo, &sb, b)
		}
		for _, idxPtr := range []int{IdxIndSimple, IdxIndDoble, IdxIndTriple} {
			if inodo.IBlock[idxPtr] != -1 {
				liberarArbolPunteros(archivo, &sb, inodo.IBlock[idxPtr], idxPtr-IdxIndSimple+1)
			}
		}
	}

	freeInode(archivo, &sb, targetInodoNum)
	removeEntryFromDir(archivo, sb, parentInodo, targetName)
	utils.EscribirSuperBloque(archivo, sb, partStart)
	fmt.Println("Eliminado:", path)
}

// liberarArbolPunteros libera recursivamente los bloques de punteros (no los de
// datos, que ya se liberaron via obtenerListaBloques) de un arbol indirecto.
func liberarArbolPunteros(archivo *os.File, sb *types.SuperBloque, blk int32, nivel int) {
	if blk == -1 {
		return
	}
	blockSize := int64(sb.SBlockS)
	if nivel > 1 {
		pb := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(blk)*blockSize)
		for _, p := range pb.BPointers {
			if p != -1 {
				liberarArbolPunteros(archivo, sb, p, nivel-1)
			}
		}
	}
	freeBlock(archivo, sb, blk)
}

// GetFileContent retorna el contenido de un archivo por ruta, validando permiso de lectura.
func GetFileContent(mp *types.MountedPartition, path string, uid, gid int32, isRoot bool) string {
	archivo, err := os.OpenFile(mp.Path, os.O_RDONLY, 0644)
	if err != nil {
		return ""
	}
	defer archivo.Close()

	partStart := getPartStart(archivo, mp)
	if partStart == -1 {
		return ""
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))

	parts := splitPath(path)
	currentInodo := resolvePath(archivo, &sb, partStart, parts, false, uid, gid)
	if currentInodo == -1 {
		return ""
	}

	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(currentInodo)*inodoSize)
	if inodo.IType != '1' {
		return ""
	}
	if !utils.TienePermiso(utils.BytesToString(inodo.IPerm[:]), inodo.IUid, inodo.IGid, uid, gid, isRoot, 'r') {
		fmt.Println("Error: permiso denegado para leer:", path)
		return ""
	}
	return readFileContent(archivo, sb, inodo)
}

// LsEntry representa una entrada de directorio para el reporte ls
type LsEntry struct {
	Name  string
	Type  byte
	Perm  string
	Uid   int32
	Gid   int32
	Mtime string
	Size  int32
}

func lsInodo(archivo *os.File, sb types.SuperBloque, dirInodoNum int32) []LsEntry {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)
	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(dirInodoNum)*inodoSize)
	if inodo.IType != '0' {
		return nil
	}

	var entries []LsEntry
	for i := 0; i < NumDirectos; i++ {
		if inodo.IBlock[i] == -1 {
			break
		}
		fb := utils.ObtenerFolderBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[i])*blockSize)
		for j := 0; j < 4; j++ {
			n := utils.BytesToString(fb.BContent[j].BName[:])
			if n == "" || n == "." || n == ".." || fb.BContent[j].BInodo == -1 {
				continue
			}
			childInodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(fb.BContent[j].BInodo)*inodoSize)
			entries = append(entries, LsEntry{
				Name:  n,
				Type:  childInodo.IType,
				Perm:  utils.BytesToString(childInodo.IPerm[:]),
				Uid:   childInodo.IUid,
				Gid:   childInodo.IGid,
				Mtime: utils.BytesToString(childInodo.IMtime[:]),
				Size:  childInodo.IS,
			})
		}
	}
	return entries
}

// LsDir retorna lista de entradas en un directorio (uso publico para reportes)
func LsDir(mp *types.MountedPartition, path string) []LsEntry {
	archivo, err := os.OpenFile(mp.Path, os.O_RDONLY, 0644)
	if err != nil {
		return nil
	}
	defer archivo.Close()

	partStart := getPartStart(archivo, mp)
	if partStart == -1 {
		return nil
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)

	parts := splitPath(path)
	currentInodo := resolvePath(archivo, &sb, partStart, parts, false, 1, 1)
	if currentInodo == -1 {
		return nil
	}
	return lsInodo(archivo, sb, currentInodo)
}
