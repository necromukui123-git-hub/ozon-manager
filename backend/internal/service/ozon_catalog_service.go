package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
	"ozon-manager/pkg/ozon"
)

const (
	defaultOzonCatalogPageSize = 20
	maxOzonCatalogPageSize     = 100
	ozonCatalogRefreshThrottle = 120 * time.Second
	ozonCatalogBatchSize       = 200
	ozonCatalogRemotePageSize  = 1000
)

type ozonCatalogCursor struct {
	ListingDate string `json:"listing_date"`
	ID          uint   `json:"id"`
}

type ozonCatalogRefreshState struct {
	Running        bool
	LastStartedAt  *time.Time
	LastFinishedAt *time.Time
	LastError      string
}

type OzonCatalogService struct {
	ozonCatalogRepo *repository.OzonCatalogRepository
	shopRepo        *repository.ShopRepository

	refreshMu      sync.RWMutex
	refreshStateBy map[uint]*ozonCatalogRefreshState
}

func NewOzonCatalogService(
	ozonCatalogRepo *repository.OzonCatalogRepository,
	shopRepo *repository.ShopRepository,
) *OzonCatalogService {
	return &OzonCatalogService{
		ozonCatalogRepo: ozonCatalogRepo,
		shopRepo:        shopRepo,
		refreshStateBy:  make(map[uint]*ozonCatalogRefreshState),
	}
}

func (s *OzonCatalogService) GetCatalog(req *dto.OzonCatalogListRequest) (*dto.OzonCatalogListResponse, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultOzonCatalogPageSize
	}
	if pageSize > maxOzonCatalogPageSize {
		pageSize = maxOzonCatalogPageSize
	}

	visibility := strings.ToUpper(strings.TrimSpace(req.Visibility))
	if visibility == "ALL" {
		visibility = ""
	}
	listingDateSource := strings.TrimSpace(req.ListingDateSource)
	if listingDateSource == "" {
		listingDateSource = "all"
	}
	if listingDateSource != "all" && listingDateSource != "ozon" && listingDateSource != "local_sync" {
		return nil, fmt.Errorf("invalid listing_date_source")
	}

	cursorDate, cursorID, err := decodeOzonCatalogCursor(req.Cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}

	listedFrom, err := parseDateOnly(req.ListedFrom)
	if err != nil {
		return nil, fmt.Errorf("invalid listed_from, expected YYYY-MM-DD")
	}
	listedTo, err := parseDateOnly(req.ListedTo)
	if err != nil {
		return nil, fmt.Errorf("invalid listed_to, expected YYYY-MM-DD")
	}
	if listedFrom != nil && listedTo != nil && listedFrom.After(*listedTo) {
		return nil, fmt.Errorf("listed_from must be <= listed_to")
	}

	offerIDs := parseCSV(req.OfferIDs)
	productIDs, err := parseCSVInt64(req.ProductIDs)
	if err != nil {
		return nil, fmt.Errorf("invalid product_ids")
	}

	items, total, err := s.ozonCatalogRepo.ListWithFilters(repository.OzonCatalogListQuery{
		ShopID:            req.ShopID,
		PageSize:          pageSize,
		CursorListingDate: cursorDate,
		CursorID:          cursorID,
		Visibility:        visibility,
		OfferIDs:          offerIDs,
		ProductIDs:        productIDs,
		ListedFrom:        listedFrom,
		ListedTo:          listedTo,
		ListingDateSource: listingDateSource,
	})
	if err != nil {
		return nil, err
	}

	hasNext := false
	if len(items) > pageSize {
		hasNext = true
		items = items[:pageSize]
	}

	nextCursor := ""
	if hasNext && len(items) > 0 {
		nextCursor = encodeOzonCatalogCursor(items[len(items)-1])
	}

	respItems := make([]dto.OzonCatalogItem, 0, len(items))
	for _, item := range items {
		listingDate := ""
		if item.ListingDate != nil {
			listingDate = item.ListingDate.Format("2006-01-02")
		}
		lastSyncedAt := ""
		if item.LastRemoteSyncedAt != nil {
			lastSyncedAt = item.LastRemoteSyncedAt.Format("2006-01-02 15:04:05")
		}

		respItems = append(respItems, dto.OzonCatalogItem{
			ID:                item.ID,
			OzonProductID:     item.OzonProductID,
			OfferID:           item.OfferID,
			SKU:               item.SKU,
			Name:              item.Name,
			PrimaryImageURL:   item.PrimaryImageURL,
			Price:             item.Price,
			OldPrice:          item.OldPrice,
			MinPrice:          item.MinPrice,
			MarketingPrice:    item.MarketingPrice,
			Currency:          item.Currency,
			Visibility:        item.Visibility,
			Status:            item.Status,
			StockTotal:        item.StockTotal,
			StockFBO:          item.StockFBO,
			StockFBS:          item.StockFBS,
			ListingDate:       listingDate,
			ListingDateSource: item.ListingDateSource,
			LastSyncedAt:      lastSyncedAt,
		})
	}

	lastCacheSyncAt := ""
	latestSyncedAt, err := s.ozonCatalogRepo.GetLatestSyncedAt(req.ShopID)
	if err == nil && latestSyncedAt != nil {
		lastCacheSyncAt = latestSyncedAt.Format("2006-01-02 15:04:05")
	}

	state := s.getRefreshState(req.ShopID)
	return &dto.OzonCatalogListResponse{
		Items:           respItems,
		Total:           total,
		NextCursor:      nextCursor,
		HasNext:         hasNext,
		RefreshStatus:   toRefreshStatusDTO(state),
		LastCacheSyncAt: lastCacheSyncAt,
	}, nil
}

