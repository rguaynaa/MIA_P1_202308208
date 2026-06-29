package report

import (
	"fmt"
	"mia/filesystem"
	"mia/mount"
	"mia/types"
	"mia/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"
)

func Rep(params map[string]string) {
	name, ok1 := params["name"]
	path, ok2 := params["path"]
	id, ok3 := params["id"]
	if !ok1 || !ok2 || !ok3 {
		fmt.Println("Error: REP requiere -name, -path, -id")
		return
	}
	name = strings.ToLower(strings.ReplaceAll(name, "\"", ""))
	path = strings.ReplaceAll(path, "\"", "")
	id = strings.ReplaceAll(id, "\"", "")
	pathFileLs := strings.ReplaceAll(params["path_file_ls"], "\"", "")

	mp, ok := mount.GetMountedPartition(id)
	if !ok {
		fmt.Println("Error: particion no montada:", id)
		return
	}

	archivo, err := os.OpenFile(mp.Path, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("Error al abrir disco")
		return
	}
	defer archivo.Close()

	switch name {
	case "mbr":
		repMBR(archivo, path, mp)
	case "disk":
		repDisk(archivo, path, mp)
	case "inode":
		repInode(archivo, path, mp)
	case "block":
		repBlock(archivo, path, mp)
	case "bm_inode":
		repBmInode(archivo, path, mp)
	case "bm_block":
		repBmBlock(archivo, path, mp)
	case "sb":
		repSB(archivo, path, mp)
	case "file":
		repFile(path, mp, pathFileLs)
	case "ls":
		repLS(path, mp, pathFileLs)
	case "tree":
		repTree(archivo, path, mp)
	default:
		fmt.Println("Error: reporte desconocido:", name)
	}
}

func prepararRuta(path string) (string, string) {
	dir := filepath.Dir(path)
	exec.Command("mkdir", "-p", dir).Output()
	nombre := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	txt := filepath.Join(dir, nombre+".dot")
	png := filepath.Join(dir, nombre+".png")
	return txt, png
}

func generarPNG(txt, png string) {
	out, err := exec.Command("dot", "-Tpng", txt, "-o", png).CombinedOutput()
	if err != nil {
		fmt.Println("Error Graphviz:", string(out), err)
	} else {
		fmt.Println("Reporte generado:", png)
	}
}

