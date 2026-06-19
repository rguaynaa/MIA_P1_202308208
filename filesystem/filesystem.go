package filesystem

import (
	"fmt"
	"mia/types"
	"mia/utils"
	"os"
	"strconv"
	"strings"
	"unsafe"
)

// ReadUsersFile lee el contenido de /users.txt en la particion montada
func ReadUsersFile(mp *types.MountedPartition) string {
	archivo, err := os.OpenFile(mp.Path, os.O_RDWR, 0644)
	if err != nil {
		return ""
	}
	defer archivo.Close()

	inodoNum, blockNum := findUsersFileInodes(archivo, mp)
	if inodoNum == -1 {
		return ""
	}
	_ = inodoNum

	mbr := utils.ObtenerMBR(archivo)
	_ = mbr

	// Leer el superbloque de la particion
	partStart := getPartStart(archivo, mp)
	if partStart == -1 {
		return ""
	}

	sb := utils.ObtenerSuperBloque(archivo, partStart)
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))

	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(blockNum)*inodoSize)
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
	blockSize := int64(sb.SBlockS)

	// Inodo 1 es siempre users.txt
	inodoOffset := sb.SInodeStart + inodoSize
	inodo := utils.ObtenerInodo(archivo, inodoOffset)

	// Escribir contenido en bloques
	data := []byte(content)
	inodo.IS = int32(len(data))

	blockIdx := 0
	for offset := 0; offset < len(data); offset += int(blockSize) {
		end := offset + int(blockSize)
		if end > len(data) {
			end = len(data)
		}
		chunk := make([]byte, blockSize)
		copy(chunk, data[offset:end])

		fb := types.FileBlock{}
		copy(fb.BContent[:], chunk)

		var blkNum int32
		if blockIdx < 12 {
			blkNum = inodo.IBlock[blockIdx]
			if blkNum == -1 {
				blkNum = allocBlock(archivo, sb, partStart)
				if blkNum == -1 {
					break
				}
				inodo.IBlock[blockIdx] = blkNum
			}
		}
		utils.EscribirFileBlock(archivo, fb, sb.SBlockStart+int64(blkNum)*blockSize)
		blockIdx++
	}

	copy(inodo.IMtime[:], utils.FechaActual())
	utils.EscribirInodo(archivo, inodo, inodoOffset)
	utils.EscribirSuperBloque(archivo, sb, partStart)
}

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

func findUsersFileInodes(archivo *os.File, mp *types.MountedPartition) (int32, int32) {
	return 1, 1
}

func readFileContent(archivo *os.File, sb types.SuperBloque, inodo types.Inodo) string {
	blockSize := int64(sb.SBlockS)
	content := ""
	remaining := int(inodo.IS)

	for i := 0; i < 12 && remaining > 0; i++ {
		if inodo.IBlock[i] == -1 {
			break
		}
		fb := utils.ObtenerFileBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[i])*blockSize)
		chunk := string(fb.BContent[:])
		if remaining < int(blockSize) {
			chunk = chunk[:remaining]
		}
		content += chunk
		remaining -= int(blockSize)
	}

	// Bloques indirectos simples
	if inodo.IBlock[12] != -1 && remaining > 0 {
		pb := utils.ObtenerPointerBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[12])*blockSize)
		for _, ptr := range pb.BPointers {
			if ptr == -1 || remaining <= 0 {
				break
			}
			fb := utils.ObtenerFileBlock(archivo, sb.SBlockStart+int64(ptr)*blockSize)
			chunk := string(fb.BContent[:])
			if remaining < int(blockSize) {
				chunk = chunk[:remaining]
			}
			content += chunk
			remaining -= int(blockSize)
		}
	}

	return content
}