func (s *OzonCatalogService) TriggerRefresh(req *dto.OzonCatalogRefreshRequest) (*dto.OzonCatalogRefreshResponse, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "manual"
	}
	if reason != "manual" && reason != "page_enter" {
		return nil, fmt.Errorf("invalid reason")
	}

	if _, err := s.shopRepo.GetWithCredentials(req.ShopID); err != nil {
		return nil, fmt.Errorf("shop not found")
	}

	now := time.Now()

	s.refreshMu.Lock()
	state, exists := s.refreshStateBy[req.ShopID]
	if !exists {
		state = &ozonCatalogRefreshState{}
		s.refreshStateBy[req.ShopID] = state
	}

	if state.Running {
		resp := toRefreshResponse("running", state)
		s.refreshMu.Unlock()
		return resp, nil
	}

	if !req.Force && reason == "page_enter" && state.LastStartedAt != nil && now.Sub(*state.LastStartedAt) < ozonCatalogRefreshThrottle {
		resp := toRefreshResponse("throttled", state)
		s.refreshMu.Unlock()
		return resp, nil
	}

	state.Running = true
	state.LastStartedAt = &now
	state.LastError = ""
	resp := toRefreshResponse("started", state)
	s.refreshMu.Unlock()

	go s.refreshShopCatalog(req.ShopID)
	return resp, nil
}

func (s *OzonCatalogService) RefreshShopCatalogSync(shopID uint) error {
	if _, err := s.shopRepo.GetWithCredentials(shopID); err != nil {
		return fmt.Errorf("shop not found")
	}

	now := time.Now()
	s.refreshMu.Lock()
	state, exists := s.refreshStateBy[shopID]
	if !exists {
		state = &ozonCatalogRefreshState{}
		s.refreshStateBy[shopID] = state
	}
	if state.Running {
		s.refreshMu.Unlock()
		return fmt.Errorf("catalog refresh already running")
	}
	state.Running = true
	state.LastStartedAt = &now
	state.LastError = ""
	s.refreshMu.Unlock()

	defer func() {
		if recovered := recover(); recovered != nil {
			s.updateRefreshState(shopID, fmt.Errorf("panic: %v", recovered))
		}
	}()

	err := s.syncCatalogFromOzon(shopID)
	s.updateRefreshState(shopID, err)
	return err
}