func repMBR(archivo *os.File, path string, mp *types.MountedPartition) {
	mbr := utils.ObtenerMBR(archivo)
	txt, png := prepararRuta(path)

	grafo := "digraph MBR {\n"
	grafo += "node [shape=plaintext fontsize=12];\n"
	grafo += "tabla [label=<\n"
	grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" BGCOLOR=\"#f0f0f0\">\n"
	grafo += "<TR><TD COLSPAN=\"2\" BGCOLOR=\"#336699\"><FONT COLOR=\"white\"><B>MBR</B></FONT></TD></TR>\n"
	grafo += fmt.Sprintf("<TR><TD>Tamanio</TD><TD>%d bytes</TD></TR>\n", mbr.MbrTamanio)
	grafo += fmt.Sprintf("<TR><TD>Fecha Creacion</TD><TD>%s</TD></TR>\n", utils.BytesToString(mbr.MbrFechaCreacion[:]))
	grafo += fmt.Sprintf("<TR><TD>Signature</TD><TD>%d</TD></TR>\n", mbr.MbrDskSignature)
	grafo += fmt.Sprintf("<TR><TD>Ajuste</TD><TD>%s</TD></TR>\n", string(mbr.DskFit))
	grafo += "</TABLE>\n>];\n"

	for i := 0; i < 4; i++ {
		p := mbr.MbrPartitions[i]
		if p.PartStart <= 0 && p.PartS <= 0 {
			continue
		}
		pname := utils.BytesToString(p.PartName[:])
		pid := utils.BytesToString(p.PartId[:])
		grafo += fmt.Sprintf("p%d [label=<\n", i)
		grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" BGCOLOR=\"#e8f4e8\">\n"
		grafo += fmt.Sprintf("<TR><TD COLSPAN=\"2\" BGCOLOR=\"#336633\"><FONT COLOR=\"white\"><B>Particion %d</B></FONT></TD></TR>\n", i+1)
		grafo += fmt.Sprintf("<TR><TD>Nombre</TD><TD>%s</TD></TR>\n", pname)
		grafo += fmt.Sprintf("<TR><TD>Tipo</TD><TD>%s</TD></TR>\n", string(p.PartType))
		grafo += fmt.Sprintf("<TR><TD>Ajuste</TD><TD>%s</TD></TR>\n", string(p.PartFit))
		grafo += fmt.Sprintf("<TR><TD>Inicio</TD><TD>%d</TD></TR>\n", p.PartStart)
		grafo += fmt.Sprintf("<TR><TD>Tamanio</TD><TD>%d</TD></TR>\n", p.PartS)
		grafo += fmt.Sprintf("<TR><TD>Status</TD><TD>%s</TD></TR>\n", string(p.PartStatus))
		grafo += fmt.Sprintf("<TR><TD>Correlativo</TD><TD>%d</TD></TR>\n", p.PartCorrelative)
		grafo += fmt.Sprintf("<TR><TD>ID</TD><TD>%s</TD></TR>\n", pid)
		grafo += "</TABLE>\n>];\n"
		grafo += fmt.Sprintf("tabla -> p%d;\n", i)

		// Si es extendida, mostrar EBRs
		if p.PartType == 'E' {
			ebrOffset := p.PartStart
			ebrIdx := 0
			for ebrOffset != -1 {
				ebr := utils.ObtenerEBR(archivo, ebrOffset)
				if ebr.PartS == -1 {
					break
				}
				ename := utils.BytesToString(ebr.PartName[:])
				grafo += fmt.Sprintf("ebr%d_%d [label=<\n", i, ebrIdx)
				grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" BGCOLOR=\"#fff0e0\">\n"
				grafo += fmt.Sprintf("<TR><TD COLSPAN=\"2\" BGCOLOR=\"#996633\"><FONT COLOR=\"white\"><B>EBR %d</B></FONT></TD></TR>\n", ebrIdx)
				grafo += fmt.Sprintf("<TR><TD>Nombre</TD><TD>%s</TD></TR>\n", ename)
				grafo += fmt.Sprintf("<TR><TD>Inicio</TD><TD>%d</TD></TR>\n", ebr.PartStart)
				grafo += fmt.Sprintf("<TR><TD>Tamanio</TD><TD>%d</TD></TR>\n", ebr.PartS)
				grafo += fmt.Sprintf("<TR><TD>Siguiente</TD><TD>%d</TD></TR>\n", ebr.PartNext)
				grafo += "</TABLE>\n>];\n"
				grafo += fmt.Sprintf("p%d -> ebr%d_%d;\n", i, i, ebrIdx)
				ebrIdx++
				ebrOffset = ebr.PartNext
			}
		}
	}
	grafo += "}\n"

	f, _ := os.Create(txt)
	f.WriteString(grafo)
	f.Close()
	generarPNG(txt, png)
}

