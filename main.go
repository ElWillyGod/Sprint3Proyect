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
	"time"

	"github.com/go-redis/redis/v8"
	lru "github.com/hashicorp/golang-lru/v2"
	_ "github.com/lib/pq"
)

type Product struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
}

var (
	db  *sql.DB
	rdb *redis.Client
	// L1: Cache de búsquedas (arrays de productos)
	l1Cache *lru.Cache[string, []Product]
	// L2: Cache de productos individuales
	l2Cache *lru.Cache[int, Product]
	// Track de las keys en L2 cache para poder listarlas
	l2Keys map[int]bool
	ctx    = context.Background()
)

func main() {
	var err error

	// Detectar si estamos en Docker o desarrollo local
	dbHost := getEnv("DB_HOST", "localhost")
	redisHost := getEnv("REDIS_HOST", "localhost")

	db, err = sql.Open("postgres", fmt.Sprintf("postgres://myuser:mypass@%s:5432/productsdb?sslmode=disable", dbHost))
	if err != nil {
		log.Fatal(err)
	}

	// Verificar conexión DB
	if err = db.Ping(); err != nil {
		log.Fatal("No se pudo conectar a PostgreSQL:", err)
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:6379", redisHost),
	})

	// Verificar conexión Redis
	if _, err = rdb.Ping(ctx).Result(); err != nil {
		log.Fatal("No se pudo conectar a Redis:", err)
	}

	l1Cache, _ = lru.New[string, []Product](128) // Búsquedas
	l2Cache, _ = lru.New[int, Product](500)      // Productos individuales
	l2Keys = make(map[int]bool)                  // Track de keys en L2

	// Endpoints con middleware de métricas
	http.HandleFunc("/search", metricsMiddleware(searchHandler))
	http.HandleFunc("/product", metricsMiddleware(productHandler)) // NUEVO
	http.HandleFunc("/recent", recentProductsHandler)              // NUEVO - Productos en cache L2
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/health", healthHandler)

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
		appMetrics.IncrementCacheHit("l1")
		// Enriquecer con L2 cache si es posible
		optimizedResults := enrichWithL2Cache(res)
		json.NewEncoder(w).Encode(optimizedResults)
		return
	}

	// Si no está en L1, hacer query a BD
	results := executeSearchQuery(q, cat, maxPrice)

	// Guardar búsqueda en L1 Cache
	l1Cache.Add(cacheKey, results)

	json.NewEncoder(w).Encode(results)
}

// Nuevo endpoint para productos individuales (L2 Cache)
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

	// L2 Cache check (productos individuales)
	if product, ok := l2Cache.Get(productID); ok {
		appMetrics.IncrementCacheHit("l2")
		json.NewEncoder(w).Encode(product)
		return
	}

	// Redis check para productos individuales
	redisKey := fmt.Sprintf("product:%d", productID)
	if val, err := rdb.Get(ctx, redisKey).Result(); err == nil {
		var product Product
		if err := json.Unmarshal([]byte(val), &product); err == nil {
			// Guardar en L2 cache local
			l2Cache.Add(productID, product)
			l2Keys[productID] = true // Registrar la key
			appMetrics.IncrementCacheHit("l2")
			json.NewEncoder(w).Encode(product)
			return
		}
	}

	// Query individual a BD
	product, err := getProductByID(productID)
	if err != nil {
		http.Error(w, "Producto no encontrado", 404)
		return
	}

	// Guardar en ambos caches
	l2Cache.Add(productID, product)
	l2Keys[productID] = true // Registrar la key

	// Guardar en Redis con TTL largo (productos cambian menos frecuentemente)
	if data, err := json.Marshal(product); err == nil {
		rdb.Set(ctx, redisKey, data, 30*time.Minute)
	}

	json.NewEncoder(w).Encode(product)
}

// Función para enriquecer resultados de búsqueda con datos de L2 cache
func enrichWithL2Cache(searchResults []Product) []Product {
	enriched := make([]Product, len(searchResults))

	for i, product := range searchResults {
		// Verificar si tenemos este producto en L2 (más actualizado)
		if cachedProduct, ok := l2Cache.Get(product.ID); ok {
			enriched[i] = cachedProduct
		} else {
			enriched[i] = product
		}
	}

	return enriched
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
		query = `SELECT id, name, category, price FROM products 
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
		query += ` ORDER BY price ASC`

	} else if cat != "" {
		// Usar índice de categoría
		query = `SELECT id, name, category, price FROM products 
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
		query += ` ORDER BY price ASC`

	} else if q != "" {
		// Priorizar búsqueda de texto (usar índices de texto)
		query = `SELECT id, name, category, price FROM products 
				WHERE name ILIKE $1`
		if len(q) >= 3 {
			args = []any{"%" + q + "%"}
		} else {
			args = []any{q + "%"}
		}

		if maxPrice > 0 {
			query += ` AND price <= $2`
			args = append(args, maxPrice)
		}
		query += ` ORDER BY name ASC`

	} else if maxPrice > 0 {
		// Solo filtro de precio
		query = `SELECT id, name, category, price FROM products 
				WHERE price <= $1 ORDER BY price ASC`
		args = []any{maxPrice}

	} else {
		// Sin filtros, obtener todos (con límite)
		query = `SELECT id, name, category, price FROM products 
				ORDER BY id ASC LIMIT 50`
		args = []any{}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error ejecutando consulta: %v", err)
		return []Product{}
	}
	defer rows.Close()

	var results []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Price); err != nil {
			log.Printf("Error scanning product: %v", err)
			continue
		}
		results = append(results, p)
	}

	return results
}

// Función para obtener un producto por ID
func getProductByID(productID int) (Product, error) {
	var product Product
	query := `SELECT id, name, category, price FROM products WHERE id = $1`

	err := db.QueryRow(query, productID).Scan(
		&product.ID, &product.Name, &product.Category, &product.Price)

	if err != nil {
		return Product{}, err
	}

	return product, nil
}

// Endpoint para mostrar todos los productos en cache L2
func recentProductsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Obtener todos los productos del cache L2
	products := []Product{}

	// Iterar sobre las keys registradas y obtener los productos del cache
	for productID := range l2Keys {
		if product, ok := l2Cache.Get(productID); ok {
			products = append(products, product)
		} else {
			// Si el producto ya no está en cache, remover la key del tracking
			delete(l2Keys, productID)
		}
	}

	// Información del cache L2
	response := map[string]interface{}{
		"cache_size":     l2Cache.Len(),
		"tracked_keys":   len(l2Keys),
		"products":       products,
		"total_products": len(products),
		"message":        "Productos almacenados en cache L2 (visitados recientemente)",
	}

	json.NewEncoder(w).Encode(response)
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"instance":  getEnv("INSTANCE_ID", "app-local"),
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Helper para variables de entorno
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