func (s *OzonCatalogService) refreshShopCatalog(shopID uint) {
	defer func() {
		if recovered := recover(); recovered != nil {
			s.updateRefreshState(shopID, fmt.Errorf("panic: %v", recovered))
		}
	}()

	err := s.syncCatalogFromOzon(shopID)
	s.updateRefreshState(shopID, err)
}

func (s *OzonCatalogService) updateRefreshState(shopID uint, refreshErr error) {
	now := time.Now()

	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	state, exists := s.refreshStateBy[shopID]
	if !exists {
		state = &ozonCatalogRefreshState{}
		s.refreshStateBy[shopID] = state
	}

	state.Running = false
	state.LastFinishedAt = &now
	if refreshErr != nil {
		state.LastError = refreshErr.Error()
	}
}

func (s *OzonCatalogService) syncCatalogFromOzon(shopID uint) error {
	shop, err := s.shopRepo.GetWithCredentials(shopID)
	if err != nil {
		return err
	}
	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	listItems := make([]ozon.ProductListV3Item, 0)
	lastID := ""
	seenCursor := map[string]struct{}{}

	for {
		resp, err := client.GetProductListV3(ozonCatalogRemotePageSize, lastID, "ALL")
		if err != nil {
			return err
		}

		listItems = append(listItems, resp.Result.Items...)
		if len(resp.Result.Items) == 0 {
			break
		}

		nextCursor := strings.TrimSpace(resp.Result.LastID)
		if nextCursor == "" {
			break
		}
		if _, exists := seenCursor[nextCursor]; exists {
			break
		}
		seenCursor[nextCursor] = struct{}{}
		lastID = nextCursor
	}

	now := time.Now()
	syncToken := now.UTC().Format("20060102150405.000000000")

	productIDs := make([]int64, 0, len(listItems))
	catalogByProductID := make(map[int64]*model.OzonProductCatalogItem, len(listItems))
	rawListByProductID := make(map[int64]map[string]interface{}, len(listItems))
	archivedByProductID := make(map[int64]bool, len(listItems))
	for _, item := range listItems {
		if item.ProductID <= 0 {
			continue
		}
		if item.Archived {
			archivedByProductID[item.ProductID] = true
		}
		entry, exists := catalogByProductID[item.ProductID]
		if !exists {
			visibility := resolveCatalogVisibility(strings.TrimSpace(item.Visibility), nil, false, archivedByProductID[item.ProductID])
			entry = &model.OzonProductCatalogItem{
				ShopID:             shopID,
				OzonProductID:      item.ProductID,
				OfferID:            strings.TrimSpace(item.OfferID),
				Visibility:         visibility,
				Status:             normalizeCatalogStatus("", visibility, false),
				ListingDateSource:  "local_sync",
				SyncToken:          syncToken,
				LastRemoteSyncedAt: &now,
			}
			catalogByProductID[item.ProductID] = entry
			productIDs = append(productIDs, item.ProductID)
		} else if entry.OfferID == "" {
			entry.OfferID = strings.TrimSpace(item.OfferID)
		}
		entry.Visibility = resolveCatalogVisibility(entry.Visibility, nil, false, archivedByProductID[item.ProductID])
		rawListByProductID[item.ProductID] = item.Raw
	}

	existingItems, err := s.ozonCatalogRepo.FindExistingByProductIDs(shopID, productIDs)
	if err != nil {
		return err
	}

	for start := 0; start < len(productIDs); start += ozonCatalogBatchSize {
		end := start + ozonCatalogBatchSize
		if end > len(productIDs) {
			end = len(productIDs)
		}
		batchIDs := productIDs[start:end]

		infoResp, err := client.GetProductInfoList(batchIDs, nil)
		if err != nil {
			return err
		}
		stocksByProductID, rawStocksByProductID, err := s.fetchStocksByProductIDs(client, batchIDs)
		if err != nil {
			return err
		}

		infoItems := infoResp.ItemsList()
		rawInfoByProductID := make(map[int64]map[string]interface{}, len(infoItems))
		for _, info := range infoItems {
			productID := resolveCatalogProductID(info.ProductID, info.ID)
			if productID <= 0 {
				continue
			}
			entry, exists := catalogByProductID[productID]
			if !exists {
				entry = &model.OzonProductCatalogItem{
					ShopID:             shopID,
					OzonProductID:      productID,
					Visibility:         resolveCatalogVisibility("", nil, false, archivedByProductID[productID]),
					ListingDateSource:  "local_sync",
					SyncToken:          syncToken,
					LastRemoteSyncedAt: &now,
				}
				catalogByProductID[productID] = entry
				productIDs = append(productIDs, productID)
			}

			entry.Visibility = resolveCatalogVisibility(entry.Visibility, info.Raw, info.Visible, archivedByProductID[productID])
			mergeCatalogInfo(entry, info)
			if stock, exists := stocksByProductID[productID]; exists {
				entry.StockTotal = stock.Total
				entry.StockFBO = stock.FBO
				entry.StockFBS = stock.FBS
			}

			if parsedDate, ok := resolveListingDate(info.Raw); ok {
				dateValue := parsedDate
				entry.ListingDate = &dateValue
				entry.ListingDateSource = "ozon"
			}

			if entry.ListingDate == nil {
				if parsedDate, ok := resolveListingDate(rawListByProductID[productID]); ok {
					dateValue := parsedDate
					entry.ListingDate = &dateValue
					entry.ListingDateSource = "ozon"
				}
			}

			rawInfoByProductID[productID] = info.Raw
		}

		// 库存端点可能返回了详情端点未返回的产品，仍补齐库存。
		for productID, stock := range stocksByProductID {
			entry, exists := catalogByProductID[productID]
			if !exists {
				entry = &model.OzonProductCatalogItem{
					ShopID:             shopID,
					OzonProductID:      productID,
					Visibility:         resolveCatalogVisibility("", nil, false, archivedByProductID[productID]),
					ListingDateSource:  "local_sync",
					SyncToken:          syncToken,
					LastRemoteSyncedAt: &now,
				}
				catalogByProductID[productID] = entry
				productIDs = append(productIDs, productID)
			}
			entry.StockTotal = stock.Total
			entry.StockFBO = stock.FBO
			entry.StockFBS = stock.FBS
		}

		// 合并 payload，便于后续排查数据来源。
		for _, id := range batchIDs {
			entry, exists := catalogByProductID[id]
			if !exists {
				continue
			}

			payload := map[string]interface{}{}
			if listRaw, ok := rawListByProductID[id]; ok && listRaw != nil {
				payload["list"] = listRaw
			}
			if infoRaw, ok := rawInfoByProductID[id]; ok && infoRaw != nil {
				payload["info"] = infoRaw
			}
			if stocksRaw, ok := rawStocksByProductID[id]; ok && stocksRaw != nil {
				payload["stocks"] = stocksRaw
			}

			if len(payload) > 0 {
				payloadBytes, _ := json.Marshal(payload)
				entry.Payload = payloadBytes
			}
		}
	}

	items := make([]model.OzonProductCatalogItem, 0, len(catalogByProductID))
	for productID, item := range catalogByProductID {
		if item.ListingDate == nil {
			if existing, ok := existingItems[productID]; ok {
				if existing.ListingDate != nil {
					dateValue := *existing.ListingDate
					item.ListingDate = &dateValue
					item.ListingDateSource = existing.ListingDateSource
				} else {
					createdAt := existing.CreatedAt
					item.ListingDate = &createdAt
					item.ListingDateSource = "local_sync"
				}
			} else {
				fallback := now
				item.ListingDate = &fallback
				item.ListingDateSource = "local_sync"
			}
		}
		item.Visibility = resolveCatalogVisibility(item.Visibility, nil, false, archivedByProductID[productID])
		if item.Status == "" {
			item.Status = normalizeCatalogStatus("", item.Visibility, false)
		}
		if item.OfferID == "" {
			item.OfferID = strconv.FormatInt(item.OzonProductID, 10)
		}
		item.SyncToken = syncToken
		item.LastRemoteSyncedAt = &now
		items = append(items, *item)
	}

	if err := s.ozonCatalogRepo.UpsertBatch(items); err != nil {
		return err
	}
	return s.ozonCatalogRepo.DeleteStaleBySyncToken(shopID, syncToken)
}

