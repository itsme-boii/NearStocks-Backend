

---- 
deploying after running 

> UPDATE market_tables
> SET is_active = false
> WHERE symbol NOT IN ('AAPL-USD', 'GOOGL_OSTRICH-USD', 'XYZ100-USD', 'TSLA-USD', 'NVDA-USD');

to invalidate cache properly for funding rate and markets