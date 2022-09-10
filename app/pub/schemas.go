package pub

// !!!!!NOTE!!!!!
// consumers should aware of changes in this file
// update app/pub/msgs.go `latestSchemaVersions`
// put old version into pub/schemas with version suffixed to filename for tracking historical version

// Backward compatibility:
// 1. publisher add field, consumer should initialize two decoder with two publisher schema, choose which decoder should be used by `lastestSchemaVersion` component in kafka message Key
// 2. consumer add field, consumer should initialize one decode with publisher schema and consumer schema. In which, consumer schema should define default value for added field

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
                                        { "name": "qty", "type": "long"    },
                                        { "name": "sid", "type": "string" },
                                        { "name": "bid", "type": "string" },
                                        { "name": "sfee", "type": "string" },
                                        { "name": "bfee", "type": "string" },
                                        { "name": "saddr", "type": "string" },
                                        { "name": "baddr", "type": "string" },
                                        { "name": "ssrc", "type": "long" },
                                        { "name": "bsrc", "type": "long" },
                                        { "name": "ssinglefee", "type": "string" },
                                        { "name": "bsinglefee", "type": "string" },
                                        { "name": "tickType", "type": "int" }
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
                                    { "name": "txHash", "type": "string" },
                                    { "name": "singlefee", "type": "string" }
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
								{"name": "sequence", "type": "long"},
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
                            { "name": "memo", "type": "string" },
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

	blockDatasSchema = `
		{
			"namespace":"com.company",
			"type":"record",
			"name":"blockData",
			"doc":"The fields in this record contain information about the data in the single block of the Binance Chain.",
			"fields":[
				{
					"name":"chainId",
					"type":"string"
				},
				{
					"name":"cryptoBlock",
					"type":{
						"type":"record",
						"name":"CryptoBlock",
						"doc":"This contains information for the given block or channel with all transactions within it.",
						"fields":[
							{
								"name":"blockHash",
								"type":"string",
								"doc":"Hash of this block"
							},
							{
								"name":"parentHash",
								"type":"string",
								"doc":"Hash of parent block"
							},
							{
								"name":"blockHeight",
								"type":"long",
								"doc":"Height of this block"
							},
							{
								"name":"timestamp",
								"type":"string",
								"doc":"Block time"
							},
							{
								"name":"txTotal",
								"type":"long",
								"default":0,
								"doc":"Overall tx count, including this block"
							},
							{
								"name":"bnbBlockMeta",
								"type":{
									"type":"record",
									"name":"bnbBlockMeta",
									"doc":"binance-specific block data",
									"fields":[
										{
											"name":"lastCommitHash",
											"type":"string",
											"default":""
										},
										{
											"name":"dataHash",
											"type":"string",
											"default":""
										},
										{
											"name":"validatorsHash",
											"type":"string",
											"default":""
										},
										{
											"name":"nextValidatorsHash",
											"type":"string",
											"default":""
										},
										{
											"name":"consensusHash",
											"type":"string",
											"default":""
										},
										{
											"name":"appHash",
											"type":"string",
											"default":""
										},
										{
											"name":"lastResultsHash",
											"type":"string",
											"default":""
										},
										{
											"name":"evidenceHash",
											"type":"string",
											"default":""
										},
										{
											"name":"proposerAddress",
											"type":"string",
											"default":""
										}
									]
								}
							},
							{
								"name":"transactions",
								"type":{
									"type":"array",
									"doc":"All transactions in the block",
									"items":{
										"name":"cryptoTx",
										"type":"record",
										"doc":"Array of fields contained within the transaction.",
										"fields":[
											{
												"name":"txHash",
												"type":"string"
											},
											{
												"name":"fee",
												"type":"string",
												"default":"",
												"doc":"Transaction fee"
											},
											{
												"name":"inputs",
												"type":{
													"type":"array",
													"doc":"Inputs into the transaction",
													"items":{
														"name":"txLineItem",
														"type":"record",
														"fields":[
															{
																"name":"address",
																"type":"string"
															},
															{
																"name":"coins",
																"type":{
																	"type":"array",
																	"items":{
																		"type":"record",
																		"name":"Coin",
																		"namespace":"com.company",
																		"fields":[
																			{
																				"name":"denom",
																				"type":"string"
																			},
																			{
																				"name":"amount",
																				"type":"long"
																			}
																		]
																	}
																}
															}
														]
													}
												}
											},
											{
												"name":"outputs",
												"type":{
													"type":"array",
													"doc":"Outputs of the transaction",
													"items":"txLineItem"
												}
											},
											{
												"name":"timestamp",
												"type":"string",
												"default":""
											},
											{
												"name":"bnbTransaction",
												"type":{
													"type":"record",
													"name":"bnbTransaction",
													"doc":"binance-specific transaction data.",
													"fields":[
														{
															"name":"source",
															"type":"long",
															"default":0
														},
														{
															"name":"txType",
															"type":"string",
															"default":"",
															"doc":"type of transaction"
														},
														{
															"name":"proposalId",
															"type":"long",
															"default":0
														},
														{
															"name":"txAsset",
															"type":"string",
															"default":""
														},
														{
															"name":"orderId",
															"type":"string",
															"default":""
														},
														{
															"name":"code",
															"type":"long",
															"default":0
														},
														{
															"name":"data",
															"type":"string",
															"doc":"Raw data of the transaction",
															"default":""
														}
													]
												}
											}
										]
									}
								}
							}
						]
					}
				}
			]
		}
    `

	stakingSchema = `
		{
			"type": "record",
			"name": "Staking",
			"namespace": "org.binance.dex.model.avro",
			"fields": [
				{"name": "height", "type": "long"},
				{"name": "timestamp", "type": "long" },
				{"name": "numOfMsgs", "type": "int" },
				{"name": "validators", "type": ["null", {
					"type": "array",
					"items": {
						"type": "record",
						"name": "Validator",
						"namespace": "org.binance.dex.model.avro",
						"fields": [
							{"name": "feeAddr", "type": "string"},
							{"name": "operatorAddr", "type": "string"},
							{"name": "consAddr", "type": ["null","string"], "default": "null"},
							{"name": "jailed", "type": "boolean"},
							{"name": "status", "type": "string"},
							{"name": "tokens", "type": "long"},
							{"name": "delegatorShares", "type": "long"},
							{"name": "description", "type": {
								"type": "record",
								"name": "Description",
								"namespace": "org.binance.dex.model.avro",
								"fields": [
									{"name": "moniker", "type": "string"},
									{"name": "identity", "type": "string"},
									{"name": "website", "type": "string"},
									{"name": "details", "type": "string"}
								]
							}},
							{"name": "bondHeight", "type": "long"},
							{"name": "bondIntraTxCounter", "type": "int"},
							{"name": "commission", "type": {
								"type": "record",
								"name": "Commission",
								"namespace": "org.binance.dex.model.avro",
								"fields": [
									{"name": "rate", "type": "long"},
									{"name": "maxRate", "type": "long"},
									{"name": "maxChangeRate", "type": "long"},
									{"name": "updateTime", "type": "long"}
								]
							}},
							{"name": "distributionAddr", "type": "string"},
							{"name": "sideChainId", "type": "string"},
							{"name": "sideConsAddr", "type": "string"},
							{"name": "sideFeeAddr", "type": "string"}
						]
					}
				}], "default": "null"},
				{"name": "removedValidators", "type": ["null", {
					"type": "map",
					"values": {
						"type": "array",
						"items": {"type": "string"}
					}
				}], "default": null},
				{"name": "delegations", "type": ["null",{
					"type": "map",
					"values": {
						"type": "array",
						"items": {
							"type": "record",
							"name": "Delegation",
							"namespace": "org.binance.dex.model.avro",
							"fields": [
								{"name": "delegator", "type": "string"},
								{"name": "validator", "type": "string"},
								{"name": "shares", "type": "long"}
							]
						}
					}
				}], "default": null},
				{"name": "unBondingDelegations", "type": ["null",{
					"type": "map",
					"values": {
						"type": "array",
						"items": {
							"type": "record",
							"name": "UnBondingDelgation",
							"namespace": "org.binance.dex.model.avro",
							"fields": [
								{"name": "delegator", "type": "string"},
								{"name": "validator", "type": "string"},
								{"name": "creationHeight", "type": "long"},
								{"name": "minTime", "type": "long"},
								{"name": "initialBalance", "type": {
									"type": "record",
									"name": "Coin",
									"namespace": "org.binance.dex.model.avro",
									"fields": [
										{ "name": "denom", "type": "string" },
										{ "name": "amount", "type": "long" }
									]
								}},
								{"name": "balance", "type": "org.binance.dex.model.avro.Coin"}
							]
						}
					}
				}], "default": null},
				{"name": "reDelegations", "type": ["null",{
					"type": "map",
					"values": {
						"type": "array",
						"items": {
							"type": "record",
							"name": "ReDelegation",
							"namespace": "org.binance.dex.model.avro",
							"fields": [
								{"name": "delegator", "type": "string"},
								{"name": "srcValidator", "type": "string"},
								{"name": "dstValidator", "type": "string"},
								{"name": "creationHeight", "type": "long"},
								{"name": "sharesSrc", "type": "long"},
								{"name": "sharesDst", "type": "long"},
								{"name": "initialBalance", "type": "org.binance.dex.model.avro.Coin" },
								{"name": "balance", "type": "org.binance.dex.model.avro.Coin" },
								{"name": "minTime", "type": "long"}
							]
						}
					}
				}], "default": null},
				{"name": "completedUBDs", "type": ["null",{
					"type": "map",
					"values": {
						"type": "array",
						"items": {
							"type": "record",
							"name": "CompletedUnbondingDelegation",
							"namespace": "org.binance.dex.model.avro",
							"fields": [
								{ "name": "validator", "type": "string" },
								{ "name": "delegator", "type": "string" },
								{ "name": "amount", "type": "org.binance.dex.model.avro.Coin"}
							]
						}
					}
				}],  "default": null},
				{"name": "completedREDs", "type": ["null",{
					"type": "map",
					"values": {
						"type": "array",
						"items": {
							"type": "record",
							"name": "CompletedReDelegation",
							"namespace": "org.binance.dex.model.avro",
							"fields": [
								{ "name": "delegator", "type": "string" },
								{ "name": "srcValidator", "type": "string" },
								{ "name": "dstValidator", "type": "string" }
							]
						}
					}
				}],  "default": null},
				{"name": "delegateEvents", "type": ["null", {
					"type": "map",
					"values": {
						"type": "array",
						"items": {
							"type": "record",
							"name": "DelegateEvent",
							"namespace": "org.binance.dex.model.avro",
							"fields": [
								{"name": "delegator", "type": "string"},
								{"name": "validator", "type": "string"},
								{"name": "amount", "type": "org.binance.dex.model.avro.Coin"},
								{"name": "txHash", "type": "string"},
								{"name": "crossStake", "type": "boolean"}
							]
						}
					}
				}], "default": null},
				{"name": "unDelegateEvents", "type": ["null", {
					"type": "map",
					"values": {
						"type": "array",
						"items": {
							"type": "record",
							"name": "UndelegateEvent",
							"namespace": "org.binance.dex.model.avro",
							"fields": [
								{"name": "delegator", "type": "string"},
								{"name": "validator", "type": "string"},
								{"name": "amount", "type": "org.binance.dex.model.avro.Coin"},
								{"name": "txHash", "type": "string"}
							]
						}
					}
				}], "default": null},
				{"name": "reDelegateEvents", "type": ["null", {
					"type": "map",
					"values": {
						"type": "array",
						"items": {
							"type": "record",
							"name": "RedelegateEvent",
							"namespace": "org.binance.dex.model.avro",
							"fields": [
								{"name": "delegator", "type": "string"},
								{"name": "srcValidator", "type": "string"},
								{"name": "dstValidator", "type": "string"},
								{"name": "amount", "type": "org.binance.dex.model.avro.Coin"},
								{"name": "txHash", "type": "string"}
							]
						}
					}
				}], "default": null},
				{"name": "electedValidators", "type": ["null", {
					"type": "map",
					"values": {
						"type": "array",
						"items": "org.binance.dex.model.avro.Validator"
					}
				}], "default": null}
			]
		}
	`

	distributionSchema = `
		{
			"type": "record",
			"name": "Distribution",
			"namespace": "org.binance.dex.model.avro",
			"fields": [
				{ "name": "height", "type": "long" },
				{ "name": "timestamp", "type": "long" },
				{ "name": "numOfMsgs", "type": "int" },
				{ "name": "distributions", "type": {
						"type": "map",
						"values": {
							"type": "array",
							"items": {
								"type": "record",
								"name": "DistributionData",
								"namespace": "org.binance.dex.model.avro",
								"fields": [
									{"name": "validator", "type": "string"},
									{"name": "selfDelegator","type": "string"},
									{"name": "distributeAddr","type": "string"},
									{"name": "valTokens", "type": "long"},
									{"name": "totalReward", "type": "long"},
									{"name": "commission", "type": "long"},
									{"name": "rewards", "type":{
										"type": "array",
										"items": {
											"type": "record",
											"name": "Reward",
											"namespace": "org.binance.dex.model.avro",
											"fields":[
												{"name": "validator", "type": "string"},
												{"name": "delegator", "type": "string"},
												{"name": "delegationTokens", "type": "long"},
												{"name": "reward", "type": "long"}
											]
										}
									}}
								]
							}
						}
					}
				}
			]
		}
	`

	slashingSchema = `
		{
			"type": "record",
			"name": "Slashing",
			"namespace": "org.binance.dex.model.avro",
			"fields": [
				{ "name": "height", "type": "long" },
				{ "name": "timestamp", "type": "long" },
				{ "name": "numOfMsgs", "type": "int" },
				{ "name": "slashData", "type": {
					"type": "map",
					"values": {
						"type": "array",
						"items": {
							"type": "record",
							"name": "SlashData",
							"namespace": "org.binance.dex.model.avro",
							"fields": [
								{"name": "validator", "type": "string"},
								{"name": "infractionType", "type": "int"},
								{"name": "infractionHeight", "type": "long"},
								{"name": "jailUtil", "type": "long"},
								{"name": "slashAmount", "type": "long"},
								{"name": "toFeePool", "type": "long"},
								{"name": "submitter", "type": "string"},
								{"name": "submitterReward", "type": "long"},
								{"name": "validatorsCompensation", "type":{
									"type": "array",
									"items": {
										"type": "record",
										"name": "AllocatedAmt",
										"namespace": "org.binance.dex.model.avro",
										"fields":[
											{"name": "address", "type": "string"},
											{"name": "amount", "type": "long"}
										]
									}
								}}
							]
						}
					}
				}}
			]
		}
	`

	crossTransferSchema = `
        {
            "type": "record",
            "name": "CrossTransfers",
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
                			{ "name": "type", "type": "string" },
							{ "name": "relayerFee", "type": "long" },
                			{ "name": "chainid", "type": "string" },
                            { "name": "from", "type": "string" },
                            { "name": "denom", "type": "string" },
                            { "name": "contract", "type": "string" },
                            { "name": "decimals", "type": "int" },
                            { "name": "to", 
                                  "type": {
                                     "type": "array",
                                    "items": {
                                        "type": "record",
                                        "name": "Receiver",
                                        "namespace": "com.company",
                                        "fields": [
                                            { "name": "addr", "type": "string" },
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
	`

	mirrorSchema = `
        {
            "type": "record",
            "name": "Mirrors",
            "namespace": "com.company",
            "fields": [
                { "name": "height", "type": "long"},
                { "name": "num", "type": "int" },
                { "name": "timestamp", "type": "long" },
                { "name": "mirrors",
                  	"type": {    
                    	"type": "array",
                    	"items": {
                        	"type": "record",
							"name": "Mirror",
                        	"namespace": "com.company",
                        	"fields": [
                            	{ "name": "txHash", "type": "string" },
                				{ "name": "chainId", "type": "string" },
								{ "name": "type", "type": "string" },
								{ "name": "relayerFee", "type": "long" },
								{ "name": "sender", "type": "string" },
								{ "name": "contract", "type": "string" },
								{ "name": "bep20Name", "type": "string" },
								{ "name": "bep20Symbol", "type": "string" },
								{ "name": "bep2Symbol", "type": "string" },
								{ "name": "oldTotalSupply", "type": "long" },
								{ "name": "totalSupply", "type": "long" },
								{ "name": "decimals", "type": "int" },
								{ "name": "fee", "type": "long" }
                        	]
                    	}
                  	}    
                }
            ]
        }
	`

	sideProposalsSchema = `
        {
            "type": "record",
            "name": "SideProposals",
            "namespace": "com.company",
            "fields": [
				{ "name": "height", "type": "long" },
				{ "name": "timestamp", "type": "long" },
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
							{ "name": "chainid", "type": "string" },
							{ "name": "status", "type": "string" }
						]
					}
				   }
				}
			]
        }
	`

	breatheBlockSchema = `
		{
			"type": "record",
			"name": "BreatheBlock",
			"namespace": "org.binance.dex.model.avro",
			"fields": [
				{"name": "height", "type": "long"},
				{"name": "timestamp", "type": "long"}
			]
		}
	`
)
