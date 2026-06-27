package format

import (
	"fmt"
	"mia/mount"
	"mia/types"
	"mia/utils"
	"os"
	"strings"
	"unsafe"
)

func Mkfs(params map[string]string) {
	id, ok := params["id"]
	if !ok {
		fmt.Println("Error: MKFS requiere -id")
		return
	}
	id = strings.ReplaceAll(id, "\"", "")

	mp, ok := mount.GetMountedPartition(id)
	if !ok {
		fmt.Println("Error: particion no montada:", id)
		return
	}

	archivo, err := os.OpenFile(mp.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir disco:", err)
		return
	}
	defer archivo.Close()

	mbr := utils.ObtenerMBR(archivo)

	// Buscar la particion por nombre
	var partStart, partSize int64
	found := false
	for i := 0; i < 4; i++ {
		n := utils.BytesToString(mbr.MbrPartitions[i].PartName[:])
		if n == mp.Name {
			partStart = mbr.MbrPartitions[i].PartStart
			partSize = mbr.MbrPartitions[i].PartS
			found = true
			break
		}
	}
	if !found {
		fmt.Println("Error: particion no encontrada en disco")
		return
	}

	// Calcular n (numero de inodos)
	sbSize := int64(unsafe.Sizeof(types.SuperBloque{}))
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(64)

	// tamanio = sb + n + 3n + n*inodoSize + 3n*blockSize
	// n*(1 + 3 + inodoSize + 3*blockSize) = partSize - sbSize
	n := (partSize - sbSize) / (1 + 3 + inodoSize + 3*blockSize)
	if n <= 0 {
		fmt.Println("Error: particion demasiado pequena para EXT2")
		return
	}

	// Calcular posiciones
	bmInodeStart := partStart + sbSize
	bmBlockStart := bmInodeStart + n
	inodeStart := bmBlockStart + 3*n
	blockStart := inodeStart + n*inodoSize

	// Inicializar bitmaps con 0
	zeros := make([]byte, n)
	archivo.Seek(bmInodeStart, 0)
	archivo.Write(zeros)
	blockZeros := make([]byte, 3*n)
	archivo.Seek(bmBlockStart, 0)
	archivo.Write(blockZeros)

	// Crear SuperBloque
	sb := types.SuperBloque{
		SFilesystemType:  2,
		SInodesCount:     int32(n),
		SBlocksCount:     int32(3 * n),
		SFreeBlocksCount: int32(3*n) - 3,
		SFreeInodesCount: int32(n) - 2,
		SMntCount:        1,
		SMagic:           0xEF53,
		SInodeS:          int32(inodoSize),
		SBlockS:          int32(blockSize),
		SFirstIno:        2,
		SFirstBlo:        3,
		SBmInodeStart:    bmInodeStart,
		SBmBlockStart:    bmBlockStart,
		SInodeStart:      inodeStart,
		SBlockStart:      blockStart,
	}
	copy(sb.SMtime[:], utils.FechaActual())
	copy(sb.SUmtime[:], utils.FechaActual())

	utils.EscribirSuperBloque(archivo, sb, partStart)

	// Inodo 0: carpeta raiz /
	inodo0 := types.Inodo{
		IUid:  1,
		IGid:  1,
		IS:    int32(blockSize),
		IType: '0',
	}
	copy(inodo0.IAtime[:], utils.FechaActual())
	copy(inodo0.ICtime[:], utils.FechaActual())
	copy(inodo0.IMtime[:], utils.FechaActual())
	copy(inodo0.IPerm[:], "777")
	for i := range inodo0.IBlock {
		inodo0.IBlock[i] = -1
	}
	inodo0.IBlock[0] = 0 // apunta al bloque 0

	// Inodo 1: users.txt
	inodo1 := types.Inodo{
		IUid:  1,
		IGid:  1,
		IType: '1',
	}
	copy(inodo1.IAtime[:], utils.FechaActual())
	copy(inodo1.ICtime[:], utils.FechaActual())
	copy(inodo1.IMtime[:], utils.FechaActual())
	copy(inodo1.IPerm[:], "664")
	for i := range inodo1.IBlock {
		inodo1.IBlock[i] = -1
	}
	inodo1.IBlock[0] = 2 // apunta al bloque 2

	// Escribir inodos
	utils.EscribirInodo(archivo, inodo0, inodeStart)
	utils.EscribirInodo(archivo, inodo1, inodeStart+inodoSize)

	// Bloque 0: FolderBlock para /
	fb0 := types.FolderBlock{}
	copy(fb0.BContent[0].BName[:], ".")
	fb0.BContent[0].BInodo = 0
	copy(fb0.BContent[1].BName[:], "..")
	fb0.BContent[1].BInodo = 0
	copy(fb0.BContent[2].BName[:], "users.txt")
	fb0.BContent[2].BInodo = 1
	fb0.BContent[3].BInodo = -1
	utils.EscribirFolderBlock(archivo, fb0, blockStart)

	// Bloque 1: reservado (solo para alinear, apuntado por carpeta raiz extra si se necesita)
	// Bloque 2: FileBlock para users.txt
	usersContent := "1,G,root\n1,U,root,root,123\n"
	fb2 := types.FileBlock{}
	copy(fb2.BContent[:], usersContent)
	utils.EscribirFileBlock(archivo, fb2, blockStart+2*blockSize)

	inodo1.IS = int32(len(usersContent))
	utils.EscribirInodo(archivo, inodo1, inodeStart+inodoSize)

	// Actualizar bitmaps
	utils.EscribirByte(archivo, bmInodeStart, '1')
	utils.EscribirByte(archivo, bmInodeStart+1, '1')
	utils.EscribirByte(archivo, bmBlockStart, '1')
	utils.EscribirByte(archivo, bmBlockStart+1, '1')
	utils.EscribirByte(archivo, bmBlockStart+2, '1')

	fmt.Printf("MKFS exitoso: n=%d inodos, %d bloques, particion=%s\n", n, 3*n, mp.Name)
}