func repDisk(archivo *os.File, path string, mp *types.MountedPartition) {
	mbr := utils.ObtenerMBR(archivo)
	txt, png := prepararRuta(path)

	grafo := "digraph Disk {\n"
	grafo += "graph [rankdir=LR];\n"
	grafo += "node [shape=plaintext fontsize=10];\n"
	grafo += "disco [label=<\n"
	grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n"
	grafo += "<TR>\n"
	grafo += "<TD BGCOLOR=\"#cccccc\"><B>MBR</B></TD>\n"

	mbrSize := int64(unsafe.Sizeof(mbr))
	type Seg struct {
		nombre string
		inicio int64
		size   int64
		tipo   string
		extIdx int // indice de particion extendida en mbr, -1 si no aplica
	}
	var segs []Seg
	for i := 0; i < 4; i++ {
		p := mbr.MbrPartitions[i]
		if p.PartStart > 0 && p.PartS > 0 {
			extIdx := -1
			if p.PartType == 'E' {
				extIdx = i
			}
			segs = append(segs, Seg{
				nombre: utils.BytesToString(p.PartName[:]),
				inicio: p.PartStart,
				size:   p.PartS,
				tipo:   string(p.PartType),
				extIdx: extIdx,
			})
		}
	}
	// Ordenar segs por inicio
	for i := 0; i < len(segs); i++ {
		for j := i + 1; j < len(segs); j++ {
			if segs[j].inicio < segs[i].inicio {
				segs[i], segs[j] = segs[j], segs[i]
			}
		}
	}

	cursor := mbrSize
	for _, s := range segs {
		if s.inicio > cursor {
			libre := s.inicio - cursor
			pct := float64(libre) * 100 / float64(mbr.MbrTamanio)
			grafo += fmt.Sprintf("<TD BGCOLOR=\"#ffffff\">Libre<BR/>%.2f%%</TD>\n", pct)
		}

		if s.extIdx >= 0 {
			// Particion extendida: desglosar logicas + EBRs + espacio libre interno
			grafo += desglosarExtendida(archivo, s.inicio, s.size, mbr.MbrTamanio)
		} else {
			pct := float64(s.size) * 100 / float64(mbr.MbrTamanio)
			grafo += fmt.Sprintf("<TD BGCOLOR=\"#99ccff\"><B>%s</B><BR/>%s<BR/>%.2f%%</TD>\n", s.nombre, s.tipo, pct)
		}
		cursor = s.inicio + s.size
	}
	if cursor < mbr.MbrTamanio {
		libre := mbr.MbrTamanio - cursor
		pct := float64(libre) * 100 / float64(mbr.MbrTamanio)
		grafo += fmt.Sprintf("<TD BGCOLOR=\"#ffffff\">Libre<BR/>%.2f%%</TD>\n", pct)
	}

	grafo += "</TR>\n</TABLE>\n>];\n}\n"

	f, _ := os.Create(txt)
	f.WriteString(grafo)
	f.Close()
	generarPNG(txt, png)
}

// desglosarExtendida recorre la cadena de EBRs de una particion extendida y
// genera celdas para cada EBR, cada particion logica, y el espacio libre
// entre ellas, de forma que el desglose interno tambien suma 100% relativo
// al disco completo.
func desglosarExtendida(archivo *os.File, extInicio, extTamanio, discoTamanio int64) string {
	ebrSize := int64(unsafe.Sizeof(types.EBR{}))
	grafo := ""
	cursor := extInicio
	extFin := extInicio + extTamanio
	offset := extInicio

	for offset != -1 && offset < extFin {
		ebr := utils.ObtenerEBR(archivo, offset)
		if ebr.PartS == -1 {
			break
		}
		if ebr.PartStart > cursor {
			libre := ebr.PartStart - cursor - ebrSize
			if libre > 0 {
				pct := float64(libre) * 100 / float64(discoTamanio)
				grafo += fmt.Sprintf("<TD BGCOLOR=\"#fff8e8\">Libre(L)<BR/>%.2f%%</TD>\n", pct)
			}
		}
		// Celda del EBR
		pctEbr := float64(ebrSize) * 100 / float64(discoTamanio)
		grafo += fmt.Sprintf("<TD BGCOLOR=\"#ffe0c0\">EBR<BR/>%.2f%%</TD>\n", pctEbr)

		// Celda de la particion logica
		ename := utils.BytesToString(ebr.PartName[:])
		pct := float64(ebr.PartS) * 100 / float64(discoTamanio)
		grafo += fmt.Sprintf("<TD BGCOLOR=\"#ffcc99\"><B>%s</B><BR/>L<BR/>%.2f%%</TD>\n", ename, pct)

		cursor = ebr.PartStart + ebr.PartS
		offset = ebr.PartNext
	}

	// Espacio libre al final de la extendida
	if cursor < extFin {
		libre := extFin - cursor
		pct := float64(libre) * 100 / float64(discoTamanio)
		grafo += fmt.Sprintf("<TD BGCOLOR=\"#fff8e8\">Libre(L)<BR/>%.2f%%</TD>\n", pct)
	}

	return grafo
}

func getPartStartFromFile(archivo *os.File, mp *types.MountedPartition) int64 {
	mbr := utils.ObtenerMBR(archivo)
	for i := 0; i < 4; i++ {
		n := utils.BytesToString(mbr.MbrPartitions[i].PartName[:])
		if n == mp.Name {
			return mbr.MbrPartitions[i].PartStart
		}
	}
	return -1
}

