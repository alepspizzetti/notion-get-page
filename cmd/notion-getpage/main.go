package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"notion-getpage/internal/browser"
	"notion-getpage/internal/core"
)

const usage = `Uso:
  notion-getpage login
  notion-getpage <URL_NOTION> [--output ARQUIVO] [--ttl 1h] [--refresh] [--headed]

Por padrão, Markdown vai para stdout e avisos vão para stderr.
O primeiro login abre um navegador; as chamadas seguintes usam a sessão salva.
`

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(stdout, usage)
		return 0
	}
	profile, err := core.ProfileDir()
	if err != nil {
		fmt.Fprintln(stderr, "Erro ao localizar dados de sessão:", err)
		return 1
	}
	if err := core.CleanupLegacyData(); err != nil {
		fmt.Fprintln(stderr, "Erro ao apagar dados antigos:", err)
		return 1
	}
	if args[0] == "login" {
		if len(args) != 1 {
			fmt.Fprint(stderr, usage)
			return 2
		}
		if err := browser.Login(profile); err != nil {
			fmt.Fprintln(stderr, "Erro:", err)
			return 3
		}
		fmt.Fprintln(stderr, "Sessão do Notion pronta.")
		return 0
	}
	if err := core.ValidateURL(args[0]); err != nil {
		fmt.Fprintln(stderr, "Erro:", err)
		return 2
	}
	flags := flag.NewFlagSet("notion-getpage", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("output", "", "grava o Markdown neste arquivo")
	ttlText := flags.String("ttl", "1h", "prazo do cache: s, m, h ou d")
	refresh := flags.Bool("refresh", false, "ignora cache válido")
	headed := flags.Bool("headed", false, "mostra o navegador durante a leitura")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	ttl, err := core.ParseDuration(*ttlText)
	if err != nil {
		fmt.Fprintln(stderr, "Erro:", err)
		return 2
	}
	cacheDir, err := core.CacheDir()
	if err != nil {
		fmt.Fprintln(stderr, "Erro ao localizar cache:", err)
		return 1
	}
	cache := core.Cache{Dir: cacheDir}
	markdown, hit := "", false
	if !*refresh {
		markdown, hit = cache.Get(args[0], time.Now())
	}
	if !hit {
		result, err := browser.Get(profile, args[0], *headed)
		if err != nil {
			fmt.Fprintln(stderr, "Erro:", err)
			switch {
			case errors.Is(err, browser.ErrAuthRequired):
				return 3
			case errors.Is(err, browser.ErrNoAccess):
				return 4
			default:
				return 1
			}
		}
		for _, warning := range result.Warnings {
			fmt.Fprintln(stderr, "Aviso:", warning)
		}
		markdown = result.Markdown
		if len(result.Warnings) == 0 {
			if err := cache.Put(args[0], markdown, ttl, time.Now()); err != nil {
				fmt.Fprintln(stderr, "Aviso: não foi possível salvar o cache:", err)
			}
		}
	}
	if *output == "" {
		if _, err := io.WriteString(stdout, markdown); err != nil {
			fmt.Fprintln(stderr, "Erro ao escrever Markdown:", err)
			return 1
		}
		return 0
	}
	path, err := filepath.Abs(*output)
	if err != nil {
		fmt.Fprintln(stderr, "Erro:", err)
		return 1
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Fprintln(stderr, "Erro ao salvar arquivo:", err)
		return 1
	}
	if err := file.Chmod(0600); err != nil {
		file.Close()
		fmt.Fprintln(stderr, "Erro ao proteger arquivo:", err)
		return 1
	}
	_, writeErr := io.WriteString(file, markdown)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		if writeErr == nil {
			writeErr = closeErr
		}
		fmt.Fprintln(stderr, "Erro ao salvar arquivo:", writeErr)
		return 1
	}
	fmt.Fprintln(stdout, path)
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
