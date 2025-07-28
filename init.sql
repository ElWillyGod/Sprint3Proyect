CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price NUMERIC NOT NULL,
    stock INTEGER NOT NULL CHECK (stock >= 0)
);
INSERT INTO products (name, category, price, stock) VALUES
('Laptop Lenovo', 'Electrónica', 800, 10),
('Batería Externa', 'Accesorios', 40, 25),
('Teclado Mecánico', 'Accesorios', 120, 15),
('Monitor 24"', 'Electrónica', 200, 8),
('Auriculares Bluetooth', 'Accesorios', 60, 20),
('Smartphone Galaxy', 'Electrónica', 650, 12),
('Mouse Inalámbrico', 'Accesorios', 35, 30),
('Impresora Multifunción', 'Electrónica', 150, 6),
('Tablet 10"', 'Electrónica', 300, 9),
('Cargador USB-C', 'Accesorios', 25, 40),
('Cámara Web HD', 'Electrónica', 75, 18),
('Altavoz Portátil', 'Accesorios', 55, 22),
('Router WiFi', 'Electrónica', 90, 14),
('Funda para Laptop', 'Accesorios', 20, 35),
('Disco Duro Externo', 'Electrónica', 110, 11),
('Cable HDMI', 'Accesorios', 15, 50),
('Smartwatch', 'Electrónica', 180, 7),
('Soporte para Monitor', 'Accesorios', 30, 28),
('Tarjeta SD 64GB', 'Accesorios', 22, 45),
('Proyector LED', 'Electrónica', 250, 5),
('Hub USB', 'Accesorios', 32, 33),
('Micrófono USB', 'Electrónica', 85, 16),
('Cinta LED RGB', 'Accesorios', 27, 38),
('Adaptador Bluetooth', 'Accesorios', 18, 42),
('SSD 512GB', 'Electrónica', 120, 10),
('Mousepad XXL', 'Accesorios', 14, 48),
('Cámara de Seguridad', 'Electrónica', 95, 13),
('Cargador Inalámbrico', 'Accesorios', 36, 29),
('Tablet Gráfica', 'Electrónica', 210, 6),
('Auriculares Gamer', 'Accesorios', 70, 21),
('Switch HDMI', 'Accesorios', 24, 37),
('Monitor 27"', 'Electrónica', 320, 4),
('Teclado Bluetooth', 'Accesorios', 45, 26),
('Impresora Láser', 'Electrónica', 190, 8),

-- Productos adicionales de Electrónica
('MacBook Pro M3', 'Electrónica', 2500, 3),
('iPhone 15 Pro', 'Electrónica', 1200, 8),
('Samsung OLED 55"', 'Electrónica', 1800, 2),
('PlayStation 5', 'Electrónica', 500, 5),
('Xbox Series X', 'Electrónica', 450, 6),
('Nintendo Switch OLED', 'Electrónica', 350, 12),
('AirPods Pro 2', 'Electrónica', 250, 15),
('iPad Air M2', 'Electrónica', 700, 7),
('Apple Watch Ultra', 'Electrónica', 800, 4),
('Surface Pro 9', 'Electrónica', 1100, 6),
('Steam Deck', 'Electrónica', 400, 9),
('RTX 4080 GPU', 'Electrónica', 1200, 3),
('AMD Ryzen 9 7900X', 'Electrónica', 550, 8),
('SSD NVMe 2TB', 'Electrónica', 200, 12),
('Monitor Gaming 144Hz', 'Electrónica', 380, 10),
('Webcam 4K Logitech', 'Electrónica', 150, 18),
('Drone DJI Mini 3', 'Electrónica', 760, 5),
('GoPro Hero 12', 'Electrónica', 400, 11),
('Kindle Paperwhite', 'Electrónica', 140, 20),
('Echo Dot 5ta Gen', 'Electrónica', 50, 25),

-- Más Accesorios
('Cable Lightning', 'Accesorios', 20, 60),
('Soporte Laptop Ajustable', 'Accesorios', 35, 40),
('Hub USB-C 7 en 1', 'Accesorios', 65, 25),
('Mousepad RGB', 'Accesorios', 28, 35),
('Funda iPhone 15', 'Accesorios', 25, 50),
('Protector Pantalla', 'Accesorios', 12, 100),
('Cable Ethernet Cat6', 'Accesorios', 15, 80),
('Adaptador USB-C a HDMI', 'Accesorios', 22, 45),
('Soporte Monitor Dual', 'Accesorios', 75, 20),
('Cargador Rápido 65W', 'Accesorios', 40, 30),
('Bandeja Enfriadora Laptop', 'Accesorios', 32, 28),
('Organizador Cables', 'Accesorios', 18, 55),
('Limpiador Pantallas', 'Accesorios', 8, 75),
('Alfombrilla Antideslizante', 'Accesorios', 12, 90),
('Atril para Tablet', 'Accesorios', 20, 40),

-- Nueva categoría: Gaming
('Silla Gaming RGB', 'Gaming', 280, 8),
('Teclado Mecánico RGB', 'Gaming', 150, 15),
('Mouse Gaming 16000 DPI', 'Gaming', 80, 22),
('Auriculares Gaming 7.1', 'Gaming', 120, 18),
('Micrófono Streaming', 'Gaming', 200, 12),
('Capturadora 4K', 'Gaming', 180, 10),
('Joystick Xbox Wireless', 'Gaming', 60, 25),
('Volante Racing Logitech', 'Gaming', 350, 6),
('Alfombrilla Gaming XXL', 'Gaming', 45, 30),
('Luz LED Gaming Strip', 'Gaming', 35, 40),

-- Nueva categoría: Oficina
('Escritorio Ejecutivo', 'Oficina', 450, 5),
('Silla Ergonómica', 'Oficina', 320, 8),
('Lámpara LED Escritorio', 'Oficina', 65, 20),
('Papelera Smart', 'Oficina', 85, 12),
('Pizarra Digital', 'Oficina', 380, 4),
('Destructora Documentos', 'Oficina', 120, 10),
('Cafetera Oficina', 'Oficina', 180, 7),
('Reloj Pared Digital', 'Oficina', 45, 25),
('Calendario Inteligente', 'Oficina', 90, 15),
('Dispensador Agua', 'Oficina', 200, 6),

-- Nueva categoría: Hogar
('Aspiradora Robot', 'Hogar', 350, 8),
('Purificador Aire', 'Hogar', 280, 12),
('Humidificador Smart', 'Hogar', 120, 15),
('Bombillas LED Smart', 'Hogar', 25, 60),
('Enchufe Inteligente', 'Hogar', 18, 80),
('Cámara Seguridad WiFi', 'Hogar', 95, 20),
('Termostato Smart', 'Hogar', 180, 10),
('Altavoz Ceiling', 'Hogar', 150, 12),
('Detector Humo Smart', 'Hogar', 75, 25),
('Cerradura Digital', 'Hogar', 220, 8);

-- Índice compuesto: category + price
CREATE INDEX idx_category_price ON products(category, price);

-- Índices para búsquedas de texto
CREATE INDEX idx_name_text ON products USING gin(to_tsvector('spanish', name));
CREATE INDEX idx_name_trigram ON products USING gin(name gin_trgm_ops);

-- Habilitar extensión para trigrams (mejora ILIKE)
CREATE EXTENSION IF NOT EXISTS pg_trgm;
