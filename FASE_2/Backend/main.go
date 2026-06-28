package main

import (
	"bufio"
	"fmt"
	"mia/commands"
	"mia/server"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]

	if len(args) > 0 && args[0] == "-server" {
		addr := ":8080"
		if len(args) > 1 {
			addr = args[1]
			if !strings.HasPrefix(addr, ":") {
				addr = ":" + addr
			}
		}
		if err := server.Start(addr); err != nil {
			fmt.Println("Error al iniciar el servidor:", err)
			os.Exit(1)
		}
		return
	}

	// Si se pasa un script como argumento
	if len(args) > 0 {
		for _, arg := range args {
			if strings.HasSuffix(arg, ".smia") {
				commands.RunScript(arg)
			} else {
				cmd, params := commands.ParseLine(arg)
				if cmd != "" {
					commands.Execute(cmd, params)
				}
			}
		}
		return
	}

	// REPL interactivo
	fmt.Println("===========================================")
	fmt.Println(" 	 			    EXT2	     		    ")
	fmt.Println("===========================================")
	fmt.Println("Comandos: MKDISK, RMDISK, FDISK, MOUNT, MKFS,")
	fmt.Println("          LOGIN, LOGOUT, MKGRP, RMGRP, MKUSR,")
	fmt.Println("          RMUSR, CHGRP, MKDIR, MKFILE, REP, EXEC, PAUSE")
	fmt.Println("Escriba 'exit' para salir")
	fmt.Println("-------------------------------------------")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("mia> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.ToLower(line) == "exit" {
			fmt.Println("Saliendo...")
			break
		}
		cmd, params := commands.ParseLine(line)
		if cmd != "" {
			commands.Execute(cmd, params)
		}
	}
}
