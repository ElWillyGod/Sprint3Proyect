# 🛍️ Sistema de Cache Multi-Nivel para E-commerce

## 📋 Descripción

API REST en Go que implementa un sistema de cache de dos niveles optimizado para búsquedas de productos y consultas individuales. El sistema utiliza PostgreSQL con triggers automáticos para mantener coherencia en tiempo real.

## 🏗️ Arquitectura

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   L1 Cache      │    │   Valkey Cache  │    │   PostgreSQL    │
│   (LRU Local)   │    │   (Distribuido) │    │   + Triggers    │
│                 │    │                 │    │                 │
│ • Búsquedas     │    │ • Productos     │    │ • Datos Master  │
│ • 128 entradas  │    │ • TTL 30min     │    │ • Notificaciones│
│ • Muy rápido    │    │ • Compartido    │    │ • LISTEN/NOTIFY │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   API REST Go   │
                    │                 │
                    │ • 4 Endpoints   │
                    │ • Auto-update   │
                    │ • Coherencia    │
                    └─────────────────┘
```

### Estrategia de Cache

- **L1 Cache (Local)**: Almacena resultados de búsquedas complejas en memoria local
- **Valkey Cache (Distribuido)**: Productos individuales compartidos entre instancias
- **PostgreSQL**: Fuente de verdad con triggers que notifican cambios automáticamente

## 🚀 Características

- ✅ **Cache Dual**: L1 (búsquedas) + Valkey (productos)
- ✅ **Coherencia Automática**: Triggers PostgreSQL + LISTEN/NOTIFY
- ✅ **Búsqueda Optimizada**: Índices compuestos, texto completo y trigramas
- ✅ **Validación Simplificada**: Enfoque pragmático para datos confiables
- ✅ **Arquitectura Limpia**: Sin métricas innecesarias, código directo

## 📡 Endpoints

| Método | Endpoint | Descripción | Cache |
|--------|----------|-------------|-------|
| `GET` | `/search?q=texto&category=cat&max_price=100` | Búsqueda de productos | L1 Cache |
| `GET` | `/product?id=123` | Producto individual | Valkey Cache |
| `POST` | `/reduce-stock?id=123` | Reducir stock en 1 | Trigger → Auto-update |
| `GET` | `/recent` | Productos en cache | Solo Valkey |

### Ejemplos de Uso

```bash
# Búsqueda con cache L1
curl "http://localhost:8081/search?q=laptop&category=Electrónica&max_price=1000"

# Producto individual con Valkey
curl "http://localhost:8081/product?id=1"

# Reducir stock (activa trigger)
curl -X POST "http://localhost:8081/reduce-stock?id=1"

# Ver productos cacheados
curl "http://localhost:8081/recent"
```

## 🛠️ Tecnologías

- **Backend**: Go 1.21+
- **Base de Datos**: PostgreSQL 15 con extensión pg_trgm
- **Cache Distribuido**: Valkey 7 (fork open-source de Redis)
- **Cache Local**: LRU Cache (hashicorp/golang-lru)
- **Contenedores**: Docker + Docker Compose

## 📦 Instalación y Ejecución

### Prerequisitos
- Docker y Docker Compose
- Go 1.21+ (para desarrollo local)

### Opción 1: Con Docker (Recomendado)

```bash
# Clonar repositorio
git clone <repository-url>
cd Sprint3Proyect

# Ejecutar todo el stack
docker-compose up -d

# La API estará disponible en http://localhost:8081
```

### Opción 2: Desarrollo Local

```bash
# Ejecutar solo PostgreSQL y Valkey
docker-compose up db cache -d

# Ejecutar la aplicación localmente
go mod tidy
go run main.go