func (s *OzonCatalogService) fetchStocksByProductIDs(client *ozon.Client, productIDs []int64) (map[int64]stockSummary, map[int64]map[string]interface{}, error) {
	stockByProductID := make(map[int64]stockSummary)
	rawByProductID := make(map[int64]map[string]interface{})

	lastID := ""
	seenCursor := map[string]struct{}{}

	for {
		resp, err := client.GetProductStocks(productIDs, nil, ozonCatalogRemotePageSize, lastID)
		if err != nil {
			return nil, nil, err
		}
		for _, stockItem := range resp.Result.Items {
			if stockItem.ProductID <= 0 {
				continue
			}
			stockByProductID[stockItem.ProductID] = summarizeStocks(stockItem)
			rawByProductID[stockItem.ProductID] = stockItem.Raw
		}

		nextCursor := strings.TrimSpace(resp.Result.LastID)
		if nextCursor == "" {
			break
		}
		if _, exists := seenCursor[nextCursor]; exists {
			break
		}
		seenCursor[nextCursor] = struct{}{}
		lastID = nextCursor
	}

	return stockByProductID, rawByProductID, nil
}

func (s *OzonCatalogService) getRefreshState(shopID uint) *ozonCatalogRefreshState {
	s.refreshMu.RLock()
	defer s.refreshMu.RUnlock()

	state, exists := s.refreshStateBy[shopID]
	if !exists {
		return &ozonCatalogRefreshState{}
	}

	copyState := *state
	return &copyState
}