func repInode(archivo *os.File, path string, mp *types.MountedPartition) {
	partStart := getPartStartFromFile(archivo, mp)
	if partStart == -1 {
		return
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	txt, png := prepararRuta(path)

	grafo := "digraph Inodos {\n"
	grafo += "node [shape=plaintext fontsize=10];\n"
	grafo += "rankdir=LR;\n"

	for i := int32(0); i < sb.SInodesCount; i++ {
		bm := make([]byte, 1)
		archivo.Seek(sb.SBmInodeStart+int64(i), 0)
		archivo.Read(bm)
		if bm[0] != '1' {
			continue
		}
		inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(i)*inodoSize)
		tipo := "Carpeta"
		if inodo.IType == '1' {
			tipo = "Archivo"
		}
		grafo += fmt.Sprintf("i%d [label=<\n", i)
		grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" BGCOLOR=\"#e0e8ff\">\n"
		grafo += fmt.Sprintf("<TR><TD COLSPAN=\"2\" BGCOLOR=\"#334499\"><FONT COLOR=\"white\"><B>Inodo %d</B></FONT></TD></TR>\n", i)
		grafo += fmt.Sprintf("<TR><TD>Tipo</TD><TD>%s</TD></TR>\n", tipo)
		grafo += fmt.Sprintf("<TR><TD>UID</TD><TD>%d</TD></TR>\n", inodo.IUid)
		grafo += fmt.Sprintf("<TR><TD>GID</TD><TD>%d</TD></TR>\n", inodo.IGid)
		grafo += fmt.Sprintf("<TR><TD>Tamanio</TD><TD>%d</TD></TR>\n", inodo.IS)
		grafo += fmt.Sprintf("<TR><TD>Permisos</TD><TD>%s</TD></TR>\n", utils.BytesToString(inodo.IPerm[:]))
		grafo += fmt.Sprintf("<TR><TD>Atime</TD><TD>%s</TD></TR>\n", utils.BytesToString(inodo.IAtime[:]))
		grafo += fmt.Sprintf("<TR><TD>Ctime</TD><TD>%s</TD></TR>\n", utils.BytesToString(inodo.ICtime[:]))
		grafo += fmt.Sprintf("<TR><TD>Mtime</TD><TD>%s</TD></TR>\n", utils.BytesToString(inodo.IMtime[:]))
		for j, blk := range inodo.IBlock {
			grafo += fmt.Sprintf("<TR><TD>Block[%d]</TD><TD>%d</TD></TR>\n", j, blk)
		}
		grafo += "</TABLE>\n>];\n"
	}

	// Conectar inodos a bloques
	for i := int32(0); i < sb.SInodesCount; i++ {
		bm := make([]byte, 1)
		archivo.Seek(sb.SBmInodeStart+int64(i), 0)
		archivo.Read(bm)
		if bm[0] != '1' {
			continue
		}
		inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(i)*inodoSize)
		for _, blk := range inodo.IBlock {
			if blk != -1 {
				grafo += fmt.Sprintf("i%d -> b%d;\n", i, blk)
			}
		}
	}
	grafo += "}\n"

	f, _ := os.Create(txt)
	f.WriteString(grafo)
	f.Close()
	generarPNG(txt, png)
}

