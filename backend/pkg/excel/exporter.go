package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// PromotableProduct 可推广商品数据
type PromotableProduct struct {
	SourceSKU string
	ShopName  string
	Name      string
	Price     float64
}

// ExportPromotableProducts 导出可推广商品到Excel
func ExportPromotableProducts(products []PromotableProduct) (*excelize.File, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sheetName := "可推广商品"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, err
	}
	f.SetActiveSheet(index)

	// 删除默认的Sheet1
	f.DeleteSheet("Sheet1")

	// 设置标题行
	headers := []string{"Source SKU", "店铺名称", "商品名称", "当前价格"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// 设置标题样式
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#CCCCCC"},
			Pattern: 1,
		},
	})
	f.SetCellStyle(sheetName, "A1", "D1", style)

	// 写入数据
	for i, product := range products {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), product.SourceSKU)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), product.ShopName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), product.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), product.Price)
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 40)
	f.SetColWidth(sheetName, "D", "D", 15)

	return f, nil
}

// ExportToBytes 将Excel文件导出为字节数组
func ExportToBytes(f *excelize.File) ([]byte, error) {
	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// CreateLossTemplate 创建亏损商品导入模板
func CreateLossTemplate() (*excelize.File, error) {
	f := excelize.NewFile()

	sheetName := "亏损商品"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, err
	}
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	// 设置标题
	f.SetCellValue(sheetName, "A1", "source_sku")
	f.SetCellValue(sheetName, "B1", "new_price")

	// 设置示例数据
	f.SetCellValue(sheetName, "A2", "SKU001")
	f.SetCellValue(sheetName, "B2", 1500.00)
	f.SetCellValue(sheetName, "A3", "SKU002")
	f.SetCellValue(sheetName, "B3", 2300.00)

	// 设置标题样式
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFFF00"},
			Pattern: 1,
		},
	})
	f.SetCellStyle(sheetName, "A1", "B1", style)

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "B", 15)

	return f, nil
}

// CreateRepriceTemplate 创建改价商品导入模板
func CreateRepriceTemplate() (*excelize.File, error) {
	return CreateLossTemplate() // 格式相同
}
