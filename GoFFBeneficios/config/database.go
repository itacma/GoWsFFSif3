package config

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/sijms/go-ora/v2" // o el driver de Oracle que utilices
)

func ConnectDB() (*sql.DB, error) {
	// Intentar cargar el archivo .env si existe en el entorno
	_ = godotenv.Load()

	// Leer la variable de entorno
	dbURL := os.Getenv("API_APP_ORCL_DB_DATA2021")
	if dbURL == "" {
		return nil, fmt.Errorf("la variable de entorno ORACLE_DB_URL no esta definida")
	}

	// Abrir la conexión usando la URL dinámica
	db, err := sql.Open("oracle", dbURL)
	if err != nil {
		return nil, fmt.Errorf("error al abrir el driver Oracle: %w", err)
	}

	// Configuración del pool de conexiones
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Validar comunicación con la base de datos
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("fallo de comunicacion (Ping) con el servidor Oracle: %w", err)
	}

	return db, nil
}
