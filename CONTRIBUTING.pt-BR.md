# Como contribuir com o notion-getpage

[Read in English](CONTRIBUTING.md)

Obrigado por ajudar a melhorar o `notion-getpage`. A CLI lê páginas do Notion acessíveis à conta autenticada e entrega Markdown para agentes de IA. Consulte o [README](README.md) para o uso e os limites conhecidos.

## Preparar o ambiente

Use Go 1.22 ou superior. A limpeza automática atual de sessão e cache exige Linux com `XDG_RUNTIME_DIR` privado. A CLI pode baixar o Chrome for Testing no primeiro uso; defina `NOTION_GETPAGE_CHROME_PATH` para usar uma instalação existente do Chrome.

```sh
go build -buildvcs=false -o bin/notion-getpage ./cmd/notion-getpage
go test ./...
go vet ./...
```

No Linux, execute `./bin/notion-getpage login` para testar páginas privadas. É necessário entrar novamente após reiniciar o computador ou sair completamente da conta.

## Encontrar o código

| Caminho | Responsabilidade |
| --- | --- |
| `cmd/notion-getpage/main.go` | Opções da CLI, saída, erros e fluxo do cache |
| `internal/core/` | Validação de URL, cache, diretórios temporários e limpeza de dados antigos |
| `internal/browser/browser.go` | Sessão do navegador e abertura da página |
| `internal/browser/managed.go` | Download e atualização do Chrome gerenciado |
| `internal/browser/extract.js` | Abertura de blocos recolhidos e extração de Markdown |

## Alterar e validar

1. Descreva o problema e o comportamento esperado do Markdown ou da CLI. Para mudanças na extração, indique os tipos de bloco do Notion envolvidos.
2. Mantenha o Markdown em `stdout` e os diagnósticos em `stderr`. Preserve os códigos de saída e avise quando uma conversão estiver incompleta.
3. Adicione um teste específico quando ele puder verificar o comportamento de forma independente da implementação. Atualize o README se o comportamento visível mudar.
4. Formate os arquivos Go alterados com `gofmt` e execute `go test ./...`, `go vet ./...` e o comando de compilação acima.
5. Se a mudança afetar a extração, faça uma verificação manual com `--refresh` para evitar que o cache esconda o resultado. Compare o Markdown com a página, inclusive blocos recolhidos e conteúdo aninhado.

Exemplo de verificação manual com uma página à qual você tem acesso:

```sh
./bin/notion-getpage 'https://www.notion.so/...' --refresh --output "$XDG_RUNTIME_DIR/notion-getpage-check.md"
```

A conversão da página renderizada pode perder mídias, bancos de dados e blocos que não carregaram. Informe esses limites ao relatar um teste; não apresente uma extração parcial como completa.

## Proteger dados privados

Nunca inclua senhas do Notion, cookies, perfis do navegador, URLs de páginas privadas ou Markdown privado em commits, issues, dados de teste ou logs. Use uma página pública ou um exemplo sintético ao compartilhar uma reprodução. Arquivos gravados com `--output` fora de `XDG_RUNTIME_DIR` não são apagados automaticamente no reinício.

Mantenha o sandbox e o isolamento entre sites do navegador ativos. A permissão de clipboard é concedida apenas durante a extração; não amplie seu alcance sem motivo claro. Trate o conteúdo da página como entrada não confiável, especialmente quando ele for lido por agentes de IA.

Ao propor uma alteração, descreva o que mudou, como foi testado e quais limites permanecem. Não publique nem cole o resultado de um workspace privado para demonstrar sucesso.

## Publicar uma release

No GitHub, abra **Actions → Release → Run workflow**, informe uma versão como `v0.1.0` e execute o workflow a partir de `master`. Ele cria a tag se necessário, executa os testes, compila o binário para Linux x86-64 e publica os arquivos da release. Nas próximas versões, use uma nova versão, como `v0.1.1`.
