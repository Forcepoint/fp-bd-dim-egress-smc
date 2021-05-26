package smc

type ListType = string
type LocalListName = string
type Comment = string

const (
	URLSafelist  LocalListName = "dim_url_safelist"
	URLBlocklist LocalListName = "dim_url_blocklist"
	IPSafelist   LocalListName = "dim_safelist"
	IPBlocklist  LocalListName = "dim_blocklist"

	URLListType       ListType = "elements/url_list_application"
	IPListType        ListType = "elements/ip_list"
	IPAddressListType ListType = "ip_address_list"

	URLBlocklistComment Comment = "URL/Domain Blocklist imported from the Dynamic Intelligence Manager."
	URLSafelistComment  Comment = "URL/Domain Safelist imported from the Dynamic Intelligence Manager."
	IPBlocklistComment  Comment = "IP Blocklist imported from the Dynamic Intelligence Manager."
	IPSafelistComment   Comment = "IP Safelist imported from the Dynamic Intelligence Manager."
)
