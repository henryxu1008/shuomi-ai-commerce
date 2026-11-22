package supplymarket

import (
	"context"
	"errors"
	"strings"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
)

var (
	ErrCategoryChanged   = errors.New("market category unavailable or changed")
	ErrCategoryNameTaken = errors.New("market category name already exists")
)

type CategoryOption struct {
	ID       string `json:"id"`
	ParentID string `json:"parentId,omitempty"`
	Name     string `json:"name"`
}

type Category struct {
	CategoryOption
	Enabled  bool  `json:"enabled"`
	Revision int64 `json:"revision"`
}

type CategoryPath struct {
	Parent CategoryOption `json:"parent"`
	Child  CategoryOption `json:"child"`
}

type CategoryAssignment struct {
	Parent Category `json:"parent"`
	Child  Category `json:"child"`
}

func (a CategoryAssignment) Path() CategoryPath {
	return CategoryPath{Parent: a.Parent.CategoryOption, Child: a.Child.CategoryOption}
}

type CategorySelection struct {
	ID             string `json:"id"`
	Revision       int64  `json:"revision"`
	ParentRevision int64  `json:"parentRevision"`
}

func (c *CategorySelection) Valid() bool {
	return c != nil && collection.ValidID(c.ID) && c.Revision > 0 && c.ParentRevision > 0
}

type CategoryChange struct {
	Action string    `json:"action"`
	Before *Category `json:"before,omitempty"`
	After  Category  `json:"after"`
}

type CategoryQuery struct {
	ParentID, After, Kind string
	Limit                 int
}

func (q CategoryQuery) Validate(platform bool) error {
	if q.Limit < 1 || q.Limit > 50 || q.ParentID != "" && !collection.ValidID(q.ParentID) ||
		q.After != "" && !collection.ValidID(q.After) {
		return ErrInvalid
	}
	if platform && q.Kind != "" || !platform && q.Kind != "official" && q.Kind != "selected" {
		return ErrInvalid
	}
	return nil
}

func ValidCategoryName(name string) bool {
	return text(name, 1, 120) && strings.TrimSpace(name) == name &&
		!strings.ContainsAny(name, "\r\n\t")
}

func validateCategoryMutation(i Mutation) error {
	if i.Selection != nil || i.Supply != nil || i.Connection != nil || len(i.FileIDs) != 0 ||
		i.Disclosure || i.CooperationConfirmed || i.Category != nil || i.Note != "" {
		return ErrInvalid
	}
	if i.Action == "category_create" {
		if i.ID != "" || i.ExpectedRevision != 0 || !ValidCategoryName(i.CategoryName) ||
			i.ParentCategoryID != "" && !collection.ValidID(i.ParentCategoryID) {
			return ErrInvalid
		}
		return nil
	}
	if !collection.ValidID(i.ID) || i.ExpectedRevision < 1 || i.ParentCategoryID != "" {
		return ErrInvalid
	}
	if i.Action == "category_rename" && ValidCategoryName(i.CategoryName) ||
		(i.Action == "category_disable" || i.Action == "category_enable") && i.CategoryName == "" {
		return nil
	}
	return ErrInvalid
}

func (s *Service) ListCategories(ctx context.Context, q CategoryQuery, platform bool) (any, error) {
	if ctx == nil || s == nil {
		return nil, ErrForbidden
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	if err := q.Validate(platform); err != nil {
		return nil, err
	}
	scope, actor, err := s.readPrincipal(ctx, platform)
	if err != nil {
		return nil, err
	}
	page, err := s.repository.ListCategories(ctx, q)
	if err != nil {
		return nil, err
	}
	if err = s.verifyReadPrincipal(ctx, scope, actor, platform); err != nil {
		return nil, err
	}
	if platform {
		return page, nil
	}
	items := make([]CategoryOption, 0, len(page.Items))
	for _, category := range page.Items {
		items = append(items, category.CategoryOption)
	}
	return Page[CategoryOption]{Items: items, NextCursor: page.NextCursor, Total: page.Total}, nil
}
