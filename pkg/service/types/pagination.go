package types

type Pagination struct {
	Page     uint32 `json:"page"`
	PageSize uint32 `json:"page_size"`
}

const DefaultPageSize = 100
const DefaultPage = 0

func NewDefaultPagination() *Pagination {
	return &Pagination{
		Page:     DefaultPage,
		PageSize: DefaultPageSize,
	}
}

func (p *Pagination) Load(pageNumber uint32, pageSize uint32) {
	p.Page = pageNumber
	if pageSize > 0 {
		p.PageSize = pageSize
	}
}
