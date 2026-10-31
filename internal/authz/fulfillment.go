package authz

var fulfillmentPermissions = []string{"workbench.fulfillment.read", "workbench.fulfillment.purchase", "workbench.fulfillment.stock", "workbench.fulfillment.address", "workbench.fulfillment.execute"}

func FulfillmentPermissions() []string { return append([]string(nil), fulfillmentPermissions...) }
