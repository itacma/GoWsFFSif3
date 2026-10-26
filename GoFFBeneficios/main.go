package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv" // <-- AGREGAR ESTA LÍNEA

	"github.com/ffleming/api-beneficios/config"
	"github.com/ffleming/api-beneficios/handler"
	"github.com/ffleming/api-beneficios/middleware"
	"github.com/ffleming/api-beneficios/repository"
)

func main() {
	// Intentar cargar .env si existe localmente (ignora si no existe en prod)
	_ = godotenv.Load()

	// 1. Inicialización de la base de datos Oracle
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("Error crítico al conectar a Oracle DB: %v", err)
	}
	defer db.Close()

	// 2. Inyección de Dependencias (Repository Pattern)
	repo := repository.NewOraclePromotionRepository(db)
	h := handler.NewPromotionHandler(repo)

	// 3. Registro de Rutas
	http.HandleFunc("/api/health", h.HealthCheckHandler)
	http.HandleFunc("/api/login", h.LoginHandler)
	http.HandleFunc("/api/cupon-vigente", middleware.ValidarJWT(h.CuponVigenteHandler))
	http.HandleFunc("/api/promocionesVenta", middleware.ValidarJWT(h.PromocionesVentaHandler))
	http.HandleFunc("/api/cambia-activo-cupon", middleware.ValidarJWT(h.CambiaActivoCuponHandler))
	http.HandleFunc("/api/campos-campana", middleware.ValidarJWT(h.CamposCampanaHandler))

	// 4. Puerto dinámico desde variables de entorno
	port := os.Getenv("API_APP_PORT")
	if port == "" {
		port = ":54302" // Puerto por defecto si no está definido
	}

	log.Printf("[Dev-Go] Servicio Windows GoBeneficios iniciado exitosamente en el puerto %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Error fatal en el servidor HTTP: %v", err)
	}
}
