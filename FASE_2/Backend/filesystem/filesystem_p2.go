package filesystem

import (
	"fmt"
	"mia/types"
	"mia/utils"
	"os"
	"unsafe"
)

// RemoveRecursive elimina un archivo o carpeta y todo su contenido. Para carpetas, elimina recursivamente solo los archivos/subcarpetas en los que el usuario tenga permiso de escritura
func RemoveRecursive(mp *types.MountedPartition, path string, uid, gid int32, isRoot bool) {
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

	ok := removeInodoRecursivo(archivo, &sb, targetInodoNum, uid, gid, isRoot)
	if !ok {
		fmt.Println("Error: no se pudo eliminar por completo (permisos insuficientes en algun elemento):", path)
		return
	}

	freeInode(archivo, &sb, targetInodoNum)
	removeEntryFromDir(archivo, sb, parentInodo, targetName)
	utils.EscribirSuperBloque(archivo, sb, partStart)
	fmt.Println("Eliminado:", path)
}

// removeInodoRecursivo libera el inodo dado y, si es carpeta, primero intenta eliminar recursivamente cada hijo. Devuelve false sin borrar nada
// si algun hijo no pudo eliminarse por falta de permiso de escritura, dejando intacto el arbol completo (no se borra nada "a medias")
func removeInodoRecursivo(archivo *os.File, sb *types.SuperBloque, inodoNum int32, uid, gid int32, isRoot bool) bool {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	inodoOffset := sb.SInodeStart + int64(inodoNum)*inodoSize
	inodo := utils.ObtenerInodo(archivo, inodoOffset)

	if !utils.TienePermiso(utils.BytesToString(inodo.IPerm[:]), inodo.IUid, inodo.IGid, uid, gid, isRoot, 'w') {
		return false
	}

	if inodo.IType == '0' {
		// Recolectar hijos primero (sin modificar nada todavia)
		type hijo struct {
			nombre string
			inodo  int32
		}
		var hijos []hijo
		for i := 0; i < NumDirectos; i++ {
			if inodo.IBlock[i] == -1 {
				break
			}
			fb := utils.ObtenerFolderBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[i])*int64(sb.SBlockS))
			for j := 0; j < 4; j++ {
				n := utils.BytesToString(fb.BContent[j].BName[:])
				if n == "" || n == "." || n == ".." || fb.BContent[j].BInodo == -1 {
					continue
				}
				hijos = append(hijos, hijo{n, fb.BContent[j].BInodo})
			}
		}

		// Verificar que TODOS los hijos se puedan eliminar antes de borrar nada
		for _, h := range hijos {
			if !puedeEliminarseRecursivo(archivo, sb, h.inodo, uid, gid, isRoot) {
				return false
			}
		}

		// Ahora si, eliminar cada hijo de verdad
		for _, h := range hijos {
			removeInodoRecursivo(archivo, sb, h.inodo, uid, gid, isRoot)
			freeInode(archivo, sb, h.inodo)
			removeEntryFromDir(archivo, *sb, inodoNum, h.nombre)
		}

		// Liberar bloques de carpeta propios
		inodoActual := utils.ObtenerInodo(archivo, inodoOffset)
		for i := 0; i < NumDirectos; i++ {
			if inodoActual.IBlock[i] == -1 {
				break
			}
			freeBlock(archivo, sb, inodoActual.IBlock[i])
		}
	} else {
		bloques := obtenerListaBloques(archivo, *sb, inodo)
		for _, b := range bloques {
			freeBlock(archivo, sb, b)
		}
		for _, idxPtr := range []int{IdxIndSimple, IdxIndDoble, IdxIndTriple} {
			if inodo.IBlock[idxPtr] != -1 {
				liberarArbolPunteros(archivo, sb, inodo.IBlock[idxPtr], idxPtr-IdxIndSimple+1)
			}
		}
	}
	return true
}

