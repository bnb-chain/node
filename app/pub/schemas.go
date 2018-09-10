package pub

const (
	tradesAndOrdersSchema = `
		{
			"type": "record",
			"name": "TradesAndOrders",
			"namespace": "org.binance.dex.model.avro",
			"fields": [
				{ "name": "height", "type": "long" },
				{ "name": "timestamp", "type": "long" },
				{ "name": "numOfMsgs", "type": "int" },
				{ "name": "trades", "type": ["null", {
					"type": "record",
					"name": "Trades",
					"namespace": "org.binance.dex.model.avro",
					"fields": [
						{ "name": "numOfMsgs", "type": "int" },
						{ "name": "trades", "type": {
							"type": "array",
							"items":
								{
									"type": "record",
									"name": "Trade",
									"namespace": "org.binance.dex.model.avro",
									"fields": [
										{ "name": "symbol", "type": "string" },
										{ "name": "id", "type": "string" },
										{ "name": "price", "type": "long" },
										{ "name": "qty", "type": "long"	},
										{ "name": "sid", "type": "string" },
										{ "name": "bid", "type": "string" },
										{ "name": "sfee", "type": "long" },
										{ "name": "bfee", "type": "long" },
										{ "name": "sfeeAsset", "type": "string" },
										{ "name": "bfeeAsset", "type": "string" }
									]
								}
							}
						}
					]
				}], "default": null },
				{ "name": "orders", "type": ["null", {
					"type": "record",
					"name": "Orders",
					"namespace": "org.binance.dex.model.avro",
					"fields": [
						{ "name": "numOfMsgs", "type": "int" },
						{ "name": "orders", "type": {
							"type": "array",
							"items":
							{
								"type": "record",
								"name": "Order",
								"namespace": "org.binance.dex.model.avro",
								"fields": [
									{ "name": "symbol", "type": "string" },
									{ "name": "status", "type": "string" },
									{ "name": "orderId", "type": "string" },
									{ "name": "tradeId", "type": "string" },
									{ "name": "owner", "type": "string" },
									{ "name": "side", "type": "string" },
									{ "name": "orderType", "type": "string" },
									{ "name": "price", "type": "long" },
									{ "name": "qty", "type": "long" },
									{ "name": "lastExecutedPrice", "type": "long" },
									{ "name": "lastExecutedQty", "type": "long" },
									{ "name": "cumQty", "type": "long" },
									{ "name": "cumQuoteAssetQty", "type": "long" },
									{ "name": "fee", "type": "long" }, 
									{ "name": "feeAsset", "type": "string" },
									{ "name": "orderCreationTime", "type": "long" },
									{ "name": "transactionTime", "type": "long" },
									{ "name": "timeInForce", "type": "string" },
									{ "name": "currentExecutionType", "type": "string" },
									{ "name": "txHash", "type": "string" }
								]
							}
						   }
						}
					]
				}], "default": null }
			]
		}
	`

	booksSchema = `
		{
			"type": "record",
			"name": "Books",
			"namespace": "com.company",
			"fields": [
				{ "name": "height", "type": "long" },
				{ "name": "timestamp", "type": "long" },
				{ "name": "numOfMsgs", "type": "int" },
				{ "name": "books", "type": {
					"type": "array",
					"items":
						{
							"type": "record",
							"name": "OrderBookDelta",
							"namespace": "com.company",
							"fields": [
								{ "name": "symbol", "type": "string" },
								{ "name": "buys", "type": {
									"type": "array",
									"items": {
										"type": "record",
										"name": "PriceLevel",
										"namespace": "com.company",
										"fields": [
											{ "name": "price", "type": "long" },
											{ "name": "lastQty", "type": "long" }
										]
									}
								} },
								{ "name": "sells", "type": {
									"type": "array",
									"items": "com.company.PriceLevel"
								} }
							]
						}
					}, "default": []
				}
			]
		}
	`

	accountSchema = `
		{
			"type": "record",
			"name": "Accounts",
			"namespace": "com.company",
			"fields": [
				{ "name": "height", "type": "long" },
				{ "name": "numOfMsgs", "type": "int" },
				{ "name": "accounts", "type": {
					"type": "array",
					"items":
						{
							"type": "record",
							"name": "Account",
							"namespace": "com.company",
							"fields": [
								{ "name": "owner", "type": "string" },
								{ "name": "balances", "type": {
										"type": "array",
										"items": {
											"type": "record",
											"name": "AssetBalance",
											"namespace": "com.company",
											"fields": [
												{ "name": "asset", "type": "string" },
												{ "name": "free", "type": "long" },
												{ "name": "frozen", "type": "long" },
												{ "name": "locked", "type": "long" }
											]
										}
									}
								}
							]
						}
				   }, "default": []
				}
			]
		}
	`
)
