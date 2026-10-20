package main

import "testing"

func TestStoreAllowed(t *testing.T) {
	t.Setenv("WHATSAPP_STORE_ALLOW", "5511942749700, 120363412404981832@g.us, 194957776314530@lid")
	cases := []struct {
		jid  string
		want bool
	}{
		{"5511942749700@s.whatsapp.net", true},   // chat da EF: libera
		{"5511942749700:3@s.whatsapp.net", true}, // mesmo número, outro aparelho: libera
		{"120363412404981832@g.us", true},        // grupo de teste: libera
		{"194957776314530@lid", true},            // conversa @lid escolhida: libera
		{"5531999990000@s.whatsapp.net", false},  // conversa pessoal: barra
		{"120363423380350510@g.us", false},       // outro grupo: barra
		{"5511942749700@g.us", false},            // grupo nunca casa por dígitos: barra
	}
	for _, c := range cases {
		if got := storeAllowed(c.jid); got != c.want {
			t.Errorf("storeAllowed(%q) = %v, want %v", c.jid, got, c.want)
		}
	}
}

func TestStoreAllowedSemLista(t *testing.T) {
	t.Setenv("WHATSAPP_STORE_ALLOW", "")
	if !storeAllowed("5531999990000@s.whatsapp.net") {
		t.Fatal("sem allowlist o comportamento antigo (gravar tudo) deve continuar")
	}
}

func TestSendBlockedReason(t *testing.T) {
	t.Setenv("WHATSAPP_SEND_ALLOW", "5511942749700,120363412404981832@g.us")
	t.Setenv("WHATSAPP_SEND_TEXT_ONLY", "1")
	t.Setenv("WHATSAPP_SEND_MAX_PER_DAY", "2")
	sendDay, sendCount = "", 0
	if r := sendBlockedReason("+55 11 94274-9700", false); r != "" {
		t.Fatalf("número da EF deveria liberar, veio %q", r)
	}
	if r := sendBlockedReason("120363412404981832@g.us", false); r != "" {
		t.Fatalf("grupo de teste deveria liberar, veio %q", r)
	}
	if r := sendBlockedReason("5511942749700", false); r == "" {
		t.Fatal("terceiro envio no dia deveria barrar pelo teto")
	}
	sendDay, sendCount = "", 0
	if r := sendBlockedReason("5531999990000", false); r == "" {
		t.Fatal("destino fora da allowlist deveria barrar")
	}
	if r := sendBlockedReason("5511942749700", true); r == "" {
		t.Fatal("arquivo com WHATSAPP_SEND_TEXT_ONLY=1 deveria barrar")
	}
	t.Setenv("WHATSAPP_READ_ONLY", "1")
	if r := sendBlockedReason("5511942749700", false); r == "" {
		t.Fatal("WHATSAPP_READ_ONLY=1 deveria barrar tudo")
	}
}