func allocBlock(archivo *os.File, sb types.SuperBloque, partStart int64) int32 {
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

func allocInode(archivo *os.File, sb types.SuperBloque, partStart int64) int32 {
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

// MkDir crea una carpeta en la ruta dada
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

	currentInodo := int32(0) // empezar en raiz
	for i, part := range parts {
		if part == "" {
			continue
		}
		next := findInDir(archivo, sb, currentInodo, part)
		if next == -1 {
			if i < len(parts)-1 && !createParents {
				fmt.Println("Error: directorio padre no existe:", part)
				return
			}
			// Crear
			newInodo := createDir(archivo, sb, partStart, currentInodo, part, uid, gid)
			if newInodo == -1 {
				fmt.Println("Error: no se pudo crear directorio:", part)
				return
			}
			// Actualizar sb
			sb = utils.ObtenerSuperBloque(archivo, partStart)
			currentInodo = newInodo
		} else {
			currentInodo = next
		}
	}
	fmt.Println("Directorio creado:", path)
}

func createDir(archivo *os.File, sb types.SuperBloque, partStart int64, parentInodo int32, name string, uid, gid int32) int32 {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)

	newInodoNum := allocInode(archivo, sb, partStart)
	newBlockNum := allocBlock(archivo, sb, partStart)
	if newInodoNum == -1 || newBlockNum == -1 {
		return -1
	}

	// Recargar sb tras alloc
	sb = utils.ObtenerSuperBloque(archivo, partStart)

	// Crear folder block
	fb := types.FolderBlock{}
	copy(fb.BContent[0].BName[:], ".")
	fb.BContent[0].BInodo = newInodoNum
	copy(fb.BContent[1].BName[:], "..")
	fb.BContent[1].BInodo = parentInodo
	fb.BContent[2].BInodo = -1
	fb.BContent[3].BInodo = -1
	utils.EscribirFolderBlock(archivo, fb, sb.SBlockStart+int64(newBlockNum)*blockSize)

	// Crear inodo
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

	// Agregar entrada en el directorio padre
	addEntryToDir(archivo, sb, parentInodo, name, newInodoNum)

	utils.EscribirSuperBloque(archivo, sb, partStart)
	return newInodoNum
}

func addEntryToDir(archivo *os.File, sb types.SuperBloque, dirInodo int32, name string, childInodo int32) {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)

	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(dirInodo)*inodoSize)

	for i := 0; i < 12; i++ {
		if inodo.IBlock[i] == -1 {
			break
		}
		fb := utils.ObtenerFolderBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[i])*blockSize)
		for j := 0; j < 4; j++ {
			if fb.BContent[j].BInodo == -1 || (fb.BContent[j].BInodo == 0 && utils.BytesToString(fb.BContent[j].BName[:]) == "") {
				copy(fb.BContent[j].BName[:], name)
				fb.BContent[j].BInodo = childInodo
				utils.EscribirFolderBlock(archivo, fb, sb.SBlockStart+int64(inodo.IBlock[i])*blockSize)
				return
			}
		}
	}

	// Necesitamos nuevo bloque para el directorio padre
	partStart := sb.SBmInodeStart - int64(unsafe.Sizeof(types.SuperBloque{}))
	newBlock := allocBlock(archivo, sb, partStart)
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
	for i := 0; i < 12; i++ {
		if inodo.IBlock[i] == -1 {
			inodo.IBlock[i] = newBlock
			break
		}
	}
	utils.EscribirInodo(archivo, inodo, sb.SInodeStart+int64(dirInodo)*inodoSize)
}