// puedeEliminarseRecursivo verifica (sin modificar nada) si un inodo y todo su contenido se podrian eliminar exitosamente, validando permisos en cada nivel de la recursion
func puedeEliminarseRecursivo(archivo *os.File, sb *types.SuperBloque, inodoNum int32, uid, gid int32, isRoot bool) bool {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(inodoNum)*inodoSize)
	if !utils.TienePermiso(utils.BytesToString(inodo.IPerm[:]), inodo.IUid, inodo.IGid, uid, gid, isRoot, 'w') {
		return false
	}
	if inodo.IType != '0' {
		return true
	}
	for i := 0; i < NumDirectos; i++ {
		if inodo.IBlock[i] == -1 {
			break
		}
		fb := utils.ObtenerFolderBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[i])*int64(sb.SBlockS))
		for j := 0; j < 4; j++ {
			n := utils.BytesToString(fb.BContent[j].BName[:])
			if n == "" || n == "." || n == ".." || fb.BContent[j].BInodo == -1 {
				continue
			}
			if !puedeEliminarseRecursivo(archivo, sb, fb.BContent[j].BInodo, uid, gid, isRoot) {
				return false
			}
		}
	}
	return true
}


// EditFile reemplaza el contenido de un archivo existente con el contenido leido desde un archivo del sistema operativo anfitrion (-contenido). Requiere permiso de lectura y escritura sobre el archivo destino.
func EditFile(mp *types.MountedPartition, path string, contenidoHost string, uid, gid int32, isRoot bool) {
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
	currentFit = getPartFit(archivo, mp)
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))

	parts := splitPath(path)
	if len(parts) == 0 {
		fmt.Println("Error: ruta invalida")
		return
	}
	dirParts := parts[:len(parts)-1]
	fileName := parts[len(parts)-1]

	parentInodo := resolvePath(archivo, &sb, partStart, dirParts, false, uid, gid)
	if parentInodo == -1 {
		fmt.Println("Error: ruta no encontrada:", path)
		return
	}
	targetInodoNum := findInDir(archivo, sb, parentInodo, fileName)
	if targetInodoNum == -1 {
		fmt.Println("Error: archivo no encontrado:", path)
		return
	}

	inodoOffset := sb.SInodeStart + int64(targetInodoNum)*inodoSize
	inodo := utils.ObtenerInodo(archivo, inodoOffset)
	if inodo.IType != '1' {
		fmt.Println("Error: la ruta no es un archivo:", path)
		return
	}
	permStr := utils.BytesToString(inodo.IPerm[:])
	if !utils.TienePermiso(permStr, inodo.IUid, inodo.IGid, uid, gid, isRoot, 'r') ||
		!utils.TienePermiso(permStr, inodo.IUid, inodo.IGid, uid, gid, isRoot, 'w') {
		fmt.Println("Error: permiso de lectura y escritura requerido para editar:", path)
		return
	}

	data, errRead := os.ReadFile(contenidoHost)
	if errRead != nil {
		fmt.Println("Error al leer archivo de contenido en el sistema operativo:", errRead)
		return
	}

	sb = writeContentToInodo(archivo, sb, partStart, &inodo, data)
	copy(inodo.IMtime[:], utils.FechaActual())
	utils.EscribirInodo(archivo, inodo, inodoOffset)
	utils.EscribirSuperBloque(archivo, sb, partStart)
	fmt.Println("Archivo editado:", path)
}

