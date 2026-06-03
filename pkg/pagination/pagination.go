package pagination

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type PaginatedResult struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

func NewPagination(page, pageSize int) *Pagination {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return &Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func (p *Pagination) Limit() int {
	return p.PageSize
}

// SetTotal 方法用于设置分页的总数
// 接收一个整数参数 total，表示总记录数
// 该方法会更新 Pagination 结构体中的 Total 字段
func (p *Pagination) SetTotal(total int) {
	p.Total = total
}

// NewPaginatedResult 创建一个新的分页结果结构体
// 参数:
//
//	data: 分页数据，可以是任意类型
//	page: 当前页码
//	pageSize: 每页显示的数据量
//	total: 总数据量
//
// 返回值:
//
//	*PaginatedResult: 返回一个初始化好的分页结果指针
func NewPaginatedResult(data interface{}, page, pageSize, total int) *PaginatedResult {
	// 创建一个新的分页对象
	pagination := NewPagination(page, pageSize)
	// 设置总数据量
	pagination.SetTotal(total)
	// 返回一个新的分页结果，包含数据和分页信息
	return &PaginatedResult{
		Data:       data,
		Pagination: *pagination,
	}
}
