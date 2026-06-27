package auth

import (
	"fmt"
	"mia/filesystem"
	"mia/mount"
	"mia/types"
	"strings"
)

func Login(params map[string]string) {
	if mount.CurrentSession != nil {
		fmt.Println("Error: ya hay una sesion activa. Haga LOGOUT primero")
		return
	}
	user, ok1 := params["user"]
	pass, ok2 := params["pass"]
	id, ok3 := params["id"]
	if !ok1 || !ok2 || !ok3 {
		fmt.Println("Error: LOGIN requiere -user, -pass, -id")
		return
	}
	user = strings.ReplaceAll(user, "\"", "")
	pass = strings.ReplaceAll(pass, "\"", "")
	id = strings.ReplaceAll(id, "\"", "")

	mp, ok := mount.GetMountedPartition(id)
	if !ok {
		fmt.Println("Error: particion no montada:", id)
		return
	}

	// Leer users.txt
	content := filesystem.ReadUsersFile(mp)
	if content == "" {
		fmt.Println("Error: no se pudo leer users.txt")
		return
	}

	lines := strings.Split(content, "\n")
	var uid, gid int32 = -1, -1

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		tipo := strings.TrimSpace(parts[1])
		if tipo == "U" && len(parts) >= 5 {
			uname := strings.TrimSpace(parts[3])
			upass := strings.TrimSpace(parts[4])
			ugrp := strings.TrimSpace(parts[2])
			if uname == user && upass == pass {
				fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &uid)
				// Buscar gid del grupo
				for _, l2 := range lines {
					l2 = strings.TrimSpace(l2)
					p2 := strings.Split(l2, ",")
					if len(p2) >= 3 && strings.TrimSpace(p2[1]) == "G" && strings.TrimSpace(p2[2]) == ugrp {
						fmt.Sscanf(strings.TrimSpace(p2[0]), "%d", &gid)
						break
					}
				}
				mount.CurrentSession = &types.Session{
					User:   user,
					Pass:   pass,
					Id:     id,
					Uid:    uid,
					Gid:    gid,
					IsRoot: user == "root",
				}
				fmt.Println("Login exitoso:", user)
				return
			}
		}
	}
	fmt.Println("Error: usuario o contrasena incorrectos")
}

func Logout() {
	if mount.CurrentSession == nil {
		fmt.Println("Error: no hay sesion activa")
		return
	}
	fmt.Println("Logout:", mount.CurrentSession.User)
	mount.CurrentSession = nil
}

func MkGrp(params map[string]string) {
	if !checkRoot() {
		return
	}
	name, ok := params["name"]
	if !ok {
		fmt.Println("Error: MKGRP requiere -name")
		return
	}
	name = strings.ReplaceAll(name, "\"", "")

	mp, ok2 := mount.GetMountedPartition(mount.CurrentSession.Id)
	if !ok2 {
		return
	}

	content := filesystem.ReadUsersFile(mp)
	lines := strings.Split(content, "\n")

	// Verificar que no exista
	for _, l := range lines {
		l = strings.TrimSpace(l)
		parts := strings.Split(l, ",")
		if len(parts) >= 3 && strings.TrimSpace(parts[1]) == "G" && strings.TrimSpace(parts[2]) == name {
			fmt.Println("Error: el grupo ya existe")
			return
		}
	}

	// Obtener siguiente GID
	maxGid := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		parts := strings.Split(l, ",")
		if len(parts) >= 2 && strings.TrimSpace(parts[1]) == "G" {
			var g int
			fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &g)
			if g > maxGid {
				maxGid = g
			}
		}
	}

	newLine := fmt.Sprintf("%d,G,%s\n", maxGid+1, name)
	filesystem.WriteUsersFile(mp, content+newLine)
	fmt.Println("Grupo creado:", name)
}

func RmGrp(params map[string]string) {
	if !checkRoot() {
		return
	}
	name, ok := params["name"]
	if !ok {
		fmt.Println("Error: RMGRP requiere -name")
		return
	}
	name = strings.ReplaceAll(name, "\"", "")

	if name == "root" {
		fmt.Println("Error: no se puede eliminar el grupo root")
		return
	}

	mp, _ := mount.GetMountedPartition(mount.CurrentSession.Id)
	content := filesystem.ReadUsersFile(mp)
	lines := strings.Split(content, "\n")
	var newLines []string
	found := false
	for _, l := range lines {
		l = strings.TrimSpace(l)
		parts := strings.Split(l, ",")
		if len(parts) >= 3 && strings.TrimSpace(parts[1]) == "G" && strings.TrimSpace(parts[2]) == name {
			found = true
			// Marcar como 0 (eliminado logico)
			newLines = append(newLines, fmt.Sprintf("0,G,%s", name))
		} else {
			if l != "" {
				newLines = append(newLines, l)
			}
		}
	}
	if !found {
		fmt.Println("Error: grupo no encontrado")
		return
	}
	filesystem.WriteUsersFile(mp, strings.Join(newLines, "\n")+"\n")
	fmt.Println("Grupo eliminado:", name)
}