// RenameFile cambia el nombre de un archivo o carpeta, validando permiso de escritura sobre el elemento y que no exista ya otro con el nuevo nombre dentro del mismo directorio padre.
func RenameFile(mp *types.MountedPartition, path string, newName string, uid, gid int32, isRoot bool) {
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
		fmt.Println("Error: no se puede renombrar la raiz")
		return
	}
	dirParts := parts[:len(parts)-1]
	oldName := parts[len(parts)-1]

	parentInodo := resolvePath(archivo, &sb, partStart, dirParts, false, uid, gid)
	if parentInodo == -1 {
		fmt.Println("Error: ruta no encontrada:", path)
		return
	}

	targetInodoNum := findInDir(archivo, sb, parentInodo, oldName)
	if targetInodoNum == -1 {
		fmt.Println("Error: archivo o carpeta no encontrada:", path)
		return
	}

	if findInDir(archivo, sb, parentInodo, newName) != -1 {
		fmt.Println("Error: ya existe un archivo o carpeta con el nombre:", newName)
		return
	}

	inodoOffset := sb.SInodeStart + int64(targetInodoNum)*inodoSize
	inodo := utils.ObtenerInodo(archivo, inodoOffset)
	if !utils.TienePermiso(utils.BytesToString(inodo.IPerm[:]), inodo.IUid, inodo.IGid, uid, gid, isRoot, 'w') {
		fmt.Println("Error: permiso de escritura requerido para renombrar:", path)
		return
	}

	if !renombrarEntradaEnDir(archivo, sb, parentInodo, oldName, newName) {
		fmt.Println("Error: no se pudo renombrar:", path)
		return
	}
	fmt.Println("Renombrado:", oldName, "->", newName)
}

// renombrarEntradaEnDir busca la entrada 'oldName' dentro del directorio y reemplaza unicamente su nombre por 'newName', preservando el inodo al que apunta
func renombrarEntradaEnDir(archivo *os.File, sb types.SuperBloque, dirInodo int32, oldName, newName string) bool {
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
			if n == oldName && fb.BContent[j].BInodo != -1 {
				for k := range fb.BContent[j].BName {
					fb.BContent[j].BName[k] = 0
				}
				copy(fb.BContent[j].BName[:], newName)
				utils.EscribirFolderBlock(archivo, fb, sb.SBlockStart+int64(inodo.IBlock[i])*blockSize)
				return true
			}
		}
	}
	return false
}


// CopyFile copia un archivo o carpeta (con todo su contenido recursivamente)hacia otro destino. Solo copia los elementos a los que el usuario tenga permiso de lectura, si un hijo no tiene permiso, se omite (no se copia solo ese elemento, el resto continua
func CopyFile(mp *types.MountedPartition, srcPath, dstPath string, uid, gid int32, isRoot bool) {
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
	currentFit = getPartFit(archivo, mp)
	sb := utils.ObtenerSuperBloque(archivo, partStart)

	srcParts := splitPath(srcPath)
	if len(srcParts) == 0 {
		fmt.Println("Error: no se puede copiar la raiz")
		return
	}
	srcDirParts := srcParts[:len(srcParts)-1]
	srcName := srcParts[len(srcParts)-1]

	srcParentInodo := resolvePath(archivo, &sb, partStart, srcDirParts, false, uid, gid)
	if srcParentInodo == -1 {
		fmt.Println("Error: ruta origen no encontrada:", srcPath)
		return
	}
	srcInodoNum := findInDir(archivo, sb, srcParentInodo, srcName)
	if srcInodoNum == -1 {
		fmt.Println("Error: archivo o carpeta origen no encontrada:", srcPath)
		return
	}

	dstParts := splitPath(dstPath)
	dstParentInodo := resolvePath(archivo, &sb, partStart, dstParts, false, uid, gid)
	if dstParentInodo == -1 {
		fmt.Println("Error: carpeta destino no encontrada:", dstPath)
		return
	}
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	dstParentInodoData := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(dstParentInodo)*inodoSize)
	if !utils.TienePermiso(utils.BytesToString(dstParentInodoData.IPerm[:]), dstParentInodoData.IUid, dstParentInodoData.IGid, uid, gid, isRoot, 'w') {
		fmt.Println("Error: permiso de escritura requerido sobre la carpeta destino:", dstPath)
		return
	}

	nuevo := copiarInodoRecursivo(archivo, &sb, partStart, srcInodoNum, dstParentInodo, uid, gid, isRoot)
	if nuevo == -1 {
		fmt.Println("Error: no se tiene permiso de lectura sobre:", srcPath)
		return
	}
	addEntryToDir(archivo, &sb, partStart, dstParentInodo, srcName, nuevo)
	utils.EscribirSuperBloque(archivo, sb, partStart)
	fmt.Println("Copiado:", srcPath, "->", dstPath)
}

