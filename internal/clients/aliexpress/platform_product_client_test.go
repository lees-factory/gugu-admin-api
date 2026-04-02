package aliexpress

import "testing"

func TestParseDropshippingResponse_AcceptsNumericProductID(t *testing.T) {
	resp := &PlatformResponse{
		RawBody: `{
			"aliexpress_ds_product_get_response": {
				"result": {
					"ae_item_base_info_dto": {
						"product_id": 1005011915547697,
						"subject": "Sample Product",
						"currency_code": "KRW"
					},
					"ae_item_sku_info_dtos": {
						"ae_item_sku_info_d_t_o": [
							{
								"sku_id": "12000057021854917",
								"id": "5:100014065;14:771#2",
								"sku_attr": "5:100014065;14:771#2",
								"offer_sale_price": "10750",
								"sku_price": "19545",
								"currency_code": "KRW",
								"ae_sku_property_dtos": {
									"ae_sku_property_d_t_o": [
										{
											"sku_property_name": "색상",
											"sku_property_value": "베이지",
											"sku_image": "https://example.com/image.jpg"
										}
									]
								}
							}
						]
					}
				}
			}
		}`,
	}

	apiResp, err := parseDropshippingResponse(resp)
	if err != nil {
		t.Fatalf("parseDropshippingResponse() error = %v", err)
	}
	if apiResp.Result.BaseInfo == nil {
		t.Fatal("expected base info")
	}
	if got := string(apiResp.Result.BaseInfo.ProductID); got != "1005011915547697" {
		t.Fatalf("product_id = %q, want %q", got, "1005011915547697")
	}
	if apiResp.Result.SKUs == nil || len(apiResp.Result.SKUs.Items) != 1 {
		t.Fatalf("expected one sku, got %+v", apiResp.Result.SKUs)
	}
}
