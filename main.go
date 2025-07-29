package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/lib/pq"
)

type Product struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
	Stock    int     `json:"stock"`
}

var (
	db  *sql.DB
	rdb *redis.Client
	// L1: Cache de búsquedas (arrays de productos)
	l1Cache *lru.Cache[string, []Product]
	ctx     = context.Background()
)

func main() {
	var err error

	// Detectar si estamos en Docker o desarrollo local
	dbHost := getEnv("DB_HOST", "localhost")
	valkeyHost := getEnv("REDIS_HOST", "localhost")

	db, err = sql.Open("postgres", fmt.Sprintf("postgres://myuser:mypass@%s:5432/productsdb?sslmode=disable", dbHost))
	if err != nil {
		log.Fatal(err)
	}

	// Verificar conexión DB
	if err = db.Ping(); err != nil {
		log.Fatal("No se pudo conectar a PostgreSQL:", err)
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:6379", valkeyHost),
	})

	// Verificar conexión Valkey
	if _, err = rdb.Ping(ctx).Result(); err != nil {
		log.Fatal("No se pudo conectar a Valkey:", err)
	}

	l1Cache, _ = lru.New[string, []Product](128) // Búsquedas

	// Iniciar listener de cambios de productos en background
	go startProductUpdateListener()

	// Endpoints
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/product", productHandler)
	http.HandleFunc("/reduce-stock", reduceStockHandler)
	http.HandleFunc("/recent", recentProductsHandler) // Productos en Valkey

	instanceID := getEnv("INSTANCE_ID", "app-local")
	port := getEnv("PORT", "8080")

	log.Printf("🚀 Servidor %s iniciado en puerto %s", instanceID, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	cat := r.URL.Query().Get("category")
	maxPriceStr := r.URL.Query().Get("max_price")

	var maxPrice float64
	if maxPriceStr != "" {
		mp, err := strconv.ParseFloat(maxPriceStr, 64)
		if err != nil {
			http.Error(w, "max_price inválido", 400)
			return
		}
		maxPrice = mp
	}

	// L1 Cache: Para búsquedas
	cacheKey := fmt.Sprintf("q=%s|cat=%s|max=%.2f", q, cat, maxPrice)

	// L1 Cache check (búsquedas)
	if res, ok := l1Cache.Get(cacheKey); ok {
		log.Printf("🟢 L1 CACHE HIT - Búsqueda: %s", cacheKey)
		json.NewEncoder(w).Encode(res)
		return
	}

	log.Printf("🔴 L1 CACHE MISS - Búsqueda: %s", cacheKey)
	// Si no está en L1, hacer query a BD
	results := executeSearchQuery(q, cat, maxPrice)

	// Guardar búsqueda en L1 Cache
	l1Cache.Add(cacheKey, results)
	log.Printf("💾 L1 CACHE STORED - Búsqueda: %s", cacheKey)

	json.NewEncoder(w).Encode(results)
}

// Endpoint para productos individuales (solo Valkey)
func productHandler(w http.ResponseWriter, r *http.Request) {
	productIDStr := r.URL.Query().Get("id")
	if productIDStr == "" {
		http.Error(w, "ID de producto requerido", 400)
		return
	}

	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		http.Error(w, "ID de producto inválido", 400)
		return
	}

	// Valkey check para productos individuales
	valkeyKey := fmt.Sprintf("product:%d", productID)
	if val, err := rdb.Get(ctx, valkeyKey).Result(); err == nil {
		var product Product
		if err := json.Unmarshal([]byte(val), &product); err == nil {
			log.Printf("🟡 VALKEY HIT - Producto ID: %d", productID)
			json.NewEncoder(w).Encode(product)
			return
		}
	}

	log.Printf("🔴 VALKEY MISS - Producto ID: %d", productID)

	// Query individual a BD
	log.Printf("🔵 CONSULTANDO BD - Producto ID: %d", productID)
	product, err := getProductByID(productID)
	if err != nil {
		log.Printf("❌ BD ERROR - Producto ID: %d, Error: %v", productID, err)
		http.Error(w, "Producto no encontrado", 404)
		return
	}

	log.Printf("✅ BD SUCCESS - Producto ID: %d obtenido", productID)

	// Guardar en Valkey con TTL largo
	if data, err := json.Marshal(product); err == nil {
		rdb.Set(ctx, valkeyKey, data, 30*time.Minute)
		log.Printf("💾 VALKEY STORED - Producto ID: %d", productID)
	}

	json.NewEncoder(w).Encode(product)
}

