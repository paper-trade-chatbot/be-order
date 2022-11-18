
-- +migrate Up
CREATE TABLE IF NOT EXISTS `be-order`.`order`
(
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'id',
    `member_id` BIGINT UNSIGNED NOT NULL COMMENT '會員id',
    `order_status` TINYINT(4) NOT NULL COMMENT '訂單狀態 1:待處理 2:失敗 3:完成 4:取消 5:回滾',
    `transaction_type` TINYINT(4) NOT NULL COMMENT '交易類別 1:開倉 2:關倉',
    `product_type` TINYINT(4) NOT NULL COMMENT '產品類別  1:stock, 2:crypto, 3:forex, 4:futures',
    `exchange_code` VARCHAR(32) NOT NULL COMMENT '交易所代號',
    `product_code` VARCHAR(32) NOT NULL COMMENT '產品代號',
    `trade_type` TINYINT(4) NOT NULL COMMENT '買賣類別 1:買 2:賣',
    `amount` DECIMAL(19,4) NOT NULL COMMENT '交易數量',
    `unit_price` DECIMAL(19,4) NULL DEFAULT NULL COMMENT '成交單價',
    `transaction_record_id` BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '轉帳紀錄',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
    `finished_at` TIMESTAMP NULL DEFAULT NULL COMMENT '成交時間',
    `rollbacker_id` BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '回滾者id',
    `rollbacked_at` TIMESTAMP NULL DEFAULT NULL COMMENT '回滾時間',
    `remark` VARCHAR(128) NULL DEFAULT NULL COMMENT '備註',

    PRIMARY KEY (`id`),
    UNIQUE INDEX (`member_id`,`exchange_code`, `product_code`,`created_at`)
) AUTO_INCREMENT=1 CHARSET=`utf8mb4` COLLATE=`utf8mb4_general_ci` COMMENT '訂單';


-- +migrate Down
SET FOREIGN_KEY_CHECKS=0;
DROP TABLE IF EXISTS `order`;