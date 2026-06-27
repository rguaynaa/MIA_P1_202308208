package types

// MBR
type MBR struct {
	MbrTamanio       int64
	MbrFechaCreacion [20]byte
	MbrDskSignature  int32
	DskFit           byte
	MbrPartitions    [4]Partition
}

// Partition
type Partition struct {
	PartStatus      byte
	PartType        byte
	PartFit         byte
	PartStart       int64
	PartS           int64
	PartName        [16]byte
	PartCorrelative int32
	PartId          [4]byte
}

// EBR
type EBR struct {
	PartMount byte
	PartFit   byte
	PartStart int64
	PartS     int64
	PartNext  int64
	PartName  [16]byte
}

// SuperBloque EXT2
type SuperBloque struct {
	SFilesystemType  int32
	SInodesCount     int32
	SBlocksCount     int32
	SFreeBlocksCount int32
	SFreeInodesCount int32
	SMtime           [20]byte
	SUmtime          [20]byte
	SMntCount        int32
	SMagic           int32
	SInodeS          int32
	SBlockS          int32
	SFirstIno        int32
	SFirstBlo        int32
	SBmInodeStart    int64
	SBmBlockStart    int64
	SInodeStart      int64
	SBlockStart      int64
}

// Inodo EXT2
type Inodo struct {
	IUid   int32
	IGid   int32
	IS     int32
	IAtime [20]byte
	ICtime [20]byte
	IMtime [20]byte
	IBlock [15]int32
	IType  byte
	IPerm  [3]byte
}

// FolderContent:contenido de bloque carpeta
type FolderContent struct {
	BName  [12]byte
	BInodo int32
}

// FolderBlock:bloque de carpeta
type FolderBlock struct {
	BContent [4]FolderContent
}

// FileBlock:bloque de archivo
type FileBlock struct {
	BContent [64]byte
}

// PointerBlock:bloque de apuntadores
type PointerBlock struct {
	BPointers [16]int32
}

// MountedPartition: particion montada en RAM
type MountedPartition struct {
	Path        string
	Name        string
	Id          string
	Correlative int32
}

// Session: sesion activa
type Session struct {
	User   string
	Pass   string
	Id     string
	Uid    int32
	Gid    int32
	IsRoot bool
}
