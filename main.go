package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	db      *sql.DB
	rdb     *redis.Client
	l1Cache *lru.Cache[string, []Product]
	ctx     = context.Background()
)

func main() {
	var err error

	db, err = sql.Open("postgres", "postgres://myuser:mypass@localhost:5432/productsdb?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	// Verificar conexión DB
	if err = db.Ping(); err != nil {
		log.Fatal("No se pudo conectar a PostgreSQL:", err)
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Cambiar a localhost para consistencia
	})

	// Verificar conexión Redis
	if _, err = rdb.Ping(ctx).Result(); err != nil {
		log.Fatal("No se pudo conectar a Redis:", err)
	}

	l1Cache, _ = lru.New[string, []Product](128)

	http.HandleFunc("/search", searchHandler)

	log.Println("Servidor en http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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

	cacheKey := fmt.Sprintf("q=%s|cat=%s|max=%.2f", q, cat, maxPrice)
	start := time.Now()

	// L1 Cache check
	if res, ok := l1Cache.Get(cacheKey); ok {
		log.Printf("L1 Cache HIT para key: %s (%.2fms)", cacheKey, time.Since(start).Seconds()*1000)
		json.NewEncoder(w).Encode(res)
		return
	}

	// L2 Cache (Redis) check
	if val, err := rdb.Get(ctx, cacheKey).Result(); err == nil {
		var products []Product
		json.Unmarshal([]byte(val), &products)
		l1Cache.Add(cacheKey, products)
		log.Printf("L2 Cache HIT para key: %s (%.2fms)", cacheKey, time.Since(start).Seconds()*1000)
		json.NewEncoder(w).Encode(products)
		return
	}

	query := `SELECT id, name, category, price FROM products WHERE 1=1`
	args := []any{}
	i := 1

	if q != "" {
		// Usar búsqueda más eficiente con trigrams o full-text search
		if len(q) >= 3 { // Para queries cortas, usar trigrams
			query += fmt.Sprintf(" AND name ILIKE $%d", i)
			args = append(args, "%"+q+"%")
		} else { // Para queries muy cortas, búsqueda por prefijo
			query += fmt.Sprintf(" AND name ILIKE $%d", i)
			args = append(args, q+"%")
		}
		i++
	}

	if cat != "" {
		query += fmt.Sprintf(" AND category = $%d", i)
		args = append(args, cat)
		i++
	}

	if maxPrice > 0 {
		query += fmt.Sprintf(" AND price <= $%d", i)
		args = append(args, maxPrice)
		i++
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error ejecutando consulta: %v\nQuery: %s\nArgs: %v", err, query, args)
		http.Error(w, "DB error", 500)
		return
	}

	defer rows.Close()

	var results []Product
	for rows.Next() {
		var p Product
		rows.Scan(&p.ID, &p.Name, &p.Category, &p.Price)
		results = append(results, p)
	}

	dbTime := time.Since(start)
	log.Printf("DB Query ejecutada en %.2fms para key: %s", dbTime.Seconds()*1000, cacheKey)

	data, _ := json.Marshal(results)
	rdb.Set(ctx, cacheKey, data, 10*time.Minute)
	l1Cache.Add(cacheKey, results)

	json.NewEncoder(w).Encode(results)
}
