package main

// Travas de uso da bridge, configuradas por variável de ambiente. Sem variáveis, o comportamento não muda.
//
//   WHATSAPP_STORE_ALLOW=a,b       grava só estes chats (número, JID de usuário, @lid ou JID de grupo @g.us)
//   WHATSAPP_SEND_ALLOW=a,b        a API /api/send só envia para estes destinos
//   WHATSAPP_SEND_TEXT_ONLY=1      recusa envio de arquivo/áudio
//   WHATSAPP_READ_ONLY=1           recusa qualquer envio
//   WHATSAPP_SEND_MAX_PER_DAY=N    teto de envios por dia (contado desde a partida do processo)
//   WHATSAPP_DOWNLOAD_DISABLED=1   recusa /api/download
//   WHATSAPP_BRIDGE_HOST=0.0.0.0   endereço da API REST (padrão 127.0.0.1)
//   WHATSAPP_NO_NOTIFY=1           não chama notify-send
//   WHATSAPP_PAIR_PHONE=55...      pareia por código de 8 dígitos em vez de QR (grava store/pair-code.txt)
//
// A trava de envio é aplicada aqui, e não só no servidor MCP, porque a API REST não tem autenticação:
// quem alcança a porta poderia furar a allowlist do MCP chamando /api/send direto.

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func envList(name string) []string {
	var out []string
	for _, a := range strings.Split(os.Getenv(name), ",") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	return out
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// userPart devolve a parte antes do @, sem o sufixo de aparelho (":12").
func userPart(jid string) string {
	base := strings.SplitN(jid, "@", 2)[0]
	if i := strings.Index(base, ":"); i >= 0 {
		base = base[:i]
	}
	return base
}

// jidMatches compara um JID com a allowlist. Grupo (@g.us) só casa por JID completo; usuário casa pelos dígitos.
func jidMatches(jid string, allow []string) bool {
	jid = strings.TrimSpace(jid)
	isGroup := strings.HasSuffix(jid, "@g.us")
	for _, a := range allow {
		if a == jid {
			return true
		}
		if isGroup || strings.HasSuffix(a, "@g.us") {
			continue
		}
		d := onlyDigits(userPart(a))
		if d != "" && d == onlyDigits(userPart(jid)) {
			return true
		}
	}
	return false
}

func storeAllowed(chatJID string) bool {
	allow := envList("WHATSAPP_STORE_ALLOW")
	if len(allow) == 0 {
		return true
	}
	return jidMatches(chatJID, allow)
}

var (
	sendMu    sync.Mutex
	sendDay   string
	sendCount int
)

// sendBlockedReason devolve o motivo da recusa, ou "" se o envio pode sair. Conta o envio quando libera.
func sendBlockedReason(recipient string, hasMedia bool) string {
	if os.Getenv("WHATSAPP_READ_ONLY") == "1" {
		return "sending is disabled on this bridge (WHATSAPP_READ_ONLY=1)"
	}
	if hasMedia && os.Getenv("WHATSAPP_SEND_TEXT_ONLY") == "1" {
		return "only text messages are allowed on this bridge (WHATSAPP_SEND_TEXT_ONLY=1)"
	}
	if allow := envList("WHATSAPP_SEND_ALLOW"); len(allow) > 0 {
		jid := strings.TrimSpace(recipient)
		if !strings.Contains(jid, "@") {
			jid = onlyDigits(jid) + "@s.whatsapp.net"
		}
		if !jidMatches(jid, allow) {
			return "recipient is not in WHATSAPP_SEND_ALLOW; message not sent"
		}
	}
	if max, err := strconv.Atoi(os.Getenv("WHATSAPP_SEND_MAX_PER_DAY")); err == nil && max > 0 {
		sendMu.Lock()
		defer sendMu.Unlock()
		today := time.Now().Format("2006-01-02")
		if today != sendDay {
			sendDay, sendCount = today, 0
		}
		if sendCount >= max {
			return "daily send limit reached (WHATSAPP_SEND_MAX_PER_DAY)"
		}
		sendCount++
	}
	return ""
}

func listenHost() string {
	if h := strings.TrimSpace(os.Getenv("WHATSAPP_BRIDGE_HOST")); h != "" {
		return h
	}
	return "127.0.0.1"
}