func toRefreshStatusDTO(state *ozonCatalogRefreshState) dto.OzonCatalogRefreshStatus {
	status := dto.OzonCatalogRefreshStatus{
		Running: state.Running,
	}
	if state.LastStartedAt != nil {
		status.LastStartedAt = state.LastStartedAt.Format("2006-01-02 15:04:05")
	}
	if state.LastFinishedAt != nil {
		status.LastFinishedAt = state.LastFinishedAt.Format("2006-01-02 15:04:05")
	}
	status.LastError = strings.TrimSpace(state.LastError)
	return status
}

func toRefreshResponse(status string, state *ozonCatalogRefreshState) *dto.OzonCatalogRefreshResponse {
	resp := &dto.OzonCatalogRefreshResponse{
		Status: status,
	}
	if state == nil {
		return resp
	}
	if state.LastStartedAt != nil {
		resp.StartedAt = state.LastStartedAt.Format("2006-01-02 15:04:05")
	}
	if state.LastFinishedAt != nil {
		resp.LastFinishedAt = state.LastFinishedAt.Format("2006-01-02 15:04:05")
	}
	resp.LastError = strings.TrimSpace(state.LastError)
	return resp
}

func parseCSV(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseCSVInt64(input string) ([]int64, error) {
	parts := parseCSV(input)
	result := make([]int64, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}

func parseDateOnly(input string) (*time.Time, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func encodeOzonCatalogCursor(item model.OzonProductCatalogItem) string {
	if item.ListingDate == nil || item.ID == 0 {
		return ""
	}
	payload, _ := json.Marshal(ozonCatalogCursor{
		ListingDate: item.ListingDate.UTC().Format(time.RFC3339Nano),
		ID:          item.ID,
	})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeOzonCatalogCursor(input string) (*time.Time, uint, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, 0, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, 0, err
	}
	cursor := ozonCatalogCursor{}
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, 0, err
	}
	if cursor.ListingDate == "" || cursor.ID == 0 {
		return nil, 0, fmt.Errorf("invalid cursor payload")
	}
	parsedDate, err := time.Parse(time.RFC3339Nano, cursor.ListingDate)
	if err != nil {
		return nil, 0, err
	}
	return &parsedDate, cursor.ID, nil
}

func parsePrice(input string) float64 {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func resolveCatalogProductID(productID int64, fallbackID int64) int64 {
	if productID > 0 {
		return productID
	}
	return fallbackID
}

func mergeCatalogInfo(target *model.OzonProductCatalogItem, info ozon.ProductInfoListItem) {
	if target == nil {
		return
	}

	if info.ProductID > 0 {
		target.OzonProductID = info.ProductID
	}
	if strings.TrimSpace(info.OfferID) != "" {
		target.OfferID = strings.TrimSpace(info.OfferID)
	}
	if info.SKU > 0 {
		target.SKU = info.SKU
	}
	if strings.TrimSpace(info.Name) != "" {
		target.Name = strings.TrimSpace(info.Name)
	}
	if strings.TrimSpace(info.PrimaryImage) != "" {
		target.PrimaryImageURL = strings.TrimSpace(info.PrimaryImage)
	} else if len(info.Images) > 0 && strings.TrimSpace(info.Images[0]) != "" {
		target.PrimaryImageURL = strings.TrimSpace(info.Images[0])
	}
	if strings.TrimSpace(info.CurrencyCode) != "" {
		target.Currency = strings.TrimSpace(info.CurrencyCode)
	}

	target.Price = parsePrice(info.Price)
	target.OldPrice = parsePrice(info.OldPrice)
	target.MinPrice = parsePrice(info.MinPrice)
	target.MarketingPrice = parsePrice(info.MarketingPrice)
	target.Status = normalizeCatalogStatus(info.Status.State, target.Visibility, info.Visible)
}

type stockSummary struct {
	Total int
	FBO   int
	FBS   int
}

func summarizeStocks(item ozon.ProductStocksItem) stockSummary {
	result := stockSummary{}
	if len(item.Stocks) > 0 {
		for _, stock := range item.Stocks {
			available := stock.Present - stock.Reserved
			if available < 0 {
				available = 0
			}
			result.Total += available

			normalizedType := strings.ToLower(strings.TrimSpace(stock.Type))
			switch {
			case strings.Contains(normalizedType, "fbo"):
				result.FBO += available
			case strings.Contains(normalizedType, "fbs"), strings.Contains(normalizedType, "seller"):
				result.FBS += available
			}
		}
		return result
	}

	stocksValue, exists := item.Raw["stocks"]
	if !exists {
		return result
	}

	switch stocks := stocksValue.(type) {
	case map[string]interface{}:
		present := intFromAny(stocks["present"])
		reserved := intFromAny(stocks["reserved"])
		available := present - reserved
		if available < 0 {
			available = 0
		}
		result.Total = available
	case []interface{}:
		for _, stockItem := range stocks {
			stockMap, ok := stockItem.(map[string]interface{})
			if !ok {
				continue
			}
			available := intFromAny(stockMap["present"]) - intFromAny(stockMap["reserved"])
			if available < 0 {
				available = 0
			}
			result.Total += available
			normalizedType := strings.ToLower(strings.TrimSpace(strFromAny(stockMap["type"])))
			switch {
			case strings.Contains(normalizedType, "fbo"):
				result.FBO += available
			case strings.Contains(normalizedType, "fbs"), strings.Contains(normalizedType, "seller"):
				result.FBS += available
			}
		}
	}

	return result
}

func intFromAny(v interface{}) int {
	switch value := v.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float32:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		i, _ := value.Int64()
		return int(i)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(value))
		return parsed
	default:
		return 0
	}
}

func resolveCatalogVisibility(current string, infoRaw map[string]interface{}, infoVisible bool, archived bool) string {
	if visible, ok := resolveVisibleFromInfoRaw(infoRaw, infoVisible); ok {
		if visible {
			return "VISIBLE"
		}
		return "INVISIBLE"
	}

	normalizedCurrent := strings.ToUpper(strings.TrimSpace(current))
	if normalizedCurrent != "" && normalizedCurrent != "ALL" {
		return normalizedCurrent
	}
	if archived {
		return "ARCHIVED"
	}
	return "ALL"
}

func resolveVisibleFromInfoRaw(raw map[string]interface{}, fallback bool) (bool, bool) {
	if raw == nil {
		return false, false
	}
	value, exists := raw["visible"]
	if !exists {
		return false, false
	}
	if parsed, ok := boolFromAny(value); ok {
		return parsed, true
	}
	return fallback, true
}

func boolFromAny(v interface{}) (bool, bool) {
	switch value := v.(type) {
	case bool:
		return value, true
	case int:
		return value != 0, true
	case int32:
		return value != 0, true
	case int64:
		return value != 0, true
	case float32:
		return value != 0, true
	case float64:
		return value != 0, true
	case json.Number:
		i, err := value.Int64()
		if err != nil {
			return false, false
		}
		return i != 0, true
	case string:
		normalized := strings.ToLower(strings.TrimSpace(value))
		switch normalized {
		case "true", "1", "yes", "y", "on":
			return true, true
		case "false", "0", "no", "n", "off":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func strFromAny(v interface{}) string {
	switch value := v.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func normalizeCatalogStatus(status string, visibility string, visible bool) string {
	normalized := strings.TrimSpace(status)
	if normalized != "" {
		return strings.ToLower(normalized)
	}

	if visible {
		return "visible"
	}

	switch strings.ToUpper(strings.TrimSpace(visibility)) {
	case "INVISIBLE", "DISABLED", "ARCHIVED":
		return "hidden"
	case "VISIBLE", "ALL":
		return "visible"
	default:
		return "unknown"
	}
}

func resolveListingDate(raw map[string]interface{}) (time.Time, bool) {
	if raw == nil {
		return time.Time{}, false
	}

	keys := map[string]struct{}{
		"created_at":   {},
		"createdat":    {},
		"created":      {},
		"create_date":  {},
		"created_date": {},
		"upload_date":  {},
		"uploaded_at":  {},
		"published_at": {},
	}

	if value, ok := findAnyByKeys(raw, keys); ok {
		if parsed, ok := parseAnyTime(value); ok {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func findAnyByKeys(node interface{}, keys map[string]struct{}) (interface{}, bool) {
	switch value := node.(type) {
	case map[string]interface{}:
		for key, nested := range value {
			normalized := strings.ToLower(strings.TrimSpace(key))
			if _, exists := keys[normalized]; exists {
				return nested, true
			}
		}
		for _, nested := range value {
			if found, ok := findAnyByKeys(nested, keys); ok {
				return found, true
			}
		}
	case []interface{}:
		for _, nested := range value {
			if found, ok := findAnyByKeys(nested, keys); ok {
				return found, true
			}
		}
	}
	return nil, false
}

func parseAnyTime(value interface{}) (time.Time, bool) {
	switch v := value.(type) {
	case string:
		return parseTimeString(v)
	case json.Number:
		if unix, err := v.Int64(); err == nil {
			return time.Unix(unix, 0).UTC(), true
		}
	case float64:
		return time.Unix(int64(v), 0).UTC(), true
	case int64:
		return time.Unix(v, 0).UTC(), true
	case int:
		return time.Unix(int64(v), 0).UTC(), true
	}
	return time.Time{}, false
}

func parseTimeString(input string) (time.Time, bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return time.Time{}, false
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}