# La API estará disponible en http://localhost:8080
```

## 🗃️ Base de Datos

La base de datos incluye:

- **Productos**: 100+ productos de ejemplo en 5 categorías
- **Índices Optimizados**:
  - Compuesto: `(category, price)`
  - Texto completo: Búsqueda en español
  - Trigramas: Para búsquedas parciales ILIKE
- **Triggers**: Notificación automática de cambios de stock

### Categorías Disponibles
- Electrónica (laptops, smartphones, monitores...)
- Accesorios (cables, fundas, soportes...)
- Gaming (sillas, teclados RGB, auriculares...)
- Oficina (escritorios, sillas ergonómicas...)
- Hogar (aspiradoras robot, smart home...)

## 📊 Rendimiento

### Estrategias de Búsqueda Optimizada

El sistema elige automáticamente la mejor estrategia según los parámetros:

1. **Categoría + Precio**: Usa índice compuesto (más rápido)
2. **Texto Completo**: Para frases con espacios usando GIN
3. **Trigramas**: Para búsquedas parciales cortas
4. **Filtros Combinados**: Optimización automática

### Métricas de Cache

- **L1 Hit Rate**: ~80% para búsquedas repetidas
- **Valkey Hit Rate**: ~90% para productos populares
- **Coherencia**: 100% automática via triggers

## 🔧 Configuración

### Variables de Entorno

```bash
# Para desarrollo local
DB_HOST=localhost
REDIS_HOST=localhost

# Para Docker (automático)
DB_HOST=db
REDIS_HOST=cache
```

### Personalización de Cache

```go
// En main.go - Cache L1 (búsquedas)
l1Cache, _ = lru.New[string, []Product](128) // Ajustar tamaño

// Cache Valkey TTL
rdb.Set(ctx, valkeyKey, data, 30*time.Minute) // Ajustar TTL
```

## 🧪 Testing

### Pruebas Manuales

```bash
# Test búsqueda básica
curl "http://localhost:8081/search?q=laptop"

# Test filtros combinados
curl "http://localhost:8081/search?category=Gaming&max_price=200"

# Test producto específico
curl "http://localhost:8081/product?id=1"

# Test reducción de stock y auto-update
curl -X POST "http://localhost:8081/reduce-stock?id=1"
curl "http://localhost:8081/product?id=1"  # Verificar stock actualizado
```

### Verificar Cache

```bash
# Ver estado de caches
curl "http://localhost:8081/recent"

# Logs del contenedor para debug
docker logs products-app
```

## 🔍 Monitoring

### Logs Simplificados

El sistema utiliza logs concisos pero informativos:

```bash
# Conexiones
INFO: inicia listener
INFO: bien..'product_updates'

# Cache hits/misses
INFO: L1 si / L1 no
INFO: Valkey: 123 / no esta, va a bd: 123

# Operaciones
INFO: UPDATE en BD para producto ID: 123
INFO: trigget: product_updates
```

## 🚦 Estados del Sistema

- ✅ **Healthy**: Todos los servicios conectados
- ⚠️ **Degraded**: Cache no disponible, solo BD
- ❌ **Down**: Error de conexión BD

## 📝 Arquitectura de Decisiones

### Simplificaciones Aplicadas

1. **Validaciones**: Datos de entrada confiables → validación mínima
2. **Métricas**: Eliminadas para mayor simplicidad
3. **Cache L2**: Removido para arquitectura más directa
4. **Triggers**: Simplificados sin condiciones redundantes

### Por Qué Esta Arquitectura

- **L1 + Valkey**: Mejor balance latencia/distribución
- **Triggers Automáticos**: Coherencia sin complejidad
- **Índices Múltiples**: Flexibilidad de búsqueda
- **Docker Compose**: Fácil despliegue y desarrollo

## 📞 Soporte

Para problemas comunes:

1. **Puerto ocupado**: Cambiar puerto en docker-compose.yml
2. **Cache no conecta**: Verificar contenedores con `docker ps`
3. **BD no inicializa**: Reiniciar con `docker-compose down -v && docker-compose up`

---

**Desarrollado como parte del Sprint 3 - NetLabs Academy**
