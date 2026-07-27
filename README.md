# WhatsApp MCP Server — Linux/KDE Fork

**🌐 Idioma / Language:** **Português** · [English](./README.en.md)

> Fork de [`lharries/whatsapp-mcp`](https://github.com/lharries/whatsapp-mcp) mantido por [@hiltoncc03](https://github.com/hiltoncc03), com foco em rodar de forma estável no Linux (testado em **CachyOS + KDE Plasma**).

Um servidor **Model Context Protocol (MCP)** para o WhatsApp. Com ele você pode buscar e ler suas mensagens pessoais do WhatsApp (incluindo imagens, vídeos, documentos e áudios), pesquisar contatos e enviar mensagens para pessoas ou grupos — tudo através de um agente de IA como o Claude.

A conexão é feita **diretamente na sua conta pessoal do WhatsApp** via API web multidispositivo (biblioteca [whatsmeow](https://github.com/tulir/whatsmeow)). Todas as mensagens ficam armazenadas **localmente** em um banco SQLite e só são enviadas ao LLM quando o agente as acessa através das ferramentas — que você controla.

![WhatsApp MCP](./example-use.png)

## O que este fork adiciona

Em relação ao projeto original, este fork traz:

- 🔔 **Notificação nativa no desktop (KDE Plasma / GNOME / qualquer libnotify)** toda vez que o bridge inicia e fica pronto (`notify-send`, sem dependência Go nova).
- 🔄 **Migração para a API atual do whatsmeow** — parâmetros `context.Context` em `Download`, `sqlstore.New`, `GetFirstDevice`, `GetGroupInfo` e `GetContact`, além de dependências atualizadas (whatsmeow, libsignal, go-sqlite3, protobuf). Corrige falhas de compilação/conexão com as versões novas do servidor do WhatsApp.
- ⚙️ **Serviço systemd de usuário** para iniciar automaticamente no boot e reiniciar sozinho (instruções abaixo).

> ⚠️ **Nota de segurança:** por padrão o bridge escuta em `:8080` em **todas as interfaces de rede** e **sem autenticação** — qualquer um na mesma rede pode enviar mensagens pela sua conta. Se a máquina não for isolada, altere `serverAddr` em `whatsapp-bridge/main.go` para `fmt.Sprintf("127.0.0.1:%d", port)` para escutar apenas em localhost (o servidor MCP roda na mesma máquina, então não precisa de acesso pela rede).

> ⚠️ **Sobre banimento:** o whatsmeow é um cliente **não-oficial** e usá-lo tecnicamente viola os Termos de Serviço do WhatsApp. Para uso pessoal e moderado (ler chats, responder contatos que já te conhecem, volume humano) o risco é baixo, mas existe. **Evite** envio em massa, automação agressiva e mensagens para números que não te conhecem.

> ⚠️ Como muitos servidores MCP, este está sujeito à [lethal trifecta](https://simonwillison.net/2025/Jun/16/the-lethal-trifecta/): injeção de prompt pode levar à exfiltração de dados privados.

## Instalação

### Pré-requisitos

- **Go** (1.25+)
- **Python 3.6+**
- **UV** (gerenciador de pacotes Python): `curl -LsSf https://astral.sh/uv/install.sh | sh`
- **Claude Desktop**, **Cursor** ou **Claude Code** (CLI)
- **FFmpeg** (_opcional_) — necessário apenas para enviar áudios como mensagens de voz do WhatsApp (formato `.ogg` Opus). Com o FFmpeg instalado, o servidor converte automaticamente outros formatos.
- **libnotify** (`notify-send`) — para as notificações de desktop deste fork (já presente na maioria dos ambientes KDE/GNOME).

### Passos

1. **Clone o fork**

   ```bash
   git clone https://github.com/hiltoncc03/whatsapp-mcp.git
   cd whatsapp-mcp
   ```

2. **Rode o WhatsApp bridge**

   ```bash
   cd whatsapp-bridge
   go run main.go
   ```

   Na primeira execução, um **QR code** será exibido no terminal. Escaneie com o WhatsApp do celular em **Configurações → Aparelhos conectados → Conectar um aparelho**.

   Quando conectar e ficar pronto, uma **notificação de desktop** será disparada. Após ~20 dias pode ser necessário reautenticar.

3. **Conecte ao servidor MCP**

   #### Claude Desktop / Cursor

   ```json
   {
     "mcpServers": {
       "whatsapp": {
         "command": "{{PATH_TO_UV}}",
         "args": [
           "--directory",
           "{{PATH_TO_SRC}}/whatsapp-mcp/whatsapp-mcp-server",
           "run",
           "main.py"
         ]
       }
     }
   }
   ```

   - `{{PATH_TO_UV}}`: saída de `which uv`
   - `{{PATH_TO_SRC}}`: caminho onde você clonou o repo (`pwd`)

   Salve como `claude_desktop_config.json` (Claude Desktop) ou `~/.cursor/mcp.json` (Cursor) e reinicie o app.

   #### Claude Code (CLI)

   ```bash
   claude mcp add whatsapp -- uv --directory /caminho/para/whatsapp-mcp/whatsapp-mcp-server run main.py
   ```

## Iniciar automaticamente no boot (systemd — Linux)

Este fork foi pensado para rodar como serviço. Compile o binário e crie um serviço de usuário:

1. **Compile:**

   ```bash
   cd whatsapp-bridge
   go build -o whatsapp-bridge .
   ```

2. **Crie** `~/.config/systemd/user/whatsapp-bridge.service`:

   ```ini
   [Unit]
   Description=WhatsApp MCP Bridge
   After=network-online.target
   Wants=network-online.target

   [Service]
   WorkingDirectory=%h/whatsapp-mcp/whatsapp-bridge
   ExecStart=%h/whatsapp-mcp/whatsapp-bridge/whatsapp-bridge
   Restart=on-failure
   RestartSec=5

   [Install]
   WantedBy=default.target
   ```

3. **Ative e habilite no boot:**

   ```bash
   systemctl --user daemon-reload
   systemctl --user enable --now whatsapp-bridge.service
   loginctl enable-linger "$USER"   # inicia no boot mesmo sem login gráfico
   ```

   Comandos úteis:

   ```bash
   systemctl --user status whatsapp-bridge      # estado
   systemctl --user restart whatsapp-bridge     # reiniciar
   journalctl --user -u whatsapp-bridge -f      # logs (ex.: QR ao reautenticar)
   ```

   > O `WorkingDirectory` **precisa** apontar para `whatsapp-bridge/`, pois o banco é gravado em `store/` relativo a esse diretório — e é dele que o servidor MCP lê. Se iniciar de outro lugar, o bridge grava num `store/` diferente e o MCP não enxerga as mensagens.

### Compatibilidade com Windows

O `go-sqlite3` exige **CGO habilitado**, que vem desligado por padrão no Windows. Instale um compilador C (via [MSYS2](https://www.msys2.org/), adicionando `ucrt64\bin` ao `PATH`) e rode:

```bash
cd whatsapp-bridge
go env -w CGO_ENABLED=1
go run main.go
```

Sem isso você verá: `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work.`

## Arquitetura

A aplicação tem dois componentes:

1. **Go WhatsApp Bridge** (`whatsapp-bridge/`): conecta na API web do WhatsApp, autentica via QR code, armazena o histórico em SQLite e expõe uma API REST em `:8080`.
2. **Python MCP Server** (`whatsapp-mcp-server/`): implementa o Model Context Protocol, fornecendo as ferramentas padronizadas para o Claude interagir com os dados do WhatsApp.

### Armazenamento

- O histórico fica em SQLite dentro de `whatsapp-bridge/store/` (`messages.db` e `whatsapp.db`).
- Os dados são gravados **em texto puro** — mantenha as permissões do diretório restritas ao seu usuário.

## Ferramentas MCP

- **search_contacts**: busca contatos por nome ou telefone
- **list_messages**: recupera mensagens com filtros e contexto
- **list_chats**: lista chats com metadados
- **get_chat**: informações de um chat específico
- **get_direct_chat_by_contact**: encontra o chat direto com um contato
- **get_contact_chats**: lista todos os chats de um contato
- **get_last_interaction**: última mensagem trocada com um contato
- **get_message_context**: contexto ao redor de uma mensagem
- **send_message**: envia mensagem para um número ou JID de grupo
- **send_file**: envia arquivo (imagem, vídeo, áudio bruto, documento)
- **send_audio_message**: envia áudio como mensagem de voz (requer `.ogg` Opus ou FFmpeg instalado)
- **download_media**: baixa a mídia de uma mensagem e retorna o caminho local

### Mídia

**Envio:** use `send_file` para imagens/vídeos/documentos e `send_audio_message` para mensagens de voz (com FFmpeg, converte MP3/WAV automaticamente para `.ogg` Opus).

**Download:** por padrão apenas os metadados da mídia ficam no banco. Para acessar o arquivo, use `download_media` com o `message_id` e o `chat_jid` (exibidos ao listar mensagens); ele baixa e retorna o caminho local.

## Como funciona

1. O Claude envia requisições ao servidor MCP em Python
2. O servidor MCP consulta o bridge Go / o banco SQLite
3. O bridge Go acessa a API do WhatsApp e mantém o SQLite atualizado
4. Os dados voltam pela cadeia até o Claude
5. Ao enviar, o fluxo vai do Claude → servidor MCP → bridge Go → WhatsApp

## Troubleshooting

- **QR code não aparece**: reinicie o bridge; verifique se o terminal suporta exibir QR codes.
- **Já logado**: se a sessão está ativa, o bridge reconecta sem mostrar QR.
- **Limite de dispositivos**: o WhatsApp limita aparelhos conectados. Remova um em **Configurações → Aparelhos conectados**.
- **Mensagens não carregam**: após autenticar, o histórico pode levar alguns minutos para sincronizar.
- **`list_chats` vazio / MCP não vê mensagens**: quase sempre é descompasso de caminho — o bridge precisa ser iniciado de dentro de `whatsapp-bridge/` (veja a nota do systemd) para gravar no mesmo `store/` que o MCP lê.
- **WhatsApp fora de sincronia**: apague `whatsapp-bridge/store/messages.db` e `whatsapp-bridge/store/whatsapp.db` e reinicie o bridge para reautenticar.

## Créditos

Fork de [`lharries/whatsapp-mcp`](https://github.com/lharries/whatsapp-mcp). Licença [MIT](./LICENSE).
