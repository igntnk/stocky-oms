-- name: CreateOrder :one
INSERT INTO orders (
    uuid, comment, user_id, staff_id, order_cost
) VALUES (
             $1, $2, $3, $4, $5
         )
    RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders
WHERE uuid = $1 LIMIT 1;

-- name: ListOrders :many
SELECT * FROM orders
where status = $1
ORDER BY creation_date DESC
limit $2 offset $3;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2, finish_date = CASE WHEN $2 = 'completed' THEN NOW() ELSE finish_date END
WHERE uuid = $1
    RETURNING *;

-- name: DeleteOrder :exec
DELETE FROM orders
WHERE uuid = $1;

-- name: DeleteOrderProducts :exec
DELETE FROM order_products where order_uuid = $1;

-- name: AddProductToOrder :one
INSERT INTO order_products (
    product_uuid, order_uuid, result_price, amount
) VALUES (
             (select uuid from product where product_code = $1), $2, $3, $4
         )
    RETURNING *;

-- name: RemoveProductFromOrder :exec
DELETE FROM order_products
WHERE product_uuid = $1 AND order_uuid = $2;

-- name: GetOrderProducts :many
SELECT op.*, p.name as product_name, p.product_code FROM order_products op
                                                             JOIN product p ON op.product_uuid = p.uuid
WHERE op.order_uuid = $1;


-- name: CalculateOrderTotal :one
SELECT SUM(result_price * amount) as total FROM order_products
WHERE order_uuid = $1;

-- name: UpdateOrder :one
UPDATE orders
SET
    comment = COALESCE($2, comment),
    user_id = COALESCE($3, user_id),
    staff_id = COALESCE($4, staff_id),
    order_cost = COALESCE($5, order_cost),
    status = COALESCE($6, status),
    finish_date = CASE
                      WHEN $6 = 'completed' AND status != 'completed' THEN NOW()
                      WHEN $6 != 'completed' THEN NULL
                      ELSE finish_date
        END
WHERE uuid = $1
    RETURNING *;