func findInDir(archivo *os.File, sb types.SuperBloque, dirInodo int32, name string) int32 {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)
	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(dirInodo)*inodoSize)
	for i := 0; i < 12; i++ {
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

// MkFile crea un archivo en la ruta dada
func MkFile(mp *types.MountedPartition, path string, size int, cont string, uid, gid int32) {
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

	// Navegar hasta el directorio padre
	dirParts := parts[:len(parts)-1]
	fileName := parts[len(parts)-1]
	currentInodo := int32(0)
	for _, part := range dirParts {
		next := findInDir(archivo, sb, currentInodo, part)
		if next == -1 {
			fmt.Println("Error: directorio no existe:", part)
			return
		}
		currentInodo = next
	}

	// Generar contenido
	var content string
	if cont != "" {
		data, err2 := os.ReadFile(cont)
		if err2 == nil {
			content = string(data)
		}
	} else if size > 0 {
		digits := "0123456789"
		for i := 0; i < size; i++ {
			content += string(digits[i%10])
		}
	}

	// Crear inodo de archivo
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)

	newInodoNum := allocInode(archivo, sb, partStart)
	if newInodoNum == -1 {
		fmt.Println("Error: no hay inodos libres")
		return
	}
	sb = utils.ObtenerSuperBloque(archivo, partStart)

	newInodo := types.Inodo{
		IUid:  uid,
		IGid:  gid,
		IS:    int32(len(content)),
		IType: '1',
	}
	copy(newInodo.IAtime[:], utils.FechaActual())
	copy(newInodo.ICtime[:], utils.FechaActual())
	copy(newInodo.IMtime[:], utils.FechaActual())
	copy(newInodo.IPerm[:], "664")
	for i := range newInodo.IBlock {
		newInodo.IBlock[i] = -1
	}

	// Escribir contenido en bloques
	data := []byte(content)
	blockIdx := 0
	for offset := 0; offset < len(data); offset += int(blockSize) {
		end := offset + int(blockSize)
		if end > len(data) {
			end = len(data)
		}
		chunk := make([]byte, blockSize)
		copy(chunk, data[offset:end])
		fb := types.FileBlock{}
		copy(fb.BContent[:], chunk)

		if blockIdx < 12 {
			blkNum := allocBlock(archivo, sb, partStart)
			if blkNum == -1 {
				break
			}
			sb = utils.ObtenerSuperBloque(archivo, partStart)
			newInodo.IBlock[blockIdx] = blkNum
			utils.EscribirFileBlock(archivo, fb, sb.SBlockStart+int64(blkNum)*blockSize)
		}
		blockIdx++
	}

	utils.EscribirInodo(archivo, newInodo, sb.SInodeStart+int64(newInodoNum)*inodoSize)
	addEntryToDir(archivo, sb, currentInodo, fileName, newInodoNum)
	sb = utils.ObtenerSuperBloque(archivo, partStart)
	utils.EscribirSuperBloque(archivo, sb, partStart)

	fmt.Println("Archivo creado:", path)
}

// GetFileContent retorna el contenido de un archivo por ruta
func GetFileContent(mp *types.MountedPartition, path string) string {
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
	currentInodo := int32(0)
	for _, part := range parts {
		next := findInDir(archivo, sb, currentInodo, part)
		if next == -1 {
			return ""
		}
		currentInodo = next
	}

	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(currentInodo)*inodoSize)
	if inodo.IType != '1' {
		return ""
	}
	return readFileContent(archivo, sb, inodo)
}

// LsDir retorna lista de entradas en un directorio
type LsEntry struct {
	Name  string
	Type  byte
	Perm  string
	Uid   int32
	Gid   int32
	Mtime string
	Size  int32
}

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
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)

	parts := splitPath(path)
	currentInodo := int32(0)
	for _, part := range parts {
		next := findInDir(archivo, sb, currentInodo, part)
		if next == -1 {
			return nil
		}
		currentInodo = next
	}

	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(currentInodo)*inodoSize)
	if inodo.IType != '0' {
		return nil
	}

	var entries []LsEntry
	for i := 0; i < 12; i++ {
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

func PermisoUGO(inodo types.Inodo, uid int32, gid int32, w bool) bool {
	if uid == 1 {
		return true
	} // Root aplastar reglas
	perm := utils.BytesToString(inodo.IPerm[:])
	if len(perm) < 3 {
		return true
	}
	u, _ := strconv.Atoi(string(perm[0]))
	g, _ := strconv.Atoi(string(perm[1]))
	o, _ := strconv.Atoi(string(perm[2]))

	p := o
	if inodo.IUid == uid {
		p = u
	} else if inodo.IGid == gid {
		p = g
	}

	if w {
		return p == 2 || p == 3 || p == 6 || p == 7 // Escritura
	}
	return p == 4 || p == 5 || p == 6 || p == 7 // Lectura
}
