-- name: CreateProduct :one
INSERT INTO product (uuid, name, product_code, customer_cost)
VALUES ($1, $2, $3, $4)
    RETURNING *;

-- name: GetProduct :one
SELECT * FROM product
WHERE uuid = $1 LIMIT 1;

-- name: ListProducts :many
SELECT * FROM product
ORDER BY name
limit $1 offset $2;

-- name: UpdateProduct :one
UPDATE product
SET name = $2, product_code = $3, customer_cost = $4
WHERE uuid = $1
    RETURNING *;

-- name: DeleteProduct :exec
DELETE FROM product
WHERE uuid = $1;

-- name: GetProductsByOrder :many
SELECT p.* FROM product p
                    JOIN order_products op ON p.uuid = op.product_uuid
WHERE op.order_uuid = $1;