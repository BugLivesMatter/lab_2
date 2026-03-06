-- Удаляем индексы (сначала индексы, потом таблицу)
DROP INDEX IF EXISTS idx_categories_name;
DROP INDEX IF EXISTS idx_categories_status;
DROP INDEX IF EXISTS idx_categories_deleted_at;

-- Удаляем таблицу категорий
DROP TABLE IF EXISTS categories;

-- Примечание: расширение uuid-ossp не удаляем, 
-- так как оно может использоваться другими таблицами