// copiarInodoRecursivo crea una copia completa (nuevo inodo + nuevos bloques) 
func copiarInodoRecursivo(archivo *os.File, sb *types.SuperBloque, partStart int64, srcInodoNum int32, padreInodo int32, uid, gid int32, isRoot bool) int32 {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	srcInodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(srcInodoNum)*inodoSize)

	if !utils.TienePermiso(utils.BytesToString(srcInodo.IPerm[:]), srcInodo.IUid, srcInodo.IGid, uid, gid, isRoot, 'r') {
		return -1
	}

	newInodoNum := allocInode(archivo, sb)
	if newInodoNum == -1 {
		return -1
	}

	if srcInodo.IType == '1' {
		// Archivo: copiar contenido tal cual
		content := readFileContent(archivo, *sb, srcInodo)
		newInodo := types.Inodo{
			IUid:  srcInodo.IUid,
			IGid:  srcInodo.IGid,
			IType: '1',
		}
		copy(newInodo.IAtime[:], utils.FechaActual())
		copy(newInodo.ICtime[:], utils.FechaActual())
		copy(newInodo.IMtime[:], utils.FechaActual())
		newInodo.IPerm = srcInodo.IPerm
		for i := range newInodo.IBlock {
			newInodo.IBlock[i] = -1
		}
		*sb = writeContentToInodo(archivo, *sb, partStart, &newInodo, []byte(content))
		utils.EscribirInodo(archivo, newInodo, sb.SInodeStart+int64(newInodoNum)*inodoSize)
		return newInodoNum
	}

	// Carpeta: crear nueva carpeta vacia y copiar hijos recursivamente
	newBlockNum := allocBlock(archivo, sb)
	if newBlockNum == -1 {
		freeInode(archivo, sb, newInodoNum)
		return -1
	}
	fb := types.FolderBlock{}
	copy(fb.BContent[0].BName[:], ".")
	fb.BContent[0].BInodo = newInodoNum
	copy(fb.BContent[1].BName[:], "..")
	fb.BContent[1].BInodo = padreInodo
	fb.BContent[2].BInodo = -1
	fb.BContent[3].BInodo = -1
	utils.EscribirFolderBlock(archivo, fb, sb.SBlockStart+int64(newBlockNum)*int64(sb.SBlockS))

	newInodo := types.Inodo{
		IUid:  srcInodo.IUid,
		IGid:  srcInodo.IGid,
		IS:    int32(sb.SBlockS),
		IType: '0',
	}
	copy(newInodo.IAtime[:], utils.FechaActual())
	copy(newInodo.ICtime[:], utils.FechaActual())
	copy(newInodo.IMtime[:], utils.FechaActual())
	newInodo.IPerm = srcInodo.IPerm
	for i := range newInodo.IBlock {
		newInodo.IBlock[i] = -1
	}
	newInodo.IBlock[0] = newBlockNum
	utils.EscribirInodo(archivo, newInodo, sb.SInodeStart+int64(newInodoNum)*inodoSize)
	utils.EscribirSuperBloque(archivo, *sb, partStart)

	// Copiar hijos (omitiendo los que no tengan permiso de lectura)
	for i := 0; i < NumDirectos; i++ {
		if srcInodo.IBlock[i] == -1 {
			break
		}
		childFb := utils.ObtenerFolderBlock(archivo, sb.SBlockStart+int64(srcInodo.IBlock[i])*int64(sb.SBlockS))
		for j := 0; j < 4; j++ {
			n := utils.BytesToString(childFb.BContent[j].BName[:])
			if n == "" || n == "." || n == ".." || childFb.BContent[j].BInodo == -1 {
				continue
			}
			nuevoHijo := copiarInodoRecursivo(archivo, sb, partStart, childFb.BContent[j].BInodo, newInodoNum, uid, gid, isRoot)
			if nuevoHijo == -1 {
				// Sin permiso de lectura sobre este hijo: se omite, no se copia
				continue
			}
			addEntryToDir(archivo, sb, partStart, newInodoNum, n, nuevoHijo)
		}
	}

	return newInodoNum
}