func repBlock(archivo *os.File, path string, mp *types.MountedPartition) {
	partStart := getPartStartFromFile(archivo, mp)
	if partStart == -1 {
		return
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)
	txt, png := prepararRuta(path)

	grafo := "digraph Bloques {\n"
	grafo += "node [shape=plaintext fontsize=9];\n"

	// Determinar tipo de cada bloque utilizado segun los inodos
	blockTypes := make(map[int32]byte) // '0'=folder, '1'=file, 'P'=pointer
	for i := int32(0); i < sb.SInodesCount; i++ {
		bm := make([]byte, 1)
		archivo.Seek(sb.SBmInodeStart+int64(i), 0)
		archivo.Read(bm)
		if bm[0] != '1' {
			continue
		}
		inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(i)*inodoSize)
		for j, blk := range inodo.IBlock {
			if blk == -1 {
				continue
			}
			if j < 12 {
				blockTypes[blk] = inodo.IType
			} else {
				blockTypes[blk] = 'P'
			}
		}
	}

	for blk, btype := range blockTypes {
		bm := make([]byte, 1)
		archivo.Seek(sb.SBmBlockStart+int64(blk), 0)
		archivo.Read(bm)
		if bm[0] != '1' {
			continue
		}
		offset := sb.SBlockStart + int64(blk)*blockSize
		switch btype {
		case '0': // folder
			fb := utils.ObtenerFolderBlock(archivo, offset)
			grafo += fmt.Sprintf("b%d [label=<\n", blk)
			grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" BGCOLOR=\"#e8ffe8\">\n"
			grafo += fmt.Sprintf("<TR><TD COLSPAN=\"2\" BGCOLOR=\"#336633\"><FONT COLOR=\"white\"><B>FolderBlock %d</B></FONT></TD></TR>\n", blk)
			for k := 0; k < 4; k++ {
				n := utils.BytesToString(fb.BContent[k].BName[:])
				grafo += fmt.Sprintf("<TR><TD>%s</TD><TD>%d</TD></TR>\n", n, fb.BContent[k].BInodo)
			}
			grafo += "</TABLE>\n>];\n"
		case '1': // file
			fb := utils.ObtenerFileBlock(archivo, offset)
			content := utils.BytesToString(fb.BContent[:])
			if len(content) > 40 {
				content = content[:40] + "..."
			}
			grafo += fmt.Sprintf("b%d [label=<\n", blk)
			grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" BGCOLOR=\"#fff8e0\">\n"
			grafo += fmt.Sprintf("<TR><TD BGCOLOR=\"#996633\"><FONT COLOR=\"white\"><B>FileBlock %d</B></FONT></TD></TR>\n", blk)
			grafo += fmt.Sprintf("<TR><TD>%s</TD></TR>\n", escapeHTML(content))
			grafo += "</TABLE>\n>];\n"
		case 'P': // pointer
			pb := utils.ObtenerPointerBlock(archivo, offset)
			grafo += fmt.Sprintf("b%d [label=<\n", blk)
			grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" BGCOLOR=\"#f0e0ff\">\n"
			grafo += fmt.Sprintf("<TR><TD COLSPAN=\"2\" BGCOLOR=\"#663399\"><FONT COLOR=\"white\"><B>PointerBlock %d</B></FONT></TD></TR>\n", blk)
			for k, ptr := range pb.BPointers {
				grafo += fmt.Sprintf("<TR><TD>P[%d]</TD><TD>%d</TD></TR>\n", k, ptr)
			}
			grafo += "</TABLE>\n>];\n"
		}
	}
	grafo += "}\n"

	f, _ := os.Create(txt)
	f.WriteString(grafo)
	f.Close()
	generarPNG(txt, png)
}

