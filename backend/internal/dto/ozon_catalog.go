package dto

type OzonCatalogListRequest struct {
	ShopID            uint   `form:"shop_id" binding:"required"`
	Cursor            string `form:"cursor"`
	PageSize          int    `form:"page_size,default=20"`
	Visibility        string `form:"visibility"`
	OfferIDs          string `form:"offer_ids"`           // comma-separated
	ProductIDs        string `form:"product_ids"`         // comma-separated
	ListedFrom        string `form:"listed_from"`         // YYYY-MM-DD
	ListedTo          string `form:"listed_to"`           // YYYY-MM-DD
	ListingDateSource string `form:"listing_date_source"` // all / ozon / local_sync
}

type OzonCatalogRefreshRequest struct {
	ShopID uint   `json:"shop_id" binding:"required"`
	Force  bool   `json:"force"`
	Reason string `json:"reason"` // page_enter / manual
}

type OzonCatalogItem struct {
	ID                uint    `json:"id"`
	OzonProductID     int64   `json:"ozon_product_id"`
	OfferID           string  `json:"offer_id"`
	SKU               int64   `json:"sku"`
	Name              string  `json:"name"`
	PrimaryImageURL   string  `json:"primary_image_url"`
	Price             float64 `json:"price"`
	OldPrice          float64 `json:"old_price"`
	MinPrice          float64 `json:"min_price"`
	MarketingPrice    float64 `json:"marketing_price"`
	Currency          string  `json:"currency"`
	Visibility        string  `json:"visibility"`
	Status            string  `json:"status"`
	StockTotal        int     `json:"stock_total"`
	StockFBO          int     `json:"stock_fbo"`
	StockFBS          int     `json:"stock_fbs"`
	ListingDate       string  `json:"listing_date,omitempty"`
	ListingDateSource string  `json:"listing_date_source"`
	LastSyncedAt      string  `json:"last_synced_at,omitempty"`
}

type OzonCatalogRefreshStatus struct {
	Running        bool   `json:"running"`
	LastStartedAt  string `json:"last_started_at,omitempty"`
	LastFinishedAt string `json:"last_finished_at,omitempty"`
	LastError      string `json:"last_error,omitempty"`
}

type OzonCatalogListResponse struct {
	Items           []OzonCatalogItem        `json:"items"`
	Total           int64                    `json:"total"`
	NextCursor      string                   `json:"next_cursor,omitempty"`
	HasNext         bool                     `json:"has_next"`
	RefreshStatus   OzonCatalogRefreshStatus `json:"refresh_status"`
	LastCacheSyncAt string                   `json:"last_cache_sync_at,omitempty"`
}

type OzonCatalogRefreshResponse struct {
	Status         string `json:"status"` // started / running / throttled
	StartedAt      string `json:"started_at,omitempty"`
	LastFinishedAt string `json:"last_finished_at,omitempty"`
	LastError      string `json:"last_error,omitempty"`
}
