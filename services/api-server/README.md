
If you have done all the installation you just need this cmd
```[bash]
air
```

### One time setup

1. Install go and setup GOPATH
2. Install `air` package from [here](https://github.com/cosmtrek/air?tab=readme-ov-file#via-go-install-recommended) for live server reload
3. Setup `air` as terminal alias for `$(go env GOPATH)/bin/air`


---- 
deploying after running 

> UPDATE market_tables
> SET is_active = false
> WHERE symbol NOT IN ('AAPL-USD', 'GOOGL_OSTRICH-USD', 'XYZ100-USD', 'TSLA-USD', 'NVDA-USD');

to invalidate cache properly for funding rate and markets