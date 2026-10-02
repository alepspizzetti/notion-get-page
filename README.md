# notion-getpage

CLI em Go para ler uma página do Notion pela sessão do usuário e entregar Markdown a agentes de IA. Para colaborar, veja [Como contribuir](CONTRIBUTING.pt-BR.md) ou [Contributing](CONTRIBUTING.md).

## Estado

Primeira implementação. A leitura de uma página privada fornecida pelo usuário foi testada. O Notion pode mudar a estrutura da página e a cópia oferecida pelo navegador.

## Instalar sem compilar

O [instalador](install.sh) baixa a [release](https://github.com/alepspizzetti/notion-get-page/releases) para Linux x86-64, verifica os checksums e instala `notion-getpage` em `~/.local/bin`, sem `sudo` nem Go na máquina. A licença e os avisos das dependências ficam em `~/.local/share/doc/notion-getpage`. O instalador usa `curl` ou `wget` e `sha256sum`. O Chrome for Testing é baixado automaticamente no primeiro uso. Ainda é necessário entrar na sua conta do Notion com `notion-getpage login` após a instalação e após cada reinício.

Após a publicação da primeira release, instale a versão mais recente com:

```sh
curl -fsSL https://raw.githubusercontent.com/alepspizzetti/notion-get-page/master/install.sh | sh
notion-getpage login
```

Para criar a primeira release, um mantenedor pode abrir [Actions → Release](https://github.com/alepspizzetti/notion-get-page/actions/workflows/release.yml), clicar em **Run workflow** e manter `v0.1.0` como versão. O workflow testa o projeto, publica o binário e seus checksums; depois disso, o comando de instalação acima estará disponível.

Se `~/.local/bin` não estiver no `PATH`, execute `~/.local/bin/notion-getpage` ou adicione esse diretório ao `PATH`. Para atualizar o binário, rode o instalador novamente. O projeto ainda não oferece uma release funcional para macOS ou Windows.

## Compilar

```sh
go build -buildvcs=false -o bin/notion-getpage ./cmd/notion-getpage
```

O executável não precisa de Go na máquina de destino. Para abrir páginas privadas, o CLI baixa automaticamente uma versão gerenciada do Chrome for Testing no primeiro uso e verifica se há atualização uma vez por dia. Se a verificação falhar, usa a versão instalada e avisa no `stderr`. O usuário precisa fazer login na janela do navegador aberta pelo comando. É possível apontar para um Chrome já instalado com `NOTION_GETPAGE_CHROME_PATH`.

## Usar

```sh
./bin/notion-getpage login
./bin/notion-getpage 'https://www.notion.so/...'
./bin/notion-getpage 'https://www.notion.so/...' --output pagina.md
./bin/notion-getpage 'https://www.notion.so/...' --ttl 2h --refresh
```

No primeiro comando, aguarde a janela do navegador abrir, entre na sua conta do Notion e volte ao terminal para pressionar **Enter**. O CLI salva essa sessão para as próximas chamadas durante a sessão do sistema. Após reiniciar o computador ou sair completamente da conta do Linux, é necessário fazer login novamente. Uma página de outro workspace deve ser aberta com a mesma conta que recebeu o convite.

Por padrão, o Markdown sai no `stdout`; avisos e erros saem no `stderr`. Com `--output`, o comando grava o arquivo e imprime seu caminho absoluto. `--ttl` aceita `s`, `m`, `h` ou `d` e vale `1h` por padrão. `--refresh` ignora o cache. `--headed` mostra o navegador ao ler uma página.

O próprio programa gerencia o prazo do cache: grava a data de vencimento e ignora entradas vencidas ao ler. O cache usa a URL exata como chave. No Linux, cache e sessão ficam em `$XDG_RUNTIME_DIR/notion-getpage/{cache,profile}`, que o sistema remove no logout completo ou reinício. O binário do Chrome permanece em `~/.cache/notion-getpage/browser` e é verificado periodicamente. No primeiro uso desta versão, dados privados dos antigos diretórios persistentes são apagados; o login precisa ser repetido. `NOTION_GETPAGE_CACHE_DIR` e `NOTION_GETPAGE_PROFILE_DIR` permitem caminhos personalizados, mas podem impedir a limpeza automática no reinício. `--output` grava no caminho solicitado e não faz parte do cache temporário.

## Limitações atuais

- O CLI abre automaticamente os toggles recolhidos antes de extrair a página. Se algum não abrir, emite um aviso no `stderr`.
- O Markdown inclui uma seção de comentários com as discussões e respostas visíveis no painel do Notion, associadas ao trecho comentado quando possível.
- O método principal seleciona o conteúdo da página e usa a cópia em Markdown do Notion. Se ela falhar, há uma conversão básica da página renderizada, com aviso no `stderr`.
- Bancos de dados, embeds, mídia e páginas muito longas podem não ser extraídos por completo no modo de conversão da página renderizada.
- A primeira execução pode precisar baixar o navegador. Para uma distribuição totalmente offline, será necessário publicar um pacote por plataforma que o inclua.
- O cache pode devolver conteúdo já obtido durante o prazo de validade mesmo após a perda do acesso no Notion.
- O armazenamento temporário automático exige Linux com `XDG_RUNTIME_DIR` privado. macOS e Windows ainda precisam de uma implementação equivalente para garantir a limpeza no reinício.
- A permissão de clipboard do navegador é concedida somente durante a extração e removida logo depois. Ela permite ler e escrever no clipboard do sistema; não concede permissão para editar a página do Notion.

## Códigos de saída

`0` sucesso; `1` falha geral; `2` uso/URL inválidos; `3` sessão ausente; `4` página não acessível.

## Licença

Este projeto é distribuído sob a licença [MIT](LICENSE).