func repBmInode(archivo *os.File, path string, mp *types.MountedPartition) {
	partStart := getPartStartFromFile(archivo, mp)
	if partStart == -1 {
		return
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	txt, png := prepararRuta(path)

	grafo := "digraph BmInode {\n"
	grafo += "node [shape=plaintext fontsize=10];\n"
	grafo += "bm [label=<\n"
	grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n"
	grafo += "<TR><TD COLSPAN=\"20\" BGCOLOR=\"#334499\"><FONT COLOR=\"white\"><B>Bitmap de Inodos</B></FONT></TD></TR>\n"

	bitmap := utils.LeerBitmap(archivo, sb.SBmInodeStart, sb.SInodesCount)
	for i, b := range bitmap {
		if i%20 == 0 {
			grafo += "<TR>\n"
		}
		color := "#ffffff"
		val := "0"
		if b == '1' {
			color = "#336699"
			val = "<FONT COLOR=\"white\">1</FONT>"
		}
		grafo += fmt.Sprintf("<TD BGCOLOR=\"%s\">%s</TD>\n", color, val)
		if (i+1)%20 == 0 || i == len(bitmap)-1 {
			grafo += "</TR>\n"
		}
	}
	grafo += "</TABLE>\n>];\n}\n"

	f, _ := os.Create(txt)
	f.WriteString(grafo)
	f.Close()
	generarPNG(txt, png)
}

func repBmBlock(archivo *os.File, path string, mp *types.MountedPartition) {
	partStart := getPartStartFromFile(archivo, mp)
	if partStart == -1 {
		return
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	txt, png := prepararRuta(path)

	grafo := "digraph BmBlock {\n"
	grafo += "node [shape=plaintext fontsize=10];\n"
	grafo += "bm [label=<\n"
	grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n"
	grafo += "<TR><TD COLSPAN=\"20\" BGCOLOR=\"#663333\"><FONT COLOR=\"white\"><B>Bitmap de Bloques</B></FONT></TD></TR>\n"

	bitmap := utils.LeerBitmap(archivo, sb.SBmBlockStart, sb.SBlocksCount)
	for i, b := range bitmap {
		if i%20 == 0 {
			grafo += "<TR>\n"
		}
		color := "#ffffff"
		val := "0"
		if b == '1' {
			color = "#993333"
			val = "<FONT COLOR=\"white\">1</FONT>"
		}
		grafo += fmt.Sprintf("<TD BGCOLOR=\"%s\">%s</TD>\n", color, val)
		if (i+1)%20 == 0 || i == len(bitmap)-1 {
			grafo += "</TR>\n"
		}
	}
	grafo += "</TABLE>\n>];\n}\n"

	f, _ := os.Create(txt)
	f.WriteString(grafo)
	f.Close()
	generarPNG(txt, png)
}

func repSB(archivo *os.File, path string, mp *types.MountedPartition) {
	partStart := getPartStartFromFile(archivo, mp)
	if partStart == -1 {
		return
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	txt, png := prepararRuta(path)

	grafo := "digraph SuperBloque {\n"
	grafo += "node [shape=plaintext fontsize=10];\n"
	grafo += "sb [label=<\n"
	grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" BGCOLOR=\"#f5f5f5\">\n"
	grafo += "<TR><TD COLSPAN=\"2\" BGCOLOR=\"#336699\"><FONT COLOR=\"white\"><B>SuperBloque</B></FONT></TD></TR>\n"
	grafo += fmt.Sprintf("<TR><TD>s_filesystem_type</TD><TD>%d</TD></TR>\n", sb.SFilesystemType)
	grafo += fmt.Sprintf("<TR><TD>s_inodes_count</TD><TD>%d</TD></TR>\n", sb.SInodesCount)
	grafo += fmt.Sprintf("<TR><TD>s_blocks_count</TD><TD>%d</TD></TR>\n", sb.SBlocksCount)
	grafo += fmt.Sprintf("<TR><TD>s_free_blocks_count</TD><TD>%d</TD></TR>\n", sb.SFreeBlocksCount)
	grafo += fmt.Sprintf("<TR><TD>s_free_inodes_count</TD><TD>%d</TD></TR>\n", sb.SFreeInodesCount)
	grafo += fmt.Sprintf("<TR><TD>s_mtime</TD><TD>%s</TD></TR>\n", utils.BytesToString(sb.SMtime[:]))
	grafo += fmt.Sprintf("<TR><TD>s_umtime</TD><TD>%s</TD></TR>\n", utils.BytesToString(sb.SUmtime[:]))
	grafo += fmt.Sprintf("<TR><TD>s_mnt_count</TD><TD>%d</TD></TR>\n", sb.SMntCount)
	grafo += fmt.Sprintf("<TR><TD>s_magic</TD><TD>0x%X</TD></TR>\n", sb.SMagic)
	grafo += fmt.Sprintf("<TR><TD>s_inode_s</TD><TD>%d</TD></TR>\n", sb.SInodeS)
	grafo += fmt.Sprintf("<TR><TD>s_block_s</TD><TD>%d</TD></TR>\n", sb.SBlockS)
	grafo += fmt.Sprintf("<TR><TD>s_first_ino</TD><TD>%d</TD></TR>\n", sb.SFirstIno)
	grafo += fmt.Sprintf("<TR><TD>s_first_blo</TD><TD>%d</TD></TR>\n", sb.SFirstBlo)
	grafo += fmt.Sprintf("<TR><TD>s_bm_inode_start</TD><TD>%d</TD></TR>\n", sb.SBmInodeStart)
	grafo += fmt.Sprintf("<TR><TD>s_bm_block_start</TD><TD>%d</TD></TR>\n", sb.SBmBlockStart)
	grafo += fmt.Sprintf("<TR><TD>s_inode_start</TD><TD>%d</TD></TR>\n", sb.SInodeStart)
	grafo += fmt.Sprintf("<TR><TD>s_block_start</TD><TD>%d</TD></TR>\n", sb.SBlockStart)
	grafo += "</TABLE>\n>];\n}\n"

	f, _ := os.Create(txt)
	f.WriteString(grafo)
	f.Close()
	generarPNG(txt, png)
}

func repFile(path string, mp *types.MountedPartition, filePath string) {
	if filePath == "" {
		fmt.Println("Error: REP file requiere -path_file_ls")
		return
	}
	// Los reportes se generan con privilegios administrativos (no dependen
	// de sesion de usuario), por eso se lee forzando isRoot=true.
	content := filesystem.GetFileContent(mp, filePath, 1, 1, true)
	txt, png := prepararRuta(path)

	grafo := "digraph File {\n"
	grafo += "node [shape=plaintext fontsize=10];\n"
	grafo += "f [label=<\n"
	grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n"
	grafo += fmt.Sprintf("<TR><TD BGCOLOR=\"#336699\"><FONT COLOR=\"white\"><B>%s</B></FONT></TD></TR>\n", filePath)
	lines := strings.Split(content, "\n")
	for _, l := range lines {
		grafo += fmt.Sprintf("<TR><TD ALIGN=\"LEFT\">%s</TD></TR>\n", escapeHTML(l))
	}
	grafo += "</TABLE>\n>];\n}\n"

	f, _ := os.Create(txt)
	f.WriteString(grafo)
	f.Close()
	generarPNG(txt, png)
}

func repLS(path string, mp *types.MountedPartition, dirPath string) {
	if dirPath == "" {
		fmt.Println("Error: REP ls requiere -path_file_ls")
		return
	}
	entries := filesystem.LsDir(mp, dirPath)
	txt, png := prepararRuta(path)

	grafo := "digraph LS {\n"
	grafo += "node [shape=plaintext fontsize=10];\n"
	grafo += "ls [label=<\n"
	grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n"
	grafo += fmt.Sprintf("<TR><TD COLSPAN=\"6\" BGCOLOR=\"#336699\"><FONT COLOR=\"white\"><B>ls %s</B></FONT></TD></TR>\n", dirPath)
	grafo += "<TR><TD><B>Permisos</B></TD><TD><B>UID</B></TD><TD><B>GID</B></TD><TD><B>Fecha</B></TD><TD><B>Tipo</B></TD><TD><B>Nombre</B></TD></TR>\n"
	for _, e := range entries {
		tipo := "C"
		if e.Type == '1' {
			tipo = "A"
		}
		grafo += fmt.Sprintf("<TR><TD>%s</TD><TD>%d</TD><TD>%d</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD></TR>\n",
			e.Perm, e.Uid, e.Gid, e.Mtime, tipo, e.Name)
	}
	grafo += "</TABLE>\n>];\n}\n"

	f, _ := os.Create(txt)
	f.WriteString(grafo)
	f.Close()
	generarPNG(txt, png)
}

func repTree(archivo *os.File, path string, mp *types.MountedPartition) {
	partStart := getPartStartFromFile(archivo, mp)
	if partStart == -1 {
		return
	}
	sb := utils.ObtenerSuperBloque(archivo, partStart)
	inodoSize := int64(unsafe.Sizeof(types.Inodo{}))
	blockSize := int64(sb.SBlockS)
	txt, png := prepararRuta(path)

	grafo := "digraph Tree {\n"
	grafo += "node [shape=plaintext fontsize=9];\n"
	grafo += "rankdir=LR;\n"

	inodoOcupado := func(inodoNum int32) bool {
		if inodoNum < 0 || inodoNum >= sb.SInodesCount {
			return false
		}
		bm := make([]byte, 1)
		archivo.Seek(sb.SBmInodeStart+int64(inodoNum), 0)
		archivo.Read(bm)
		return bm[0] == '1'
	}

	//hace la misma verificacion pero sobre el bitmap de bloques
	bloqueOcupado := func(blkNum int32) bool {
		if blkNum < 0 || blkNum >= sb.SBlocksCount {
			return false
		}
		bm := make([]byte, 1)
		archivo.Seek(sb.SBmBlockStart+int64(blkNum), 0)
		archivo.Read(bm)
		return bm[0] == '1'
	}

	var buildTree func(inodoNum int32, depth int)
	buildTree = func(inodoNum int32, depth int) {
		if !inodoOcupado(inodoNum) { //evita inodos no usados o vacios
			return
		}
		inodo := utils.ObtenerInodo(archivo, sb.SInodeStart+int64(inodoNum)*inodoSize)
		tipo := "C"
		color := "#e0ffe0"
		headerColor := "#336633"
		if inodo.IType == '1' {
			tipo = "A"
			color = "#fff8e0"
			headerColor = "#996633"
		}
		grafo += fmt.Sprintf("i%d [label=<\n", inodoNum)
		grafo += fmt.Sprintf("<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" BGCOLOR=\"%s\">\n", color)
		grafo += fmt.Sprintf("<TR><TD COLSPAN=\"2\" BGCOLOR=\"%s\"><FONT COLOR=\"white\"><B>Inodo %d [%s]</B></FONT></TD></TR>\n", headerColor, inodoNum, tipo)
		grafo += fmt.Sprintf("<TR><TD>Perm</TD><TD>%s</TD></TR>\n", utils.BytesToString(inodo.IPerm[:]))
		grafo += "</TABLE>\n>];\n"

		for j, blkNum := range inodo.IBlock {
			if blkNum == -1 {
				break
			}
			if !bloqueOcupado(blkNum) { //se omiten bloques no usados o vacios, para evitar errores de lectura
				continue
			}
			if j < 12 {
				if inodo.IType == '0' {
					// Folder block
					fb := utils.ObtenerFolderBlock(archivo, sb.SBlockStart+int64(blkNum)*blockSize)
					grafo += fmt.Sprintf("b%d_%d [label=<\n", inodoNum, blkNum)
					grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" BGCOLOR=\"#e8f8e8\">\n"
					grafo += fmt.Sprintf("<TR><TD COLSPAN=\"2\"><B>FolderBlock %d</B></TD></TR>\n", blkNum)
					for k := 0; k < 4; k++ {
						n := utils.BytesToString(fb.BContent[k].BName[:])
						grafo += fmt.Sprintf("<TR><TD>%s</TD><TD>%d</TD></TR>\n", n, fb.BContent[k].BInodo)
					}
					grafo += "</TABLE>\n>];\n"
					grafo += fmt.Sprintf("i%d -> b%d_%d;\n", inodoNum, inodoNum, blkNum)

					// Recursivo para hijos
					for k := 0; k < 4; k++ {
						n := utils.BytesToString(fb.BContent[k].BName[:])
						child := fb.BContent[k].BInodo
						if n == "." || n == ".." || n == "" || child == -1 {
							continue
						}
						if !inodoOcupado(child) {
							continue
						}
						buildTree(child, depth+1)
						grafo += fmt.Sprintf("b%d_%d -> i%d;\n", inodoNum, blkNum, child)
					}
				} else {
					// File block
					fb := utils.ObtenerFileBlock(archivo, sb.SBlockStart+int64(blkNum)*blockSize)
					content := utils.BytesToString(fb.BContent[:])
					if len(content) > 30 {
						content = content[:30] + "..."
					}
					grafo += fmt.Sprintf("b%d_%d [label=<\n", inodoNum, blkNum)
					grafo += "<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" BGCOLOR=\"#fffde0\">\n"
					grafo += fmt.Sprintf("<TR><TD><B>FileBlock %d</B></TD></TR>\n", blkNum)
					grafo += fmt.Sprintf("<TR><TD ALIGN=\"LEFT\">%s</TD></TR>\n", escapeHTML(content))
					grafo += "</TABLE>\n>];\n"
					grafo += fmt.Sprintf("i%d -> b%d_%d;\n", inodoNum, inodoNum, blkNum)
				}
			}
		}
	}

	buildTree(0, 0)
	grafo += "}\n"

	f, _ := os.Create(txt)
	f.WriteString(grafo)
	f.Close()
	generarPNG(txt, png)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
