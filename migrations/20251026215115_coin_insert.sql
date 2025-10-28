-- +goose Up
-- +goose StatementBegin
INSERT INTO coin (ticker, name, precision) 
VALUES 
('BTC', 'Bitcoin', 8),
('SOL', 'Solana', 8),
('ETH', 'Ethereum', 8),
('WLD', 'WorldCoin', 8),
('OP', 'Optimism', 8),
('ONT', 'Ontology', 8),
('STX', 'Stacks', 8),
('FLOW', 'Flow', 8),
('YFI', 'Yearn.Finance', 8),
('RVN', 'Ravencoin', 8),
('1INCH', '1inch', 8),
('NANO', 'Nano', 8),
('LSK', 'Lisk', 8),
('ZEN', 'Horizen', 8),
('NEXO', 'Nexo', 8),
('AR', 'Arweave', 8)
ON CONFLICT (ticker) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
