package utils

import (
	"bytes"
	"encoding/binary"
	"mia/types"
	"os"
	"strings"
	"time"
	"unsafe"
)

func Tamanio(tamanio int64, unit string) int64 {
	u := strings.ToLower(unit)
	switch u {
	case "k":
		return tamanio * 1024
	case "m":
		return tamanio * 1024 * 1024
	case "b":
		return tamanio
	}
	return -1
}

func ObtenerMBR(archivo *os.File) types.MBR {
	mbr := types.MBR{}
	content := make([]byte, int(unsafe.Sizeof(mbr)))
	archivo.Seek(0, 0)
	archivo.Read(content)
	buffer := bytes.NewBuffer(content)
	binary.Read(buffer, binary.BigEndian, &mbr)
	return mbr
}

func EscribirMBR(archivo *os.File, mbr types.MBR) {
	archivo.Seek(0, 0)
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, &mbr)
	archivo.Write(buffer.Bytes())
}

func ObtenerEBR(archivo *os.File, offset int64) types.EBR {
	ebr := types.EBR{}
	content := make([]byte, int(unsafe.Sizeof(ebr)))
	archivo.Seek(offset, 0)
	archivo.Read(content)
	buffer := bytes.NewBuffer(content)
	binary.Read(buffer, binary.BigEndian, &ebr)
	return ebr
}

func EscribirEBR(archivo *os.File, ebr types.EBR, offset int64) {
	archivo.Seek(offset, 0)
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, &ebr)
	archivo.Write(buffer.Bytes())
}

func ObtenerSuperBloque(archivo *os.File, offset int64) types.SuperBloque {
	sb := types.SuperBloque{}
	content := make([]byte, int(unsafe.Sizeof(sb)))
	archivo.Seek(offset, 0)
	archivo.Read(content)
	buffer := bytes.NewBuffer(content)
	binary.Read(buffer, binary.BigEndian, &sb)
	return sb
}

func EscribirSuperBloque(archivo *os.File, sb types.SuperBloque, offset int64) {
	archivo.Seek(offset, 0)
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, &sb)
	archivo.Write(buffer.Bytes())
}

func ObtenerInodo(archivo *os.File, offset int64) types.Inodo {
	inodo := types.Inodo{}
	content := make([]byte, int(unsafe.Sizeof(inodo)))
	archivo.Seek(offset, 0)
	archivo.Read(content)
	buffer := bytes.NewBuffer(content)
	binary.Read(buffer, binary.BigEndian, &inodo)
	return inodo
}

func EscribirInodo(archivo *os.File, inodo types.Inodo, offset int64) {
	archivo.Seek(offset, 0)
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, &inodo)
	archivo.Write(buffer.Bytes())
}

func ObtenerFolderBlock(archivo *os.File, offset int64) types.FolderBlock {
	fb := types.FolderBlock{}
	content := make([]byte, int(unsafe.Sizeof(fb)))
	archivo.Seek(offset, 0)
	archivo.Read(content)
	buffer := bytes.NewBuffer(content)
	binary.Read(buffer, binary.BigEndian, &fb)
	return fb
}

func EscribirFolderBlock(archivo *os.File, fb types.FolderBlock, offset int64) {
	archivo.Seek(offset, 0)
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, &fb)
	archivo.Write(buffer.Bytes())
}

func ObtenerFileBlock(archivo *os.File, offset int64) types.FileBlock {
	fb := types.FileBlock{}
	content := make([]byte, int(unsafe.Sizeof(fb)))
	archivo.Seek(offset, 0)
	archivo.Read(content)
	buffer := bytes.NewBuffer(content)
	binary.Read(buffer, binary.BigEndian, &fb)
	return fb
}

func EscribirFileBlock(archivo *os.File, fb types.FileBlock, offset int64) {
	archivo.Seek(offset, 0)
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, &fb)
	archivo.Write(buffer.Bytes())
}

func ObtenerPointerBlock(archivo *os.File, offset int64) types.PointerBlock {
	pb := types.PointerBlock{}
	content := make([]byte, int(unsafe.Sizeof(pb)))
	archivo.Seek(offset, 0)
	archivo.Read(content)
	buffer := bytes.NewBuffer(content)
	binary.Read(buffer, binary.BigEndian, &pb)
	return pb
}

func EscribirPointerBlock(archivo *os.File, pb types.PointerBlock, offset int64) {
	archivo.Seek(offset, 0)
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, &pb)
	archivo.Write(buffer.Bytes())
}

func FechaActual() string {
	t := time.Now()
	return t.Format("02/01/2006 15:04:05")
}

func BytesToString(b []byte) string {
	result := ""
	for _, c := range b {
		if c == 0 {
			break
		}
		result += string(c)
	}
	return result
}

func LeerBitmap(archivo *os.File, offset int64, cantidad int32) []byte {
	bitmap := make([]byte, cantidad)
	archivo.Seek(offset, 0)
	archivo.Read(bitmap)
	return bitmap
}

func EscribirByte(archivo *os.File, offset int64, valor byte) {
	archivo.Seek(offset, 0)
	archivo.Write([]byte{valor})
}

// TienePermiso verifica si un usuario (uid, gid, esRoot) puede realizar
// la accion deseada ('r'=leer, 'w'=escribir, 'x'=ejecutar) sobre un inodo
// dado su propietario (ownerUid, ownerGid) y sus permisos octales UGO (perm).
func TienePermiso(perm string, ownerUid, ownerGid, uid, gid int32, isRoot bool, accion byte) bool {
	if isRoot {
		return true
	}
	if len(perm) != 3 {
		return false
	}

	var grupo string
	switch {
	case uid == ownerUid:
		grupo = string(perm[0]) // Usuario (propietario)
	case gid == ownerGid:
		grupo = string(perm[1]) // Grupo
	default:
		grupo = string(perm[2]) // Otros
	}

	val := digitoOctal(grupo)

	switch accion {
	case 'r':
		return val == 4 || val == 5 || val == 6 || val == 7
	case 'w':
		return val == 2 || val == 3 || val == 6 || val == 7
	case 'x':
		return val == 1 || val == 3 || val == 5 || val == 7
	}
	return false
}

// digitoOctal convierte un caracter '0'-'7' a su valor numerico entero.
func digitoOctal(s string) int {
	if len(s) == 0 {
		return 0
	}
	c := s[0]
	if c >= '0' && c <= '7' {
		return int(c - '0')
	}
	return 0
}

func SizeOf(v interface{}) int64 {
	switch v.(type) {
	case types.MBR:
		return int64(unsafe.Sizeof(types.MBR{}))
	case types.Partition:
		return int64(unsafe.Sizeof(types.Partition{}))
	case types.EBR:
		return int64(unsafe.Sizeof(types.EBR{}))
	case types.SuperBloque:
		return int64(unsafe.Sizeof(types.SuperBloque{}))
	case types.Inodo:
		return int64(unsafe.Sizeof(types.Inodo{}))
	case types.FolderBlock:
		return int64(unsafe.Sizeof(types.FolderBlock{}))
	case types.FileBlock:
		return int64(unsafe.Sizeof(types.FileBlock{}))
	case types.PointerBlock:
		return int64(unsafe.Sizeof(types.PointerBlock{}))
	}
	return 0
}
