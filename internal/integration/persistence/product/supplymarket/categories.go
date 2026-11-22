package supplymarketpersistence

import (
	"context"
	"errors"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/supplymarket"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const categoryTable = "supply_market_categories"

func readCategory(db *gorm.DB, id string, lock bool) (supplymarket.Category, error) {
	var category supplymarket.Category
	query := db.Table(categoryTable).Where("id = ?", id)
	if lock {
		query = query.Clauses(lockUpdate)
	}
	err := query.Take(&category).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = supplymarket.ErrCategoryChanged
	}
	return category, err
}

// Category paths have immutable parents. Discover the parent first, then lock
// it before the child so that confirmation and directory edits serialize.
func readCategoryAssignment(db *gorm.DB, id string, lock bool) (supplymarket.CategoryAssignment, error) {
	child, err := readCategory(db, id, false)
	if err != nil {
		return supplymarket.CategoryAssignment{}, err
	}
	if child.ParentID == "" {
		return supplymarket.CategoryAssignment{}, supplymarket.ErrCategoryChanged
	}
	parent, err := readCategory(db, child.ParentID, lock)
	if err != nil {
		return supplymarket.CategoryAssignment{}, err
	}
	child, err = readCategory(db, id, lock)
	if err != nil {
		return supplymarket.CategoryAssignment{}, err
	}
	return supplymarket.CategoryAssignment{Parent: parent, Child: child}, nil
}

func checkCategory(db *gorm.DB, pick *supplymarket.CategorySelection) (supplymarket.CategoryPath, error) {
	if !pick.Valid() {
		return supplymarket.CategoryPath{}, supplymarket.ErrInvalid
	}
	a, err := readCategoryAssignment(db, pick.ID, true)
	if err != nil {
		return supplymarket.CategoryPath{}, err
	}
	if !a.Parent.Enabled || !a.Child.Enabled || a.Parent.ParentID != "" ||
		a.Parent.Revision != pick.ParentRevision || a.Child.Revision != pick.Revision {
		return supplymarket.CategoryPath{}, supplymarket.ErrCategoryChanged
	}
	return a.Path(), nil
}

func executeCategory(tx *gorm.DB, c supplymarket.Command) (*supplymarket.CategoryChange, error) {
	if c.PlatformActor == "" {
		return nil, supplymarket.ErrForbidden
	}
	i := c.Input
	change := &supplymarket.CategoryChange{Action: i.Action}
	if i.Action == "category_create" {
		if i.ParentCategoryID != "" {
			parent, err := readCategory(tx, i.ParentCategoryID, true)
			if err != nil {
				return nil, err
			}
			if parent.ParentID != "" {
				return nil, supplymarket.ErrInvalid
			}
		}
		cat := supplymarket.Category{CategoryOption: supplymarket.CategoryOption{
			ID: collection.StableID(c.OperationID, "category"), ParentID: i.ParentCategoryID, Name: i.CategoryName,
		}, Enabled: true, Revision: 1}
		result := tx.Exec("INSERT INTO supply_market_categories(id,parent_id,name,enabled,revision) VALUES(?,nullif(?,'')::uuid,?,true,1) ON CONFLICT DO NOTHING", cat.ID, cat.ParentID, cat.Name)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected != 1 {
			return nil, supplymarket.ErrCategoryNameTaken
		}
		change.After = cat
		return change, nil
	}

	category, err := readCategory(tx, i.ID, false)
	if err != nil {
		return nil, err
	}
	if category.ParentID != "" {
		if _, err = readCategory(tx, category.ParentID, true); err != nil {
			return nil, err
		}
	}
	category, err = readCategory(tx, i.ID, true)
	if err != nil {
		return nil, err
	}
	if category.Revision != i.ExpectedRevision {
		return nil, supplymarket.ErrConflict
	}
	before := category
	switch i.Action {
	case "category_rename":
		category.Name = i.CategoryName
	case "category_disable":
		category.Enabled = false
	case "category_enable":
		category.Enabled = true
	default:
		return nil, supplymarket.ErrInvalid
	}
	category.Revision++
	err = tx.Exec("UPDATE supply_market_categories SET name=?,enabled=?,revision=? WHERE id=?", category.Name, category.Enabled, category.Revision, category.ID).Error
	if errorState(err) == "23505" {
		return nil, supplymarket.ErrCategoryNameTaken
	}
	if err != nil {
		return nil, err
	}
	change.Before, change.After = &before, category
	return change, nil
}

func errorState(err error) string {
	var state interface{ SQLState() string }
	if errors.As(err, &state) {
		return state.SQLState()
	}
	return ""
}

const visibleCategory = `(
 (c.enabled AND (c.parent_id IS NULL OR p.enabled) AND
   (c.parent_id IS NOT NULL OR EXISTS (SELECT 1 FROM supply_market_categories child WHERE child.parent_id=c.id AND child.enabled)))
 OR EXISTS (
  SELECT 1 FROM supply_market_releases rel JOIN supply_market_categories assigned ON assigned.id=rel.category_id
  WHERE rel.active AND rel.channel=? AND (assigned.id=c.id OR assigned.parent_id=c.id)
 ))`

func categoryVisibility(db *gorm.DB, kind string) *gorm.DB {
	query := db.Table(categoryTable + " AS c").Joins("LEFT JOIN " + categoryTable + " AS p ON p.id=c.parent_id")
	if kind != "" {
		query = query.Where(visibleCategory, kind)
	}
	return query
}

func (r *Repository) ListCategories(ctx context.Context, q supplymarket.CategoryQuery) (supplymarket.Page[supplymarket.Category], error) {
	page := supplymarket.Page[supplymarket.Category]{Items: []supplymarket.Category{}}
	if err := q.Validate(q.Kind == ""); err != nil {
		return page, err
	}
	query := categoryVisibility(r.db.WithContext(ctx), q.Kind)
	if q.ParentID == "" {
		query = query.Where("c.parent_id IS NULL")
	} else {
		query = query.Where("c.parent_id = ?", q.ParentID)
	}
	if err := query.Count(&page.Total).Error; err != nil {
		return page, err
	}
	if q.After != "" {
		query = query.Where("c.id > ?", q.After)
	}
	err := query.Select("c.id, c.parent_id, c.name, c.enabled, c.revision").
		Order(clause.OrderByColumn{Column: clause.Column{Table: "c", Name: "id"}}).
		Limit(q.Limit + 1).Scan(&page.Items).Error
	if err != nil {
		return page, err
	}
	if len(page.Items) > q.Limit {
		page.Items = page.Items[:q.Limit]
		page.NextCursor = page.Items[len(page.Items)-1].ID
	}
	return page, nil
}
