-- +goose Up
-- +goose StatementBegin
create extension if not exists "uuid-ossp";

CREATE TYPE order_status AS ENUM ('new', 'processing', 'completed', 'cancelled');

CREATE TABLE product (
                         uuid UUID PRIMARY KEY,
                         name VARCHAR(80) NOT NULL,
                         product_code UUID NOT NULL,
                         customer_cost DECIMAL(10, 2) NOT NULL
);

CREATE TABLE orders (
                        uuid UUID PRIMARY KEY,
                        comment TEXT,
                        user_id UUID NOT NULL,
                        staff_id UUID NOT NULL,
                        order_cost DECIMAL(10, 2) NOT NULL,
                        creation_date TIMESTAMP NOT NULL DEFAULT NOW(),
                        finish_date TIMESTAMP,
                        status order_status NOT NULL DEFAULT 'new'
);

CREATE TABLE order_products (
                                product_uuid UUID NOT NULL REFERENCES product(uuid),
                                order_uuid UUID NOT NULL REFERENCES orders(uuid),
                                result_price DECIMAL(10, 2) NOT NULL,
                                amount INTEGER NOT NULL,
                                PRIMARY KEY (product_uuid, order_uuid)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin


DROP TABLE order_products;
DROP TABLE orders;
DROP TABLE product;
DROP TYPE order_status;

-- +goose StatementEnd
