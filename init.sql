CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price NUMERIC NOT NULL
);

INSERT INTO products (name, category, price) VALUES
('Laptop Lenovo', 'Electronics', 800),
('Batería Externa', 'Accessories', 40),
('Teclado Mecánico', 'Accessories', 120),
('Monitor 24"', 'Electronics', 200),
('Auriculares Bluetooth', 'Accessories', 60);

-- Índice compuesto: category + price
CREATE INDEX idx_category_price ON products(category, price);

-- Índices para búsquedas de texto
CREATE INDEX idx_name_text ON products USING gin(to_tsvector('spanish', name));
CREATE INDEX idx_name_trigram ON products USING gin(name gin_trgm_ops);

-- Habilitar extensión para trigrams (mejora ILIKE)
CREATE EXTENSION IF NOT EXISTS pg_trgm;