// Función para ejecutar búsquedas en BD
func executeSearchQuery(q, cat string, maxPrice float64) []Product {
	// ESTRATEGIA OPTIMIZADA según índices disponibles:
	// idx_category_price (category, price) - índice compuesto
	// idx_name_text (name) - índice de texto
	// idx_name_trigram (name) - índice trigram para ILIKE

	var query string
	var args []any

	// Elegir la mejor estrategia según los parámetros
	if cat != "" && maxPrice > 0 {
		// CASO ÓPTIMO: Usar índice compuesto category + price
		query = `SELECT id, name, category, price, stock FROM products 
				WHERE category = $1 AND price <= $2`
		args = []any{cat, maxPrice}

		if q != "" {
			query += ` AND name ILIKE $3`
			if len(q) >= 3 {
				args = append(args, "%"+q+"%")
			} else {
				args = append(args, q+"%")
			}
		}

	} else if cat != "" {
		// Usar índice de categoría
		query = `SELECT id, name, category, price, stock FROM products 
				WHERE category = $1`
		args = []any{cat}

		if q != "" {
			query += ` AND name ILIKE $2`
			if len(q) >= 3 {
				args = append(args, "%"+q+"%")
			} else {
				args = append(args, q+"%")
			}
		}

		if maxPrice > 0 {
			query += fmt.Sprintf(` AND price <= $%d`, len(args)+1)
			args = append(args, maxPrice)
		}

	} else if q != "" {
		if len(q) >= 3 && containsSpaces(q) {
			// Búsqueda de TEXTO COMPLETO para frases (usa índice GIN)
			query = `SELECT id, name, category, price, stock FROM products 
					WHERE to_tsvector('spanish', name) @@ to_tsquery('spanish', $1)`
			// Convertir consulta para texto completo (espacios → &)
			searchTerm := prepareFullTextQuery(q)
			args = []any{searchTerm}
		} else {
			// Búsqueda ILIKE para términos cortos/parciales (usa trigrama)
			query = `SELECT id, name, category, price, stock FROM products 
					WHERE name ILIKE $1`
			if len(q) >= 3 {
				args = []any{"%" + q + "%"}
			} else {
				args = []any{q + "%"}
			}
		}

		if maxPrice > 0 {
			query += fmt.Sprintf(` AND price <= $%d`, len(args)+1)
			args = append(args, maxPrice)
		}

	} else if maxPrice > 0 {
		query = `SELECT id, name, category, price, stock FROM products 
				WHERE price <= $1`
		args = []any{maxPrice}

	} else {
		query = `SELECT id, name, category, price, stock FROM products 
				LIMIT 50`
		args = []any{}
	}

	log.Printf("🔵 CONSULTANDO BD - Búsqueda con query: %s, args: %v", strings.ReplaceAll(query, "\n", " "), args)
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("❌ BD ERROR en búsqueda: %v", err)
		return []Product{}
	}
	defer rows.Close()

	var results []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Price, &p.Stock); err != nil {
			log.Printf("❌ Error scanning product: %v", err)
			continue
		}
		results = append(results, p)
	}

	log.Printf("✅ BD SUCCESS - Búsqueda retornó %d productos", len(results))
	return results
}

// Función para obtener un producto por ID
func getProductByID(productID int) (Product, error) {
	var product Product
	query := `SELECT id, name, category, price, stock FROM products WHERE id = $1`

	err := db.QueryRow(query, productID).Scan(
		&product.ID, &product.Name, &product.Category, &product.Price, &product.Stock)

	if err != nil {
		return Product{}, err
	}

	return product, nil
}

// Endpoint para mostrar todos los productos en Valkey
func recentProductsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	valkeyKeys, err := rdb.Keys(ctx, "product:*").Result()
	if err != nil {
		http.Error(w, "Error obteniendo productos recientes", 500)
		return
	}

	products := []Product{}
	for _, valkeyKey := range valkeyKeys {
		if val, err := rdb.Get(ctx, valkeyKey).Result(); err == nil {
			var product Product
			if err := json.Unmarshal([]byte(val), &product); err == nil {
				products = append(products, product)
			}
		}
	}

	// Información del cache
	response := map[string]interface{}{
		"valkey_keys":    len(valkeyKeys),
		"products":       products,
		"total_products": len(products),
		"message":        "Productos almacenados en Valkey (visitados recientemente)",
		"note":           "Cache distribuido - compartido entre todas las instancias",
	}

	json.NewEncoder(w).Encode(response)
}

