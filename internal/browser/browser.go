package browser

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"notion-getpage/internal/core"
)

//go:embed extract.js
var extractScript string

var ErrAuthRequired = errors.New("sessão do Notion ausente ou expirada; execute 'notion-getpage login'")
var ErrNoAccess = errors.New("a página não abriu; confira o link e o acesso da sua conta")

type Result struct {
	Markdown string   `json:"markdown"`
	Warnings []string `json:"warnings"`
	Source   string   `json:"source"`
	Error    string   `json:"error"`
}

func launch(profile string, headless bool) (*rod.Browser, error) {
	if err := core.EnsurePrivateDir(profile); err != nil {
		return nil, err
	}
	bin := os.Getenv("NOTION_GETPAGE_CHROME_PATH")
	if bin == "" {
		managedBin, err := managedChrome()
		if err != nil {
			return nil, fmt.Errorf("não foi possível obter o navegador automaticamente: %w", err)
		}
		bin = managedBin
	}
	control, err := launcher.New().
		Bin(bin).
		UserDataDir(profile).
		Headless(headless).
		NoSandbox(false).
		Delete("disable-site-isolation-trials").
		Delete("disable-features").
		Launch()
	if err != nil {
		return nil, fmt.Errorf("não foi possível abrir o navegador: %w", err)
	}
	browser := rod.New().NoDefaultDevice().ControlURL(control)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("não foi possível conectar ao navegador: %w", err)
	}
	return browser, nil
}

func openPage(browser *rod.Browser, rawURL string) (*rod.Page, error) {
	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, err
	}
	version, err := (proto.BrowserGetVersion{}).Call(browser)
	if err != nil {
		page.Close()
		return nil, err
	}
	userAgent := strings.ReplaceAll(version.UserAgent, "HeadlessChrome", "Chrome")
	if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{UserAgent: userAgent}); err != nil {
		page.Close()
		return nil, err
	}
	if err := page.Navigate(rawURL); err != nil {
		page.Close()
		return nil, err
	}
	return page, nil
}

func Login(profile string) error {
	b, err := launch(profile, false)
	if err != nil {
		return err
	}
	defer b.Close()
	page, err := openPage(b, "https://www.notion.so/login")
	if err != nil {
		return err
	}
	defer page.Close()
	fmt.Fprintln(os.Stderr, "Entre na sua conta do Notion na janela aberta. Quando terminar, pressione Enter aqui.")
	if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
		return fmt.Errorf("login requer terminal interativo: %w", err)
	}
	info, err := page.Info()
	if err != nil {
		return err
	}
	if strings.Contains(info.URL, "/login") || strings.Contains(info.URL, "/signup") {
		return errors.New("o login ainda não foi concluído")
	}
	return nil
}

func Get(profile, rawURL string, headed bool) (Result, error) {
	b, err := launch(profile, !headed)
	if err != nil {
		return Result{}, err
	}
	defer b.Close()
	page, err := openPage(b, rawURL)
	if err != nil {
		return Result{}, err
	}
	defer page.Close()
	page = page.Timeout(35 * time.Second)
	if _, err := page.Element(".notion-page-content"); err != nil {
		info, infoErr := page.Info()
		if infoErr == nil && (strings.Contains(info.URL, "/login") || strings.Contains(info.URL, "/signup")) {
			return Result{}, ErrAuthRequired
		}
		return Result{}, ErrNoAccess
	}
	info, err := page.Info()
	if err != nil {
		return Result{}, err
	}
	if err := core.ValidateURL(info.URL); err != nil {
		return Result{}, ErrNoAccess
	}
	parsed, _ := url.Parse(info.URL)
	if err := (proto.BrowserGrantPermissions{
		Permissions: []proto.BrowserPermissionType{proto.BrowserPermissionTypeClipboardReadWrite},
		Origin:      parsed.Scheme + "://" + parsed.Host,
	}).Call(b); err == nil {
		defer (proto.BrowserResetPermissions{}).Call(b)
	}
	value, err := page.Evaluate(rod.Eval(extractScript).ByPromise().ByUser())
	if err != nil {
		return Result{}, fmt.Errorf("falha ao extrair a página: %w", err)
	}
	var result Result
	if err := value.Value.Unmarshal(&result); err != nil {
		return Result{}, fmt.Errorf("resposta de extração inválida: %w", err)
	}
	if result.Error != "" {
		return Result{}, fmt.Errorf("não foi possível extrair a página: %s", result.Error)
	}
	return result, nil
}