// MoveFile mueve un archivo o carpeta hacia otro destino. Si origen y destino estan en la misma particion (siempre es el caso aqui, ya que se opera sobre una sola particion montada), solo se actualizan las
// referencias en los bloques de carpeta (no se mueve contenido fisico). Solo se valida permiso de escritura sobre el elemento origen y sobre la carpeta destino
func MoveFile(mp *types.MountedPartition, srcPath, dstPath string, uid, gid int32, isRoot bool) {
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

	srcParts := splitPath(srcPath)
	if len(srcParts) == 0 {
		fmt.Println("Error: no se puede mover la raiz")
		return
	}
	srcDirParts := srcParts[:len(srcParts)-1]
	srcName := srcParts[len(srcParts)-1]

	srcParentInodo := resolvePath(archivo, &sb, partStart, srcDirParts, false, uid, gid)
	if srcParentInodo == -1 {
		fmt.Println("Error: ruta origen no encontrada:", srcPath)
		return
	}
	srcInodoNum := findInDir(archivo, sb, srcParentInodo, srcName)
	if srcInodoNum == -1 {
		fmt.Println("Error: archivo o carpeta origen no encontrada:", srcPath)
		return
	}

	srcInodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(srcInodoNum)*inodoSize)
	if !utils.TienePermiso(utils.BytesToString(srcInodo.IPerm[:]), srcInodo.IUid, srcInodo.IGid, uid, gid, isRoot, 'w') {
		fmt.Println("Error: permiso de escritura requerido sobre el origen:", srcPath)
		return
	}

	dstParts := splitPath(dstPath)
	dstParentInodo := resolvePath(archivo, &sb, partStart, dstParts, false, uid, gid)
	if dstParentInodo == -1 {
		fmt.Println("Error: carpeta destino no encontrada:", dstPath)
		return
	}
	dstParentData := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(dstParentInodo)*inodoSize)
	if !utils.TienePermiso(utils.BytesToString(dstParentData.IPerm[:]), dstParentData.IUid, dstParentData.IGid, uid, gid, isRoot, 'w') {
		fmt.Println("Error: permiso de escritura requerido sobre la carpeta destino:", dstPath)
		return
	}

	if findInDir(archivo, sb, dstParentInodo, srcName) != -1 {
		fmt.Println("Error: ya existe un elemento con ese nombre en el destino:", srcName)
		return
	}

	// Quitar referencia del padre origen y agregar en el padre destino
	if !removeEntryFromDir(archivo, sb, srcParentInodo, srcName) {
		fmt.Println("Error: no se pudo desvincular del directorio origen:", srcPath)
		return
	}
	addEntryToDir(archivo, &sb, partStart, dstParentInodo, srcName, srcInodoNum)

	// Si es carpeta, actualizar su entrada ".." para que apunte al nuevo padre
	if srcInodo.IType == '0' {
		actualizarPuntoPadre(archivo, sb, srcInodoNum, dstParentInodo)
	}

	utils.EscribirSuperBloque(archivo, sb, partStart)
	fmt.Println("Movido:", srcPath, "->", dstPath)
}

// actualizarPuntoPadre busca la entrada ".." dentro del primer bloque de
// carpeta del inodo dado y actualiza el inodo al que apunta, para que siga
// siendo coherente despues de un MOVE.
func actualizarPuntoPadre(archivo *os.File, sb types.SuperBloque, dirInodoNum int32, nuevoPadre int32) {
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)
	inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(dirInodoNum)*inodoSize)
	if inodo.IBlock[0] == -1 {
		return
	}
	fb := utils.ObtenerFolderBlock(archivo, sb.SBlockStart+int64(inodo.IBlock[0])*blockSize)
	for j := 0; j < 4; j++ {
		n := utils.BytesToString(fb.BContent[j].BName[:])
		if n == ".." {
			fb.BContent[j].BInodo = nuevoPadre
			utils.EscribirFolderBlock(archivo, fb, sb.SBlockStart+int64(inodo.IBlock[0])*blockSize)
			return
		}
	}
}

