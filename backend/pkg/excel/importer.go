package excel

import (
	"fmt"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// LossProductRow 亏损商品Excel行数据
type LossProductRow struct {
	SourceSKU string
	NewPrice  float64
}

// RepriceRow 改价商品Excel行数据
type RepriceRow struct {
	SourceSKU string
	NewPrice  float64
}

// ImportLossProducts 导入亏损商品Excel
// Excel格式: 第一列为source_sku, 第二列为new_price
func ImportLossProducts(filePath string) ([]LossProductRow, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	// 获取第一个工作表
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	var result []LossProductRow

	// 跳过标题行（第一行）
	for i, row := range rows {
		if i == 0 {
			continue // 跳过标题
		}

		if len(row) < 2 {
			continue // 跳过不完整的行
		}

		sourceSKU := row[0]
		if sourceSKU == "" {
			continue
		}

		newPrice, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			continue // 跳过无效价格
		}

		result = append(result, LossProductRow{
			SourceSKU: sourceSKU,
			NewPrice:  newPrice,
		})
	}

	return result, nil
}

// ImportRepriceProducts 导入改价商品Excel
// Excel格式: 第一列为source_sku, 第二列为new_price
func ImportRepriceProducts(filePath string) ([]RepriceRow, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	var result []RepriceRow

	for i, row := range rows {
		if i == 0 {
			continue
		}

		if len(row) < 2 {
			continue
		}

		sourceSKU := row[0]
		if sourceSKU == "" {
			continue
		}

		newPrice, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			continue
		}

		result = append(result, RepriceRow{
			SourceSKU: sourceSKU,
			NewPrice:  newPrice,
		})
	}

	return result, nil
}

// ImportFromReader 从Reader导入Excel（用于处理上传的文件）
func ImportLossProductsFromBytes(data []byte) ([]LossProductRow, error) {
	f, err := excelize.OpenReader(bytesReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to open excel data: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	var result []LossProductRow

	for i, row := range rows {
		if i == 0 {
			continue
		}

		if len(row) < 2 {
			continue
		}

		sourceSKU := row[0]
		if sourceSKU == "" {
			continue
		}

		newPrice, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			continue
		}

		result = append(result, LossProductRow{
			SourceSKU: sourceSKU,
			NewPrice:  newPrice,
		})
	}

	return result, nil
}

type bytesReader []byte

func (b bytesReader) Read(p []byte) (n int, err error) {
	return copy(p, b), nil
}
