// backup_handler.go — Handlers HTTP para exportar e importar datos
package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"mtg-tracker/internal/db"
)

// AdminExportarJSON genera el JSON de backup y lo envía como descarga
func (a *AppHandlers) AdminExportarJSON(w http.ResponseWriter, r *http.Request) {
	datos, err := db.ExportarBackup(a.DB)
	if err != nil {
		log.Println("Error al exportar backup:", err)
		http.Error(w, "Error al generar el backup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Nombre de archivo con la fecha actual
	nombreArchivo := fmt.Sprintf("mtg-backup-%s.json", time.Now().Format("2006-01-02"))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, nombreArchivo))
	// No establecer Content-Length manualmente: Go lo calcula correctamente
	w.Write(datos)
}

// AdminImportarJSONForm muestra el formulario de importación
func (a *AppHandlers) AdminImportarJSONForm(w http.ResponseWriter, r *http.Request) {
	a.renderizar(w, "importar.html", map[string]interface{}{})
}

// AdminImportarJSON procesa el archivo JSON subido e importa los datos
func (a *AppHandlers) AdminImportarJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	// Limitar tamaño del archivo a 10MB (más que suficiente para cualquier backup)
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		a.renderizar(w, "importar.html", map[string]interface{}{
			"Error": "Error al procesar el archivo. ¿Pesa más de 10MB?",
		})
		return
	}

	archivo, _, err := r.FormFile("backup_file")
	if err != nil {
		a.renderizar(w, "importar.html", map[string]interface{}{
			"Error": "No se recibió ningún archivo.",
		})
		return
	}
	defer archivo.Close()

	datos, err := io.ReadAll(archivo)
	if err != nil {
		a.renderizar(w, "importar.html", map[string]interface{}{
			"Error": "Error al leer el archivo.",
		})
		return
	}

	resultado, err := db.ImportarBackup(a.DB, datos)
	if err != nil {
		log.Println("Error al importar backup:", err)
		a.renderizar(w, "importar.html", map[string]interface{}{
			"Error": "Error al importar: " + err.Error(),
		})
		return
	}

	a.renderizar(w, "importar.html", map[string]interface{}{
		"Resultado": resultado,
	})
}
