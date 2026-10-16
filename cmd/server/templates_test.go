package main

import (
	"os"
	"testing"
)

// Las plantillas se parsean al arrancar con template.Must: un error de sintaxis en
// cualquiera de ellas tumbaría el servidor en Render. Este test lo detecta antes.
func TestCargarTemplates(t *testing.T) {
	// cargarTemplates usa rutas relativas a la raíz del repositorio
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	tmpl := cargarTemplates()
	for _, nombre := range []string{"nav", "head", "selectorTemporada", "eloDelta",
		"inicio.html", "historial.html", "sesion.html", "estadisticas.html", "perfil_jugador.html",
		"tv.html", "changelog.html", "resumen_temporada.html", "dado.html", "vidas.html",
		"login.html", "panel.html", "resultados.html", "editar_jugador.html", "importar.html"} {
		if tmpl.Lookup(nombre) == nil {
			t.Errorf("falta la plantilla %q", nombre)
		}
	}
}
