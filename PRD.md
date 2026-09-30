# PRD — notion-getpage

Status: **implementação inicial em validação**
Data: 2026-09-29

## 1. Problema

Agentes de IA precisam consumir o conteúdo de uma página do Notion a partir de sua URL, em Markdown, sem que o usuário tenha de exportar e copiar a página manualmente. O usuário acessa páginas de vários workspaces, inclusive como convidado com permissão apenas para visualizar.

## 2. Objetivo

Oferecer um comando local no formato `notion-getpage <URL_NOTION>` que obtenha o conteúdo de **uma página acessível à conta do usuário**, entregue Markdown utilizável por agentes e reutilize uma cópia local por um período configurável.

### Critérios de sucesso

1. Uma página privada de outro workspace, compartilhada com o usuário como convidado com permissão **Pode visualizar**, produz um `.md` sem ação do dono do workspace e sem instalar uma integração nesse workspace.
2. Chamadas repetidas para a mesma URL durante o prazo do cache não acessam o Notion novamente.
3. O agente consegue distinguir conteúdo, avisos e erros de modo automático.
4. O resultado indica quando algum conteúdo da página não pôde ser representado com segurança em Markdown.

O primeiro critério exige uma **prova de viabilidade** com uma página real antes de escolher a técnica de extração. A API oficial não cobre diretamente esse caso: convidados não podem criar tokens pessoais e integrações precisam receber acesso à página. A exportação de Markdown pela interface exige acesso completo para convidados. Ver [tokens pessoais](https://developers.notion.com/guides/get-started/personal-access-tokens), [autorização](https://developers.notion.com/guides/get-started/authorization) e [exportação](https://www.notion.com/help/export-your-content).

## 3. Usuários e fluxo principal

- **Usuário humano:** autentica sua própria conta do Notion uma vez em um navegador local e pode renovar a sessão quando ela expirar.
- **Agente de IA:** chama `notion-getpage <URL_NOTION>`, lê o Markdown e usa o código de saída para saber se a operação deu certo.
- **Primeiro uso:** o comando orienta a autenticação interativa. A forma exata do comando de login ainda será definida.
- **Uso recorrente:** o comando usa o cache válido; após o vencimento, consulta a página novamente pela sessão do usuário.

## 4. Requisitos funcionais

| ID | Requisito | Prioridade |
| --- | --- | --- |
| RF-01 | Aceitar URL de página do Notion e rejeitar URL inválida com erro claro. | Obrigatório |
| RF-02 | Ler páginas que a conta autenticada consegue visualizar, inclusive em workspaces externos onde ela é convidada, sem instalação ou aprovação adicional do dono. | Obrigatório |
| RF-03 | Produzir Markdown com título, parágrafos, títulos de seção, listas, links, citações, código e tabelas quando presentes. | Obrigatório |
| RF-04 | Preservar a ordem do conteúdo e informar blocos que não puderem ser convertidos. | Obrigatório |
| RF-05 | Manter cache local indexado pela URL, com prazo de validade configurável e opção de atualização forçada. | Obrigatório |
| RF-06 | Permitir gravar o resultado em um arquivo `.md`. | Obrigatório |
| RF-07 | Usar a sessão da conta do usuário sem colocar credenciais na URL, nos argumentos do processo, no Markdown ou nos logs. | Obrigatório |
| RF-08 | Devolver códigos de saída distintos para URL inválida, falta de autenticação, falta de acesso e falha temporária, se a técnica escolhida permitir essa distinção. | Desejável |
| RF-09 | Permitir uso não interativo após a autenticação inicial. | Obrigatório |

### Contrato de CLI proposto

```text
notion-getpage <URL_NOTION> [--output CAMINHO] [--ttl DURAÇÃO] [--refresh]
```

`--output` grava o Markdown no caminho indicado. Sem `--output`, Markdown vai para `stdout` e diagnósticos para `stderr`. A primeira implementação usa `notion-getpage login`, duração em `s`, `m`, `h` ou `d`, e TTL padrão de `1h`.

## 5. Cache e dados locais

- A **chave lógica é a URL** recebida; a implementação pode usar seu hash como nome do arquivo no disco.
- Cada entrada deve registrar quando foi obtida e quando vence. O cache guarda apenas resultados completos; falhas de autenticação, acesso ou conversão não entram nele.
- Arquivos de cache e de sessão contêm dados privados e devem ficar em diretórios locais com permissões restritas. No Linux, usar `XDG_RUNTIME_DIR` para que sejam eliminados no logout completo ou reinício; manter apenas o binário do navegador em cache persistente.
- `--refresh` ignora uma entrada válida e tenta atualizar a página.
- Uma URL diferente pode apontar para a mesma página; a regra para tratar variantes da URL ainda precisa ser definida.
- O comportamento quando o acesso é revogado mas ainda existe cache válido precisa ser definido explicitamente.

## 6. Qualidade e limites do Markdown

- O `.md` deve conter o conteúdo da página solicitada, com links para subpáginas; seguir e incorporar subpáginas não faz parte do fluxo inicial até decidirmos essa regra.
- Imagens, anexos, embeds, bancos de dados, conteúdo recolhido e blocos especiais precisam de uma política explícita antes da implementação: link, texto alternativo, marcador de omissão ou falha.
- A ferramenta não deve apresentar uma extração parcial como se fosse completa. Avisos devem ser legíveis por agentes sem contaminar o Markdown.
- A extração deve evitar menus, barra lateral, comentários e outros elementos da interface que não pertencem à página.

## 7. Restrições técnicas verificadas

| Caminho | Vantagem | Limite para este produto |
| --- | --- | --- |
| [API oficial de Markdown](https://developers.notion.com/reference/retrieve-page-markdown) | Markdown direto e estruturado. | Exige credencial com acesso à página. Um convidado não pode criar token pessoal para o workspace externo; uma integração precisa ser autorizada. |
| [Exportação pela interface](https://www.notion.com/help/export-your-content) | Exportação oficial em Markdown e CSV. | Convidados precisam de **acesso completo** para ver a opção Exportar; o workspace pode desativá-la. |
| Cópia ou leitura da página no navegador autenticado | Usa a sessão do próprio usuário e pode funcionar com permissão de visualização. | Precisa de validação prática quanto a conteúdo longo, blocos recolhidos, tabelas, links, mídia e mudanças na interface do Notion. |

**Direção técnica:** usar Go para distribuir um executável sem runtime ou gerenciador de pacotes. Validar a leitura pela sessão autenticada no navegador com uma página de convidado em modo “Pode visualizar”. A técnica exata de extração ainda depende dessa prova.

O navegador continua sendo um componente necessário para abrir páginas privadas com a sessão do usuário. A primeira implementação baixa Chrome for Testing automaticamente no primeiro uso, sem instalação manual de dependências. O primeiro login na conta Notion continua sendo uma ação do usuário. Um pacote totalmente offline ainda não foi construído.

## 8. Segurança e operação

- Sessão e cache ficam apenas no computador do usuário e, no Linux, são descartados no logout completo ou reinício.
- O navegador deve manter o sandbox e o isolamento entre sites ativos. O Chrome gerenciado deve verificar atualizações periodicamente.
- A permissão de clipboard deve ficar restrita ao período de extração. Ela não modifica a permissão da conta no Notion.
- Não coletar senha nem pedir que o usuário copie cookies para a CLI.
- Uma falha de login deve pedir uma nova autenticação interativa, sem sugerir acesso administrativo.
- Erros, URLs e logs não devem expor tokens de sessão.
- O projeto deve funcionar em chamadas repetidas de agentes sem abrir uma janela ou pedir interação a cada execução.

## 9. Validação antes do lançamento

1. Testar uma página privada com permissão de convidado “Pode visualizar”.
2. Comparar Markdown com a página original em uma amostra que contenha títulos, listas aninhadas, links, código, tabela, imagem e toggle.
3. Verificar que a segunda chamada usa cache e que `--refresh` atualiza o conteúdo.
4. Verificar login expirado, URL inválida, página sem permissão e falha de rede.
5. Verificar que a saída padrão contém apenas Markdown e que erros saem no canal apropriado.

## 10. Decisões em aberto

1. **Plataformas:** armazenamento temporário com limpeza no reinício implementado no Linux; definir mecanismo equivalente para macOS e Windows.
2. **Subpáginas e mídia:** preservar só links, ou baixar e montar uma pasta com assets?
3. **Qualidade mínima:** quais blocos devem causar erro se não forem extraídos?
4. **Cache após perda de acesso:** a implementação inicial devolve o conteúdo até o fim do TTL; confirmar se essa regra serve.
5. **Distribuição offline:** criar um pacote por plataforma que inclua Chrome, caso o primeiro uso não possa depender de rede.
6. **Técnica de extração:** validar o acesso como convidado “Pode visualizar”; a primeira implementação usa cópia do conteúdo pelo navegador e conversão da página renderizada como alternativa.

## 11. Fora do escopo inicial proposto

- Modificar páginas do Notion.
- Exportar um workspace inteiro ou sincronizar todas as páginas.
- Usar uma integração instalada pelo dono do workspace como requisito para o fluxo principal.