func MkUsr(params map[string]string) {
	if !checkRoot() {
		return
	}
	user, ok1 := params["user"]
	pass, ok2 := params["pass"]
	grp, ok3 := params["grp"]
	if !ok1 || !ok2 || !ok3 {
		fmt.Println("Error: MKUSR requiere -user, -pass, -grp")
		return
	}
	user = strings.ReplaceAll(user, "\"", "")
	pass = strings.ReplaceAll(pass, "\"", "")
	grp = strings.ReplaceAll(grp, "\"", "")

	if len(user) > 10 {
		fmt.Println("Error: usuario max 10 caracteres")
		return
	}

	mp, _ := mount.GetMountedPartition(mount.CurrentSession.Id)
	content := filesystem.ReadUsersFile(mp)
	lines := strings.Split(content, "\n")

	// Verificar usuario unico
	for _, l := range lines {
		l = strings.TrimSpace(l)
		parts := strings.Split(l, ",")
		if len(parts) >= 4 && strings.TrimSpace(parts[1]) == "U" && strings.TrimSpace(parts[3]) == user {
			fmt.Println("Error: usuario ya existe")
			return
		}
	}

	// Verificar grupo existe
	gid := -1
	for _, l := range lines {
		l = strings.TrimSpace(l)
		parts := strings.Split(l, ",")
		if len(parts) >= 3 && strings.TrimSpace(parts[1]) == "G" && strings.TrimSpace(parts[2]) == grp {
			fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &gid)
			break
		}
	}
	if gid <= 0 {
		fmt.Println("Error: grupo no existe")
		return
	}

	// Siguiente UID
	maxUid := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		parts := strings.Split(l, ",")
		if len(parts) >= 2 && strings.TrimSpace(parts[1]) == "U" {
			var u int
			fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &u)
			if u > maxUid {
				maxUid = u
			}
		}
	}

	newLine := fmt.Sprintf("%d,U,%s,%s,%s\n", maxUid+1, grp, user, pass)
	filesystem.WriteUsersFile(mp, content+newLine)
	fmt.Println("Usuario creado:", user)
}

func RmUsr(params map[string]string) {
	if !checkRoot() {
		return
	}
	user, ok := params["user"]
	if !ok {
		fmt.Println("Error: RMUSR requiere -user")
		return
	}
	user = strings.ReplaceAll(user, "\"", "")

	if user == "root" {
		fmt.Println("Error: no se puede eliminar al usuario root")
		return
	}

	mp, _ := mount.GetMountedPartition(mount.CurrentSession.Id)
	content := filesystem.ReadUsersFile(mp)
	lines := strings.Split(content, "\n")
	var newLines []string
	found := false
	for _, l := range lines {
		l = strings.TrimSpace(l)
		parts := strings.Split(l, ",")
		if len(parts) >= 4 && strings.TrimSpace(parts[1]) == "U" && strings.TrimSpace(parts[3]) == user {
			found = true
			newLines = append(newLines, fmt.Sprintf("0,U,%s,%s,%s", parts[2], user, parts[4]))
		} else {
			if l != "" {
				newLines = append(newLines, l)
			}
		}
	}
	if !found {
		fmt.Println("Error: usuario no encontrado")
		return
	}
	filesystem.WriteUsersFile(mp, strings.Join(newLines, "\n")+"\n")
	fmt.Println("Usuario eliminado:", user)
}

func ChGrp(params map[string]string) {
	if !checkRoot() {
		return
	}
	user, ok1 := params["user"]
	grp, ok2 := params["grp"]
	if !ok1 || !ok2 {
		fmt.Println("Error: CHGRP requiere -user y -grp")
		return
	}
	user = strings.ReplaceAll(user, "\"", "")
	grp = strings.ReplaceAll(grp, "\"", "")

	mp, _ := mount.GetMountedPartition(mount.CurrentSession.Id)
	content := filesystem.ReadUsersFile(mp)
	lines := strings.Split(content, "\n")
	var newLines []string
	found := false
	for _, l := range lines {
		l = strings.TrimSpace(l)
		parts := strings.Split(l, ",")
		if len(parts) >= 5 && strings.TrimSpace(parts[1]) == "U" && strings.TrimSpace(parts[3]) == user {
			found = true
			newLines = append(newLines, fmt.Sprintf("%s,U,%s,%s,%s", parts[0], grp, user, parts[4]))
		} else {
			if l != "" {
				newLines = append(newLines, l)
			}
		}
	}
	if !found {
		fmt.Println("Error: usuario no encontrado")
		return
	}
	filesystem.WriteUsersFile(mp, strings.Join(newLines, "\n")+"\n")
	fmt.Println("Grupo de", user, "cambiado a", grp)
}

func checkRoot() bool {
	if mount.CurrentSession == nil {
		fmt.Println("Error: no hay sesion activa")
		return false
	}
	if !mount.CurrentSession.IsRoot {
		fmt.Println("Error: solo root puede realizar esta operacion")
		return false
	}
	return true
}