// Endpoint para reducir stock en 1 unidad
func reduceStockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	productIDStr := r.URL.Query().Get("id")
	if productIDStr == "" {
		http.Error(w, "ID de producto requerido", 400)
		return
	}

	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		http.Error(w, "ID de producto inválido", 400)
		return
	}

	log.Printf("🛒 INICIANDO reducción de stock para producto ID: %d", productID)

	// Reducir stock en la BD (el trigger se encargará del cache)
	log.Printf("🔵 EJECUTANDO UPDATE en BD para producto ID: %d", productID)
	result, err := db.Exec(`
		UPDATE products 
		SET stock = stock - 1 
		WHERE id = $1 AND stock > 0
	`, productID)

	if err != nil {
		log.Printf("❌ Error reduciendo stock para producto %d: %v", productID, err)
		http.Error(w, "Error interno", 500)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("📊 UPDATE ejecutado - Filas afectadas: %d", rowsAffected)

	if rowsAffected == 0 {
		log.Printf("⚠️  No se actualizó ninguna fila - Producto %d no encontrado o sin stock", productID)
		http.Error(w, "Producto no encontrado o sin stock", 404)
		return
	}

	log.Printf("✅ Stock reducido exitosamente para producto ID: %d", productID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "Stock reducido exitosamente",
		"product_id": productID,
	})
}

// Listener simple para actualizaciones de productos
func startProductUpdateListener() {
	log.Printf("🎧 Iniciando listener de actualizaciones de productos...")

	dbHost := getEnv("DB_HOST", "localhost")
	connStr := fmt.Sprintf("postgres://myuser:mypass@%s:5432/productsdb?sslmode=disable", dbHost)

	log.Printf("🔗 Conectando listener a: %s", connStr)

	listener := pq.NewListener(connStr, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("❌ Listener error: %v", err)
		} else {
			log.Printf("🔄 Listener event: %v", ev)
		}
	})
	defer listener.Close()

	err := listener.Listen("product_updates")
	if err != nil {
		log.Printf("❌ Error listening to product_updates: %v", err)
		return
	}

	log.Printf("✅ Listener conectado exitosamente al canal 'product_updates'")
	log.Println("🎧 Escuchando actualizaciones de productos...")

	for {
		select {
		case notification := <-listener.Notify:
			if notification != nil {
				log.Printf("📨 Notificación recibida del canal: %s", notification.Channel)
				handleProductUpdate(notification.Extra)
			} else {
				log.Printf("⚠️  Notificación nula recibida")
			}
		case <-time.After(90 * time.Second):
			log.Printf("⏰ Ping de keepalive del listener...")
			go func() {
				if err := listener.Ping(); err != nil {
					log.Printf("❌ Listener ping failed: %v", err)
				} else {
					log.Printf("✅ Listener ping successful")
				}
			}()
		}
	}
}

// Handler simple: actualizar caches con producto actualizado
func handleProductUpdate(payload string) {
	log.Printf("🔔 TRIGGER NOTIFICATION RECEIVED: %s", payload)

	var product Product
	if err := json.Unmarshal([]byte(payload), &product); err != nil {
		log.Printf("❌ Error parsing product update: %v", err)
		return
	}

	log.Printf("📦 Producto actualizado recibido: ID %d, Name: %s, Stock: %d", product.ID, product.Name, product.Stock)

	// 1. Actualizar Valkey solo si la key ya existe (producto previamente cacheado)
	valkeyKey := fmt.Sprintf("product:%d", product.ID)
	exists, err := rdb.Exists(ctx, valkeyKey).Result()
	if err != nil {
		log.Printf("❌ Error checking if key exists in Valkey: %v", err)
	} else if exists > 0 {
		// La key existe, actualizarla
		if data, err := json.Marshal(product); err == nil {
			err = rdb.Set(ctx, valkeyKey, data, 30*time.Minute).Err()
			if err != nil {
				log.Printf("❌ Error updating Valkey: %v", err)
			} else {
				log.Printf("💾 VALKEY UPDATED via trigger - Producto ID: %d (key existía)", product.ID)
			}
		}
	} else {
		log.Printf("⏭️  VALKEY SKIP - Producto ID: %d no estaba cacheado previamente", product.ID)
	}

	// 2. Invalidar L1 Cache (búsquedas)
	l1Cache.Purge()
	log.Printf("🧹 L1 CACHE PURGED - Todas las búsquedas invalidadas")
}

// Helper para variables de entorno
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Funciones auxiliares para búsqueda de texto completo
func containsSpaces(s string) bool {
	return strings.Contains(s, " ")
}

func prepareFullTextQuery(query string) string {
	// Convertir "mouse gaming" → "mouse & gaming" para texto completo
	// Limpiar y preparar términos para tsquery
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return query
	}

	// Unir con & para búsqueda AND de todos los términos
	return strings.Join(terms, " & ")
}
