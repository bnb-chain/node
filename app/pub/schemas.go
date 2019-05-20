package pub

const (
	executionResultSchema = `
		{
			"type": "record",
			"name": "ExecutionResults",
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
										{ "name": "sfee", "type": "string" },
										{ "name": "bfee", "type": "string" },
										{ "name": "saddr", "type": "string" },
										{ "name": "baddr", "type": "string" },
										{ "name": "ssrc", "type": "long" },
										{ "name": "bsrc", "type": "long" }
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
									{ "name": "side", "type": "int" },
									{ "name": "orderType", "type": "int" },
									{ "name": "price", "type": "long" },
									{ "name": "qty", "type": "long" },
									{ "name": "lastExecutedPrice", "type": "long" },
									{ "name": "lastExecutedQty", "type": "long" },
									{ "name": "cumQty", "type": "long" },
									{ "name": "fee", "type": "string" }, 
									{ "name": "orderCreationTime", "type": "long" },
									{ "name": "transactionTime", "type": "long" },
									{ "name": "timeInForce", "type": "int" },
									{ "name": "currentExecutionType", "type": "string" },
									{ "name": "txHash", "type": "string" }
								]
							}
						   }
						}
					]
				}], "default": null },
				{ "name": "proposals", "type": ["null", {
					"type": "record",
					"name": "Proposals",
					"namespace": "org.binance.dex.model.avro",
					"fields": [
						{ "name": "numOfMsgs", "type": "int" },
						{ "name": "proposals", "type": {
							"type": "array",
							"items":
							{
								"type": "record",
								"name": "Proposal",
								"namespace": "org.binance.dex.model.avro",
								"fields": [
									{ "name": "id", "type": "long" },
									{ "name": "status", "type": "string" }
								]
							}
						   }
						}
					]
				}], "default": null },
				{ "name": "stakeUpdates", "type": ["null", {
					"type": "record",
					"name": "StakeUpdates",
					"namespace": "org.binance.dex.model.avro",
					"fields": [
						{ "name": "numOfMsgs", "type": "int" },
						{ "name": "completedUnbondingDelegations", "type": {
							"type": "array",
							"items":
							{
								"type": "record",
								"name": "CompletedUnbondingDelegation",
								"namespace": "org.binance.dex.model.avro",
								"fields": [
									{ "name": "validator", "type": "string" },
									{ "name": "delegator", "type": "string" },
									{ "name": "amount", "type": {
											"type": "record",
											"name": "Coin",
											"namespace": "org.binance.dex.model.avro",
											"fields": [
												{ "name": "denom", "type": "string" },
												{ "name": "amount", "type": "long" }
											]
										}
									}
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
								{ "name": "fee", "type": "string" },
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

	blockfeeSchema = `
		{
			"type": "record",
			"name": "BlockFee",
			"namespace": "com.company",
			"fields": [
				{ "name": "height", "type": "long"},
				{ "name": "fee", "type": "string"},
				{ "name": "validators", "type": { "type": "array", "items": "string" }}
			]
		}
	`
	transfersSchema = `
		{
			"type": "record",
			"name": "Transfers",
			"namespace": "com.company",
			"fields": [
				{ "name": "height", "type": "long"},
				{ "name": "num", "type": "int" },
				{ "name": "timestamp", "type": "long" },
				{ "name": "transfers",
				  "type": {	
				  	"type": "array",
					"items": {
						"type": "record",
						"name": "Transfer",
						"namespace": "com.company",
						"fields": [
							{ "name": "txhash", "type": "string" },
							{ "name": "from", "type": "string" },
							{ "name": "to", 
                  				"type": {
 									"type": "array",
                    				"items": {
										"type": "record",
                       				 	"name": "Receiver",
										"namespace": "com.company",
										"fields": [
											{ "name": "addr", "type": "string" },
											{ "name": "coins",
												"type": {
													"type": "array",
                                  					"items": {
														"type": "record",
														"name": "Coin",
														"namespace": "com.company",
														"fields": [
															{ "name": "denom", "type": "string" },
															{ "name": "amount", "type": "long" }
														]
								  					}
												}	
											}
										]
									}
				  				}
							}
						]
					}
                  }	
				}
			]
		}
	`
)
