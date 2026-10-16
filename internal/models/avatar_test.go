package models

import (
	"strings"
	"testing"
)

func TestNormalizarYRenderAvatar(t *testing.T) {
	malicioso := `ms-x"><img src=x onerror=alert(1)>`
	if got := NormalizarAvatar(malicioso); got != AvatarPorDefecto {
		t.Errorf("NormalizarAvatar(malicioso) = %q", got)
	}
	if got := NormalizarAvatar("ms-ability-flying"); got != "ms-ability-flying" {
		t.Errorf("clase válida alterada: %q", got)
	}
	if got := NormalizarAvatar("🐉"); got != "🐉" {
		t.Errorf("emoji legacy alterado: %q", got)
	}
	if got := NormalizarAvatar(""); got != AvatarPorDefecto {
		t.Errorf("vacío = %q", got)
	}
	// Aunque ya estuviera guardado en la BD, al pintarlo no se inyecta nada
	if html := string(RenderAvatar(malicioso)); strings.Contains(html, "<img") || !strings.Contains(html, AvatarPorDefecto) {
		t.Errorf("RenderAvatar inyecta HTML: %s", html)
	}
	if html := string(RenderAvatar("<b>")); strings.Contains(html, "<b>") {
		t.Errorf("emoji legacy sin escapar: %s", html)
	}
}
